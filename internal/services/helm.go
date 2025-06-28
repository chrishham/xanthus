package services

import (
	"fmt"
	"strings"
)

// HelmService handles Helm chart operations on K3s clusters
type HelmService struct {
	sshService *SSHService
}

// NewHelmService creates a new Helm service instance
func NewHelmService() *HelmService {
	return &HelmService{
		sshService: NewSSHService(),
	}
}

// InstallChart installs a Helm chart on the specified VPS using a values file
func (h *HelmService) InstallChart(vpsIP, sshUser, privateKey, releaseName, chartName, chartVersion, namespace, valuesFile string) error {
	// Try to get existing connection from session manager
	sessionManager := GetGlobalSessionManager()
	var conn *SSHConnection
	var shouldCloseConn bool = true

	// Check if there's an active session for this VPS
	sessionManager.mutex.RLock()
	for _, session := range sessionManager.sessions {
		if session.Host == vpsIP && session.User == sshUser && session.isValid() {
			conn = session.Connection
			shouldCloseConn = false // Don't close shared session connection
			break
		}
	}
	sessionManager.mutex.RUnlock()

	// Fallback to creating new connection if no session available
	if conn == nil {
		var err error
		conn, err = h.sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
		if err != nil {
			return fmt.Errorf("failed to connect to VPS: %v", err)
		}
	}

	if shouldCloseConn {
		defer conn.Close()
	}

	// Create namespace if it doesn't exist
	createNamespaceCmd := fmt.Sprintf("kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -", namespace)
	if _, err := h.sshService.ExecuteCommand(conn, createNamespaceCmd); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	// Build Helm install command
	var helmCmd string
	
	// For ArgoCD charts, remove any existing CRDs to avoid ownership conflicts
	if strings.Contains(chartName, "argo-cd") {
		// Remove existing ArgoCD CRDs if they exist (they're tied to other deployments)
		removeCRDsCmd := `
		kubectl delete crd applications.argoproj.io --ignore-not-found=true
		kubectl delete crd appprojects.argoproj.io --ignore-not-found=true 
		kubectl delete crd applicationsets.argoproj.io --ignore-not-found=true
		`
		if _, err := h.sshService.ExecuteCommand(conn, removeCRDsCmd); err != nil {
			// Log warning but don't fail deployment - CRDs might not exist
			fmt.Printf("Warning: Failed to remove existing ArgoCD CRDs: %v\n", err)
		}
		// Now let Helm install fresh CRDs with the new deployment
	}
	
	if strings.HasPrefix(chartName, "/") || strings.HasPrefix(chartName, "./") {
		// Local chart path - don't use --version flag
		helmCmd = fmt.Sprintf("helm install %s %s --namespace %s --create-namespace",
			releaseName, chartName, namespace)
	} else if chartVersion == "stable" || chartVersion == "latest" || chartVersion == "" {
		// Repository chart with stable/latest version - omit --version flag to get latest
		helmCmd = fmt.Sprintf("helm install %s %s --namespace %s --create-namespace",
			releaseName, chartName, namespace)
	} else {
		// Repository chart with specific version
		helmCmd = fmt.Sprintf("helm install %s %s --version %s --namespace %s --create-namespace",
			releaseName, chartName, chartVersion, namespace)
	}

	// Add values file if provided
	if valuesFile != "" {
		helmCmd += fmt.Sprintf(" -f %s", valuesFile)
	}

	// Execute Helm install
	result, err := h.sshService.ExecuteCommand(conn, helmCmd)
	if err != nil {
		return fmt.Errorf("failed to install Helm chart: command failed: %v, output: %s", err, result.Output)
	}

	return nil
}

