package services

import (
	"fmt"
	"log"
	"strings"
)

// VPSService provides high-level business logic for VPS operations
type VPSService struct {
	hetzner *HetznerService
	kv      *KVService
	ssh     *SSHService
	cf      *CloudflareService
}

// NewVPSService creates a new VPS service instance
func NewVPSService() *VPSService {
	return &VPSService{
		hetzner: NewHetznerService(),
		kv:      NewKVService(),
		ssh:     NewSSHService(),
		cf:      NewCloudflareService(),
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

// EnhanceServersWithCosts adds cost information to a list of Hetzner servers
func (vs *VPSService) EnhanceServersWithCosts(token, accountID string, servers []HetznerServer) error {
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
	// Create server using Hetzner service
	server, err := vs.hetzner.CreateServer(hetznerKey, name, serverType, location, sshKeyName, domain, domainCert, domainKey)
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
		Timezone:    vs.getTimezoneForLocation(location),
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
	// Get all VPS configurations from KV
	vpsConfigsMap, err := vs.kv.ListVPSConfigs(token, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VPS configs from KV: %w", err)
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
				"managed_by":       "xanthus",
				"accumulated_cost": fmt.Sprintf("%.2f", accumulatedCost),
				"monthly_cost":     fmt.Sprintf("%.2f", vpsConfig.MonthlyRate),
				"hourly_cost":      fmt.Sprintf("%.4f", vpsConfig.HourlyRate),
			},
		}

		servers = append(servers, server)
	}

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
