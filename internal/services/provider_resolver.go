package services

import (
	"fmt"
	"log"
	"strings"
)

// ProviderResolver provides provider-specific configurations and utilities
type ProviderResolver struct {
	kv *KVService
}

// NewProviderResolver creates a new provider resolver service
func NewProviderResolver(kv *KVService) *ProviderResolver {
	return &ProviderResolver{
		kv: kv,
	}
}

// ProviderDefaults contains default settings for each provider
type ProviderDefaults struct {
	DefaultSSHUser      string
	DefaultSSHPort      int
	SupportsAPICreation bool
	LocationTimezones   map[string]string
}

// GetProviderDefaults returns default settings for a given provider
func (pr *ProviderResolver) GetProviderDefaults(provider string) *ProviderDefaults {
	switch provider {
	case "Hetzner":
		return &ProviderDefaults{
			DefaultSSHUser:      "root",
			DefaultSSHPort:      22,
			SupportsAPICreation: true,
			LocationTimezones: map[string]string{
				"nbg1":    "Europe/Berlin",    // Nuremberg, Germany
				"fsn1":    "Europe/Berlin",    // Falkenstein, Germany
				"hel1":    "Europe/Helsinki",  // Helsinki, Finland
				"ash":     "America/New_York", // Ashburn, USA
				"hil":     "America/New_York", // Hillsboro, USA
				"cax":     "America/New_York", // Central US
				"default": "Europe/Athens",    // Default for Greece-based deployments
			},
		}
	case "Oracle Cloud Infrastructure (OCI)", "oci", "OCI":
		return &ProviderDefaults{
			DefaultSSHUser:      "ubuntu", // Common default for OCI instances
			DefaultSSHPort:      22,
			SupportsAPICreation: true, // Now supports API creation
			LocationTimezones: map[string]string{
				"us-phoenix-1":   "America/Phoenix",
				"us-ashburn-1":   "America/New_York",
				"eu-frankfurt-1": "Europe/Berlin",
				"eu-zurich-1":    "Europe/Zurich",
				"uk-london-1":    "Europe/London",
				"ap-mumbai-1":    "Asia/Kolkata",
				"ap-seoul-1":     "Asia/Seoul",
				"ap-sydney-1":    "Australia/Sydney",
				"ap-tokyo-1":     "Asia/Tokyo",
				"sa-saopaulo-1":  "America/Sao_Paulo",
				"ca-toronto-1":   "America/Toronto",
				"ca-montreal-1":  "America/Montreal",
				"oracle-cloud":   "UTC",
				"default":        "UTC",
			},
		}
	case "AWS":
		return &ProviderDefaults{
			DefaultSSHUser:      "ec2-user", // Common default for Amazon Linux
			DefaultSSHPort:      22,
			SupportsAPICreation: false,
			LocationTimezones: map[string]string{
				"us-east-1":    "America/New_York",
				"us-west-2":    "America/Los_Angeles",
				"eu-west-1":    "Europe/Dublin",
				"eu-central-1": "Europe/Berlin",
				"default":      "UTC",
			},
		}
	case "DigitalOcean":
		return &ProviderDefaults{
			DefaultSSHUser:      "root",
			DefaultSSHPort:      22,
			SupportsAPICreation: false,
			LocationTimezones: map[string]string{
				"nyc1":    "America/New_York",
				"lon1":    "Europe/London",
				"fra1":    "Europe/Berlin",
				"default": "UTC",
			},
		}
	default:
		log.Printf("Warning: Unknown provider '%s', using generic defaults", provider)
		return &ProviderDefaults{
			DefaultSSHUser:      "root",
			DefaultSSHPort:      22,
			SupportsAPICreation: false,
			LocationTimezones: map[string]string{
				"default": "UTC",
			},
		}
	}
}

// ResolveSSHUser resolves the SSH user for a VPS based on provider and configuration
func (pr *ProviderResolver) ResolveSSHUser(token, accountID string, serverID int) (string, error) {
	// Get VPS configuration
	vpsConfig, err := pr.kv.GetVPSConfig(token, accountID, serverID)
	if err != nil {
		return "", fmt.Errorf("failed to get VPS config: %w", err)
	}

	log.Printf("🔍 ProviderResolver.ResolveSSHUser - ServerID: %d, Provider: %s, StoredSSHUser: %s",
		serverID, vpsConfig.Provider, vpsConfig.SSHUser)

	// If SSH user is explicitly set in config, use it
	if vpsConfig.SSHUser != "" {
		log.Printf("✅ Using stored SSH user: %s", vpsConfig.SSHUser)
		return vpsConfig.SSHUser, nil
	}

	// Otherwise, use provider default
	defaults := pr.GetProviderDefaults(vpsConfig.Provider)
	log.Printf("⚠️ No stored SSH user, using provider default: %s", defaults.DefaultSSHUser)
	return defaults.DefaultSSHUser, nil
}

// ResolveSSHUserFromConfig resolves the SSH user from a VPS configuration
func (pr *ProviderResolver) ResolveSSHUserFromConfig(vpsConfig *VPSConfig) string {
	// If SSH user is explicitly set in config, use it
	if vpsConfig.SSHUser != "" {
		return vpsConfig.SSHUser
	}

	// Otherwise, use provider default
	defaults := pr.GetProviderDefaults(vpsConfig.Provider)
	return defaults.DefaultSSHUser
}

// GetCorrectSSHUserFromConfig always returns the correct SSH user based on provider defaults
// This method ignores the stored SSH user and always uses provider defaults
func (pr *ProviderResolver) GetCorrectSSHUserFromConfig(vpsConfig *VPSConfig) string {
	defaults := pr.GetProviderDefaults(vpsConfig.Provider)
	return defaults.DefaultSSHUser
}

// ResolveTimezone resolves the timezone for a location within a provider
func (pr *ProviderResolver) ResolveTimezone(provider, location string) string {
	defaults := pr.GetProviderDefaults(provider)

	// Extract location prefix (e.g., "nbg1" from "nbg1-dc3")
	for prefix, timezone := range defaults.LocationTimezones {
		if strings.HasPrefix(location, prefix) {
			return timezone
		}
	}

	// Default fallback
	if defaults.LocationTimezones["default"] != "" {
		return defaults.LocationTimezones["default"]
	}

	return "UTC"
}

// ValidateProviderSupport checks if a provider is supported
func (pr *ProviderResolver) ValidateProviderSupport(provider string) error {
	switch provider {
	case "Hetzner", "Oracle Cloud Infrastructure (OCI)", "oci", "OCI", "AWS", "DigitalOcean":
		return nil
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// GetSupportedProviders returns a list of supported providers
func (pr *ProviderResolver) GetSupportedProviders() []string {
	return []string{
		"Hetzner",
		"Oracle Cloud Infrastructure (OCI)",
		"AWS",
		"DigitalOcean",
	}
}
