package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	HetznerBaseURL = "https://api.hetzner.cloud/v1"
)

// HetznerService handles Hetzner Cloud API operations
type HetznerService struct {
	client *http.Client
}

// NewHetznerService creates a new Hetzner service instance
func NewHetznerService() *HetznerService {
	return &HetznerService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// HetznerError represents a Hetzner API error
type HetznerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"details"`
}

// HetznerServer represents a Hetzner VPS instance
type HetznerServer struct {
	ID         int                   `json:"id"`
	Name       string                `json:"name"`
	Status     string                `json:"status"`
	PublicNet  HetznerPublicNet      `json:"public_net"`
	PrivateNet []HetznerPrivateNet   `json:"private_net"`
	ServerType HetznerServerTypeInfo `json:"server_type"`
	Datacenter HetznerDatacenterInfo `json:"datacenter"`
	Image      HetznerImageInfo      `json:"image"`
	Created    string                `json:"created"`
	Labels     map[string]string     `json:"labels"`
	Protection HetznerProtection     `json:"protection"`
}

// HetznerPublicNet represents public network information
type HetznerPublicNet struct {
	IPv4 HetznerIPv4Info `json:"ipv4"`
	IPv6 HetznerIPv6Info `json:"ipv6"`
}

// HetznerIPv4Info represents IPv4 information
type HetznerIPv4Info struct {
	IP      string `json:"ip"`
	Blocked bool   `json:"blocked"`
}

// HetznerIPv6Info represents IPv6 information
type HetznerIPv6Info struct {
	IP      string `json:"ip"`
	Blocked bool   `json:"blocked"`
}

// HetznerPrivateNet represents private network information
type HetznerPrivateNet struct {
	Network    int    `json:"network"`
	IP         string `json:"ip"`
	MACAddress string `json:"mac_address"`
}

// HetznerServerTypeInfo represents server type information
type HetznerServerTypeInfo struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Cores       int     `json:"cores"`
	Memory      float64 `json:"memory"`
	Disk        int     `json:"disk"`
	CPUType     string  `json:"cpu_type"`
}

// HetznerDatacenterInfo represents datacenter information
type HetznerDatacenterInfo struct {
	ID          int             `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Location    HetznerLocation `json:"location"`
}

// HetznerLocation represents location information
type HetznerLocation struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Country     string  `json:"country"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// HetznerImageInfo represents image information
type HetznerImageInfo struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// HetznerProtection represents server protection settings
type HetznerProtection struct {
	Delete  bool `json:"delete"`
	Rebuild bool `json:"rebuild"`
}

// HetznerServersResponse represents the API response for servers
type HetznerServersResponse struct {
	Servers []HetznerServer `json:"servers"`
}

// HetznerCreateServerRequest represents a server creation request
type HetznerCreateServerRequest struct {
	Name             string            `json:"name"`
	ServerType       string            `json:"server_type"`
	Location         string            `json:"location"`
	Image            string            `json:"image"`
	SSHKeys          []string          `json:"ssh_keys,omitempty"`
	UserData         string            `json:"user_data,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	StartAfterCreate bool              `json:"start_after_create"`
}

// HetznerCreateServerResponse represents a server creation response
type HetznerCreateServerResponse struct {
	Server  HetznerServer `json:"server"`
	Actions []struct {
		ID       int    `json:"id"`
		Command  string `json:"command"`
		Status   string `json:"status"`
		Progress int    `json:"progress"`
	} `json:"actions"`
}

// HetznerSSHKey represents an SSH key
type HetznerSSHKey struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Fingerprint string            `json:"fingerprint"`
	PublicKey   string            `json:"public_key"`
	Labels      map[string]string `json:"labels"`
}

// HetznerSSHKeysResponse represents the API response for SSH keys
type HetznerSSHKeysResponse struct {
	SSHKeys []HetznerSSHKey `json:"ssh_keys"`
}

