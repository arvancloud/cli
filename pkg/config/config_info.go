package config

import (
	"errors"
	"os"
	"os/user"

	"gopkg.in/yaml.v2"

	"git.arvan.me/arvan/cli/pkg/api"
)

const (
	configFileApiVersion = "v1"
)

// ConfigInfo is a struct to access authorization information and global configurations of arvan cli save based on `arvan login` command.
type ConfigInfo struct {
	// base url to access arvan api server
	Server string

	// Arvan region
	Region string

	// an api key used to authorize request to arvan api server
	ApiKey string

	// path to arvan config file e.g /home/jane/.arvan/config
	ConfigFilePath string

	// path to arvan config directroy e.g /home/jane/.arvan
	HomeDir string
}

func (c *ConfigInfo) Complete() error {
	if !c.RegionProvided() {
		return errors.New("No region has been set.")
	}

	if !c.ServerProvided() {
		c.Server = serverAddress(c.Region)
	}

	if !c.HomeDirProvided() {
		c.HomeDir, _ = defaultHomeDir()
	}

	if !c.ConfigFileProvided() {
		c.ConfigFilePath = defaultConfigFilePath(c.HomeDir)
	}
	return nil
}

func (c *ConfigInfo) IsAuthorized() (bool, error) {
	if _, err := api.GetUserInfo(c.ApiKey); err != nil {
		return false, err
	}
	return true, nil
}

// SaveConfig save config info to ConfigFilePath
// It requires to have ConfigFilePath and HomeDir
func (c *ConfigInfo) SaveConfig() (bool, error) {
	if !c.ConfigFileProvided() {
		return false, errors.New("no config file provided")
	}
	if !c.HomeDirProvided() {
		return false, errors.New("no home directory provided")
	}
	if _, err := os.Stat(c.HomeDir); os.IsNotExist(err) {
		err = os.MkdirAll(c.HomeDir, os.ModePerm)
		if err != nil {
			return false, err
		}
	}
	file, err := os.Create(c.ConfigFilePath)
	if err != nil {
		return false, err
	}

	defer file.Close()

	configFileStruct := configFile{
		ApiVersion: configFileApiVersion,
		Server:     c.Server,
		Region:     c.Region,
		ApiKey:     c.ApiKey,
	}

	configFileStr, err := yaml.Marshal(&configFileStruct)
	if err != nil {
		return false, err
	}

	_, err = file.Write(configFileStr)
	if err != nil {
		return false, err
	}
	return true, nil
}
func (c *ConfigInfo) RegionProvided() bool {
	return len(c.Region) > 0
}
func (c *ConfigInfo) ServerProvided() bool {
	return len(c.Server) > 0
}
func (c *ConfigInfo) HomeDirProvided() bool {
	return len(c.HomeDir) > 0
}
func (c *ConfigInfo) ConfigFileProvided() bool {
	return len(c.ConfigFilePath) > 0
}

func defaultHomeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir + "/.arvan", nil
}

func defaultConfigFilePath(homeDir string) string {
	return homeDir + "/config"
}

// #TODO implement default server address
func serverAddress(region string) string {
	if len(region) > 0 {
		return "https://console.paas.arvaniaas.com:8443"
	}
	return "https://console.paas.arvaniaas.com:8443"
}
