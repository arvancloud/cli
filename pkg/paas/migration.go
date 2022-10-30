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
	migrationEndpoint = "/paas/v1/migrate"
	redColor          = "\033[31m"
	greenColor        = "\033[32m"
	yellowColor       = "\033[33m"
	resetColor        = "\033[0m"
	bamdad            = "ba1"
	interval          = 10
)

type State string

const (
	Pending   State = "Pending"
	Doing     State = "Doing"
	Completed State = "Completed"
	Failed    State = "Failed"
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

type Route struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	IsFree bool   `json:"is_free"`
}

type ZoneInfo struct {
	Services []Service `json:"services"`
	Routes   []Route   `json:"routes"`
	Gateway  string    `json:"gateway"`
}

type StepData struct {
	Time     time.Time
	Message  string
	Percent  int
	Response Response
}

type Step struct {
	Order int
	Step  string
	Title string
	State string
	Data  StepData
}

type ProgressResponse struct {
	State       State
	Source      string
	Destination string
	Namespace   string
	Steps       []Step
}

type Response struct {
	Source      ZoneInfo `json:"source"`
	Destination ZoneInfo `json:"destination"`
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

			project, err := getSelectedProject(in, explainOut)
			if err != nil {
				failureOutput(err.Error())
				return
			}

			currentRegionName := getCurrentRegion()

			if currentRegionName == bamdad {
				log.Printf("migration from region %s is not possible now\nplease first switch your region using command:\n\n \tarvan paas region\n\n", currentRegionName)
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

			requset := Request{
				Namespace:   project,
				Source:      currentRegionName,
				Destination: fmt.Sprintf("%s-%s", destinationRegion.RegionName, destinationRegion.Name),
			}

			err = migrate(requset)
			if err != nil {
				log.Println(err)
			}
		},
	}

	return cmd
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
	inputExplain := fmt.Sprintf(yellowColor+"\nWARNING: This will STOP your applications during migration process."+resetColor+"\n\nPlease enter project's name [%s] to proceed: ", project)

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
	postResponse, err := httpPost(migrationEndpoint, request)
	if err != nil {
		failureOutput(err.Error())
		return err
	}

	if postResponse.StatusCode != http.StatusCreated {
		failureOutput(fmt.Sprint(postResponse.StatusCode))
		return errors.New(fmt.Sprint(postResponse.StatusCode))
	}

	// init writer to update lines
	uiliveWriter := uilive.New()
	uiliveWriter.Start()

	// init writer to display lines in column
	tabWriter := new(tabwriter.Writer)
	tabWriter.Init(uiliveWriter, 0, 8, 0, '\t', 0)

	stopChannel := make(chan bool, 1)

	doEvery(interval*time.Second, stopChannel, func() {
		response, _ := httpGet(migrationEndpoint)

		sprintResponse(*response, tabWriter)

		if response.State == Completed {
			close(stopChannel)
			tabWriter.Flush()
			uiliveWriter.Stop()

			successOutput(&response.Steps[len(response.Steps)-1].Data.Response)
		}

		if response.State == Failed {
			failureOutput(response.Steps[len(response.Steps)-1].Data.Message)
		}
	})

	return nil
}

// doEvery runs given function in periods of 'd' and stops using stopChannel.
func doEvery(d time.Duration, stopChannel chan bool, f func()) {
	ticker := time.NewTicker(d)

	for {
		select {
		case <-ticker.C:
			f()
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
		responseStr += fmt.Sprintf("\t%s...\t\t%s\t%s\n", s.Title, s.State, s.Data.Message)
	}

	fmt.Fprintf(w, "%s", responseStr)
	time.Sleep(time.Millisecond * 100)

	return nil
}

// httpPost sends POST request to inserted url.
func httpPost(endpoint string, payload interface{}) (*http.Response, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	arvanConfig := config.GetConfigInfo()
	arvanURL, err := url.Parse(arvanConfig.GetServer())
	if err != nil {
		return nil, fmt.Errorf("invalid config")
	}

	httpReq, err := http.NewRequest(http.MethodPost, arvanURL.Scheme+"://"+arvanURL.Host+endpoint, bytes.NewBuffer(requestBody))
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

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.New("server error. try again later")
	}

	return httpResp, nil
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

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.New("server error. try again later")
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
	return &response, nil
}

// failureOutput displays failure output.
func failureOutput(message string) {
	fmt.Println("failed:", message)
}

// successOutput displays success output.
func successOutput(response *Response) {
	fmt.Println("\nYour IPs changed successfully")

	ipTable := tablewriter.NewWriter(os.Stdout)
	ipTable.SetHeader([]string{"Old IPs", "New IPs"})

	for i := 0; i < len(response.Source.Services); i++ {
		ipTable.Append([]string{redColor + response.Source.Services[i].IP + resetColor, greenColor + response.Destination.Services[i].IP + resetColor})
	}

	ipTable.Render()

	freeSourceRoutes := make([]Route, 0)
	freeDestinationRoutes := make([]Route, 0)
	nonfreeSourceRoutes := make([]Route, 0)
	nonfreeDestinationRoutes := make([]Route, 0)

	for i := 0; i < len(response.Source.Routes); i++ {
		if response.Source.Routes[i].IsFree {
			freeSourceRoutes = append(freeSourceRoutes, response.Source.Routes[i])
			freeDestinationRoutes = append(freeDestinationRoutes, response.Destination.Routes[i])
		} else {
			nonfreeSourceRoutes = append(nonfreeSourceRoutes, response.Source.Routes[i])
			nonfreeDestinationRoutes = append(nonfreeDestinationRoutes, response.Destination.Routes[i])
		}
	}

	if len(freeSourceRoutes) > 0 {
		fmt.Println("Your free routes changed successfully:")

		freeRouteTable := tablewriter.NewWriter(os.Stdout)
		freeRouteTable.SetHeader([]string{"old free routes", "new free routes"})

		for i := 0; i < len(freeSourceRoutes); i++ {
			freeRouteTable.Append([]string{redColor + freeSourceRoutes[i].Host + resetColor, greenColor + freeDestinationRoutes[i].Host + resetColor})
		}

		freeRouteTable.Render()
	}

	if len(nonfreeSourceRoutes) > 0 {
		nonFreeRouteTable := tablewriter.NewWriter(os.Stdout)
		nonFreeRouteTable.SetHeader([]string{"non-free routes"})

		for i := 0; i < len(nonfreeSourceRoutes); i++ {
			nonFreeRouteTable.Append([]string{yellowColor + nonfreeDestinationRoutes[i].Host + resetColor})
		}

		nonFreeRouteTable.Render()
	}

	gatewayTable := tablewriter.NewWriter(os.Stdout)
	gatewayTable.SetHeader([]string{"old gateway", "new gateway"})

	fmt.Println("For non-free domains above, please change your gateway in your DNS provider as bellow:")
	gatewayTable.Append([]string{redColor + response.Source.Gateway + resetColor, greenColor + response.Destination.Gateway + resetColor})
	gatewayTable.Render()
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
