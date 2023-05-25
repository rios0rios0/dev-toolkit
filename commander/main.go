package commander

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os/exec"
	"strings"

	"runtime"
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
func (f *FileReader) ReadYAML(toBeExecuted string) (YamlData, error) {
	logrus.Info("i entered in the function readyaml")
	data, err := ioutil.ReadFile(f.filePath)
	if err != nil {
		logrus.WithError(err).Error("Error reading file")
		return YamlData{}, err
	}

	fileContents := string(data)
	yamlStart := fmt.Sprintf("# lol:%s", toBeExecuted)
	yamlEnd := "```"

	startIndex := strings.Index(fileContents, yamlStart)
	if startIndex == -1 {
		return YamlData{}, errors.New("YAML start not found")
	}

	endIndex := strings.Index(fileContents[startIndex:], yamlEnd)
	if endIndex == -1 {
		return YamlData{}, errors.New("YAML end not found")
	}

	yamlString := fileContents[startIndex+len(yamlStart) : startIndex+endIndex]
	var yamlData YamlData
	err = yaml.Unmarshal([]byte(yamlString), &yamlData)
	if err != nil {
		logrus.WithError(err).Error("Error parsing YAML")
		return YamlData{}, err
	}

	return yamlData, nil
}

// ExecCommand executes a command in the operating system
func ExecCommand(cmd string) error {
	if runtime.GOOS == "windows" {
		execwindows(cmd)
	} else {
		execlinux(cmd)

	}
	return nil
}
func execlinux(cmd string) error {

	log.Info("o comando executado é " + cmd)
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
		log.Info("i entered in the function upCmd  ")
		fileReader := &FileReader{filePath: args[0]}
		yamlData, err := fileReader.ReadYAML("up")

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
		log.Info("i entered in the function downCmd ")
		fileReader := &FileReader{filePath: args[0]}
		yamlData, err := fileReader.ReadYAML("down")
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
func execwindows(command string) {
	log.Info("o comando executado é " + command)
	cmd := exec.Command("cmd", "/c", command)
	output, err := cmd.Output()
	if err != nil {
		log.Error(err)
		return
	}
	log.Info(string(output))
}
