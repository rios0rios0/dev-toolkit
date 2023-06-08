package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	LanguagesConfig map[string]LanguageConfig `yaml:"languages"`
	CacheDir        string                    `yaml:"cache_dir"`
	Dependencies    []Dependency              `yaml:"dependencies"`
}

type LanguageConfig struct {
	Start           string   `yaml:"start"`
	Stop            string   `yaml:"stop"`
	Build           string   `yaml:"build"`
	Extensions      []string `yaml:"extensions"`
	SpecialPatterns []string `yaml:"special_patterns"`
}

type Dependency struct {
	URL  string `yaml:"url"`
	Path string `yaml:"path"`
}

var Globalconf *GlobalConfig

func InitConfig(configPath string) error {
	var err error
	Globalconf, err = readConfig(configPath)
	return err
}

func MergeConfigs(config1, config2 *GlobalConfig) *GlobalConfig {
	merged := &GlobalConfig{
		LanguagesConfig: make(map[string]LanguageConfig),
	}

	for key, value := range config1.LanguagesConfig {
		merged.LanguagesConfig[key] = value
	}

	for key, value := range config2.LanguagesConfig {
		merged.LanguagesConfig[key] = value
	}

	merged.CacheDir = config2.CacheDir

	merged.Dependencies = config2.Dependencies

	return merged
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

func loadLocalConfig(cwd string) (*GlobalConfig, error) {
	localConfigPath := filepath.Join(cwd, "vinit.yaml")

	// return the global config if the local config file does not exist
	_, err := os.Stat(localConfigPath)
	if os.IsNotExist(err) {
		return MergeConfigs(Globalconf, &GlobalConfig{}), nil
	}

	localConfig, err := readConfig(localConfigPath)
	if err != nil {
		return nil, err
	}
	return MergeConfigs(Globalconf, localConfig), nil
}
