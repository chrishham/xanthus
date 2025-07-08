package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/gin-gonic/gin"
)

// VersionHandler handles version management operations
type VersionHandler struct {
	*BaseHandler
	versionService *services.SelfUpdateService
}

// NewVersionHandler creates a new version handler instance
func NewVersionHandler() *VersionHandler {
	return &VersionHandler{
		BaseHandler:    NewBaseHandler(),
		versionService: services.NewSelfUpdateService(),
	}
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// VersionInfo represents version information
type VersionInfo struct {
	Current   string          `json:"current"`
	Available []GitHubRelease `json:"available"`
	Status    string          `json:"status"`
}

// UpdateRequest represents an update request
type UpdateRequest struct {
	Version string `json:"version"`
	Force   bool   `json:"force"`
}

// UpdateStatus represents the status of an update
type UpdateStatus struct {
	InProgress bool   `json:"in_progress"`
	Version    string `json:"version"`
	Status     string `json:"status"`
	Progress   int    `json:"progress"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
}

// GetCurrentVersion returns the current running version
func (h *VersionHandler) GetCurrentVersion(c *gin.Context) {
	// Get current version from build info or environment
	currentVersion := h.versionService.GetCurrentVersion()

	c.JSON(http.StatusOK, gin.H{
		"version": currentVersion,
		"status":  "running",
	})
}

// GetAboutInfo returns about information including version and platform details
func (h *VersionHandler) GetAboutInfo(c *gin.Context) {
	_, _, valid := utils.ValidateJWTAndGetAccountJSON(c)
	if !valid {
		return
	}

	aboutInfo := h.versionService.GetAboutInfo()
	c.JSON(http.StatusOK, aboutInfo)
}

// GetAvailableVersions returns available versions from GitHub releases
func (h *VersionHandler) GetAvailableVersions(c *gin.Context) {
	_, _, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	releases, err := h.fetchGitHubReleases()
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to fetch available versions: %v", err))
		return
	}

	// Filter out drafts and sort by version
	var availableReleases []GitHubRelease
	for _, release := range releases {
		if !release.Draft {
			availableReleases = append(availableReleases, release)
		}
	}

	// Sort by version (newest first)
	sort.Slice(availableReleases, func(i, j int) bool {
		return h.compareVersions(availableReleases[i].TagName, availableReleases[j].TagName) > 0
	})

	currentVersion := h.versionService.GetCurrentVersion()

	versionInfo := VersionInfo{
		Current:   currentVersion,
		Available: availableReleases,
		Status:    "ready",
	}

	c.JSON(http.StatusOK, versionInfo)
}

// TriggerUpdate triggers an update to a specific version
func (h *VersionHandler) TriggerUpdate(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONBadRequest(c, "Invalid request body")
		return
	}

	if req.Version == "" {
		utils.JSONBadRequest(c, "Version is required")
		return
	}

	// Check if update is already in progress
	if h.versionService.IsUpdateInProgress() {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Update already in progress",
		})
		return
	}

	// Validate version exists
	releases, err := h.fetchGitHubReleases()
	if err != nil {
		utils.JSONInternalServerError(c, fmt.Sprintf("Failed to validate version: %v", err))
		return
	}

	var targetRelease *GitHubRelease
	for _, release := range releases {
		if release.TagName == req.Version {
			targetRelease = &release
			break
		}
	}

	if targetRelease == nil {
		utils.JSONBadRequest(c, "Version not found")
		return
	}

	// Start update process
	go h.versionService.StartUpdate(token, accountID, req.Version, targetRelease.Body)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Update started",
		"version": req.Version,
	})
}

// GetUpdateStatus returns the current update status
func (h *VersionHandler) GetUpdateStatus(c *gin.Context) {
	_, _, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	status := h.versionService.GetUpdateStatus()
	c.JSON(http.StatusOK, status)
}

// RollbackVersion triggers a rollback to the previous version
func (h *VersionHandler) RollbackVersion(c *gin.Context) {
	token, accountID, valid := h.validateTokenAndAccount(c)
	if !valid {
		return
	}

	// Check if rollback is possible
	if !h.versionService.CanRollback() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No previous version available for rollback",
		})
		return
	}

	// Check if update is in progress
	if h.versionService.IsUpdateInProgress() {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Cannot rollback while update is in progress",
		})
		return
	}

	previousVersion := h.versionService.GetPreviousVersion()
	if previousVersion == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No previous version found",
		})
		return
	}

	// Start rollback process
	go h.versionService.StartRollback(token, accountID, previousVersion)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Rollback started",
		"version": previousVersion,
	})
}

// fetchGitHubReleases fetches releases from GitHub API
func (h *VersionHandler) fetchGitHubReleases() ([]GitHubRelease, error) {
	url := "https://api.github.com/repos/chrishham/xanthus/releases"
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "Xanthus/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	return releases, nil
}

// compareVersions compares two version strings
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if v1 == v2
func (h *VersionHandler) compareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Split by dots
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		var err error

		if i < len(parts1) {
			p1, err = strconv.Atoi(parts1[i])
			if err != nil {
				return 0 // Invalid version format
			}
		}

		if i < len(parts2) {
			p2, err = strconv.Atoi(parts2[i])
			if err != nil {
				return 0 // Invalid version format
			}
		}

		if p1 > p2 {
			return 1
		} else if p1 < p2 {
			return -1
		}
	}

	return 0
}
