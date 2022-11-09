package config

import (
	"errors"
	"io/ioutil"
	"sync"

	"github.com/arvancloud/cli/pkg/utl"

	"gopkg.in/yaml.v2"
)

const (
	regionsEndpoint = "/paas/v1/regions/"
)

type configFile struct {
	ApiVersion string `yaml:"apiVersion"`
	Server     string `yaml:"server"`
	ApiKey     string `yaml:"apikey"`
	Region     string `yaml:"region"`
}

var instance *ConfigInfo
var once sync.Once

// GetConfigInfo return ConfigInfo instance including the information about server url and authorization info
func GetConfigInfo() *ConfigInfo {
	once.Do(func() {
		instance = &ConfigInfo{}
		_ = instance.Complete()
	})
	return instance
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
		arvanConfig.server = configFileStruct.Server

		if configFileStruct.Region != "" {
			arvanConfig.server = configFileStruct.Server + regionsEndpoint + configFileStruct.Region
			_, err = arvanConfig.SaveConfig()
			utl.CheckErr(err)
		}

		return true, nil
	}

	return false, errors.New("no config file provided")
}
