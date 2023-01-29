package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os/exec"
)

var log = logrus.New()

// FileReader is a struct that will handle the file reading
type FileReader struct {
	filePath string
}

// YamlData holds the yaml content
type YamlData struct {
	Up   []string `yaml:"up"`
	Down []string `yaml:"down"`
}

// ReadYAML reads the YAML slice of code from the file
func (f *FileReader) ReadYAML() (YamlData, error) {
	data, err := ioutil.ReadFile(f.filePath)
	if err != nil {
		log.WithError(err).Error("Error reading file")
		return YamlData{}, err
	}
	var yamlData YamlData
	err = yaml.Unmarshal(data, &yamlData)
	if err != nil {
		log.WithError(err).Error("Error parsing YAML")
		return YamlData{}, err
	}
	return yamlData, nil
}

// ExecCommand executes a command in the operating system
func ExecCommand(cmd string) error {
	command := exec.Command("sh", "-c", cmd)
	output, err := command.CombinedOutput()
	if err != nil {
		return err
	}
	log.Infof("Command output: %s", string(output))
	return nil
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Runs the 'up' commands specified in the yaml file",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileReader := &FileReader{filePath: args[0]}
		yamlData, err := fileReader.ReadYAML()
		if err != nil {
			log.WithError(err).Error("Error reading YAML data")
			return
		}
		for _, cmd := range yamlData.Up {
			err = ExecCommand(cmd)
			if err != nil {
				log.WithError(err).Error("Error running command")
				return
			}
		}
		log.Info("Commands completed successfully")
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Runs the 'down' commands specified in the yaml file",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fileReader := &FileReader{filePath: args[0]}
		yamlData, err := fileReader.ReadYAML()
		if err != nil {
			log.WithError(err).Error("Error reading YAML data")
			return
		}
		for _, cmd := range yamlData.Down {
			err = ExecCommand(cmd)
			if err != nil {
				log.WithError(err).Error("Error running command")
				return
			}
		}
		log.Info("Commands completed successfully")
	},
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "lol [README.md]",
		Short: "LocalLaunch is a CLI to read a YAML slice of code inside a README.md file",
	}
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.Execute()
}
