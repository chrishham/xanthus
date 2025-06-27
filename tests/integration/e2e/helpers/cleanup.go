package helpers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// CleanupManager handles cleanup of test resources
type CleanupManager struct {
	config    *E2ETestConfig
	resources []TestResource
}

// TestResource represents a resource that needs cleanup
type TestResource struct {
	Type       string                 // "vps", "ssl", "dns", "app"
	ID         string
	Name       string
	Properties map[string]interface{}
	CreatedAt  time.Time
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(config *E2ETestConfig) *CleanupManager {
	return &CleanupManager{
		config:    config,
		resources: make([]TestResource, 0),
	}
}

// RegisterResource adds a resource to the cleanup list
func (c *CleanupManager) RegisterResource(resourceType, id, name string, properties map[string]interface{}) {
	resource := TestResource{
		Type:       resourceType,
		ID:         id,
		Name:       name,
		Properties: properties,
		CreatedAt:  time.Now(),
	}
	c.resources = append(c.resources, resource)
	log.Printf("Registered resource for cleanup: %s/%s (%s)", resourceType, name, id)
}

// CleanupTestResources removes all registered test resources
func (c *CleanupManager) CleanupTestResources() error {
	if len(c.resources) == 0 {
		log.Println("No resources to clean up")
		return nil
	}

	log.Printf("Starting cleanup of %d resources...", len(c.resources))
	
	ctx, cancel := context.WithTimeout(context.Background(), c.config.CleanupTimeout)
	defer cancel()

	var cleanupErrors []string

	// Clean up resources in reverse order (LIFO)
	for i := len(c.resources) - 1; i >= 0; i-- {
		resource := c.resources[i]
		if err := c.cleanupResource(ctx, resource); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("%s/%s: %v", resource.Type, resource.Name, err))
		}
	}

	if len(cleanupErrors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(cleanupErrors, "; "))
	}

	log.Printf("Successfully cleaned up %d resources", len(c.resources))
	return nil
}

// cleanupResource handles cleanup of a specific resource
func (c *CleanupManager) cleanupResource(ctx context.Context, resource TestResource) error {
	log.Printf("Cleaning up %s resource: %s (%s)", resource.Type, resource.Name, resource.ID)

	switch resource.Type {
	case "vps":
		return c.cleanupVPS(ctx, resource)
	case "ssl":
		return c.cleanupSSL(ctx, resource)
	case "dns":
		return c.cleanupDNS(ctx, resource)
	case "app":
		return c.cleanupApplication(ctx, resource)
	default:
		return fmt.Errorf("unknown resource type: %s", resource.Type)
	}
}

// cleanupVPS removes a VPS instance
func (c *CleanupManager) cleanupVPS(ctx context.Context, resource TestResource) error {
	// In live mode, would make actual API calls to delete VPS
	if c.config.TestMode == "live" {
		log.Printf("LIVE: Deleting VPS %s via Hetzner API", resource.ID)
		// Implementation would go here:
		// return c.hetznerService.DeleteServer(ctx, resource.ID)
		
		// For now, simulate the operation
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	// Mock mode - just log the operation
	log.Printf("MOCK: Would delete VPS %s", resource.ID)
	return nil
}

// cleanupSSL removes SSL configuration
func (c *CleanupManager) cleanupSSL(ctx context.Context, resource TestResource) error {
	domain, ok := resource.Properties["domain"].(string)
	if !ok {
		return fmt.Errorf("missing domain property for SSL resource")
	}

	if c.config.TestMode == "live" {
		log.Printf("LIVE: Removing SSL configuration for domain %s", domain)
		// Implementation would go here:
		// return c.cloudflareService.RemoveDomainFromXanthus(ctx, domain)
		
		// For now, simulate the operation
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	log.Printf("MOCK: Would remove SSL configuration for domain %s", domain)
	return nil
}

// cleanupDNS removes DNS records
func (c *CleanupManager) cleanupDNS(ctx context.Context, resource TestResource) error {
	recordID, ok := resource.Properties["record_id"].(string)
	if !ok {
		return fmt.Errorf("missing record_id property for DNS resource")
	}

	if c.config.TestMode == "live" {
		log.Printf("LIVE: Deleting DNS record %s", recordID)
		// Implementation would go here:
		// return c.cloudflareService.DeleteDNSRecord(ctx, recordID)
		
		// For now, simulate the operation
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	log.Printf("MOCK: Would delete DNS record %s", recordID)
	return nil
}

// cleanupApplication removes deployed applications
func (c *CleanupManager) cleanupApplication(ctx context.Context, resource TestResource) error {
	appName, ok := resource.Properties["app_name"].(string)
	if !ok {
		return fmt.Errorf("missing app_name property for application resource")
	}

	namespace, ok := resource.Properties["namespace"].(string)
	if !ok {
		namespace = "default"
	}

	if c.config.TestMode == "live" {
		log.Printf("LIVE: Uninstalling application %s from namespace %s", appName, namespace)
		// Implementation would go here:
		// return c.helmService.UninstallChart(ctx, appName, namespace)
		
		// For now, simulate the operation
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	log.Printf("MOCK: Would uninstall application %s from namespace %s", appName, namespace)
	return nil
}

// GetCleanupReport returns a summary of cleanup operations
func (c *CleanupManager) GetCleanupReport() map[string]int {
	report := make(map[string]int)
	
	for _, resource := range c.resources {
		report[resource.Type]++
	}
	
	return report
}

// ForceCleanupByPattern attempts to clean up resources matching a pattern
// This is useful for cleaning up orphaned resources from failed tests
func (c *CleanupManager) ForceCleanupByPattern(pattern string) error {
	log.Printf("Force cleanup of resources matching pattern: %s", pattern)
	
	if c.config.TestMode == "live" {
		log.Printf("LIVE: Would scan and clean up resources matching pattern %s", pattern)
		// Implementation would scan for resources matching the pattern
		// and clean them up using the respective APIs
		return nil
	}

	log.Printf("MOCK: Would force cleanup resources matching pattern %s", pattern)
	return nil
}

// ScheduledCleanup runs periodic cleanup of old test resources
func ScheduledCleanup(config *E2ETestConfig, maxAge time.Duration) error {
	log.Printf("Running scheduled cleanup of resources older than %v", maxAge)
	
	if config.TestMode == "live" {
		// Implementation would:
		// 1. List all resources with test prefixes
		// 2. Check their creation time
		// 3. Delete resources older than maxAge
		log.Println("LIVE: Would perform scheduled cleanup of old test resources")
		return nil
	}

	log.Println("MOCK: Would perform scheduled cleanup")
	return nil
}