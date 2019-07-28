package login

import (
	"bufio"
	"io"
	"fmt"
	"strings"
	"errors"

	"github.com/openshift/origin/pkg/cmd/util/term"
	"github.com/spf13/cobra"

	"git.arvan.me/arvan/cli/pkg/api"
	"git.arvan.me/arvan/cli/pkg/util"
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
			util.CheckErr(err)

			config.LoadConfigFile()
			arvanConfig := config.GetConfigInfo()

			reader := bufio.NewReader(in)
			if len(arvanConfig.GetApiKey()) > 0 {
				fmt.Fprintf(explainOut, "Enter arvan API token (%s): ", arvanConfig.GetApiKey())
			} else {
				fmt.Fprintf(explainOut, "Enter arvan API token: ")
			}
			apiKey, err := reader.ReadString('\n')
			util.CheckErr(err)

			apiKey = strings.TrimSpace(apiKey)
			if len(apiKey) == 0 {
				apiKey = arvanConfig.GetApiKey()
			}
			arvanConfig.Initiate(apiKey, region)

			util.CheckErr(arvanConfig.Complete())

			// _, err = arvanConfig.IsAuthorized()
			// util.CheckErr(err)

			fmt.Fprintf(explainOut, "Logged in successfully!\n")

			_, err = arvanConfig.SaveConfig()
			util.CheckErr(err)
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


// getSelectedRegion #TODO implement getSelectedRegion
func getSelectedRegion(in io.Reader, writer io.Writer) (string, error) {
	regions, err := api.GetRegions()
	if err != nil {
		return "", err
	}
	if len(regions) < 1 {
		return "", errors.New("Invalid region info.")
	}
	fmt.Fprintf(writer, "Select arvan region:\n")
	printRegions(regions, writer)
	if len(regions) == 1 {
		fmt.Fprintf(writer, "Region Number[1]: 1\n")
		return regions[0], nil
	}

	region := ""
	for {
		fmt.Fprintf(writer, "Region Number:")
		var i int
    	_, err := fmt.Fscan(in, &i)
		if err != nil || i<1 || i > len(regions) {
			fmt.Fprintf(writer, "Error: Enter a number between '1' and '%d'\n", len(regions))
		} else {
			region = regions[i-1]
			break
		}
	}

	return region, nil
}

func printRegions(regions []string, writer io.Writer) {
	for i := 0; i < len(regions); i++ {
		fmt.Fprintf(writer, "  [%d] %s\n", i+1 , regions[i])
	}
}
