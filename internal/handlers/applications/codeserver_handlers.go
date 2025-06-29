package applications

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
)

// CodeServerHandlers provides code-server specific operations
type CodeServerHandlers struct {
	vpsHelper      *VPSConnectionHelper
	passwordHelper *PasswordHelper
}

// NewCodeServerHandlers creates a new CodeServerHandlers instance
func NewCodeServerHandlers() *CodeServerHandlers {
	return &CodeServerHandlers{
		vpsHelper:      NewVPSConnectionHelper(),
		passwordHelper: NewPasswordHelper(),
	}
}

// UpdatePassword updates the password for a code-server application
func (cs *CodeServerHandlers) UpdatePassword(token, accountID, appID, newPassword string, app interface{}) error {
	// Get VPS connection
	appData := app.(struct {
		VPSID     string
		Subdomain string
		ID        string
		Namespace string
	})

	conn, err := cs.vpsHelper.GetVPSConnection(token, accountID, appData.VPSID)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Update the Kubernetes secret with new password
	releaseName := fmt.Sprintf("%s-%s", appData.Subdomain, appData.ID)
	secretName := fmt.Sprintf("%s-code-server", releaseName)
	encodedPassword := utils.Base64Encode(newPassword)
	cmd := fmt.Sprintf("kubectl patch secret --namespace %s %s -p '{\"data\":{\"password\":\"%s\"}}'", appData.Namespace, secretName, encodedPassword)

	sshService := services.NewSSHService()
	_, err = sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return fmt.Errorf("failed to update Kubernetes secret: %v", err)
	}

	// Restart the code-server deployment to pick up the new password
	deploymentName := fmt.Sprintf("%s-code-server", releaseName)
	restartCmd := fmt.Sprintf("kubectl rollout restart deployment --namespace %s %s", appData.Namespace, deploymentName)
	_, err = sshService.ExecuteCommand(conn, restartCmd)
	if err != nil {
		return fmt.Errorf("failed to restart deployment: %v", err)
	}

	// Update stored password in KV
	return cs.passwordHelper.StoreEncryptedPassword(token, accountID, appID, newPassword)
}

// RetrieveAndStorePassword retrieves the auto-generated password from Kubernetes secret
func (cs *CodeServerHandlers) RetrieveAndStorePassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := services.NewSSHService()
	vpsIDInt, _ := strconv.Atoi(strings.Split(appID, "-")[1]) // Extract VPS ID from app ID

	conn, err := sshService.GetOrCreateConnection(vpsIP, sshUser, privateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Retrieve password from Kubernetes secret
	secretName := fmt.Sprintf("%s-code-server", releaseName)

	// First, let's check if the secret exists and list available secrets for debugging
	listCmd := fmt.Sprintf("kubectl get secrets --namespace %s", namespace)
	listResult, err := sshService.ExecuteCommand(conn, listCmd)
	if err != nil {
		log.Printf("Debug: Failed to list secrets in namespace %s: %v", namespace, err)
	} else {
		log.Printf("Debug: Available secrets in namespace %s: %s", namespace, listResult.Output)
	}

	cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' | base64 --decode", namespace, secretName)
	result, err := sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return fmt.Errorf("failed to retrieve password from secret '%s' in namespace '%s': %v", secretName, namespace, err)
	}

	// Store password using helper
	password := strings.TrimSpace(result.Output)
	return cs.passwordHelper.StoreEncryptedPassword(token, accountID, appID, password)
}

// ValidateVersion checks if a given version exists in GitHub releases
func (cs *CodeServerHandlers) ValidateVersion(version string) (bool, error) {
	githubService := services.NewGitHubService()
	releases, err := githubService.GetCodeServerVersions(50) // Check last 50 releases
	if err != nil {
		return false, err
	}

	// Check if the version exists in the releases
	for _, release := range releases {
		dockerTag := strings.TrimPrefix(release.TagName, "v")
		if dockerTag == version || release.TagName == version {
			return true, nil
		}
	}

	return false, nil
}

// CreateVSCodeSettingsConfigMap creates a ConfigMap with default VS Code settings for persistence
func (cs *CodeServerHandlers) CreateVSCodeSettingsConfigMap(conn *services.SSHConnection, releaseName, namespace string) error {
	sshService := services.NewSSHService()

	// Default VS Code settings with theme persistence and other user preferences
	settingsJSON := `{
    "workbench.colorTheme": "Default Dark+",
    "workbench.iconTheme": "vs-seti",
    "editor.fontSize": 14,
    "editor.tabSize": 4,
    "editor.insertSpaces": true,
    "editor.detectIndentation": true,
    "editor.renderWhitespace": "selection",
    "editor.rulers": [80, 120],
    "files.autoSave": "afterDelay",
    "files.autoSaveDelay": 1000,
    "explorer.confirmDelete": false,
    "explorer.confirmDragAndDrop": false,
    "git.enableSmartCommit": true,
    "git.confirmSync": false,
    "terminal.integrated.fontSize": 14,
    "workbench.startupEditor": "newUntitledFile"
}`

	// Create ConfigMap with the settings
	configMapName := fmt.Sprintf("%s-vscode-settings", releaseName)
	createConfigMapCmd := fmt.Sprintf(`kubectl create configmap %s -n %s --from-literal=settings.json='%s' --dry-run=client -o yaml | kubectl apply -f -`,
		configMapName, namespace, settingsJSON)

	_, err := sshService.ExecuteCommand(conn, createConfigMapCmd)
	if err != nil {
		return fmt.Errorf("failed to create VS Code settings ConfigMap: %v", err)
	}

	log.Printf("Created VS Code settings ConfigMap: %s in namespace %s", configMapName, namespace)
	return nil
}
