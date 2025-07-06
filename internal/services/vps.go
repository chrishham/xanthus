package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// VPSService provides high-level business logic for VPS operations
type VPSService struct {
	hetzner  *HetznerService
	kv       *KVService
	ssh      *SSHService
	cf       *CloudflareService
	cache    *CacheService
	provider *ProviderResolver
}

// NewVPSService creates a new VPS service instance
func NewVPSService() *VPSService {
	kvService := NewKVService()
	return &VPSService{
		hetzner:  NewHetznerService(),
		kv:       kvService,
		ssh:      NewSSHService(),
		cf:       NewCloudflareService(),
		cache:    NewCacheService(),
		provider: NewProviderResolver(kvService),
	}
}

// CreateOCIVPSWithConfig creates an OCI VPS instance with full configuration
func (vs *VPSService) CreateOCIVPSWithConfig(
	token, accountID, ociAuthToken,
	name, shape, region,
	sshPublicKey, timezone string,
	hourlyRate, monthlyRate float64,
) (*OCIInstance, *VPSConfig, error) {
	// Create OCI service
	ociService, err := NewOCIService(ociAuthToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OCI service: %w", err)
	}

	// Create VPS instance
	instance, err := ociService.CreateVPSWithK3s(context.Background(), name, shape, sshPublicKey, timezone)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OCI instance: %w", err)
	}

	// Create VPS configuration
	vpsConfig := &VPSConfig{
		ServerID:           vs.parseOCIInstanceID(instance.ID),
		Name:               instance.DisplayName,
		Provider:           "Oracle Cloud Infrastructure (OCI)",
		Location:           region,
		ServerType:         shape,
		PublicIPv4:         instance.PublicIP,
		CreatedAt:          time.Now().Format(time.RFC3339),
		Timezone:           timezone,
		SSHUser:            "ubuntu", // OCI default
		SSHPort:            22,
		HourlyRate:         hourlyRate,
		MonthlyRate:        monthlyRate,
		ProviderInstanceID: instance.ID,
	}

	// Store VPS configuration
	err = vs.kv.StoreVPSConfig(token, accountID, vpsConfig)
	if err != nil {
		// Attempt to cleanup the created instance if config storage fails
		_ = ociService.TerminateInstance(context.Background(), instance.ID)
		return nil, nil, fmt.Errorf("failed to store VPS configuration: %w", err)
	}

	return instance, vpsConfig, nil
}

// parseOCIInstanceID converts OCI instance OCID to integer for storage compatibility
func (vs *VPSService) parseOCIInstanceID(instanceID string) int {
	// Since OCI uses OCIDs (not simple integers), we'll use a hash approach
	// This is a simple approach - in production you might want a more sophisticated mapping
	hash := 0
	for _, char := range instanceID {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	// Use last 8 digits to avoid overflow and ensure uniqueness within reasonable bounds
	return hash % 100000000
}

// DeleteOCIVPS deletes an OCI VPS instance and cleans up configuration
func (vs *VPSService) DeleteOCIVPS(token, accountID, ociAuthToken string, serverID int) error {
	// Get VPS configuration to get the actual OCI instance ID
	vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		return fmt.Errorf("failed to get VPS config: %w", err)
	}

	// We need to store the actual OCI instance ID in the config for proper deletion
	// For now, this is a limitation - we'll need to improve the storage approach

	// Create OCI service
	ociService, err := NewOCIService(ociAuthToken)
	if err != nil {
		return fmt.Errorf("failed to create OCI service: %w", err)
	}

	// List instances to find the one with matching name/IP
	instances, err := ociService.ListInstances(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list OCI instances: %w", err)
	}

	var instanceID string
	for _, instance := range instances {
		if instance.DisplayName == vpsConfig.Name || instance.PublicIP == vpsConfig.PublicIPv4 {
			instanceID = instance.ID
			break
		}
	}

	if instanceID == "" {
		return fmt.Errorf("could not find OCI instance for VPS %d", serverID)
	}

	// Delete the instance
	err = ociService.TerminateInstance(context.Background(), instanceID)
	if err != nil {
		return fmt.Errorf("failed to terminate OCI instance: %w", err)
	}

	// Remove VPS configuration
	err = vs.kv.DeleteVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Warning: Failed to delete VPS config for %d: %v", serverID, err)
	}

	return nil
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
	timezone := vs.provider.ResolveTimezone("Hetzner", location)

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
		SSHUser:     vs.provider.GetProviderDefaults("Hetzner").DefaultSSHUser,
		SSHPort:     vs.provider.GetProviderDefaults("Hetzner").DefaultSSHPort,
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

	// Get application counts for each VPS (re-enabled with performance optimization)
	appCounts, err := vs.getApplicationCountsPerVPS(token, accountID)
	if err != nil {
		log.Printf("Warning: Could not get application counts: %v", err)
		appCounts = make(map[string]int)
	}

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

		// Get resource specs based on provider and server type
		resourceSpecs := vs.getResourceSpecs(vpsConfig.Provider, vpsConfig.ServerType)

		// Create HetznerServer from VPS config for UI compatibility
		server := HetznerServer{
			ID:      vpsConfig.ServerID,
			Name:    vpsConfig.Name,
			Status:  "unknown", // Status will be fetched from live Hetzner API
			Created: vpsConfig.CreatedAt,
			ServerType: HetznerServerTypeInfo{
				Name:        vpsConfig.ServerType,
				Description: resourceSpecs.Description,
				Cores:       resourceSpecs.Cores,
				Memory:      resourceSpecs.Memory,
				Disk:        resourceSpecs.Disk,
				CPUType:     resourceSpecs.CPUType,
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
				"managed_by":           "xanthus",
				"accumulated_cost":     fmt.Sprintf("%.2f", accumulatedCost),
				"monthly_cost":         fmt.Sprintf("%.2f", vpsConfig.MonthlyRate),
				"hourly_cost":          fmt.Sprintf("%.4f", vpsConfig.HourlyRate),
				"provider":             vpsConfig.Provider,
				"application_count":    fmt.Sprintf("%d", applicationCount),
				"configured_timezone":  vpsConfig.Timezone,
			},
		}

		servers = append(servers, server)
	}

	// Cache the result for 60 seconds
	vs.cache.Set(cacheKey, servers, 60*time.Second)

	return servers, nil
}

