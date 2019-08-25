package login

import (
	"io"
	"fmt"
	"errors"
	"strconv"
	"regexp"

	"github.com/openshift/origin/pkg/cmd/util/term"
	"github.com/spf13/cobra"

	"git.arvan.me/arvan/cli/pkg/api"
	"git.arvan.me/arvan/cli/pkg/utl"
	"git.arvan.me/arvan/cli/pkg/config"
)

var (
	loginLong = `
    Log in to Arvan API and save login for subsequent use

    First-time users of the client should run this command to connect to a Arvan API,
    establish an authenticated session, and save connection to the configuration file.`
)

// NewCmdLogin returns new cobra commad enables user to login to arvan servers
func NewCmdLogin(in io.Reader, out, errout io.Writer) *cobra.Command {
	// Main command
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Arvan server",
		Long:  loginLong,
		Run: func(c *cobra.Command, args []string) {
			explainOut := term.NewResponsiveWriter(out)
			c.SetOutput(explainOut)

			region, err := getSelectedRegion(in, explainOut)
			utl.CheckErr(err)

			apiKey := getApiKey(in, explainOut)

			config.LoadConfigFile()
			arvanConfig := config.GetConfigInfo()

			arvanConfig.Initiate(apiKey, region)

			utl.CheckErr(arvanConfig.Complete())

			// _, err = arvanConfig.IsAuthorized()
			// utl.CheckErr(err)

			fmt.Fprintf(explainOut, "Logged in successfully!\n")

			_, err = arvanConfig.SaveConfig()
			utl.CheckErr(err)
		},
	}

	return cmd
}

// func (c *ConfigInfo) IsAuthorized() (bool, error) {
// 	if _, err := api.GetUserInfo(c.apiKey); err != nil {
// 		return false, err
// 	}
// 	return true, nil
// }

func getApiKey(in io.Reader, writer io.Writer) string {
	arvanConfig := config.GetConfigInfo()

	inputExplain := "Enter arvan API token: "
	defaultVal := arvanConfig.GetApiKey()

	if len(defaultVal) > 0 {
		inputExplain = fmt.Sprintf("%s[%s]: ",inputExplain , defaultVal)
	}

	return utl.ReadInput(inputExplain, defaultVal, writer, in, apiKeyValidator)
}

func apiKeyValidator(input string) (bool, error) {
	var validApiKey = regexp.MustCompile(`^Apikey [a-z0-9\-]+$$`)
	if (!validApiKey.MatchString(input)){
		return false, errors.New("API token should be in format: 'Apikey xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'")
	}
	return true, nil
}


// getSelectedRegion #TODO implement getSelectedRegion
func getSelectedRegion(in io.Reader, writer io.Writer) (string, error) {
	regions, err := api.GetRegions()
	if err != nil {
		return "", err
	}
	if len(regions) < 1 {
		return "", errors.New("invalid region info")
	}

	activeRegions, inactiveRegions := getActiveAndInactiveRegins(regions)

	if len(activeRegions) < 1 {
		return "", errors.New("no active region available")
	}

	explain := "Select arvan region:\n"
	explain += sprintRegions(activeRegions, inactiveRegions)

	fmt.Fprintf(writer, explain)

	inputExplain := "Region Number[1]: "

	defaultVal := "1"

	if len(activeRegions) == 1 {
		fmt.Fprintf(writer, inputExplain + "1\n")
		return activeRegions[0].Name, nil
	}

	validator := regionValidator{len(activeRegions)}

	regionIndex := utl.ReadInput(inputExplain, defaultVal, writer, in, validator.validate)
	intIndex, _ := strconv.Atoi(regionIndex)

	return activeRegions[intIndex-1].Name, nil
}

type regionValidator struct {
	upperBound int
}

func (r regionValidator) validate(input string) (bool, error) {
	intInput, err := strconv.Atoi(input)
	if err != nil || intInput < 1 || intInput > r.upperBound {
		return false, fmt.Errorf("enter a number between '1' and '%d'\n", r.upperBound)
	} 
	return true, nil
}

func sprintRegions(activeRegions, inactiveRegions []api.Region) string {
	result := ""
	for i := 0; i < len(activeRegions); i++ {
		result += fmt.Sprintf("  [%d] %s\n", i+1 , activeRegions[i].Name)
	}
	for i := 0; i < len(inactiveRegions); i++ {
		result += fmt.Sprintf("  [-] %s (inactive)\n", inactiveRegions[i].Name)
	}
	return result
}

func getActiveAndInactiveRegins(regions []api.Region) ([]api.Region, []api.Region){
	var activeRegions, inactiveRegions []api.Region
	for i := 0; i < len(regions); i++ {
		if regions[i].Active {
			activeRegions = append(activeRegions, regions[i])
		} else {
			inactiveRegions = append(inactiveRegions, regions[i])
		}
	}
	return activeRegions, inactiveRegions
}
