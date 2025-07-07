package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	GitHubBaseURL = "https://api.github.com"
	cacheTTL      = 10 * time.Minute // Cache for 10 minutes to avoid rate limits
)

// GitHubService handles GitHub API operations
type GitHubService struct {
	client *http.Client
	cache  *versionCache
}

// versionCache implements a simple in-memory cache for version data
type versionCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
}

type cacheEntry struct {
	releases  []GitHubRelease
	timestamp time.Time
}

// NewGitHubService creates a new GitHub service instance
func NewGitHubService() *GitHubService {
	return &GitHubService{
		client: &http.Client{Timeout: 30 * time.Second},
		cache: &versionCache{
			data: make(map[string]cacheEntry),
		},
	}
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
}

// GitHubError represents a GitHub API error
type GitHubError struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url"`
}

// GetLatestRelease fetches the latest release for a given repository
func (gs *GitHubService) GetLatestRelease(owner, repo string) (*GitHubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", GitHubBaseURL, owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Xanthus/1.0")

	resp, err := gs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var ghError GitHubError
		if err := json.NewDecoder(resp.Body).Decode(&ghError); err != nil {
			return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("GitHub API error: %s", ghError.Message)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// GetReleases fetches all releases for a given repository
func (gs *GitHubService) GetReleases(owner, repo string, perPage int) ([]GitHubRelease, error) {
	if perPage == 0 {
		perPage = 30 // Default GitHub per_page
	}
	if perPage > 100 {
		perPage = 100 // GitHub max per_page
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=%d", GitHubBaseURL, owner, repo, perPage)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Xanthus/1.0")

	resp, err := gs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var ghError GitHubError
		if err := json.NewDecoder(resp.Body).Decode(&ghError); err != nil {
			return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("GitHub API error: %s", ghError.Message)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return releases, nil
}

// GetCodeServerVersions is a convenience method to get code-server releases with caching
func (gs *GitHubService) GetCodeServerVersions(limit int) ([]GitHubRelease, error) {
	cacheKey := fmt.Sprintf("coder/code-server:%d", limit)

	// Check cache first
	gs.cache.mu.RLock()
	if entry, exists := gs.cache.data[cacheKey]; exists {
		if time.Since(entry.timestamp) < cacheTTL {
			gs.cache.mu.RUnlock()
			return entry.releases, nil
		}
	}
	gs.cache.mu.RUnlock()

	// Cache miss or expired, fetch from API
	releases, err := gs.GetReleases("coder", "code-server", limit)
	if err != nil {
		return nil, err
	}

	// Filter out drafts and pre-releases for stable versions
	var stableReleases []GitHubRelease
	for _, release := range releases {
		if !release.Draft {
			stableReleases = append(stableReleases, release)
		}
	}

	// Update cache
	gs.cache.mu.Lock()
	gs.cache.data[cacheKey] = cacheEntry{
		releases:  stableReleases,
		timestamp: time.Now(),
	}
	gs.cache.mu.Unlock()

	return stableReleases, nil
}

// GetCodeServerLatestVersion is a convenience method to get the latest stable code-server release
func (gs *GitHubService) GetCodeServerLatestVersion() (*GitHubRelease, error) {
	return gs.GetLatestRelease("coder", "code-server")
}
