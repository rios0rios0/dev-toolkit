package main

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/rios0rios0/locallaunch/changelog"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	logger "github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

type GlobalConfig struct {
	ProjectsPath      string `yaml:"projects_path"`
	UserName          string `yaml:"user_name"`
	UserEmail         string `yaml:"user_email"`
	GitLabAccessToken string `yaml:"gitlab_access_token"`
	Projects          []Config
}

type Config struct {
	Path       string `yaml:"path"`
	Language   string `yaml:"language"`
	NewVersion string
}

// LanguageAdapter is the interface for language-specific adapters
type LanguageAdapter interface {
	UpdateVersion(path string, config *Config) error
	VersionFile() string
	VersionIdentifier() string
}

// PythonAdapter is the adapter for Python projects
type PythonAdapter struct{}

func (p *PythonAdapter) UpdateVersion(path string, config *Config) error {
	projectName := filepath.Base(config.Path)
	versionFilePath := filepath.Join(path, projectName, p.VersionFile())
	if _, err := os.Stat(versionFilePath); os.IsNotExist(err) {
		return nil
	}

	content, err := ioutil.ReadFile(versionFilePath)
	if err != nil {
		return err
	}

	versionIdentifier := p.VersionIdentifier()
	versionPattern := fmt.Sprintf(`%s(\d+\.\d+\.\d+)`, regexp.QuoteMeta(versionIdentifier))
	re := regexp.MustCompile(versionPattern)

	updatedContent := re.ReplaceAllString(string(content), versionIdentifier+config.NewVersion)
	err = ioutil.WriteFile(versionFilePath, []byte(updatedContent), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (p *PythonAdapter) VersionFile() string {
	return "__init__.py"
}

func (p *PythonAdapter) VersionIdentifier() string {
	return "__version__ = "
}

func getRemoteServiceType(repo *git.Repository) (string, error) {
	cfg, err := repo.Config()
	if err != nil {
		return "", err
	}

	for _, remote := range cfg.Remotes {
		if strings.Contains(remote.URLs[0], "gitlab.com") {
			return "GitLab", nil
		} else if strings.Contains(remote.URLs[0], "github.com") {
			return "GitHub", nil
		}
	}

	return "Unknown", nil
}

func createGitLabMergeRequest(globalConfig *GlobalConfig, projectPath string, repo *git.Repository) error {
	gitlabClient, err := gitlab.NewClient(globalConfig.GitLabAccessToken)
	if err != nil {
		return err
	}

	remoteURL, err := getRemoteServiceType(repo)
	if err != nil {
		return err
	}

	remoteURLParsed, err := url.Parse(remoteURL)
	if err != nil {
		return err
	}

	namespace, project := filepath.Split(remoteURLParsed.Path)
	namespace = strings.TrimSuffix(namespace, "/")

	projectID := url.PathEscape(fmt.Sprintf("%s/%s", namespace, project))
	mrTitle := "Bump version"

	mergeRequestOptions := &gitlab.CreateMergeRequestOptions{
		SourceBranch:       gitlab.String("chore/bump"),
		TargetBranch:       gitlab.String("main"),
		Title:              &mrTitle,
		RemoveSourceBranch: gitlab.Bool(true),
	}

	_, _, err = gitlabClient.MergeRequests.CreateMergeRequest(projectID, mergeRequestOptions)
	if err != nil {
		return err
	}

	fmt.Printf("Merge Request created for project at %s\n", projectPath)
	return nil
}

func processProject(globalConfig *GlobalConfig, config *Config) error {
	logger.Info("Getting adapter by name")
	adapter := getAdapterByName(config.Language)
	if adapter == nil {
		return fmt.Errorf("invalid adapter: %s", config.Language)
	}

	logger.Info("Joining project path")
	projectPath := filepath.Join(globalConfig.ProjectsPath, config.Path)

	logger.Info("Joining changelog path")
	changelogPath := filepath.Join(projectPath, "CHANGELOG.md")
	version, err := changelog.UpdateChangelogFile(changelogPath)
	if err != nil {
		fmt.Printf("No version found in CHANGELOG.md for project at %s\n", config.Path)
		return err
	}

	logger.Info("Updating adapter version")
	config.NewVersion = version.String()
	err = adapter.UpdateVersion(projectPath, config)
	if err != nil {
		return err
	}

	logger.Info("Opening git repository")
	repo, err := git.PlainOpen(projectPath)
	if err != nil {
		return err
	}

	logger.Info("Getting worktree")
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	changelogfile := filepath.Join(projectPath, "CHANGELOG.md")
	logger.Info("Adding version file to the worktree")
	result, err := w.Add(changelogfile)
	if err != nil {
		logger.Errorf("Result not expected: %v", result)
		return err
	}

	logger.Info("Committing the updated version")
	commit, err := w.Commit("Bump version to "+config.NewVersion, &git.CommitOptions{
		Author: &object.Signature{
			Name:  globalConfig.UserName,
			Email: globalConfig.UserEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	logger.Info("Committing the object")
	_, err = repo.CommitObject(commit)
	if err != nil {
		return err
	}

	logger.Info("Getting repository head")
	head, err := repo.Head()
	if err != nil {
		return err
	}

	logger.Info("Creating hash reference")
	ref := plumbing.NewHashReference("refs/heads/chore/bump", head.Hash())
	err = repo.Storer.SetReference(ref)
	if err != nil {
		return err
	}

	logger.Info("Determining remote service type")
	serviceType, err := getRemoteServiceType(repo)
	if err != nil {
		return err
	}

	if serviceType == "GitLab" {
		logger.Info("Creating GitLab merge request")
		err = createGitLabMergeRequest(globalConfig, projectPath, repo)
		if err != nil {
			return err
		}
	}

	logger.Info("Project processing completed")
	return nil
}

func getAdapterByName(name string) LanguageAdapter {
	switch name {
	case "Python":
		return &PythonAdapter{}
	default:
		return nil
	}
}

func iterateProjects(globalConfig *GlobalConfig) error {
	for _, project := range globalConfig.Projects {
		err := processProject(globalConfig, &project)
		if err != nil {
			fmt.Printf("Error processing project at %s: %v\n", project.Path, err)
		}
	}
	return nil
}

func main() {
	var globalConfig GlobalConfig
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(data, &globalConfig)
	if err != nil {
		panic(err)
	}

	err = iterateProjects(&globalConfig)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
