package paas

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/openshift/oc/pkg/helpers/term"
	"github.com/spf13/cobra"

	"github.com/arvancloud/cli/pkg/api"
	"github.com/arvancloud/cli/pkg/config"
	"github.com/arvancloud/cli/pkg/utl"
)

var (
	loginLong = `
    Log in to Arvan API and save login for subsequent use

    First-time users of the client should run this command to connect to a Arvan API,
    establish an authenticated session, and save connection to the configuration file.`

	SwitchRegionLong = `
	Switch region to connect to different zones.
	`
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

			_, _ = config.LoadConfigFile()

			arvanConfig := config.GetConfigInfo()

			tempApiKey := arvanConfig.GetApiKey()

			arvanConfig.Initiate(apiKey, *region)

			utl.CheckErr(arvanConfig.Complete())

			_, err = arvanConfig.SaveConfig()
			utl.CheckErr(err)

			isAuthorized, authErr := isAuthorized(apiKey)
			if !isAuthorized {
				arvanConfig.Initiate(tempApiKey, *region)
				_, err = arvanConfig.SaveConfig()
				utl.CheckErr(err)
			}
			utl.CheckErr(authErr)

			if c != nil {
				err = prepareConfig(c)
			}
			utl.CheckErr(err)
			fmt.Fprintf(explainOut, "Valid Authorization credentials. Logged in successfully!\n")
		},
	}

	return cmd
}

// NewCmdLogin returns new cobra commad enables user to switch region
func NewCmdSwitchRegion(in io.Reader, out, errout io.Writer) *cobra.Command {
	// Main command
	cmd := &cobra.Command{
		Use:   "region",
		Short: "Switch region",
		Long:  SwitchRegionLong,
		Run: func(c *cobra.Command, args []string) {
			explainOut := term.NewResponsiveWriter(out)
			c.SetOutput(explainOut)

			region, err := getSelectedRegion(in, explainOut)
			utl.CheckErr(err)

			_, _ = config.LoadConfigFile()

			arvanConfig := config.GetConfigInfo()

			arvanConfig.Initiate(arvanConfig.GetApiKey(), *region)

			utl.CheckErr(arvanConfig.Complete())

			_, err = arvanConfig.SaveConfig()
			utl.CheckErr(err)

			err = prepareConfigSwtichRegion(c)
			utl.CheckErr(err)

			fmt.Fprintf(explainOut, "Region Switched successfully.\n")
		},
	}

	return cmd
}

func isAuthorized(apiKey string) (bool, error) {
	if _, err := api.GetUserInfo(apiKey); err != nil {
		return false, err
	}
	return true, nil
}

func getApiKey(in io.Reader, writer io.Writer) string {
	arvanConfig := config.GetConfigInfo()

	inputExplain := "Enter arvan API token: "
	defaultVal := arvanConfig.GetApiKey()

	if len(defaultVal) > 0 {
		inputExplain = fmt.Sprintf("%s[%s]: ", inputExplain, defaultVal)
	}

	return utl.ReadInput(inputExplain, defaultVal, writer, in, apiKeyValidator)
}

func apiKeyValidator(input string) (bool, error) {
	var validApiKey = regexp.MustCompile(`^(A|a)pikey [a-z0-9\-]+$$`)
	if !validApiKey.MatchString(input) {
		return false, errors.New("API token should be in format: 'Apikey xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'")
	}
	return true, nil
}

// getSelectedRegion #TODO implement getSelectedRegion
func getSelectedRegion(in io.Reader, writer io.Writer) (*config.Zone, error) {
	regions, err := api.GetZones()
	if err != nil {
		return nil, err
	}
	if len(regions.Zones) < 1 {
		return nil, errors.New("invalid region info")
	}

	upZones, downZones := getUpAndDownZones(regions.Zones)

	if len(upZones) < 1 {
		return nil, errors.New("no active region available")
	}

	explain := "Select arvan region:\n"
	explain += sprintRegions(upZones, downZones)

	_, err = fmt.Fprint(writer, explain)
	if err != nil {
		return nil, err
	}
	inputExplain := "Region Number[1]: "

	defaultVal := "1"

	if len(upZones) == 1 {
		fmt.Fprintf(writer, inputExplain+"1\n")
		return &upZones[0], nil
	}

	validator := regionValidator{len(upZones)}

	regionIndex := utl.ReadInput(inputExplain, defaultVal, writer, in, validator.validate)
	intIndex, _ := strconv.Atoi(regionIndex)

	return &upZones[intIndex-1], nil
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

func sprintRegions(activeZones, inactiveRegions []config.Zone) string {
	result := ""
	var activeZoneIndex int
	for i := 0; i < len(activeZones); i++ {
		if activeZones[i].Default {
			activeZoneIndex++

			if activeZones[i].Release == "STABLE" {
				result += fmt.Sprintf("  [%d] %s-%s\n", activeZoneIndex, activeZones[i].RegionName, activeZones[i].Name)
			} else {
				result += fmt.Sprintf("  [%d] %s-%s(%s)\n", activeZoneIndex, activeZones[i].RegionName, activeZones[i].Name, activeZones[i].Release)
			}

		}
	}
	for i := 0; i < len(activeZones); i++ {
		if !activeZones[i].Default {
			activeZoneIndex++
			if activeZones[i].Release == "STABLE" {
				result += fmt.Sprintf("  [%d] %s-%s\n", activeZoneIndex, activeZones[i].RegionName, activeZones[i].Name)
			} else {
				result += fmt.Sprintf("  [%d] %s-%s(%s)\n", activeZoneIndex, activeZones[i].RegionName, activeZones[i].Name, activeZones[i].Release)
			}
		}
	}
	for i := 0; i < len(inactiveRegions); i++ {
		result += fmt.Sprintf("  [-] %s-%s (down)\n", inactiveRegions[i].RegionName, inactiveRegions[i].Name)
	}
	return result
}

func getUpAndDownZones(zones []config.Zone) ([]config.Zone, []config.Zone) {
	var activeZones, inactiveZones []config.Zone
	for i := 0; i < len(zones); i++ {
		if zones[i].Status == "UP" {
			activeZones = append(activeZones, zones[i])
		} else {
			inactiveZones = append(inactiveZones, zones[i])
		}
	}
	return activeZones, inactiveZones
}
