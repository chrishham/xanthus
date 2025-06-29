package services

import (
	"fmt"
	"log"
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
		Domain:      domain,
		ServerType:  serverType,
		Location:    location,
		PublicIPv4:  server.PublicNet.IPv4.IP,
		Status:      server.Status,
		CreatedAt:   server.Created,
		SSHKeyName:  sshKeyName,
		SSHUser:     "root",
		SSHPort:     22,
		HourlyRate:  hourlyRate,
		MonthlyRate: monthlyRate,
	}

	// Store VPS configuration
	if err := vs.kv.StoreVPSConfig(token, accountID, vpsConfig); err != nil {
		log.Printf("Warning: Failed to store VPS config: %v", err)
		// Don't fail the creation, just log the warning
	}

	return server, vpsConfig, nil
}

// DeleteVPSAndCleanup deletes a VPS and cleans up its configuration
func (vs *VPSService) DeleteVPSAndCleanup(token, accountID, hetznerKey string, serverID int) (*VPSConfig, error) {
	// Get VPS configuration before deletion (for logging)
	vpsConfig, err := vs.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		log.Printf("Warning: Could not get VPS config for server %d: %v", serverID, err)
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