// UpdateVPSTimezone updates the timezone for an existing VPS configuration
func (vs *VPSService) UpdateVPSTimezone(token, accountID string, serverID int) error {
	// Get existing VPS configuration
	vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		return fmt.Errorf("failed to get VPS config: %w", err)
	}

	// Update timezone based on location
	vpsConfig.Timezone = vs.provider.ResolveTimezone(vpsConfig.Provider, vpsConfig.Location)

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
	// Default to Frankfurt region for Oracle Cloud (most common for European users)
	location := "eu-frankfurt-1"
	
	// Resolve timezone for Oracle Cloud
	timezone := vs.provider.ResolveTimezone("Oracle Cloud Infrastructure (OCI)", location)

	// Create VPS configuration for OCI
	vpsConfig := &VPSConfig{
		ServerID:    serverID,
		Name:        name,
		ServerType:  shape,
		Location:    location,
		PublicIPv4:  publicIP,
		CreatedAt:   time.Now().Format(time.RFC3339),
		SSHKeyName:  "xanthus-oci-key",
		SSHUser:     username,
		SSHPort:     22,
		HourlyRate:  0.0, // OCI instances are managed externally
		MonthlyRate: 0.0, // Cost tracking handled outside Xanthus
		Timezone:    timezone,
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
		Provider:   "Oracle Cloud Infrastructure (OCI)",
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

	// Set up KUBECONFIG environment variable for both ubuntu and root users
	log.Printf("Setting up KUBECONFIG environment for OCI instance %s", vpsConfig.Name)

	// Set up for ubuntu user
	kubeconfigSetupUbuntu := `echo 'export KUBECONFIG=/etc/rancher/k3s/k3s.yaml' >> /home/ubuntu/.bashrc && 
echo 'source <(kubectl completion bash)' >> /home/ubuntu/.bashrc && 
echo 'alias k=kubectl' >> /home/ubuntu/.bashrc && 
echo 'complete -F __start_kubectl k' >> /home/ubuntu/.bashrc`
	if _, err := vs.ssh.ExecuteCommand(sshConn, kubeconfigSetupUbuntu); err != nil {
		log.Printf("Warning: Failed to set up KUBECONFIG for ubuntu user on OCI instance %s: %v", vpsConfig.Name, err)
	}

	// Set up for root user (for sudo operations)
	kubeconfigSetupRoot := `sudo sh -c 'echo "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml" >> /root/.bashrc && 
echo "source <(kubectl completion bash)" >> /root/.bashrc && 
echo "alias k=kubectl" >> /root/.bashrc && 
echo "complete -F __start_kubectl k" >> /root/.bashrc'`
	if _, err := vs.ssh.ExecuteCommand(sshConn, kubeconfigSetupRoot); err != nil {
		log.Printf("Warning: Failed to set up KUBECONFIG for root user on OCI instance %s: %v", vpsConfig.Name, err)
	}

	// Set up globally in environment
	globalKubeconfigSetup := `sudo sh -c 'echo "KUBECONFIG=/etc/rancher/k3s/k3s.yaml" >> /etc/environment'`
	if _, err := vs.ssh.ExecuteCommand(sshConn, globalKubeconfigSetup); err != nil {
		log.Printf("Warning: Failed to set up global KUBECONFIG on OCI instance %s: %v", vpsConfig.Name, err)
	}

	// Install Helm
	log.Printf("Installing Helm on OCI instance %s", vpsConfig.Name)
	helmInstallCommand := "curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"
	if _, err := vs.ssh.ExecuteCommand(sshConn, helmInstallCommand); err != nil {
		log.Printf("Warning: Failed to install Helm on OCI instance %s: %v", vpsConfig.Name, err)
	}

	log.Printf("✅ K3s setup completed for OCI instance %s (ID: %d)", vpsConfig.Name, vpsConfig.ServerID)
}

