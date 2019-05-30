package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/openshift/origin/pkg/cmd/util/term"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"git.arvan.me/arvan/cli/pkg/api"
	"git.arvan.me/arvan/cli/pkg/util"
)

type configFile struct {
	ApiVersion string `yaml:"apiVersion"`
	Server     string `yaml:"server"`
	Region     string `yaml:"region"`
	ApiKey     string `yaml:"apiKey"`
}

var instance *ConfigInfo
var once sync.Once
var (
	loginLong = `
    Log in to Arvan API and save login for subsequent use

    First-time users of the client should run this command to connect to a Arvan API,
    establish an authenticated session, and save connection to the configuration file.`
)

// GetConfigInfo return ConfigInfo instance including the information about server url and authorization info
func GetConfigInfo() *ConfigInfo {
	once.Do(func() {
		defaultHome, _ := defaultHomeDir()
		instance = &ConfigInfo{
			homeDir:        defaultHome,
			configFilePath: defaultConfigFilePath(defaultHome),
		}
	})
	return instance
}

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

			region, err := getSelectedRegion(explainOut)
			util.CheckErr(err)

			LoadConfigFile()
			arvanConfig := GetConfigInfo()

			reader := bufio.NewReader(in)
			if len(arvanConfig.apiKey) > 0 {
				fmt.Fprintf(explainOut, "Enter arvan API token (%s): ", arvanConfig.apiKey)
			} else {
				fmt.Fprintf(explainOut, "Enter arvan API token: ")
			}
			apiKey, err := reader.ReadString('\n')
			util.CheckErr(err)

			apiKey = strings.TrimSpace(apiKey)
			if len(apiKey) > 0 {
				arvanConfig.apiKey = apiKey
			}
			arvanConfig.region = region

			util.CheckErr(arvanConfig.Complete())

			_, err = arvanConfig.IsAuthorized()
			util.CheckErr(err)

			fmt.Fprintf(explainOut, "Logged in successfully!\n")

			_, err = arvanConfig.SaveConfig()
			util.CheckErr(err)
		},
	}

	return cmd
}

// LoadConfigFile load config info from ConfigFilePath into ConfigInfo which is accessible using GetConfigInfo()
func LoadConfigFile() (bool, error) {
	arvanConfig := GetConfigInfo()

	if arvanConfig.ConfigFileProvided() {
		data, err := ioutil.ReadFile(arvanConfig.configFilePath)
		if err != nil {
			return false, err
		}
		configFileStruct := configFile{}
		err = yaml.Unmarshal(data, &configFileStruct)
		if err != nil {
			return false, err
		}
		arvanConfig.apiKey = configFileStruct.ApiKey
		arvanConfig.region = configFileStruct.Region
		arvanConfig.server = configFileStruct.Server
		return true, nil
	}

	return false, errors.New("no config file provided")
}

// getSelectedRegion #TODO implement getSelectedRegion
func getSelectedRegion(writer io.Writer) (string, error) {
	paasRegions, err := api.GetRegions()
	if err != nil {
		return "", err
	}

	fmt.Fprintf(writer, "Select arvan region:\n  [1] ir-thr-mn1\n  [2] ir-thr-at1\nRegion Number: 1\n")
	region := paasRegions[0]

	return region, nil
}
