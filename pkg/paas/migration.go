package paas

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/arvancloud/cli/pkg/api"
	"github.com/arvancloud/cli/pkg/config"
	"github.com/arvancloud/cli/pkg/utl"

	"github.com/gosuri/uilive"
	"github.com/olekukonko/tablewriter"
	"github.com/openshift/oc/pkg/helpers/term"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
)

const (
	migrationEndpoint = "/paas/v1/%s/migrate"
	redColor          = "\033[31m"
	greenColor        = "\033[32m"
	yellowColor       = "\033[33m"
	resetColor        = "\033[0m"
	bamdad            = "ba1"
	interval          = 2
)

type State string

const (
	Pending   State = "pending"
	Running   State = "running"
	Completed State = "completed"
	Failed    State = "failed"
)

type Request struct {
	Namespace   string `json:"namespace"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type Service struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type Domain struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	IsFree bool   `json:"is_free"`
}

type ZoneInfo struct {
	Services []Service `json:"services"`
	Domains  []Domain  `json:"domains"`
	Gateway  string    `json:"gateway"`
}

type StepData struct {
	Detail      string   `json:"detail"`
	Source      ZoneInfo `json:"source"`
	Destination ZoneInfo `json:"destination"`
}

type Step struct {
	Order int
	Step  string
	Title string
	State string
	Data  StepData
}

type ProgressResponse struct {
	State       State  `json:"state"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Namespace   string `json:"namespace"`
	Message     string `json:"message"`
	StatusCode  int    `json:"StatusCode"`
	Steps       []Step `json:"steps"`
}

// NewCmdMigrate returns new cobra commad enables user to migrate namespaces to another region on arvan servers.
func NewCmdMigrate(in io.Reader, out, errout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate namespaces to destination region",
		Long:  loginLong,
		Run: func(c *cobra.Command, args []string) {
			explainOut := term.NewResponsiveWriter(out)
			c.SetOutput(explainOut)

			currentRegionName := getCurrentRegion()

			request := Request{
				Source: currentRegionName,
			}

			response, err := httpGet(fmt.Sprintf(migrationEndpoint, request.Source))
			if err != nil {
				failureOutput(err.Error())
				return
			}

			if response.StatusCode == http.StatusBadRequest {
				failureOutput(response.Message)

				return
			}

			if response.StatusCode == http.StatusOK {
				if response.State == Completed || response.State == Failed {
					fmt.Println("\nLast migration report is as bellow:")
					migrate(request)
					reMigrationConfirmed := reMigrationConfirm(in, explainOut)
					if !reMigrationConfirmed {
						return
					}
				}
			}

			if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusOK {
				project, err := getSelectedProject(in, explainOut)
				if err != nil {
					failureOutput(err.Error())

					return
				}

				destinationRegion, err := getZoneByName(bamdad)
				utl.CheckErr(err)

				if currentRegionName == getRegionFromEndpoint(destinationRegion.Endpoint) {
					log.Printf("can not migrate to this region")
					return
				}

				confirmed := migrationConfirm(project, getRegionFromEndpoint(destinationRegion.Endpoint), in, explainOut)
				if !confirmed {
					return
				}

				request.Namespace = project
				request.Destination = fmt.Sprintf("%s-%s", destinationRegion.RegionName, destinationRegion.Name)

			}

			if response.StatusCode != http.StatusFound {
				err := httpPost(fmt.Sprintf(migrationEndpoint, request.Source), request)
				if err != nil {
					failureOutput(err.Error())
					return
				}
			}

			err = migrate(request)
			if err != nil {
				failureOutput(err.Error())
			}
		},
	}

	return cmd
}

func reMigrationConfirm(in io.Reader, writer io.Writer) bool {
	inputExplain := "Do you want to run a new migration?[y/N]: "

	defaultVal := "N"

	value := utl.ReadInput(inputExplain, defaultVal, writer, in, confirmationValidate)
	return value == "y"
}

func confirmationValidate(input string) (bool, error) {
	if input != "y" && input != "N" {
		return false, fmt.Errorf("enter a valid answer 'y' for \"yes\" or 'N' for \"no\"")
	}
	return true, nil
}

// getSelectedProject gets intending namespace to migrate.
func getSelectedProject(in io.Reader, writer io.Writer) (string, error) {
	projects, err := projectList()

	if err != nil {
		return "", err
	}

	if len(projects) < 1 {
		return "", errors.New("no project to migrate")
	}

	explain := "Select project:\n"
	explain += sprintProjects(projects)

	_, err = fmt.Fprint(writer, explain)
	if err != nil {
		return "", err
	}
	inputExplain := "Project Number[1]: "

	defaultVal := "1"

	validator := projectValidator{len(projects)}

	projectIndex, err := strconv.Atoi(utl.ReadInput(inputExplain, defaultVal, writer, in, validator.validate))
	if err != nil {
		return "", err
	}

	return projects[projectIndex-1], nil
}

