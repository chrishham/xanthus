package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// DNSRecord represents a Cloudflare DNS record
type DNSRecord struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Name    string                 `json:"name"`
	Content string                 `json:"content"`
	Proxied bool                   `json:"proxied"`
	TTL     int                    `json:"ttl"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// DNSRecordsResponse represents the API response for DNS records
type DNSRecordsResponse struct {
	Success bool        `json:"success"`
	Result  []DNSRecord `json:"result"`
	Errors  []CFError   `json:"errors"`
}

// GetDNSRecords retrieves all DNS records for a zone
func (cs *CloudflareService) GetDNSRecords(token, zoneID string) ([]DNSRecord, error) {
	resp, err := cs.makeRequest("GET", fmt.Sprintf("/zones/%s/dns_records", zoneID), token, nil)
	if err != nil {
		return nil, err
	}

	// Parse DNS records from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var records []DNSRecord
	if err := json.Unmarshal(resultBytes, &records); err != nil {
		return nil, fmt.Errorf("failed to parse DNS records: %w", err)
	}

	return records, nil
}

// DeleteDNSRecord removes a DNS record by ID
func (cs *CloudflareService) DeleteDNSRecord(token, zoneID, recordID string) error {
	_, err := cs.makeRequest("DELETE", fmt.Sprintf("/zones/%s/dns_records/%s", zoneID, recordID), token, nil)
	return err
}

// CreateDNSRecord creates a new DNS record
func (cs *CloudflareService) CreateDNSRecord(token, zoneID, recordType, name, content string, proxied bool) (*DNSRecord, error) {
	body := map[string]interface{}{
		"type":    recordType,
		"name":    name,
		"content": content,
		"proxied": proxied,
		"ttl":     1, // Auto TTL when proxied, otherwise minimum
	}

	resp, err := cs.makeRequest("POST", fmt.Sprintf("/zones/%s/dns_records", zoneID), token, body)
	if err != nil {
		return nil, err
	}

	// Parse DNS record from result
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var record DNSRecord
	if err := json.Unmarshal(resultBytes, &record); err != nil {
		return nil, fmt.Errorf("failed to parse DNS record: %w", err)
	}

	return &record, nil
}

// ConfigureDNSForVPS configures DNS records for a VPS deployment
func (cs *CloudflareService) ConfigureDNSForVPS(token, domain, vpsIP string) error {
	// Get zone ID for the domain
	zoneID, err := cs.GetZoneID(token, domain)
	if err != nil {
		return fmt.Errorf("failed to get zone ID: %w", err)
	}

	// Get existing DNS records
	existingRecords, err := cs.GetDNSRecords(token, zoneID)
	if err != nil {
		return fmt.Errorf("failed to get existing DNS records: %w", err)
	}

	log.Printf("📋 Found %d existing DNS records for domain %s", len(existingRecords), domain)
	for _, record := range existingRecords {
		if record.Type == "A" {
			log.Printf("📍 Existing A record: %s -> %s", record.Name, record.Content)
		}
	}

	// Delete all existing A records for the domain
	recordsDeleted := 0
	for _, record := range existingRecords {
		// Normalize record name (remove trailing dot if present)
		recordName := strings.TrimSuffix(record.Name, ".")

		// Check if this is an A record we should delete
		shouldDelete := record.Type == "A" && (recordName == domain ||
			recordName == "*."+domain ||
			recordName == "www."+domain ||
			record.Name == domain ||
			record.Name == "*."+domain ||
			record.Name == "www."+domain)

		if shouldDelete {
			log.Printf("🗑️ Deleting existing A record: %s -> %s", record.Name, record.Content)
			if err := cs.DeleteDNSRecord(token, zoneID, record.ID); err != nil {
				return fmt.Errorf("failed to delete existing A record %s: %w", record.Name, err)
			}
			recordsDeleted++
		}
	}
	log.Printf("🗑️ Deleted %d existing A records", recordsDeleted)

	// Create new A records pointing to the VPS IP
	recordsToCreate := []struct {
		name    string
		proxied bool
	}{
		{domain, true},          // Root domain with proxy
		{"*." + domain, true},   // Wildcard subdomain with proxy
		{"www." + domain, true}, // www subdomain with proxy
	}

	for _, recordInfo := range recordsToCreate {
		log.Printf("➕ Creating A record: %s -> %s (proxied: %v)", recordInfo.name, vpsIP, recordInfo.proxied)
		if _, err := cs.CreateDNSRecord(token, zoneID, "A", recordInfo.name, vpsIP, recordInfo.proxied); err != nil {
			return fmt.Errorf("failed to create A record for %s: %w", recordInfo.name, err)
		}
		log.Printf("✅ Created A record: %s -> %s", recordInfo.name, vpsIP)
	}

	// Verify DNS records were created successfully
	log.Printf("🔍 Verifying DNS records were created...")
	updatedRecords, err := cs.GetDNSRecords(token, zoneID)
	if err != nil {
		log.Printf("Warning: Could not verify DNS records: %v", err)
	} else {
		foundRecords := 0
		for _, record := range updatedRecords {
			if record.Type == "A" && record.Content == vpsIP {
				log.Printf("✅ Verified A record: %s -> %s", record.Name, record.Content)
				foundRecords++
			}
		}
		log.Printf("✅ Verified %d DNS records pointing to VPS IP %s", foundRecords, vpsIP)
	}

	return nil
}