// ResolveSSHUser resolves the SSH user for a VPS using the provider resolver
func (vs *VPSService) ResolveSSHUser(token, accountID string, serverID int) (string, error) {
	return vs.provider.ResolveSSHUser(token, accountID, serverID)
}

// GetProviderDefaults returns provider defaults for a given provider string
func (vs *VPSService) GetProviderDefaults(provider string) *ProviderDefaults {
	return vs.provider.GetProviderDefaults(provider)
}

// GetCorrectSSHUserFromProvider returns the correct SSH user for a provider (ignoring stored config)
func (vs *VPSService) GetCorrectSSHUserFromProvider(provider string) string {
	defaults := vs.provider.GetProviderDefaults(provider)
	return defaults.DefaultSSHUser
}

// UpdateVPSSSHUser updates the SSH user for a VPS configuration based on provider defaults
func (vs *VPSService) UpdateVPSSSHUser(token, accountID string, serverID int) error {
	// Get existing VPS configuration
	vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		return fmt.Errorf("failed to get VPS config: %w", err)
	}

	// Resolve correct SSH user based on provider (always use provider defaults)
	correctSSHUser := vs.provider.GetCorrectSSHUserFromConfig(vpsConfig)

	// Update if different
	if vpsConfig.SSHUser != correctSSHUser {
		log.Printf("🔧 Updating VPS %d SSH user from '%s' to '%s' (provider: %s)",
			serverID, vpsConfig.SSHUser, correctSSHUser, vpsConfig.Provider)

		vpsConfig.SSHUser = correctSSHUser

		// Store updated configuration
		if err := vs.kv.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
			return fmt.Errorf("failed to update VPS SSH user: %w", err)
		}

		log.Printf("✅ Updated SSH user for VPS %d to %s", serverID, correctSSHUser)
	} else {
		log.Printf("✅ VPS %d SSH user already correct: %s (provider: %s)",
			serverID, correctSSHUser, vpsConfig.Provider)
	}

	return nil
}

// UpdateVPSConfig updates a VPS configuration
func (vs *VPSService) UpdateVPSConfig(token, accountID string, serverID int, vpsConfig *VPSConfig) error {
	return vs.kv.StoreVPSConfig(token, accountID, vpsConfig)
}

// GetVPSConfig retrieves a VPS configuration
func (vs *VPSService) GetVPSConfig(token, accountID string, serverID int) (*VPSConfig, error) {
	return vs.kv.GetVPSConfig(token, accountID, serverID)
}

// DeleteVPSConfig deletes a VPS configuration
func (vs *VPSService) DeleteVPSConfig(token, accountID string, serverID int) error {
	return vs.kv.DeleteVPSConfig(token, accountID, serverID)
}

// ResourceSpecs represents the resource specifications of a server type
type ResourceSpecs struct {
	Description string
	Cores       int
	Memory      float64 // in GB
	Disk        int     // in GB
	CPUType     string
}

// getResourceSpecs returns resource specifications for a given provider and server type
func (vs *VPSService) getResourceSpecs(provider, serverType string) ResourceSpecs {
	// Default specs for unknown configurations
	defaultSpecs := ResourceSpecs{
		Description: "Unknown Configuration",
		Cores:       0,
		Memory:      0,
		Disk:        0,
		CPUType:     "Unknown",
	}

	// Oracle Cloud Infrastructure (OCI) shapes
	if strings.Contains(provider, "Oracle") || strings.Contains(provider, "OCI") {
		return vs.getOCIResourceSpecs(serverType)
	}

	// Hetzner Cloud server types - could be expanded in the future
	if strings.Contains(provider, "Hetzner") {
		return vs.getHetznerResourceSpecs(serverType)
	}

	return defaultSpecs
}

