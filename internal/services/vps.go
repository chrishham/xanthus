package services

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// VPSService provides high-level business logic for VPS operations
type VPSService struct {
	hetzner *HetznerService
	kv      *KVService
	ssh     *SSHService
	cf      *CloudflareService
	cache   *CacheService
}

// NewVPSService creates a new VPS service instance
func NewVPSService() *VPSService {
	return &VPSService{
		hetzner: NewHetznerService(),
		kv:      NewKVService(),
		ssh:     NewSSHService(),
		cf:      NewCloudflareService(),
		cache:   NewCacheService(),
	}
}

// EnhancedVPS represents a VPS with additional cost and status information
type EnhancedVPS struct {
	*VPSConfig
	AccumulatedCost float64           `json:"accumulated_cost"`
	Server          *HetznerServer    `json:"server,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
}

// GetVPSWithCosts retrieves VPS configuration with calculated costs
func (vs *VPSService) GetVPSWithCosts(token, accountID string, serverID int) (*EnhancedVPS, error) {
	// Get VPS configuration
	vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPS config: %w", err)
	}

	// Calculate accumulated costs
	accumulatedCost, err := vs.kv.CalculateVPSCosts(vpsConfig)
	if err != nil {
		log.Printf("Warning: Could not calculate costs for VPS %d: %v", serverID, err)
		accumulatedCost = 0
	}

	enhanced := &EnhancedVPS{
		VPSConfig:       vpsConfig,
		AccumulatedCost: accumulatedCost,
		Labels:          make(map[string]string),
	}

	// Add cost labels for compatibility with existing code
	enhanced.Labels["accumulated_cost"] = fmt.Sprintf("%.2f", accumulatedCost)
	enhanced.Labels["monthly_cost"] = fmt.Sprintf("%.2f", vpsConfig.MonthlyRate)
	enhanced.Labels["hourly_cost"] = fmt.Sprintf("%.4f", vpsConfig.HourlyRate)

	return enhanced, nil
}

// ValidateVPSAccess validates that a VPS exists and user has access
func (vs *VPSService) ValidateVPSAccess(token, accountID string, serverID int) (*VPSConfig, error) {
	return vs.kv.GetVPSConfig(token, accountID, serverID)
}

// EnhanceServersWithCosts adds cost information and application counts to a list of Hetzner servers
func (vs *VPSService) EnhanceServersWithCosts(token, accountID string, servers []HetznerServer) error {
	// Get application counts for all VPS instances
	appCounts, err := vs.getApplicationCountsPerVPS(token, accountID)
	if err != nil {
		log.Printf("Warning: Could not get application counts: %v", err)
		appCounts = make(map[string]int)
	}

	for i := range servers {
		// Get VPS configuration if it exists
		if vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, servers[i].ID); err == nil {
			// Calculate accumulated cost
			if accumulatedCost, err := vs.kv.CalculateVPSCosts(vpsConfig); err == nil {
				// Initialize labels map if needed
				if servers[i].Labels == nil {
					servers[i].Labels = make(map[string]string)
				}

				// Add cost information to server labels
				servers[i].Labels["accumulated_cost"] = fmt.Sprintf("%.2f", accumulatedCost)
				servers[i].Labels["monthly_cost"] = fmt.Sprintf("%.2f", vpsConfig.MonthlyRate)
				servers[i].Labels["hourly_cost"] = fmt.Sprintf("%.4f", vpsConfig.HourlyRate)
				servers[i].Labels["configured_timezone"] = vpsConfig.Timezone
				servers[i].Labels["provider"] = vpsConfig.Provider
				servers[i].Labels["managed_by"] = "xanthus"

				// Add application count
				vpsIDStr := fmt.Sprintf("%d", servers[i].ID)
				applicationCount := appCounts[vpsIDStr]
				servers[i].Labels["application_count"] = fmt.Sprintf("%d", applicationCount)
			}
		}
	}
	return nil
}

// CreateVPSWithConfig creates a VPS and stores its configuration
func (vs *VPSService) CreateVPSWithConfig(
	token, accountID, hetznerKey string,
	name, serverType, location, domain string,
	sshKeyName, sshPublicKey string,
	domainCert, domainKey string,
	hourlyRate, monthlyRate float64,
) (*HetznerServer, *VPSConfig, error) {
	// Calculate timezone for the location
	timezone := vs.getTimezoneForLocation(location)

	// Create server using Hetzner service
	server, err := vs.hetzner.CreateServer(hetznerKey, name, serverType, location, sshKeyName, domain, domainCert, domainKey, timezone)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Create VPS configuration
	vpsConfig := &VPSConfig{
		ServerID:    server.ID,
		Name:        server.Name,
		ServerType:  serverType,
		Location:    location,
		PublicIPv4:  server.PublicNet.IPv4.IP,
		CreatedAt:   server.Created,
		SSHKeyName:  sshKeyName,
		SSHUser:     "root",
		SSHPort:     22,
		HourlyRate:  hourlyRate,
		MonthlyRate: monthlyRate,
		Timezone:    timezone,
		Provider:    "Hetzner",
	}

	// Store VPS configuration
	if err := vs.kv.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		log.Printf("Warning: Failed to store VPS config: %v", err)
		// Don't fail the creation, just log the warning
	}

	return server, vpsConfig, nil
}

// DeleteVPSAndCleanup deletes a VPS and cleans up its configuration and all associated applications
func (vs *VPSService) DeleteVPSAndCleanup(token, accountID, hetznerKey string, serverID int) (*VPSConfig, error) {
	// Get VPS configuration before deletion (for logging)
	vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Warning: Could not get VPS config for server %d: %v", serverID, err)
	}

	// Delete all applications associated with this VPS
	if err := vs.deleteAssociatedApplications(token, accountID, fmt.Sprintf("%d", serverID)); err != nil {
		log.Printf("Warning: Failed to delete associated applications for VPS %d: %v", serverID, err)
		// Continue with VPS deletion even if application cleanup fails
	}

	// Delete server from Hetzner
	if err := vs.hetzner.DeleteServer(hetznerKey, serverID); err != nil {
		return vpsConfig, fmt.Errorf("failed to delete server: %w", err)
	}

	// Clean up VPS configuration from KV
	if err := vs.kv.DeleteVPSConfig(token, accountID, serverID); err != nil {
		log.Printf("Warning: Could not delete VPS config for server %d: %v", serverID, err)
		// Don't fail the deletion, just log the warning
	}

	return vpsConfig, nil
}

// GetServersFromKV retrieves server list from KV store instead of Hetzner API
func (vs *VPSService) GetServersFromKV(token, accountID string) ([]HetznerServer, error) {
	// Check cache first - use accountID for proper user isolation
	cacheKey := "vps_servers:" + accountID
	if cached, exists := vs.cache.Get(cacheKey); exists {
		return cached.([]HetznerServer), nil
	}

	// Get all VPS configurations from KV
	vpsConfigsMap, err := vs.kv.ListVPSConfigs(token, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPS configs from KV: %w", err)
	}

	// Get application counts for each VPS
	// DISABLED: Application counting causes 8+ second delays
	// TODO: Move to background job or async endpoint
	// appCounts, err := vs.getApplicationCountsPerVPS(token, accountID)
	// if err != nil {
	// 	log.Printf("Warning: Could not get application counts: %v", err)
	// 	appCounts = make(map[string]int)
	// }
	appCounts := make(map[string]int) // Empty map for now

	// Convert VPS configs to HetznerServer format for compatibility
	servers := make([]HetznerServer, 0, len(vpsConfigsMap))
	for _, vpsConfig := range vpsConfigsMap {
		// Calculate accumulated cost
		accumulatedCost, err := vs.kv.CalculateVPSCosts(vpsConfig)
		if err != nil {
			log.Printf("Warning: Could not calculate costs for VPS %d: %v", vpsConfig.ServerID, err)
			accumulatedCost = 0
		}

		// Get application count for this VPS
		vpsIDStr := fmt.Sprintf("%d", vpsConfig.ServerID)
		applicationCount := appCounts[vpsIDStr]

		// Create HetznerServer from VPS config for UI compatibility
		server := HetznerServer{
			ID:      vpsConfig.ServerID,
			Name:    vpsConfig.Name,
			Status:  "unknown", // Status will be fetched from live Hetzner API
			Created: vpsConfig.CreatedAt,
			ServerType: HetznerServerTypeInfo{
				Name: vpsConfig.ServerType,
				// Add other fields as needed
			},
			Datacenter: HetznerDatacenterInfo{
				Location: HetznerLocation{
					Description: vpsConfig.Location,
				},
			},
			PublicNet: HetznerPublicNet{
				IPv4: HetznerIPv4Info{
					IP: vpsConfig.PublicIPv4,
				},
			},
			Labels: map[string]string{
				"managed_by":        "xanthus",
				"accumulated_cost":  fmt.Sprintf("%.2f", accumulatedCost),
				"monthly_cost":      fmt.Sprintf("%.2f", vpsConfig.MonthlyRate),
				"hourly_cost":       fmt.Sprintf("%.4f", vpsConfig.HourlyRate),
				"provider":          vpsConfig.Provider,
				"application_count": fmt.Sprintf("%d", applicationCount),
			},
		}

		servers = append(servers, server)
	}

	// Cache the result for 60 seconds
	vs.cache.Set(cacheKey, servers, 60*time.Second)

	return servers, nil
}

// getTimezoneForLocation maps Hetzner datacenter locations to appropriate timezones
func (vs *VPSService) getTimezoneForLocation(location string) string {
	locationTimezones := map[string]string{
		"nbg1":    "Europe/Berlin",    // Nuremberg, Germany
		"fsn1":    "Europe/Berlin",    // Falkenstein, Germany
		"hel1":    "Europe/Helsinki",  // Helsinki, Finland
		"ash":     "America/New_York", // Ashburn, USA
		"hil":     "America/New_York", // Hillsboro, USA
		"cax":     "America/New_York", // Central US
		"default": "Europe/Athens",    // Default for Greece-based deployments
	}

	// Extract location prefix (e.g., "nbg1" from "nbg1-dc3")
	for prefix, timezone := range locationTimezones {
		if strings.HasPrefix(location, prefix) {
			return timezone
		}
	}

	// Default fallback
	return locationTimezones["default"]
}

// UpdateVPSTimezone updates the timezone for an existing VPS configuration
func (vs *VPSService) UpdateVPSTimezone(token, accountID string, serverID int) error {
	// Get existing VPS configuration
	vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		return fmt.Errorf("failed to get VPS config: %w", err)
	}

	// Update timezone based on location
	vpsConfig.Timezone = vs.getTimezoneForLocation(vpsConfig.Location)

	// Store updated configuration
	if err := vs.kv.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		return fmt.Errorf("failed to update VPS config: %w", err)
	}

	log.Printf("✅ Updated timezone for VPS %d to %s", serverID, vpsConfig.Timezone)
	return nil
}

// deleteAssociatedApplications deletes all applications associated with a VPS
func (vs *VPSService) deleteAssociatedApplications(token, accountID, vpsID string) error {
	// Use the application service to get all applications
	appService := NewSimpleApplicationService()

	// Get all applications for this account
	applications, err := appService.ListApplications(token, accountID)
	if err != nil {
		return fmt.Errorf("failed to list applications: %w", err)
	}

	// Filter applications that belong to this VPS
	var appsToDelete []string
	for _, app := range applications {
		if app.VPSID == vpsID {
			appsToDelete = append(appsToDelete, app.ID)
			log.Printf("Found application %s (%s) on VPS %s - will be deleted", app.Name, app.ID, vpsID)
		}
	}

	// Delete each application (VPS deletion mode - DNS and KV only)
	var deletionErrors []string
	for _, appID := range appsToDelete {
		log.Printf("Deleting application %s from VPS %s (DNS and KV only)", appID, vpsID)
		if err := appService.DeleteApplicationForVPSDeletion(token, accountID, appID); err != nil {
			deletionErrors = append(deletionErrors, fmt.Sprintf("failed to delete application %s: %v", appID, err))
			log.Printf("Error deleting application %s: %v", appID, err)
		} else {
			log.Printf("Successfully deleted application %s (DNS and KV cleanup)", appID)
		}
	}

	if len(deletionErrors) > 0 {
		return fmt.Errorf("some applications failed to delete: %s", strings.Join(deletionErrors, "; "))
	}

	if len(appsToDelete) > 0 {
		log.Printf("Successfully deleted %d applications from VPS %s", len(appsToDelete), vpsID)
	} else {
		log.Printf("No applications found on VPS %s to delete", vpsID)
	}

	return nil
}

// getApplicationCountsPerVPS counts applications for each VPS
func (vs *VPSService) getApplicationCountsPerVPS(token, accountID string) (map[string]int, error) {
	// Use the application service to get all applications
	appService := NewSimpleApplicationService()

	// Get all applications for this account
	applications, err := appService.ListApplications(token, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	// Get all VPS configurations to create a mapping between VPS names and IDs
	vpsConfigsMap, err := vs.kv.ListVPSConfigs(token, accountID)
	if err != nil {
		log.Printf("Warning: Could not get VPS configs for application counting: %v", err)
		vpsConfigsMap = make(map[int]*VPSConfig)
	}

	// Create mapping from VPS name to VPS ID
	vpsNameToID := make(map[string]string)
	for _, vpsConfig := range vpsConfigsMap {
		vpsNameToID[vpsConfig.Name] = fmt.Sprintf("%d", vpsConfig.ServerID)
	}

	// Count applications per VPS
	counts := make(map[string]int)
	for _, app := range applications {
		var targetVPSID string

		// First try to use VPSID directly (for numeric IDs)
		if app.VPSID != "" {
			// Check if VPSID is numeric (convert to int and back to string)
			if vpsIDInt, err := strconv.Atoi(app.VPSID); err == nil {
				targetVPSID = fmt.Sprintf("%d", vpsIDInt)
			} else {
				// VPSID is not numeric, try to map it as a VPS name
				if vpsID, exists := vpsNameToID[app.VPSID]; exists {
					targetVPSID = vpsID
				}
			}
		}

		// If no VPSID or couldn't resolve it, try VPS name
		if targetVPSID == "" && app.VPSName != "" {
			if vpsID, exists := vpsNameToID[app.VPSName]; exists {
				targetVPSID = vpsID
			}
		}

		// Count the application for the resolved VPS ID
		if targetVPSID != "" {
			counts[targetVPSID]++
		}
	}

	return counts, nil
}

// CreateOCIVPSConfig creates a VPS configuration for a manually added OCI instance
func (vs *VPSService) CreateOCIVPSConfig(
	token, accountID string,
	name, publicIP, username, shape string,
	serverID int, privateKey, publicKey string,
) (*VPSConfig, error) {
	// Create VPS configuration for OCI
	vpsConfig := &VPSConfig{
		ServerID:    serverID,
		Name:        name,
		ServerType:  shape,
		Location:    "oracle-cloud",
		PublicIPv4:  publicIP,
		CreatedAt:   time.Now().Format(time.RFC3339),
		SSHKeyName:  "xanthus-oci-key",
		SSHUser:     username,
		SSHPort:     22,
		HourlyRate:  0.0,   // OCI instances are managed externally
		MonthlyRate: 0.0,   // Cost tracking handled outside Xanthus
		Timezone:    "UTC", // Default timezone, can be updated later
		Provider:    "Oracle Cloud Infrastructure (OCI)",
	}

	// Store VPS configuration
	if err := vs.kv.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		return nil, fmt.Errorf("failed to store OCI VPS config: %w", err)
	}

	// Store SSH private key for this VPS
	sshKeyData := struct {
		PrivateKey string `json:"private_key"`
		PublicKey  string `json:"public_key"`
		VPSName    string `json:"vps_name"`
		Provider   string `json:"provider"`
		CreatedAt  string `json:"created_at"`
	}{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		VPSName:    name,
		Provider:   "oci",
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	kvKey := fmt.Sprintf("vps:%d:ssh", serverID)
	if err := vs.kv.PutValue(token, accountID, kvKey, sshKeyData); err != nil {
		log.Printf("Warning: Failed to store SSH key for OCI VPS %d: %v", serverID, err)
	}

	// Start K3s setup in the background
	go vs.setupOCIK3s(token, accountID, vpsConfig, privateKey)

	return vpsConfig, nil
}

// setupOCIK3s sets up K3s on the OCI instance in the background
func (vs *VPSService) setupOCIK3s(token, accountID string, vpsConfig *VPSConfig, privateKey string) {
	log.Printf("Starting K3s setup for OCI instance %s (ID: %d)", vpsConfig.Name, vpsConfig.ServerID)

	// Create SSH connection to the OCI instance
	sshConn, err := vs.ssh.ConnectToVPS(vpsConfig.PublicIPv4, vpsConfig.SSHUser, privateKey)
	if err != nil {
		log.Printf("Error connecting to OCI instance %s: %v", vpsConfig.Name, err)
		return
	}
	defer sshConn.Close()

	// Update system packages
	log.Printf("Updating system packages on OCI instance %s", vpsConfig.Name)
	if _, err := vs.ssh.ExecuteCommand(sshConn, "sudo apt update && sudo apt upgrade -y"); err != nil {
		log.Printf("Warning: Failed to update packages on OCI instance %s: %v", vpsConfig.Name, err)
	}

	// Install K3s
	log.Printf("Installing K3s on OCI instance %s", vpsConfig.Name)
	k3sInstallCommand := "curl -sfL https://get.k3s.io | sudo sh -s - --write-kubeconfig-mode 644"
	if _, err := vs.ssh.ExecuteCommand(sshConn, k3sInstallCommand); err != nil {
		log.Printf("Error installing K3s on OCI instance %s: %v", vpsConfig.Name, err)
		return
	}

	// Wait for K3s to be ready
	log.Printf("Waiting for K3s to be ready on OCI instance %s", vpsConfig.Name)
	readyCommand := "sudo k3s kubectl wait --for=condition=Ready nodes --all --timeout=300s"
	if _, err := vs.ssh.ExecuteCommand(sshConn, readyCommand); err != nil {
		log.Printf("Warning: K3s readiness check timeout on OCI instance %s: %v", vpsConfig.Name, err)
	}

	// Install Helm
	log.Printf("Installing Helm on OCI instance %s", vpsConfig.Name)
	helmInstallCommand := "curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"
	if _, err := vs.ssh.ExecuteCommand(sshConn, helmInstallCommand); err != nil {
		log.Printf("Warning: Failed to install Helm on OCI instance %s: %v", vpsConfig.Name, err)
	}

	log.Printf("✅ K3s setup completed for OCI instance %s (ID: %d)", vpsConfig.Name, vpsConfig.ServerID)
}
