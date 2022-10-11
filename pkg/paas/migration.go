package paas

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/arvancloud/cli/pkg/config"
	"github.com/arvancloud/cli/pkg/utl"
	"github.com/openshift/oc/pkg/helpers/term"
	"github.com/spf13/cobra"
)

const (
	checkmark = "[\u2713]"
)

// NewCmdMigrate returns new cobra commad enables user to migrate objects to another region on arvan servers
func NewCmdMigrate(in io.Reader, out, errout io.Writer) *cobra.Command {
	// Main command
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate to region",
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

			currentRegionName := getCurrentRegion()

			if currentRegionName == "ir-thr-ba1" {
				log.Printf("migration from region %s is not possible now\nplease first switch your region using command:\n\n \tarvan paas region\n\n", currentRegionName)
				return
			}
			log.Println(currentRegionName)

			region, err := getSelectedRegion(in, explainOut)
			utl.CheckErr(err)

			if currentRegionName == getRegionFromEndpoint(region.Endpoint) {
				log.Printf("can not migrate to this region")
				return
			}

			fmt.Println(getRegionFromEndpoint(region.Endpoint))

			project, _ := getSelectedProject(in, explainOut)
			fmt.Println("migrating project: ", project)
			migrationSteps()

			fmt.Fprintf(explainOut, "All objects migrated successfully!\n")
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

	return utl.ReadInput(inputExplain, defaultVal, writer, in, projectValidator), nil
}

func projectValidator(input string) (bool, error) {
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
		log.Println(projects[i])

		result += fmt.Sprintf("  [%d] %s\n", projectIndex, projects[i])
	}
	return result
}

func migrationSteps() {
	fmt.Print("- Step1")
	time.Sleep(2 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Step2")
	time.Sleep(3 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Step3")
	time.Sleep(1 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Step4")
	time.Sleep(1 * time.Second)
	fmt.Println(" ", checkmark)
	fmt.Print("- Step5")
	time.Sleep(2 * time.Second)
	fmt.Println(" ", checkmark)
}
