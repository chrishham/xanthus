package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
		// Generic GitHub API implementation for any repository
		parts := strings.Split(g.repository, "/")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid repository format: expected owner/repo, got %s", g.repository)
		}

		release, err := g.githubService.GetLatestRelease(parts[0], parts[1])
		if err != nil {
			return "", fmt.Errorf("failed to fetch latest release for %s: %w", g.repository, err)
		}
		return strings.TrimPrefix(release.TagName, "v"), nil
	}
}

func (g *GitHubVersionSource) GetVersionHistory() ([]string, error) {
	log.Printf("Fetching version history from GitHub repository: %s", g.repository)

	parts := strings.Split(g.repository, "/")
	if len(parts) != 2 {
		return []string{}, fmt.Errorf("invalid repository format: expected owner/repo, got %s", g.repository)
	}

	releases, err := g.githubService.GetReleases(parts[0], parts[1], 20)
	if err != nil {
		return []string{}, fmt.Errorf("failed to fetch releases: %w", err)
	}

	var versions []string
	for _, release := range releases {
		if !release.Draft && !release.Prerelease {
			version := strings.TrimPrefix(release.TagName, "v")
			versions = append(versions, version)
		}
	}

	return versions, nil
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
	client     *http.Client
}

// DockerHubResponse represents the Docker Hub API response for tags
type DockerHubResponse struct {
	Count    int            `json:"count"`
	Next     string         `json:"next"`
	Previous string         `json:"previous"`
	Results  []DockerHubTag `json:"results"`
}

// DockerHubTag represents a Docker Hub tag
type DockerHubTag struct {
	Name        string    `json:"name"`
	FullSize    int64     `json:"full_size"`
	LastUpdated time.Time `json:"last_updated"`
}

// NewDockerHubVersionSource creates a new Docker Hub version source
func NewDockerHubVersionSource(repository string) VersionSource {
	return &DockerHubVersionSource{
		repository: repository,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (d *DockerHubVersionSource) GetLatestVersion() (string, error) {
	log.Printf("Fetching latest version from Docker Hub repository: %s", d.repository)

	// Docker Hub API endpoint for tags
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags/?page_size=100", d.repository)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Xanthus/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Docker Hub API request failed with status %d", resp.StatusCode)
	}

	var response DockerHubResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Results) == 0 {
		return "", fmt.Errorf("no tags found for repository %s", d.repository)
	}

	// Look for latest tag first, then fall back to most recent semantic version
	for _, tag := range response.Results {
		if tag.Name == "latest" {
			continue // Skip 'latest' tag, we want actual version numbers
		}
		// Return the first non-latest tag (Docker Hub returns tags by most recent)
		if isSemanticVersion(tag.Name) {
			return tag.Name, nil
		}
	}

	// If no semantic version found, return the first non-latest tag
	for _, tag := range response.Results {
		if tag.Name != "latest" {
			return tag.Name, nil
		}
	}

	return "latest", nil
}

func (d *DockerHubVersionSource) GetVersionHistory() ([]string, error) {
	log.Printf("Fetching version history from Docker Hub repository: %s", d.repository)

	// Docker Hub API endpoint for tags
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags/?page_size=100", d.repository)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Xanthus/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return []string{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{}, fmt.Errorf("Docker Hub API request failed with status %d", resp.StatusCode)
	}

	var response DockerHubResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return []string{}, fmt.Errorf("failed to decode response: %w", err)
	}

	var versions []string
	for _, tag := range response.Results {
		if tag.Name != "latest" { // Exclude 'latest' from version history
			versions = append(versions, tag.Name)
		}
	}

	return versions, nil
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
	client     *http.Client
}

// HelmChartResponse represents the Helm repository API response
type HelmChartResponse struct {
	APIVersion string                 `json:"apiVersion"`
	Entries    map[string][]HelmChart `json:"entries"`
}

// HelmChart represents a Helm chart version entry
type HelmChart struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	AppVersion  string    `json:"appVersion"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	URLs        []string  `json:"urls"`
}

// NewHelmVersionSource creates a new Helm version source
func NewHelmVersionSource(repository, chartName string) VersionSource {
	return &HelmVersionSource{
		repository: repository,
		chartName:  chartName,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *HelmVersionSource) GetLatestVersion() (string, error) {
	log.Printf("Fetching latest version from Helm repository: %s, chart: %s", h.repository, h.chartName)

	// Helm repository index.yaml endpoint
	url := fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(h.repository, "/"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/yaml, text/yaml")
	req.Header.Set("User-Agent", "Xanthus/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Helm repository request failed with status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse YAML response
	var response HelmChartResponse
	if err := yaml.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse YAML response: %w", err)
	}

	charts, exists := response.Entries[h.chartName]
	if !exists || len(charts) == 0 {
		return "", fmt.Errorf("chart %s not found in repository %s", h.chartName, h.repository)
	}

	// Return the first version (usually the latest)
	return charts[0].Version, nil
}

func (h *HelmVersionSource) GetVersionHistory() ([]string, error) {
	log.Printf("Fetching version history from Helm repository: %s, chart: %s", h.repository, h.chartName)

	// Helm repository index.yaml endpoint
	url := fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(h.repository, "/"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/yaml, text/yaml")
	req.Header.Set("User-Agent", "Xanthus/1.0")

	resp, err := h.client.Do(req)
	if err != nil {
		return []string{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{}, fmt.Errorf("Helm repository request failed with status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse YAML response
	var response HelmChartResponse
	if err := yaml.Unmarshal(body, &response); err != nil {
		return []string{}, fmt.Errorf("failed to parse YAML response: %w", err)
	}

	charts, exists := response.Entries[h.chartName]
	if !exists {
		return []string{}, fmt.Errorf("chart %s not found in repository %s", h.chartName, h.repository)
	}

	var versions []string
	for _, chart := range charts {
		versions = append(versions, chart.Version)
	}

	return versions, nil
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

// isSemanticVersion checks if a version string follows semantic versioning pattern
func isSemanticVersion(version string) bool {
	// Simple check for semantic versioning pattern (e.g., 1.2.3, v1.2.3, 1.2.3-beta)
	version = strings.TrimPrefix(version, "v")
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return false
	}

	// Check if first two parts are numeric
	for i := 0; i < 2 && i < len(parts); i++ {
		part := strings.SplitN(parts[i], "-", 2)[0] // Handle pre-release suffixes
		if len(part) == 0 {
			return false
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}
