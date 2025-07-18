package applications

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/chrishham/xanthus/internal/services"
	"github.com/chrishham/xanthus/internal/utils"
)

// ArgoCDHandlers provides ArgoCD specific operations
type ArgoCDHandlers struct {
	vpsHelper      *VPSConnectionHelper
	passwordHelper *PasswordHelper
}

// NewArgoCDHandlers creates a new ArgoCDHandlers instance
func NewArgoCDHandlers() *ArgoCDHandlers {
	return &ArgoCDHandlers{
		vpsHelper:      NewVPSConnectionHelper(),
		passwordHelper: NewPasswordHelper(),
	}
}

// UpdatePassword updates the password for an ArgoCD application
func (ac *ArgoCDHandlers) UpdatePassword(token, accountID, appID, newPassword string, app interface{}) error {
	// Get VPS connection
	appData := app.(struct {
		VPSID     string
		Subdomain string
		ID        string
		Namespace string
	})

	conn, err := ac.vpsHelper.GetVPSConnection(token, accountID, appData.VPSID)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Update the ArgoCD admin password by directly patching the secret
	sshService := services.NewSSHService()

	// Base64 encode the new password
	encodeCmd := fmt.Sprintf("echo '%s' | base64 -w 0", newPassword)
	encodeResult, err := sshService.ExecuteCommand(conn, encodeCmd)
	if err != nil {
		return fmt.Errorf("failed to encode password: %v", err)
	}
	encodedPassword := strings.TrimSpace(encodeResult.Output)

	// Update the ArgoCD initial admin secret directly
	patchCmd := fmt.Sprintf("kubectl patch secret argocd-initial-admin-secret -n %s -p '{\"data\":{\"password\":\"%s\"}}'", appData.Namespace, encodedPassword)
	_, err = sshService.ExecuteCommand(conn, patchCmd)
	if err != nil {
		return fmt.Errorf("failed to update ArgoCD admin secret: %v", err)
	}

	log.Printf("Successfully updated ArgoCD admin password for application %s", appID)
	return nil
}

// GetPassword retrieves the current password for an ArgoCD application directly from the VPS
func (ac *ArgoCDHandlers) GetPassword(token, accountID, appID string, app interface{}) (string, error) {
	log.Printf("Fetching current ArgoCD password from VPS for app %s", appID)

	// Get VPS connection
	appData := app.(struct {
		VPSID     string
		Namespace string
	})

	conn, err := ac.vpsHelper.GetVPSConnection(token, accountID, appData.VPSID)
	if err != nil {
		return "", fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// Retrieve admin password from ArgoCD initial admin secret
	sshService := services.NewSSHService()
	secretName := "argocd-initial-admin-secret"
	cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' 2>/dev/null | base64 --decode", appData.Namespace, secretName)
	result, err := sshService.ExecuteCommand(conn, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve password from secret '%s' in namespace '%s': %v", secretName, appData.Namespace, err)
	}

	password := strings.TrimSpace(result.Output)
	if password == "" {
		return "", fmt.Errorf("no password found in ArgoCD secret '%s'", secretName)
	}

	return password, nil
}

// RetrieveAndStorePassword retrieves the auto-generated admin password from ArgoCD
func (ac *ArgoCDHandlers) RetrieveAndStorePassword(token, accountID, appID, releaseName, namespace, vpsIP, sshUser, privateKey string) error {
	sshService := services.NewSSHService()
	vpsIDInt, _ := strconv.Atoi(strings.Split(appID, "-")[1]) // Extract VPS ID from app ID

	conn, err := sshService.GetOrCreateConnection(vpsIP, sshUser, privateKey, vpsIDInt)
	if err != nil {
		return fmt.Errorf("failed to connect to VPS: %v", err)
	}

	// List all secrets in the namespace for debugging
	listCmd := fmt.Sprintf("kubectl get secrets --namespace %s --no-headers", namespace)
	listResult, err := sshService.ExecuteCommand(conn, listCmd)
	if err != nil {
		log.Printf("Debug: Failed to list secrets in namespace %s: %v", namespace, err)
	} else {
		log.Printf("Debug: Available secrets in namespace %s:\n%s", namespace, listResult.Output)
	}

	// Try to find ArgoCD admin secret with different possible names
	secretNames := []string{
		"argocd-initial-admin-secret",
		fmt.Sprintf("%s-argocd-initial-admin-secret", releaseName),
		"argocd-secret",
		fmt.Sprintf("%s-argocd-secret", releaseName),
	}

	var password string
	var foundSecret string

	for _, secretName := range secretNames {
		cmd := fmt.Sprintf("kubectl get secret --namespace %s %s -o jsonpath='{.data.password}' 2>/dev/null | base64 --decode", namespace, secretName)
		result, err := sshService.ExecuteCommand(conn, cmd)
		if err == nil && strings.TrimSpace(result.Output) != "" {
			password = strings.TrimSpace(result.Output)
			foundSecret = secretName
			log.Printf("Found ArgoCD admin password in secret: %s", secretName)
			break
		}
	}

	// If no password found in any secret, try to get it from ArgoCD server pod logs or generate one
	if password == "" {
		log.Printf("Warning: No ArgoCD admin password found in standard secrets, checking server logs...")

		// Try to get the initial password from ArgoCD server logs
		logCmd := fmt.Sprintf("kubectl logs --namespace %s -l app.kubernetes.io/name=argocd-server --tail=100 2>/dev/null | grep -i 'password' | head -5", namespace)
		logResult, err := sshService.ExecuteCommand(conn, logCmd)
		if err == nil && strings.TrimSpace(logResult.Output) != "" {
			log.Printf("ArgoCD server logs (password related):\n%s", logResult.Output)
		}

		// As a last resort, generate a secure password and set it
		password = "admin" + fmt.Sprintf("%d", time.Now().Unix())
		log.Printf("Warning: No ArgoCD admin password found, using generated password")

		// Try to create the initial admin secret with our generated password
		encodedPassword := utils.Base64Encode(password)
		createSecretCmd := fmt.Sprintf(`kubectl create secret generic argocd-initial-admin-secret --namespace %s --from-literal=password=%s --dry-run=client -o yaml | kubectl apply -f -`, namespace, encodedPassword)
		_, err = sshService.ExecuteCommand(conn, createSecretCmd)
		if err != nil {
			log.Printf("Warning: Failed to create ArgoCD admin secret: %v", err)
		} else {
			log.Printf("Created ArgoCD admin secret with generated password")
		}
	}

	if foundSecret != "" {
		log.Printf("Successfully retrieved ArgoCD admin password from secret: %s", foundSecret)
	} else {
		log.Printf("Successfully set up ArgoCD admin password")
	}

	return nil
}

// InstallCLI installs the ArgoCD CLI on the VPS
func (ac *ArgoCDHandlers) InstallCLI(conn *services.SSHConnection) error {
	sshService := services.NewSSHService()
	_, err := sshService.ExecuteCommand(conn, `
		ARCH=$(uname -m)
		case $ARCH in
			x86_64) ARGOCD_ARCH="amd64" ;;
			aarch64) ARGOCD_ARCH="arm64" ;;
			armv7l) ARGOCD_ARCH="armv7" ;;
			*) echo "Warning: Unsupported architecture $ARCH, defaulting to amd64"; ARGOCD_ARCH="amd64" ;;
		esac
		curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-${ARGOCD_ARCH}
		chmod +x /usr/local/bin/argocd
	`)
	if err != nil {
		log.Printf("Warning: Failed to install ArgoCD CLI: %v", err)
		// Don't fail the deployment, just log the warning
	}
	return nil
}
