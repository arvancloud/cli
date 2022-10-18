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
	"text/tabwriter"
	"time"

	"github.com/arvancloud/cli/pkg/config"
	"github.com/arvancloud/cli/pkg/utl"
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
	resetColor                 = "\033[0m"
	bamdad                     = "ir-thr-ba1"
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
			fmt.Println("migrating project:", project)

			currentRegionName := getCurrentRegion()

			if currentRegionName == bamdad {
				log.Printf("migration from region %s is not possible now\nplease first switch your region using command:\n\n \tarvan paas region\n\n", currentRegionName)
				return
			}

			destinationRegion, err := getSelectedRegion(in, explainOut)
			utl.CheckErr(err)

			if currentRegionName == getRegionFromEndpoint(destinationRegion.Endpoint) {
				log.Printf("can not migrate to this region")
				return
			}

			fmt.Println(getRegionFromEndpoint(destinationRegion.Endpoint))

			confirmed := migrationConfirm(project, destinationRegion.Name, in, explainOut)
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
	explain := fmt.Sprintf("You're about to migrate \"%s\" to region \"%s\" :\n", project, region)

	_, err := fmt.Fprint(writer, explain)
	if err != nil {
		return false
	}
	inputExplain := "Are you sure?[Y/n]: "

	defaultVal := "Y"

	value := utl.ReadInput(inputExplain, defaultVal, writer, in, confirmationValidate)
	return value == "Y"
}

func confirmationValidate(input string) (bool, error) {
	if input != "Y" && input != "n" {
		return false, fmt.Errorf("enter a valid answer 'Y' for \"yes\" or 'n' for \"no\"")
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
	_, err := Prepare(migration.Namespace, migration.DestinationRegion, targetMigrationDestination)
	if err != nil {
		failureOutput()

		return err
	}

	time.Sleep(70 * time.Millisecond)

	prepareSourceResponse, err := Prepare(migration.Namespace, migration.SourceRegion, targetMigrationSource)
	if err != nil {
		Finilize(prepareSourceResponse.Status, migration.SourceRegion, targetMigrationSource)
		failureOutput()

		return err
	}
	time.Sleep(200 * time.Millisecond)
	backupManifestsResponse, err := BackupManifests(migration.Namespace, migration.SourceRegion)
	if err != nil {
		Finilize(backupManifestsResponse.Status, migration.SourceRegion, targetMigrationSource)
		Finilize(backupManifestsResponse.Status, migration.DestinationRegion, targetMigrationDestination)
		failureOutput()

		return err
	}
	time.Sleep(500 * time.Millisecond)
	restoreManifestsResponse, err := RestoreManifests(migration.Namespace, migration.DestinationRegion)
	if err != nil {
		Finilize(restoreManifestsResponse.Status, migration.SourceRegion, targetMigrationSource)
		Finilize(restoreManifestsResponse.Status, migration.DestinationRegion, targetMigrationDestination)
		failureOutput()

		return err
	}
	time.Sleep(1000 * time.Microsecond)
	syncImagesResponse, err := SyncImages(migration.Namespace)
	if err != nil {
		Finilize(syncImagesResponse.Status, migration.SourceRegion, targetMigrationSource)
		Finilize(syncImagesResponse.Status, migration.DestinationRegion, targetMigrationDestination)
		failureOutput()

		return err
	}
	time.Sleep(70 * time.Millisecond)
	cloneVolumesResponse, err := CloneVolumes(migration.Namespace)
	if err != nil {
		Finilize(cloneVolumesResponse.Status, migration.SourceRegion, targetMigrationSource)
		Finilize(cloneVolumesResponse.Status, migration.DestinationRegion, targetMigrationDestination)
		failureOutput()

		return err
	}
	time.Sleep(3000 * time.Millisecond)
	sourceRegionFinalizeResponse, err := Finilize(200, migration.SourceRegion, targetMigrationSource)
	if err != nil {
		failureOutput()

		return err
	}
	time.Sleep(700 * time.Millisecond)
	destinationbRegionFinalizeResponse, err := Finilize(200, migration.DestinationRegion, targetMigrationDestination)
	if err != nil {
		failureOutput()

		return err
	}

	successOutput(sourceRegionFinalizeResponse, destinationbRegionFinalizeResponse)

	return nil
}

func Prepare(ns, region, target string) (*Response, error) {
	fmt.Print("- Preparing ", region)

	completeURL, err := url.Parse(migrationServer + prepareEndpoint)
	if err != nil {
		return nil, err
	}

	payload := Request{
		Namespace: ns,
		Target:    target,
	}

	return HttpPost(*completeURL, payload)
}

func SyncImages(ns string) (*Response, error) {
	fmt.Println(" ", checkmark)
	fmt.Print("- Syncing Images")

	completeURL, err := url.Parse(migrationServer + syncImagesEndpoint)
	if err != nil {
		return nil, err
	}

	return HttpPost(*completeURL, nil)
}

func CloneVolumes(ns string) (*Response, error) {
	fmt.Println(" ", checkmark)
	fmt.Print("- Cloning Volumes")

	completeURL, err := url.Parse(migrationServer + cloneVolumesEndpoint)
	if err != nil {
		return nil, err
	}

	return HttpPost(*completeURL, nil)
}

func BackupManifests(ns, region string) (*Response, error) {
	fmt.Println(" ", checkmark)
	fmt.Print("- Saving Manifests Backups")

	completeURL, err := url.Parse(migrationServer + backupManifestsEndpoint)
	if err != nil {
		return nil, err
	}

	return HttpPost(*completeURL, nil)
}

func RestoreManifests(ns, region string) (*Response, error) {
	fmt.Println(" ", checkmark)
	fmt.Print("- Restoring Manifests")

	completeURL, err := url.Parse(migrationServer + restoreManifestsEndpoint)
	if err != nil {
		return nil, err
	}

	return HttpPost(*completeURL, nil)
}

func Finilize(status int, region, target string) (*Response, error) {
	fmt.Println(" ", checkmark)
	fmt.Print("- Finilize ", region)

	_, err := url.Parse(migrationServer + finalizeEndpoint)
	if err != nil {
		return nil, err
	}

	completeURL, err := url.Parse(migrationServer + finalizeEndpoint)
	if err != nil {
		return nil, err
	}

	payload := Request{
		Target: target,
	}

	return HttpPost(*completeURL, payload)
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
	fmt.Println(" ", xmark)
	fmt.Println("failed to migrate")
}

func successOutput(source, destination *Response) {
	fmt.Println(" ", checkmark)

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 12, 1, '\t', tabwriter.AlignRight)

	defer w.Flush()

	for i := 0; i < len(source.Services); i++ {
		fmt.Fprintln(w, "\t", redColor, source.Services[i].IP, "\t", resetColor, "-->", "\t", greenColor, destination.Services[i].IP)
	}

	for i := 0; i < len(source.Routes); i++ {
		fmt.Fprintln(w, "\t", redColor, source.Routes[i].Name, "\t", resetColor, "-->", "\t", greenColor, destination.Routes[i].Name)
	}

	fmt.Fprintln(w, "\t", redColor, source.Gateway, "\t", resetColor, "-->", "\t", greenColor, destination.Gateway)
}