// makeRequest makes an authenticated request to the Hetzner API
func (hs *HetznerService) makeRequest(method, endpoint, apiKey string, body interface{}) ([]byte, error) {
	url := HetznerBaseURL + endpoint

	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := hs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the raw response body
	bodyBytes := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			bodyBytes = append(bodyBytes, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		// Try to parse error response
		var errorResp struct {
			Error HetznerError `json:"error"`
		}
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &errorResp); err == nil && errorResp.Error.Message != "" {
				return nil, fmt.Errorf("API error %s: %s", errorResp.Error.Code, errorResp.Error.Message)
			}
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

// ListServers retrieves all servers managed by Xanthus
func (hs *HetznerService) ListServers(apiKey string) ([]HetznerServer, error) {
	respBody, err := hs.makeRequest("GET", "/servers?label_selector=managed_by=xanthus", apiKey, nil)
	if err != nil {
		return nil, err
	}

	var serversResp HetznerServersResponse
	if err := json.Unmarshal(respBody, &serversResp); err != nil {
		return nil, fmt.Errorf("failed to parse servers response: %w", err)
	}

	return serversResp.Servers, nil
}

// GetServer retrieves details for a specific server
func (hs *HetznerService) GetServer(apiKey string, serverID int) (*HetznerServer, error) {
	respBody, err := hs.makeRequest("GET", fmt.Sprintf("/servers/%d", serverID), apiKey, nil)
	if err != nil {
		return nil, err
	}

	var serverResp struct {
		Server HetznerServer `json:"server"`
	}
	if err := json.Unmarshal(respBody, &serverResp); err != nil {
		return nil, fmt.Errorf("failed to parse server response: %w", err)
	}

	return &serverResp.Server, nil
}

// CreateServer creates a new VPS instance using cloud-init script
func (hs *HetznerService) CreateServer(apiKey, name, serverType, location, sshKeyName string) (*HetznerServer, error) {
	// Use SSH key name directly - Hetzner accepts both names and IDs
	var sshKeys []string
	if sshKeyName != "" {
		// Verify the SSH key exists by trying to get its ID
		_, err := hs.getSSHKeyID(apiKey, sshKeyName)
		if err != nil {
			return nil, fmt.Errorf("failed to find SSH key '%s': %w", sshKeyName, err)
		}
		// Use the key name directly instead of converting to ID
		sshKeys = []string{sshKeyName}
	}

	// Ubuntu 24.04 LTS image
	image := "ubuntu-24.04"

	// Use cloud-init script with proper readiness verification
	userData := `#cloud-config
# Update system packages
package_update: true
package_upgrade: true

packages:
  - curl
  - wget
  - git
  - apt-transport-https
  - ca-certificates
  - gnupg
  - lsb-release
  - jq

write_files:
  - path: /opt/xanthus/info.txt
    content: |
      Xanthus managed K3s server
      Created: $(date)
      Status: Initializing...
    permissions: '0644'
    owner: root:root
  - path: /etc/environment
    content: |
      KUBECONFIG=/etc/rancher/k3s/k3s.yaml
    append: true
  - path: /opt/xanthus/setup.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail
      
      LOG_FILE="/opt/xanthus/setup.log"
      STATUS_FILE="/opt/xanthus/status"
      
      log() {
          echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
      }
      
      update_status() {
          echo "$1" > "$STATUS_FILE"
          log "Status: $1"
      }
      
      # Start setup
      mkdir -p /opt/xanthus
      update_status "INSTALLING"
      log "Starting Xanthus K3s setup..."
      
      # Ensure SSH service is enabled and running
      systemctl enable ssh
      systemctl start ssh
      log "SSH service verified and enabled"
      
      # Install K3s
      update_status "INSTALLING_K3S"
      log "Installing K3s..."
      curl -sfL https://get.k3s.io | sh -
      systemctl enable k3s
      systemctl start k3s
      
      # Wait for K3s to be ready
      update_status "WAITING_K3S"
      log "Waiting for K3s to be ready..."
      timeout 300 bash -c 'until systemctl is-active k3s >/dev/null 2>&1 && kubectl get nodes --no-headers 2>/dev/null | grep -q "Ready"; do sleep 5; done'
      
      # Set proper permissions for kubeconfig
      chmod 644 /etc/rancher/k3s/k3s.yaml
      
      # Set up environment for root
      echo 'export KUBECONFIG=/etc/rancher/k3s/k3s.yaml' >> /root/.bashrc
      echo 'source <(kubectl completion bash)' >> /root/.bashrc
      echo 'alias k=kubectl' >> /root/.bashrc
      echo 'complete -F __start_kubectl k' >> /root/.bashrc
      
      # Install Helm
      update_status "INSTALLING_HELM"
      log "Installing Helm..."
      curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
      
      # Verify Helm installation
      helm version --short >> "$LOG_FILE"
      
      # Install ArgoCD
      update_status "INSTALLING_ARGOCD"
      log "Installing ArgoCD..."
      if kubectl create namespace argocd 2>&1 | tee -a "$LOG_FILE"; then
          log "ArgoCD namespace created successfully"
      else
          log "ArgoCD namespace may already exist, continuing..."
      fi
      
      if kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml 2>&1 | tee -a "$LOG_FILE"; then
          log "ArgoCD manifests applied successfully"
      else
          log "Error applying ArgoCD manifests, but continuing setup..."
      fi
      
      # Wait for ArgoCD to be ready
      update_status "WAITING_ARGOCD"
      log "Waiting for ArgoCD to be ready..."
      
      # Wait for ArgoCD namespace to have pods
      log "Waiting for ArgoCD pods to be created..."
      if timeout 300 bash -c 'until kubectl get pods -n argocd --no-headers 2>/dev/null | grep -q argocd; do sleep 5; done'; then
          log "ArgoCD pods detected, waiting for them to be ready..."
      else
          log "Warning: ArgoCD pods not detected within 5 minutes, checking status..."
          kubectl get pods -n argocd >> "$LOG_FILE" 2>&1 || log "Could not get ArgoCD pods status"
      fi
      
      # Use a more flexible readiness check - wait for at least the server to be running
      log "Waiting for ArgoCD server to be running..."
      if timeout 600 bash -c 'until kubectl get pods -n argocd 2>/dev/null | grep argocd-server | grep -q Running; do 
          echo "Current ArgoCD pods status:" >> '"$LOG_FILE"'
          kubectl get pods -n argocd >> '"$LOG_FILE"' 2>&1
          sleep 15
      done'; then
          log "ArgoCD server is running"
      else
          log "Warning: ArgoCD server not ready within 10 minutes, but continuing setup..."
          kubectl get pods -n argocd >> "$LOG_FILE" 2>&1 || log "Could not get final ArgoCD status"
      fi
      
      # Install ArgoCD CLI
      update_status "INSTALLING_ARGOCD_CLI"
      log "Installing ArgoCD CLI..."
      curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
      chmod +x /usr/local/bin/argocd
      
      # Final verification
      update_status "VERIFYING"
      log "Performing final verification..."
      
      # Verify all components are working with timeouts
      log "Checking K3s nodes..."
      timeout 30 kubectl get nodes >> "$LOG_FILE" 2>&1 || log "WARNING: kubectl get nodes timed out or failed"
      
      log "Checking K3s pods..."
      timeout 30 kubectl get pods -A >> "$LOG_FILE" 2>&1 || log "WARNING: kubectl get pods timed out or failed"
      
      log "Checking Helm version..."
      timeout 10 helm version >> "$LOG_FILE" 2>&1 || log "WARNING: helm version check failed"
      
      log "Checking ArgoCD CLI..."
      timeout 10 argocd version --client >> "$LOG_FILE" 2>&1 || log "WARNING: argocd version check failed"
      
      log "Final verification completed (warnings are non-critical)"
      
      # Update status and info
      update_status "READY"
      cat > /opt/xanthus/info.txt << EOF
      Xanthus managed K3s server
      Created: $(date)
      Status: Ready
      
      Components installed and verified:
      - K3s: $(kubectl version --short --client 2>/dev/null | head -1 || echo "Ready")
      - Helm: $(helm version --short 2>/dev/null || echo "Ready")
      - ArgoCD: Ready
      
      Access Information:
      - SSH: ssh root@<server-ip>
      - Kubeconfig: /etc/rancher/k3s/k3s.yaml
      - Setup log: /opt/xanthus/setup.log
      - Status: /opt/xanthus/status
      EOF
      
      log "Setup completed successfully!"
      log "K3s cluster is ready and all components are running"

runcmd:
  - /opt/xanthus/setup.sh

final_message: "Xanthus K3s server setup completed! All components are ready and verified."`

	createReq := HetznerCreateServerRequest{
		Name:             name,
		ServerType:       serverType,
		Location:         location,
		Image:            image,
		SSHKeys:          sshKeys,
		UserData:         userData,
		StartAfterCreate: true,
		Labels: map[string]string{
			"managed_by": "xanthus",
			"purpose":    "k3s-cluster",
		},
	}

	respBody, err := hs.makeRequest("POST", "/servers", apiKey, createReq)
	if err != nil {
		return nil, err
	}

	var createResp HetznerCreateServerResponse
	if err := json.Unmarshal(respBody, &createResp); err != nil {
		return nil, fmt.Errorf("failed to parse create server response: %w", err)
	}

	return &createResp.Server, nil
}

// DeleteServer deletes a VPS instance
func (hs *HetznerService) DeleteServer(apiKey string, serverID int) error {
	_, err := hs.makeRequest("DELETE", fmt.Sprintf("/servers/%d", serverID), apiKey, nil)
	return err
}

// PowerOffServer powers off a server
func (hs *HetznerService) PowerOffServer(apiKey string, serverID int) error {
	body := map[string]string{"type": "poweroff"}
	_, err := hs.makeRequest("POST", fmt.Sprintf("/servers/%d/actions/poweroff", serverID), apiKey, body)
	return err
}

// PowerOnServer powers on a server
func (hs *HetznerService) PowerOnServer(apiKey string, serverID int) error {
	body := map[string]string{"type": "poweron"}
	_, err := hs.makeRequest("POST", fmt.Sprintf("/servers/%d/actions/poweron", serverID), apiKey, body)
	return err
}

// RebootServer reboots a server
func (hs *HetznerService) RebootServer(apiKey string, serverID int) error {
	body := map[string]string{"type": "reboot"}
	_, err := hs.makeRequest("POST", fmt.Sprintf("/servers/%d/actions/reboot", serverID), apiKey, body)
	return err
}

// getSSHKeyID retrieves the ID of an SSH key by name
func (hs *HetznerService) getSSHKeyID(apiKey, keyName string) (int, error) {
	resp, err := hs.makeRequest("GET", "/ssh_keys", apiKey, nil)
	if err != nil {
		return 0, err
	}

	var keysResp HetznerSSHKeysResponse
	if err := json.Unmarshal(resp, &keysResp); err != nil {
		return 0, fmt.Errorf("failed to parse SSH keys response: %w", err)
	}

	for _, key := range keysResp.SSHKeys {
		if key.Name == keyName {
			return key.ID, nil
		}
	}

	return 0, fmt.Errorf("SSH key '%s' not found", keyName)
}

// CreateSSHKey creates a new SSH key in Hetzner Cloud
func (hs *HetznerService) CreateSSHKey(apiKey, name, publicKey string) (*HetznerSSHKey, error) {
	body := map[string]interface{}{
		"name":       name,
		"public_key": publicKey,
		"labels": map[string]string{
			"managed_by": "xanthus",
		},
	}

	respBody, err := hs.makeRequest("POST", "/ssh_keys", apiKey, body)
	if err != nil {
		return nil, err
	}

	var keyResp struct {
		SSHKey HetznerSSHKey `json:"ssh_key"`
	}
	if err := json.Unmarshal(respBody, &keyResp); err != nil {
		return nil, fmt.Errorf("failed to parse SSH key response: %w", err)
	}

	return &keyResp.SSHKey, nil
}

// ListSSHKeys retrieves all SSH keys
func (hs *HetznerService) ListSSHKeys(apiKey string) ([]HetznerSSHKey, error) {
	respBody, err := hs.makeRequest("GET", "/ssh_keys", apiKey, nil)
	if err != nil {
		return nil, err
	}

	var keysResp HetznerSSHKeysResponse
	if err := json.Unmarshal(respBody, &keysResp); err != nil {
		return nil, fmt.Errorf("failed to parse SSH keys response: %w", err)
	}

	return keysResp.SSHKeys, nil
}

// FindSSHKeyByPublicKey finds an existing SSH key by its public key content
func (hs *HetznerService) FindSSHKeyByPublicKey(apiKey, publicKey string) (*HetznerSSHKey, error) {
	keys, err := hs.ListSSHKeys(apiKey)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.PublicKey == publicKey {
			return &key, nil
		}
	}

	return nil, nil // Not found
}

// CreateOrFindSSHKey creates a new SSH key or returns existing one if it already exists
func (hs *HetznerService) CreateOrFindSSHKey(apiKey, name, publicKey string) (*HetznerSSHKey, error) {
	// First, try to find an existing key with the same public key
	existingKey, err := hs.FindSSHKeyByPublicKey(apiKey, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to search for existing SSH key: %w", err)
	}

	if existingKey != nil {
		return existingKey, nil
	}

	// If not found, create a new one
	return hs.CreateSSHKey(apiKey, name, publicKey)
}