type projectValidator struct {
	upperBound int
}

// validate makes sure the inserted namespace is correct.
func (p projectValidator) validate(input string) (bool, error) {
	intInput, err := strconv.Atoi(input)
	if err != nil || intInput < 1 || intInput > p.upperBound {
		return false, fmt.Errorf("enter a number between '1' and '%d'", p.upperBound)
	}
	return true, nil
}

// getCurrentRegion returns users current region, fetched from config, in string.
func getCurrentRegion() string {
	_, err := config.LoadConfigFile()
	utl.CheckErr(err)

	arvanConfig := config.GetConfigInfo()

	return getRegionFromEndpoint(arvanConfig.GetServer())
}

// getRegionFromEndpoint parses endpoint to return region name.
func getRegionFromEndpoint(endpoint string) string {
	currentRegionNameIndex := strings.LastIndex(endpoint, "/")

	return endpoint[currentRegionNameIndex+1:]
}

// sprintProjects displays projects to select in lines.
func sprintProjects(projects []string) string {
	result := ""
	var projectIndex int

	for i := 0; i < len(projects); i++ {
		projectIndex++
		result += fmt.Sprintf("  [%d] %s\n", projectIndex, projects[i])
	}

	return result
}

// migrationConfirm gets confirmation of proceeding namespace migration by asking user to enter namespace's name.
func migrationConfirm(project, region string, in io.Reader, writer io.Writer) bool {
	explain := fmt.Sprintf("\nYou're about to migrate \"%s\" from region \"%s\" to \"%s\".\n", project, getCurrentRegion(), region)

	_, err := fmt.Fprint(writer, explain)
	if err != nil {
		return false
	}
	inputExplain := fmt.Sprintf(yellowColor+"\nWARNING:\nThis will STOP applications during migration process. Your data would still be safe and available in source region. Migration is running in the background and may take a while. You can optionally detach(Ctrl+C) for now and continue monitoring the process after using 'arvan paas migrate'."+resetColor+"\n\nPlease enter project's name [%s] to proceed: ", project)

	defaultVal := ""

	v := confirmationValidator{project: project}

	value := utl.ReadInput(inputExplain, defaultVal, writer, in, v.confirmationValidate)
	return value == project
}

type confirmationValidator struct {
	project string
}

// confirmationValidate makes sure that user enters namespace correctly.
func (v confirmationValidator) confirmationValidate(input string) (bool, error) {
	if input != v.project {
		return false, fmt.Errorf("please enter project name correctly: \"%s\"", v.project)
	}
	return true, nil
}

// migrate sends migration request and displays response.
func migrate(request Request) error {
	// init writer to update lines
	uiliveWriter := uilive.New()
	uiliveWriter.Start()

	// init writer to display lines in column
	tabWriter := new(tabwriter.Writer)
	tabWriter.Init(uiliveWriter, 0, 8, 0, '\t', 0)

	stopChannel := make(chan bool, 1)

	doEvery(interval*time.Second, stopChannel, func() {
		response, err := httpGet(fmt.Sprintf(migrationEndpoint, request.Source))
		if err != nil {
			failureOutput(err.Error())
			stopChannel <- true
			return
		}

		sprintResponse(*response, tabWriter)

		if response.State == Completed {
			stopChannel <- true
			tabWriter.Flush()
			uiliveWriter.Stop()

			successOutput(response.Steps[len(response.Steps)-1].Data)
		}

		if response.State == Failed {
			stopChannel <- true
			tabWriter.Flush()
			uiliveWriter.Stop()

			failureOutput(response.Steps[len(response.Steps)-1].Data.Detail)
		}
	})

	return nil
}

// doEvery runs given function in periods of 'd' and stops using stopChannel.
func doEvery(d time.Duration, stopChannel chan bool, f func()) {
	ticker := time.NewTicker(d)

	for {
		f()
		select {
		case <-ticker.C:
			continue
		case <-stopChannel:
			ticker.Stop()
			return
		}
	}
}

// sprintResponse displays steps of migration.
func sprintResponse(response ProgressResponse, w io.Writer) error {
	responseStr := fmt.Sprintln("")
	for _, s := range response.Steps {
		responseStr += fmt.Sprintf("\t%d-%s   \t\t\t%s\t%s\n", s.Order, s.Title, strings.Title(s.State), s.Data.Detail)
	}

	fmt.Fprintf(w, "%s", responseStr)
	time.Sleep(time.Millisecond * 100)

	return nil
}

