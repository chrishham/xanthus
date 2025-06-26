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

// InstallChart installs a Helm chart on the specified VPS
func (h *HelmService) InstallChart(vpsIP, sshUser, privateKey, releaseName, chartName, chartVersion, namespace string, values map[string]string) error {
	// Connect to VPS
	conn, err := h.sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

	// Create namespace if it doesn't exist
	createNamespaceCmd := fmt.Sprintf("kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -", namespace)
	if _, err := h.sshService.ExecuteCommand(conn, createNamespaceCmd); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	// Build Helm install command
	helmCmd := fmt.Sprintf("helm install %s %s --version %s --namespace %s --create-namespace", 
		releaseName, chartName, chartVersion, namespace)

	// Add custom values if provided
	if len(values) > 0 {
		var setArgs []string
		for key, value := range values {
			setArgs = append(setArgs, fmt.Sprintf("%s=%s", key, value))
		}
		helmCmd += " --set " + strings.Join(setArgs, ",")
	}

	// Execute Helm install
	if _, err := h.sshService.ExecuteCommand(conn, helmCmd); err != nil {
		return fmt.Errorf("failed to install Helm chart: %v", err)
	}

	return nil
}

// UpgradeChart upgrades an existing Helm release
func (h *HelmService) UpgradeChart(vpsIP, sshUser, privateKey, releaseName, chartName, chartVersion, namespace string, values map[string]string) error {
	// Connect to VPS
	conn, err := h.sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

	// Build Helm upgrade command
	helmCmd := fmt.Sprintf("helm upgrade %s %s --version %s --namespace %s", 
		releaseName, chartName, chartVersion, namespace)

	// Add custom values if provided
	if len(values) > 0 {
		var setArgs []string
		for key, value := range values {
			setArgs = append(setArgs, fmt.Sprintf("%s=%s", key, value))
		}
		helmCmd += " --set " + strings.Join(setArgs, ",")
	}

	// Execute Helm upgrade
	if _, err := h.sshService.ExecuteCommand(conn, helmCmd); err != nil {
		return fmt.Errorf("failed to upgrade Helm chart: %v", err)
	}

	return nil
}

// UninstallChart removes a Helm release
func (h *HelmService) UninstallChart(vpsIP, sshUser, privateKey, releaseName, namespace string) error {
	// Connect to VPS
	conn, err := h.sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

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
	// Connect to VPS
	conn, err := h.sshService.ConnectToVPS(vpsIP, sshUser, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to connect to VPS: %v", err)
	}
	defer conn.Close()

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