package main

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type GlobalConfig struct {
	ProjectsConfig map[string]LanguageConfig `yaml:"languages"`
}

type LanguageConfig struct {
	Start           string   `yaml:"start"`
	Build           string   `yaml:"build"`
	Extensions      []string `yaml:"extensions"`
	SpecialPatterns []string `yaml:"special_patterns"`
}

var Globalconf *GlobalConfig

func InitConfig(configPath string) error {
	var err error
	Globalconf, err = readConfig(configPath)
	return err
}

func readConfig(configPath string) (*GlobalConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var globalConfig GlobalConfig
	err = yaml.Unmarshal(data, &globalConfig)
	if err != nil {
		return nil, err
	}

	return &globalConfig, nil
}