// httpPost sends POST request to inserted url.
func httpPost(endpoint string, payload interface{}) error {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	arvanConfig := config.GetConfigInfo()
	arvanURL, err := url.Parse(arvanConfig.GetServer())
	if err != nil {
		return fmt.Errorf("invalid config")
	}

	httpReq, err := http.NewRequest(http.MethodPost, arvanURL.Scheme+"://"+arvanURL.Host+endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	apikey := arvanConfig.GetApiKey()
	if apikey != "" {
		httpReq.Header.Add("Authorization", apikey)
	}

	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("User-Agent", rest.DefaultKubernetesUserAgent())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}

	// read body
	defer httpResp.Body.Close()
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return err
	}

	// parse response
	var response ProgressResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return err
	}

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusFound {
		return errors.New(response.Message)
	}

	return nil
}

// httpGet sends GET request to inserted url.
func httpGet(endpoint string) (*ProgressResponse, error) {
	arvanConfig := config.GetConfigInfo()
	arvanURL, err := url.Parse(arvanConfig.GetServer())
	if err != nil {
		return nil, fmt.Errorf("invalid config")
	}

	httpReq, err := http.NewRequest(http.MethodGet, arvanURL.Scheme+"://"+arvanURL.Host+endpoint, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, err
	}
	apikey := arvanConfig.GetApiKey()
	if apikey != "" {
		httpReq.Header.Add("Authorization", apikey)
	}

	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("User-Agent", rest.DefaultKubernetesUserAgent())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// read body
	defer httpResp.Body.Close()
	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	// parse response
	var response ProgressResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, err
	}

	response.StatusCode = httpResp.StatusCode

	return &response, nil
}

// failureOutput displays failure output.
func failureOutput(message string) {
	fmt.Println(redColor + "\nFAILED: " + message + resetColor)
}

// successOutput displays success output.
func successOutput(data StepData) {
	fmt.Println("\nNamespaces successfully migrated!")

	if len(data.Source.Services) > 0 {
		ipTable := tablewriter.NewWriter(os.Stdout)
		ipTable.SetHeader([]string{"Old IPs", "New IPs"})

		for i := 0; i < len(data.Source.Services); i++ {
			ipTable.Append([]string{redColor + data.Source.Services[i].IP + resetColor, greenColor + data.Destination.Services[i].IP + resetColor})
		}

		ipTable.Render()
	}

	freeSourceDomains := make([]Domain, 0)
	freeDestinationDomains := make([]Domain, 0)
	nonfreeSourceDomains := make([]Domain, 0)
	nonfreeDestinationDomains := make([]Domain, 0)

	for i := 0; i < len(data.Source.Domains); i++ {
		if data.Source.Domains[i].IsFree {
			freeSourceDomains = append(freeSourceDomains, data.Source.Domains[i])
			freeDestinationDomains = append(freeDestinationDomains, data.Destination.Domains[i])
		} else {
			nonfreeSourceDomains = append(nonfreeSourceDomains, data.Source.Domains[i])
			nonfreeDestinationDomains = append(nonfreeDestinationDomains, data.Destination.Domains[i])
		}
	}

	if len(freeSourceDomains) > 0 {
		fmt.Println("Free domains changed successfully:")

		freeDomainTable := tablewriter.NewWriter(os.Stdout)
		freeDomainTable.SetHeader([]string{"old free domains", "new free domains"})

		for i := 0; i < len(freeSourceDomains); i++ {
			freeDomainTable.Append([]string{redColor + freeSourceDomains[i].Host + resetColor, greenColor + freeDestinationDomains[i].Host + resetColor})
		}

		freeDomainTable.Render()
	}

	if len(nonfreeSourceDomains) > 0 {
		nonFreeDomainTable := tablewriter.NewWriter(os.Stdout)
		nonFreeDomainTable.SetHeader([]string{"non-free domains"})

		for i := 0; i < len(nonfreeSourceDomains); i++ {
			nonFreeDomainTable.Append([]string{yellowColor + nonfreeDestinationDomains[i].Host + resetColor})
		}

		nonFreeDomainTable.Render()
	}

	if len(freeSourceDomains) > 0 {
		gatewayTable := tablewriter.NewWriter(os.Stdout)
		gatewayTable.SetHeader([]string{"old gateway", "new gateway"})

		fmt.Println("For non-free domains above, please change gateway in DNS provider as bellow:")
		gatewayTable.Append([]string{redColor + data.Source.Gateway + resetColor, greenColor + data.Destination.Gateway + resetColor})
		gatewayTable.Render()
	}
}

// getZoneByName gets zone from list of active zones giving it's name.
func getZoneByName(name string) (*config.Zone, error) {
	regions, err := api.GetZones()
	if err != nil {
		return nil, err
	}
	if len(regions.Zones) < 1 {
		return nil, errors.New("invalid region info")
	}

	activeZones, _ := getActiveAndInactiveZones(regions.Zones)

	if len(activeZones) < 1 {
		return nil, errors.New("no active region available")
	}

	for i, zone := range activeZones {
		if zone.Name == name {
			return &activeZones[i], nil
		}
	}

	log.Printf("destination region not found")

	return nil, nil
}