// getOCIResourceSpecs returns resource specifications for Oracle Cloud shapes
func (vs *VPSService) getOCIResourceSpecs(shape string) ResourceSpecs {
	// Oracle Cloud Infrastructure shape mappings
	ociShapes := map[string]ResourceSpecs{
		"VM.Standard.A1.Flex": {
			Description: "Ampere Altra ARM64 Flexible Shape",
			Cores:       1,   // Always Free: 1 OCPU, can scale up to 4
			Memory:      6.0, // Always Free: 6GB RAM, can scale up to 24GB
			Disk:        47,  // Always Free: 47GB boot volume + block storage
			CPUType:     "ARM64 Ampere Altra",
		},
		"VM.Standard2.1": {
			Description: "Standard Intel Xeon E5-2690 v4",
			Cores:       1,
			Memory:      15.0,
			Disk:        47,
			CPUType:     "Intel Xeon E5-2690 v4",
		},
		"VM.Standard2.2": {
			Description: "Standard Intel Xeon E5-2690 v4",
			Cores:       2,
			Memory:      30.0,
			Disk:        47,
			CPUType:     "Intel Xeon E5-2690 v4",
		},
		"VM.Standard2.4": {
			Description: "Standard Intel Xeon E5-2690 v4",
			Cores:       4,
			Memory:      60.0,
			Disk:        47,
			CPUType:     "Intel Xeon E5-2690 v4",
		},
		"VM.Standard.E3.Flex": {
			Description: "AMD EPYC 7742 Flexible Shape",
			Cores:       1,    // Flexible, can scale from 1-64
			Memory:      16.0, // Flexible, can scale from 1-1024GB
			Disk:        47,
			CPUType:     "AMD EPYC 7742",
		},
		"VM.Standard.E4.Flex": {
			Description: "AMD EPYC 7J13 Flexible Shape",
			Cores:       1,    // Flexible, can scale from 1-64
			Memory:      16.0, // Flexible, can scale from 1-1024GB
			Disk:        47,
			CPUType:     "AMD EPYC 7J13",
		},
	}

	if specs, exists := ociShapes[shape]; exists {
		return specs
	}

	// Fallback for unknown OCI shapes
	return ResourceSpecs{
		Description: fmt.Sprintf("Oracle Cloud %s", shape),
		Cores:       1,
		Memory:      6.0,
		Disk:        47,
		CPUType:     "Oracle Cloud",
	}
}

// UpdateOCIVPSLocation updates the location for Oracle Cloud VPS instances from "oracle-cloud" to actual region
func (vs *VPSService) UpdateOCIVPSLocation(token, accountID string, serverID int, newLocation string) error {
	// Get existing VPS configuration
	vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		return fmt.Errorf("failed to get VPS config: %w", err)
	}

	// Only update Oracle Cloud instances with generic "oracle-cloud" location
	if !strings.Contains(vpsConfig.Provider, "Oracle") || vpsConfig.Location != "oracle-cloud" {
		log.Printf("VPS %d is not an Oracle Cloud instance with generic location, skipping", serverID)
		return nil
	}

	// Update location and timezone
	vpsConfig.Location = newLocation
	vpsConfig.Timezone = vs.provider.ResolveTimezone("Oracle Cloud Infrastructure (OCI)", newLocation)

	// Store updated configuration
	if err := vs.kv.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		return fmt.Errorf("failed to update VPS location: %w", err)
	}

	log.Printf("✅ Updated VPS %d location from 'oracle-cloud' to '%s' with timezone '%s'", 
		serverID, newLocation, vpsConfig.Timezone)
	return nil
}

// getHetznerResourceSpecs returns resource specifications for Hetzner Cloud server types
func (vs *VPSService) getHetznerResourceSpecs(serverType string) ResourceSpecs {
	// Common Hetzner server types (this could be expanded or made dynamic)
	hetznerTypes := map[string]ResourceSpecs{
		"cx11": {
			Description: "Intel/AMD, Dedicated vCPU",
			Cores:       1,
			Memory:      4.0,
			Disk:        20,
			CPUType:     "Intel/AMD x86",
		},
		"cx21": {
			Description: "Intel/AMD, Dedicated vCPU",
			Cores:       2,
			Memory:      8.0,
			Disk:        40,
			CPUType:     "Intel/AMD x86",
		},
		"cx31": {
			Description: "Intel/AMD, Dedicated vCPU",
			Cores:       2,
			Memory:      16.0,
			Disk:        80,
			CPUType:     "Intel/AMD x86",
		},
		"cx41": {
			Description: "Intel/AMD, Dedicated vCPU",
			Cores:       4,
			Memory:      32.0,
			Disk:        160,
			CPUType:     "Intel/AMD x86",
		},
		"cx51": {
			Description: "Intel/AMD, Dedicated vCPU",
			Cores:       8,
			Memory:      64.0,
			Disk:        240,
			CPUType:     "Intel/AMD x86",
		},
	}

	if specs, exists := hetznerTypes[serverType]; exists {
		return specs
	}

	// Fallback for unknown Hetzner server types
	return ResourceSpecs{
		Description: fmt.Sprintf("Hetzner %s", serverType),
		Cores:       1,
		Memory:      4.0,
		Disk:        20,
		CPUType:     "Intel/AMD x86",
	}
}
