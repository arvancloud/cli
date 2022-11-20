package config

import (
	"errors"
	"os"
	"os/user"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	configFileApiVersion = "v1"
)

// ConfigInfo is a struct to access authorization information and global configurations of arvan cli save based on `arvan login` command.
type ConfigInfo struct {
	// base url to access arvan api server
	server string

	// an api key used to authorize request to arvan api server
	apiKey string

	// path to arvan config file e.g /home/jane/.arvan/config
	configFilePath string

	// path to arvan config directroy e.g /home/jane/.arvan
	homeDir string

	region string
}

// GetServer returns base url to access arvan api server
func (c *ConfigInfo) GetServer() string {
	return strings.Replace(c.server, "arvancloud.com", "arvancloud.ir", -1)
}

// GetApiKey returns an api key used to authorize request to arvan api server
func (c *ConfigInfo) GetApiKey() string {
	return c.apiKey
}

// GetConfigFilePath returns path to arvan config file e.g /home/jane/.arvan/config
func (c *ConfigInfo) GetConfigFilePath() string {
	return c.configFilePath
}

// GetHomeDir returns path to arvan config directroy e.g /home/jane/.arvan
func (c *ConfigInfo) GetHomeDir() string {
	return c.homeDir
}

func (c *ConfigInfo) Initiate(apiKey string, zone Zone) {
	c.server = "https://" + zone.Endpoint
	c.apiKey = apiKey
}

func (c *ConfigInfo) Complete() error {

	if !c.ServerProvided() {
		c.server = serverAddress()
	}

	if !c.HomeDirProvided() {
		c.homeDir, _ = defaultHomeDir()
	}

	if !c.ConfigFileProvided() {
		c.configFilePath = defaultConfigFilePath(c.homeDir)
	}
	return nil
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
	if _, err := os.Stat(c.homeDir); os.IsNotExist(err) {
		err = os.MkdirAll(c.homeDir, os.ModePerm)
		if err != nil {
			return false, err
		}
	}
	file, err := os.Create(c.configFilePath)
	if err != nil {
		return false, err
	}

	defer file.Close()

	configFileStruct := configFile{
		ApiVersion: configFileApiVersion,
		Server:     c.server,
		ApiKey:     c.apiKey,
		Region:     c.region,
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

func (c *ConfigInfo) ServerProvided() bool {
	return len(c.server) > 0
}
func (c *ConfigInfo) HomeDirProvided() bool {
	return len(c.homeDir) > 0
}
func (c *ConfigInfo) ConfigFileProvided() bool {
	return len(c.configFilePath) > 0
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

func serverAddress() string {
	return "https://napi.arvancloud.ir"
}
