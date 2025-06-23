package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	PublicNet   HetznerPublicNet       `json:"public_net"`
	PrivateNet  []HetznerPrivateNet    `json:"private_net"`
	ServerType  HetznerServerTypeInfo  `json:"server_type"`
	Datacenter  HetznerDatacenterInfo  `json:"datacenter"`
	Image       HetznerImageInfo       `json:"image"`
	Created     string                 `json:"created"`
	Labels      map[string]string      `json:"labels"`
	Protection  HetznerProtection      `json:"protection"`
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
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Location    HetznerLocation   `json:"location"`
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
	Delete   bool `json:"delete"`
	Rebuild  bool `json:"rebuild"`
}

// HetznerServersResponse represents the API response for servers
type HetznerServersResponse struct {
	Servers []HetznerServer `json:"servers"`
}

// HetznerCreateServerRequest represents a server creation request
type HetznerCreateServerRequest struct {
	Name       string            `json:"name"`
	ServerType string            `json:"server_type"`
	Location   string            `json:"location"`
	Image      string            `json:"image"`
	SSHKeys    []string          `json:"ssh_keys,omitempty"`
	UserData   string            `json:"user_data,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	StartAfterCreate bool        `json:"start_after_create"`
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

// CreateServer creates a new VPS instance with SSL certificates and K3s
func (hs *HetznerService) CreateServer(apiKey, name, serverType, location, sshKeyName, sslCert, sslKey string) (*HetznerServer, error) {
	// Get SSH key ID if provided
	var sshKeys []string
	if sshKeyName != "" {
		sshKeyID, err := hs.getSSHKeyID(apiKey, sshKeyName)
		if err != nil {
			return nil, fmt.Errorf("failed to get SSH key: %w", err)
		}
		sshKeys = []string{fmt.Sprintf("%d", sshKeyID)}
	}

	// Ubuntu 24.04 LTS image
	image := "ubuntu-24.04"

	// Escape the SSL certificate and key for YAML
	escapedCert := hs.escapeForYAML(sslCert)
	escapedKey := hs.escapeForYAML(sslKey)

	// Cloud-init user data for K3s with SSL certificates
	userData := fmt.Sprintf(`#cloud-config
packages:
  - curl
  - wget
  - git
  - apt-transport-https
  - ca-certificates
  - gnupg
  - lsb-release

write_files:
  - path: /opt/xanthus/ssl/server.crt
    content: |
%s
    permissions: '0600'
    owner: root:root
  - path: /opt/xanthus/ssl/server.key
    content: |
%s
    permissions: '0600'
    owner: root:root
  - path: /opt/xanthus/info.txt
    content: |
      Xanthus managed K3s server
      Created: $(date)
      SSL certificates installed
    permissions: '0644'
    owner: root:root

runcmd:
  - mkdir -p /opt/xanthus/ssl
  - curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--write-kubeconfig-mode 644 --tls-cert-file /opt/xanthus/ssl/server.crt --tls-private-key-file /opt/xanthus/ssl/server.key" sh -
  - systemctl enable k3s
  - systemctl start k3s
  - echo "K3s installed with custom SSL certificates" >> /opt/xanthus/info.txt

final_message: "Xanthus K3s server with SSL certificates is ready!"
`, escapedCert, escapedKey)

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

// escapeForYAML properly indents and escapes content for YAML
func (hs *HetznerService) escapeForYAML(content string) string {
	lines := strings.Split(content, "\n")
	var escapedLines []string
	for _, line := range lines {
		if line != "" {
			escapedLines = append(escapedLines, "      "+line)
		}
	}
	return strings.Join(escapedLines, "\n")
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