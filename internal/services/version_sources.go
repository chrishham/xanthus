package services

import (
	"fmt"
	"log"
	"strings"
)

// VersionSource interface defines methods for fetching version information
type VersionSource interface {
	GetLatestVersion() (string, error)
	GetVersionHistory() ([]string, error)
	GetSourceType() string
	GetSourceName() string
}

// GitHubVersionSource fetches versions from GitHub releases
type GitHubVersionSource struct {
	repository    string
	githubService *GitHubService
}

// NewGitHubVersionSource creates a new GitHub version source
func NewGitHubVersionSource(repository string) VersionSource {
	return &GitHubVersionSource{
		repository:    repository,
		githubService: NewGitHubService(),
	}
}

func (g *GitHubVersionSource) GetLatestVersion() (string, error) {
	log.Printf("Fetching latest version from GitHub repository: %s", g.repository)
	
	switch g.repository {
	case "coder/code-server":
		release, err := g.githubService.GetCodeServerLatestVersion()
		if err != nil {
			return "", fmt.Errorf("failed to fetch code-server version: %w", err)
		}
		return strings.TrimPrefix(release.TagName, "v"), nil
	default:
		// For now, we'll use the existing implementation pattern
		// Future enhancement: implement generic GitHub API calls
		log.Printf("Warning: Generic GitHub version fetching not yet implemented for %s", g.repository)
		return "latest", fmt.Errorf("repository %s not supported yet", g.repository)
	}
}

func (g *GitHubVersionSource) GetVersionHistory() ([]string, error) {
	// TODO: Implement version history fetching
	return []string{}, fmt.Errorf("version history not implemented for GitHub source")
}

func (g *GitHubVersionSource) GetSourceType() string {
	return "github"
}

func (g *GitHubVersionSource) GetSourceName() string {
	return g.repository
}

// DockerHubVersionSource fetches versions from Docker Hub
type DockerHubVersionSource struct {
	repository string
}

// NewDockerHubVersionSource creates a new Docker Hub version source
func NewDockerHubVersionSource(repository string) VersionSource {
	return &DockerHubVersionSource{
		repository: repository,
	}
}

func (d *DockerHubVersionSource) GetLatestVersion() (string, error) {
	log.Printf("Fetching latest version from Docker Hub repository: %s", d.repository)
	
	// TODO: Implement Docker Hub API integration
	// For now, return a placeholder
	log.Printf("Warning: Docker Hub version fetching not yet implemented for %s", d.repository)
	return "latest", fmt.Errorf("Docker Hub integration not implemented yet")
}

func (d *DockerHubVersionSource) GetVersionHistory() ([]string, error) {
	// TODO: Implement version history fetching
	return []string{}, fmt.Errorf("version history not implemented for Docker Hub source")
}

func (d *DockerHubVersionSource) GetSourceType() string {
	return "dockerhub"
}

func (d *DockerHubVersionSource) GetSourceName() string {
	return d.repository
}

// HelmVersionSource fetches versions from Helm repositories
type HelmVersionSource struct {
	repository string
	chartName  string
}

// NewHelmVersionSource creates a new Helm version source
func NewHelmVersionSource(repository, chartName string) VersionSource {
	return &HelmVersionSource{
		repository: repository,
		chartName:  chartName,
	}
}

func (h *HelmVersionSource) GetLatestVersion() (string, error) {
	log.Printf("Fetching latest version from Helm repository: %s, chart: %s", h.repository, h.chartName)
	
	// TODO: Implement Helm repository API integration
	// For now, return a placeholder
	log.Printf("Warning: Helm version fetching not yet implemented for %s/%s", h.repository, h.chartName)
	return "latest", fmt.Errorf("Helm integration not implemented yet")
}

func (h *HelmVersionSource) GetVersionHistory() ([]string, error) {
	// TODO: Implement version history fetching
	return []string{}, fmt.Errorf("version history not implemented for Helm source")
}

func (h *HelmVersionSource) GetSourceType() string {
	return "helm"
}

func (h *HelmVersionSource) GetSourceName() string {
	return fmt.Sprintf("%s/%s", h.repository, h.chartName)
}

// StaticVersionSource provides a static version (no fetching)
type StaticVersionSource struct {
	version string
	name    string
}

// NewStaticVersionSource creates a new static version source
func NewStaticVersionSource(version, name string) VersionSource {
	return &StaticVersionSource{
		version: version,
		name:    name,
	}
}

func (s *StaticVersionSource) GetLatestVersion() (string, error) {
	log.Printf("Using static version: %s for %s", s.version, s.name)
	return s.version, nil
}

func (s *StaticVersionSource) GetVersionHistory() ([]string, error) {
	return []string{s.version}, nil
}

func (s *StaticVersionSource) GetSourceType() string {
	return "static"
}

func (s *StaticVersionSource) GetSourceName() string {
	return s.name
}

// VersionSourceFactory creates version sources based on configuration
type VersionSourceFactory struct{}

// NewVersionSourceFactory creates a new version source factory
func NewVersionSourceFactory() *VersionSourceFactory {
	return &VersionSourceFactory{}
}

// CreateVersionSource creates a version source based on type and configuration
func (f *VersionSourceFactory) CreateVersionSource(sourceType, source, chart string) (VersionSource, error) {
	switch sourceType {
	case "github":
		return NewGitHubVersionSource(source), nil
	case "dockerhub":
		return NewDockerHubVersionSource(source), nil
	case "helm":
		if chart == "" {
			return nil, fmt.Errorf("chart name is required for helm version source")
		}
		return NewHelmVersionSource(source, chart), nil
	case "static":
		return NewStaticVersionSource(source, "static"), nil
	default:
		return nil, fmt.Errorf("unsupported version source type: %s", sourceType)
	}
}