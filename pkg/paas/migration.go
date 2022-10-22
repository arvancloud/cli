package paas

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/arvancloud/cli/pkg/api"
	"github.com/arvancloud/cli/pkg/config"
	"github.com/arvancloud/cli/pkg/utl"
	"github.com/olekukonko/tablewriter"
	"k8s.io/client-go/rest"

	"github.com/openshift/oc/pkg/helpers/term"
	"github.com/spf13/cobra"
)

const (
	prepareEndpoint            = "/prepare"
	backupManifestsEndpoint    = "/backup"
	restoreManifestsEndpoint   = "/restore"
	syncImagesEndpoint         = "/sync"
	cloneVolumesEndpoint       = "/clone"
	finalizeEndpoint           = "/finalize"
	migrationServer            = "https://cli.arvan.run"
	checkmark                  = "[\u2713]"
	xmark                      = "[x]"
	redColor                   = "\033[31m"
	greenColor                 = "\033[32m"
	yellowColor                = "\033[33m"
	resetColor                 = "\033[0m"
	bamdad                     = "ba1"
	targetMigrationDestination = "destination"
	targetMigrationSource      = "source"
)

type Migration struct {
	Namespace         string
	SourceRegion      string
	DestinationRegion string
}

type Request struct {
	Namespace string
	Target    string
}

type Response struct {
	Services []Service
	Routes   []Route
	Gateway  string
	Status   int
}

type Service struct {
	Name string
	IP   string
}

type Route struct {
	Name    string
	Address string
	IsFree  bool
}

// NewCmdMigrate returns new cobra commad enables user to migrate projects to another region on arvan servers
func NewCmdMigrate(in io.Reader, out, errout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate namespaces to destination region",
		Long:  loginLong,
		Run: func(c *cobra.Command, args []string) {
			explainOut := term.NewResponsiveWriter(out)
			c.SetOutput(explainOut)

			project, _ := getSelectedProject(in, explainOut)

			currentRegionName := getCurrentRegion()

			if currentRegionName == bamdad {
				log.Printf("migration from region %s is not possible now\nplease first switch your region using command:\n\n \tarvan paas region\n\n", currentRegionName)
				return
			}

			destinationRegion, err := GetZone(bamdad)
			utl.CheckErr(err)

			if currentRegionName == getRegionFromEndpoint(destinationRegion.Endpoint) {
				log.Printf("can not migrate to this region")
				return
			}

			confirmed := migrationConfirm(project, getRegionFromEndpoint(destinationRegion.Endpoint), in, explainOut)
			if !confirmed {
				return
			}

			migration := Migration{
				Namespace:         project,
				SourceRegion:      currentRegionName,
				DestinationRegion: destinationRegion.RegionName,
			}

			migrationSteps(migration)
		},
	}

	return cmd
}

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

func (p projectValidator) validate(input string) (bool, error) {
	intInput, err := strconv.Atoi(input)
	if err != nil || intInput < 1 || intInput > p.upperBound {
		return false, fmt.Errorf("enter a number between '1' and '%d'", p.upperBound)
	}
	return true, nil
}

func getCurrentRegion() string {
	_, err := config.LoadConfigFile()
	utl.CheckErr(err)

	arvanConfig := config.GetConfigInfo()

	return getRegionFromEndpoint(arvanConfig.GetServer())
}

func getRegionFromEndpoint(endpoint string) string {
	currentRegionNameIndex := strings.LastIndex(endpoint, "/")

	return endpoint[currentRegionNameIndex+1:]
}

func sprintProjects(projects []string) string {
	result := ""
	var projectIndex int

	for i := 0; i < len(projects); i++ {
		projectIndex++
		result += fmt.Sprintf("  [%d] %s\n", projectIndex, projects[i])
	}

	return result
}

func migrationConfirm(project, region string, in io.Reader, writer io.Writer) bool {
	explain := fmt.Sprintf("\nYou're about to migrate \"%s\" from region \"%s\" to \"%s\".\n", project, getCurrentRegion(), region)

	_, err := fmt.Fprint(writer, explain)
	if err != nil {
		return false
	}
	inputExplain := fmt.Sprintf(yellowColor+"\nWARNING: This will STOP your applications."+resetColor+"\n\nPlease enter project's name [%s] to proceed: ", project)

	defaultVal := ""

	v := confirmationValidator{project: project}

	value := utl.ReadInput(inputExplain, defaultVal, writer, in, v.confirmationValidate)
	return value == project
}

type confirmationValidator struct {
	project string
}

func (v confirmationValidator) confirmationValidate(input string) (bool, error) {
	if input != v.project {
		return false, fmt.Errorf("please enter project name correctly: \"%s\"", v.project)
	}
	return true, nil
}

