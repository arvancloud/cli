package paas

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/arvancloud/cli/pkg/config"
	"github.com/arvancloud/cli/pkg/utl"

	"github.com/openshift/oc/pkg/helpers/term"
	"github.com/spf13/cobra"
)

const (
	checkmark = "[\u2713]"
	at1       = "ir-thr-ba1"
)

// NewCmdMigrate returns new cobra commad enables user to migrate projects to another region on arvan servers
func NewCmdMigrate(in io.Reader, out, errout io.Writer) *cobra.Command {
	// Main command
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate namespaces to destination region",
		Long:  loginLong,
		Run: func(c *cobra.Command, args []string) {
			/* TODO
			check current region(if asiatech migration is acceptable)
			select projects/namespaces to migrate
			select destination region2
			migration proccess
			output(if failed why and what to do/ if success message)
			*/
			explainOut := term.NewResponsiveWriter(out)
			c.SetOutput(explainOut)

			project, _ := getSelectedProject(in, explainOut)
			fmt.Println("migrating project:", project)

			currentRegionName := getCurrentRegion()

			if currentRegionName == at1 {
				log.Printf("migration from region %s is not possible now\nplease first switch your region using command:\n\n \tarvan paas region\n\n", currentRegionName)
				return
			}

			region, err := getSelectedRegion(in, explainOut)
			utl.CheckErr(err)

			if currentRegionName == getRegionFromEndpoint(region.Endpoint) {
				log.Printf("can not migrate to this region")
				return
			}

			fmt.Println(getRegionFromEndpoint(region.Endpoint))

			migrationSteps()

			fmt.Fprintf(explainOut, "All namespaces migrated successfully!\n")
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
		return false, fmt.Errorf("enter a number between '1' and '%d'\n", p.upperBound)
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

func migrationSteps() {
	fmt.Print("- Getting all resources")
	time.Sleep(2 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Step 2")
	time.Sleep(2 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Transfering volumes to new region")
	time.Sleep(3 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Checking memories with old region")
	time.Sleep(1 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Transfering manifests to new region")
	time.Sleep(1 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Checking get resources result")
	time.Sleep(2 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Connecting volumes to pods and starting services")
	time.Sleep(1 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Creating checklist of all services")
	time.Sleep(1 * time.Second)
	fmt.Println(" ", checkmark)
}