// UpgradeChart upgrades an existing Helm release using a values file
func (h *HelmService) UpgradeChart(vpsIP, sshUser, privateKey, releaseName, chartName, chartVersion, namespace, valuesFile string) error {
	// Try to get existing connection from session manager
	sessionManager := GetGlobalSessionManager()
	var conn *SSHConnection
	var shouldCloseConn bool = true

	// Check if there's an active session for this VPS
	sessionManager.mutex.RLock()
	for _, session := range sessionManager.sessions {
		if session.Host == vpsIP && session.User == sshUser && session.isValid() {
			conn = session.Connection
			shouldCloseConn = false // Don't close shared session connection
			break
		}
	}
	sessionManager.mutex.RUnlock()

	// Fallback to creating new connection if no session available
	if conn == nil {
		var err error
		conn, err = h.sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
		if err != nil {
			return fmt.Errorf("failed to connect to VPS: %v", err)
		}
	}

	if shouldCloseConn {
		defer conn.Close()
	}

	// Build Helm upgrade command
	// For local charts, don't use --version flag
	var helmCmd string
	if strings.HasPrefix(chartName, "/") || strings.HasPrefix(chartName, "./") {
		// Local chart path
		helmCmd = fmt.Sprintf("helm upgrade %s %s --namespace %s",
			releaseName, chartName, namespace)
	} else {
		// Repository chart
		helmCmd = fmt.Sprintf("helm upgrade %s %s --version %s --namespace %s",
			releaseName, chartName, chartVersion, namespace)
	}

	// Add values file if provided
	if valuesFile != "" {
		helmCmd += fmt.Sprintf(" -f %s", valuesFile)
	}

	// Execute Helm upgrade
	result, err := h.sshService.ExecuteCommand(conn, helmCmd)
	if err != nil {
		return fmt.Errorf("failed to upgrade Helm chart: command failed: %v, output: %s", err, result.Output)
	}

	return nil
}

// UninstallChart removes a Helm release
func (h *HelmService) UninstallChart(vpsIP, sshUser, privateKey, releaseName, namespace string) error {
	// Try to get existing connection from session manager
	sessionManager := GetGlobalSessionManager()
	var conn *SSHConnection
	var shouldCloseConn bool = true

	// Check if there's an active session for this VPS
	sessionManager.mutex.RLock()
	for _, session := range sessionManager.sessions {
		if session.Host == vpsIP && session.User == sshUser && session.isValid() {
			conn = session.Connection
			shouldCloseConn = false // Don't close shared session connection
			break
		}
	}
	sessionManager.mutex.RUnlock()

	// Fallback to creating new connection if no session available
	if conn == nil {
		var err error
		conn, err = h.sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
		if err != nil {
			return fmt.Errorf("failed to connect to VPS: %v", err)
		}
	}

	if shouldCloseConn {
		defer conn.Close()
	}

	// Execute Helm uninstall
	helmCmd := fmt.Sprintf("helm uninstall %s --namespace %s", releaseName, namespace)
	if _, err := h.sshService.ExecuteCommand(conn, helmCmd); err != nil {
		return fmt.Errorf("failed to uninstall Helm chart: %v", err)
	}

	// Optionally delete namespace if empty (be careful with this)
	// checkNamespaceCmd := fmt.Sprintf("kubectl get all -n %s --no-headers | wc -l", namespace)
	// output, err := h.sshService.ExecuteCommandWithOutput(conn, checkNamespaceCmd)
	// if err == nil && strings.TrimSpace(output) == "0" {
	//     deleteNamespaceCmd := fmt.Sprintf("kubectl delete namespace %s", namespace)
	//     h.sshService.ExecuteCommand(conn, deleteNamespaceCmd)
	// }

	return nil
}

// GetReleaseStatus gets the status of a Helm release
func (h *HelmService) GetReleaseStatus(vpsIP, sshUser, privateKey, releaseName, namespace string) (string, error) {
	// Try to get existing connection from session manager
	sessionManager := GetGlobalSessionManager()
	var conn *SSHConnection
	var shouldCloseConn bool = true

	// Check if there's an active session for this VPS
	sessionManager.mutex.RLock()
	for _, session := range sessionManager.sessions {
		if session.Host == vpsIP && session.User == sshUser && session.isValid() {
			conn = session.Connection
			shouldCloseConn = false // Don't close shared session connection
			break
		}
	}
	sessionManager.mutex.RUnlock()

	// Fallback to creating new connection if no session available
	if conn == nil {
		var err error
		conn, err = h.sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
		if err != nil {
			return "", fmt.Errorf("failed to connect to VPS: %v", err)
		}
	}

	if shouldCloseConn {
		defer conn.Close()
	}

	// Get Helm release status
	helmCmd := fmt.Sprintf("helm status %s --namespace %s -o json", releaseName, namespace)
	result, err := h.sshService.ExecuteCommand(conn, helmCmd)
	if err != nil {
		return "unknown", fmt.Errorf("failed to get release status: %v", err)
	}

	// Parse basic status from output (could be enhanced with proper JSON parsing)
	if strings.Contains(result.Output, "deployed") {
		return "deployed", nil
	} else if strings.Contains(result.Output, "failed") {
		return "failed", nil
	} else if strings.Contains(result.Output, "pending") {
		return "pending", nil
	}

	return "unknown", nil
}