func migrationSteps(migration Migration) error {
	/*
		1. Prepare Destination (input: ns, region, target(source/destination)) (failure output: Display error and Return)
		2. Prepare Source (input: ns, region, target(source/destination)) (failure output: SourceFinalize(input: fail status))
		3. Backup Manifests (input: ns, region(source)) (failure output: SourceFinalize(input: fail status), DestinationFinalize(input: fail status))
		4. Restore Manifests (input: ns, region(destination)) (failure output: SourceFinalize(input: fail status), DestinationFinalize(input: fail status))
		5. SyncData including sync images and clone volumes (input: ns) (failure output: SourceFinalize(input: fail status), DestinationFinalize(input: fail status))
		6. Finalize Source (input: status(call on success) - region(source), target(source/destination)) (success output: {[]services[{name,ip}], routes[{name,address,isFree(bool)}], gateway(string)})
		7. Finalize Destination (input: status(call on success) - region(destination), target(source/destination)) (success output: {[]services[{name,ip}], routes[{name,address,isFree(bool)}], gateway(string)})
		Final Output report: compare services name to display old and new ips also for routes
	*/

	sourceRegionFinalizeResponse := &Response{Services: []Service{{Name: "name1", IP: "1.1.1.1"}}, Routes: []Route{{Name: "route1-0", Address: "https://route1-0/", IsFree: true}, {Name: "route1-1", Address: "https://route1-1/", IsFree: false}}, Gateway: "gateway1"}
	destinationbRegionFinalizeResponse := &Response{Services: []Service{{Name: "name2", IP: "2.2.2.2"}}, Routes: []Route{{Name: "route2-0", Address: "https://route2-0/", IsFree: true}, {Name: "route2-1", Address: "https://route2-1/", IsFree: false}}, Gateway: "gateway2"}

	successOutput(sourceRegionFinalizeResponse, destinationbRegionFinalizeResponse)

	return nil
}

func HttpPost(u url.URL, payload interface{}) (*Response, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, migrationServer+prepareEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("User-Agent", rest.DefaultKubernetesUserAgent())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.New("server error. try again later")
	}

	// read body
	defer httpResp.Body.Close()
	responseBody, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	// parse response
	var response Response
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func failureOutput() {
	fmt.Println("failed to migrate")
}

func successOutput(source, destination *Response) {
	fmt.Println("\nYour IPs changed successfully")

	ipTable := tablewriter.NewWriter(os.Stdout)
	ipTable.SetHeader([]string{"Old IPs", "New IPs"})

	for i := 0; i < len(source.Services); i++ {
		ipTable.Append([]string{redColor + source.Services[i].IP + resetColor, greenColor + destination.Services[i].IP + resetColor})
	}

	ipTable.Render()

	freeSourceRoutes := make([]Route, 0)
	freeDestinationRoutes := make([]Route, 0)
	nonfreeSourceRoutes := make([]Route, 0)
	nonfreeDestinationRoutes := make([]Route, 0)

	for i := 0; i < len(source.Routes); i++ {
		if source.Routes[i].IsFree {
			freeSourceRoutes = append(freeSourceRoutes, source.Routes[i])
			freeDestinationRoutes = append(freeDestinationRoutes, destination.Routes[i])
		} else {
			nonfreeSourceRoutes = append(nonfreeSourceRoutes, source.Routes[i])
			nonfreeDestinationRoutes = append(nonfreeDestinationRoutes, destination.Routes[i])
		}
	}

	if len(freeSourceRoutes) > 0 {
		fmt.Println("Your free routes changed successfully:")

		freeRouteTable := tablewriter.NewWriter(os.Stdout)
		freeRouteTable.SetHeader([]string{"old free routes", "new free routes"})

		for i := 0; i < len(freeSourceRoutes); i++ {
			freeRouteTable.Append([]string{redColor + freeSourceRoutes[i].Address + resetColor, greenColor + freeDestinationRoutes[i].Address + resetColor})
		}

		freeRouteTable.Render()
	}

	if len(nonfreeSourceRoutes) > 0 {
		nonFreeRouteTable := tablewriter.NewWriter(os.Stdout)
		nonFreeRouteTable.SetHeader([]string{"non-free routes"})

		for i := 0; i < len(nonfreeSourceRoutes); i++ {
			nonFreeRouteTable.Append([]string{yellowColor + nonfreeDestinationRoutes[i].Address + resetColor})
		}

		nonFreeRouteTable.Render()
	}

	gatewayTable := tablewriter.NewWriter(os.Stdout)
	gatewayTable.SetHeader([]string{"old gateway", "new gateway"})

	fmt.Println("For non-free domains above, please change your gateway in your DNS provider as bellow:")
	gatewayTable.Append([]string{redColor + source.Gateway + resetColor, greenColor + destination.Gateway + resetColor})
	gatewayTable.Render()
}

func GetZone(target string) (*config.Zone, error) {
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
		if zone.Name == target {
			return &activeZones[i], nil
		}
	}

	log.Printf("destination region not found")

	return nil, nil
}
