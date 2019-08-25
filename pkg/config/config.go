package config

import (
	"errors"
	"io/ioutil"
	"sync"

	"gopkg.in/yaml.v2"
)

type configFile struct {
	ApiVersion string `yaml:"apiVersion"`
	Server     string `yaml:"server"`
	Region     string `yaml:"region"`
	ApiKey     string `yaml:"apikey"`
}

var instance *ConfigInfo
var once sync.Once

// GetConfigInfo return ConfigInfo instance including the information about server url and authorization info
func GetConfigInfo() *ConfigInfo {
	once.Do(func() {
		instance = &ConfigInfo{}
		instance.Complete()
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
		arvanConfig.region = configFileStruct.Region
		arvanConfig.server = configFileStruct.Server
		return true, nil
	}

	return false, errors.New("no config file provided")
}
