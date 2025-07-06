package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	_ "embed"

	"github.com/chrishham/xanthus/internal/utils"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
)

//go:embed oci-cloudinit.yaml
var ociCloudInitScript string

// OCIService handles Oracle Cloud Infrastructure operations
type OCIService struct {
	computeClient  *core.ComputeClient
	identityClient *identity.IdentityClient
	networkClient  *core.VirtualNetworkClient
	tenancyOCID    string
	region         string
	creds          *utils.OCICredentials
}

// NewOCIService creates a new OCI service instance
func NewOCIService(authToken string) (*OCIService, error) {
	// Decode auth token
	creds, err := utils.DecodeOCIAuthToken(authToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decode OCI auth token: %w", err)
	}

	// Validate credentials
	if err := utils.ValidateOCICredentials(creds); err != nil {
		return nil, fmt.Errorf("invalid OCI credentials: %w", err)
	}

	// Create configuration provider
	configProvider := common.NewRawConfigurationProvider(
		creds.Tenancy,
		creds.User,
		creds.Region,
		creds.Fingerprint,
		creds.PrivateKey,
		nil, // passphrase
	)

	// Create compute client
	computeClient, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}

	// Create identity client
	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity client: %w", err)
	}

	// Create network client
	networkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create network client: %w", err)
	}

	return &OCIService{
		computeClient:  &computeClient,
		identityClient: &identityClient,
		networkClient:  &networkClient,
		tenancyOCID:    creds.Tenancy,
		region:         creds.Region,
		creds:          creds,
	}, nil
}

// OCIInstance represents an OCI compute instance
type OCIInstance struct {
	ID               string            `json:"id"`
	DisplayName      string            `json:"display_name"`
	LifecycleState   string            `json:"lifecycle_state"`
	AvailabilityDomain string          `json:"availability_domain"`
	CompartmentID    string            `json:"compartment_id"`
	Shape            string            `json:"shape"`
	TimeCreated      *common.SDKTime   `json:"time_created"`
	PublicIP         string            `json:"public_ip"`
	PrivateIP        string            `json:"private_ip"`
	FreeformTags     map[string]string `json:"freeform_tags"`
	DefinedTags      map[string]interface{} `json:"defined_tags"`
}

// OCIShape represents an OCI compute shape
type OCIShape struct {
	Shape                   string   `json:"shape"`
	ProcessorDescription    string   `json:"processor_description"`
	Ocpus                   float32  `json:"ocpus"`
	MemoryInGBs             float32  `json:"memory_in_gbs"`
	NetworkingBandwidthInGbps float32 `json:"networking_bandwidth_in_gbps"`
	MaxVnicAttachments      int      `json:"max_vnic_attachments"`
	GPUs                    int      `json:"gpus"`
	LocalDisks              int      `json:"local_disks"`
	LocalDisksTotalSizeInGBs float32 `json:"local_disks_total_size_in_gbs"`
	IsLiveMigrationSupported bool    `json:"is_live_migration_supported"`
	IsFlexible              bool     `json:"is_flexible"`
}

// OCIVirtualCloudNetwork represents a VCN
type OCIVirtualCloudNetwork struct {
	ID             string            `json:"id"`
	DisplayName    string            `json:"display_name"`
	LifecycleState string            `json:"lifecycle_state"`
	CidrBlock      string            `json:"cidr_block"`
	CompartmentID  string            `json:"compartment_id"`
	TimeCreated    *common.SDKTime   `json:"time_created"`
	FreeformTags   map[string]string `json:"freeform_tags"`
}

// OCISubnet represents a subnet
type OCISubnet struct {
	ID             string            `json:"id"`
	DisplayName    string            `json:"display_name"`
	LifecycleState string            `json:"lifecycle_state"`
	CidrBlock      string            `json:"cidr_block"`
	CompartmentID  string            `json:"compartment_id"`
	VcnID          string            `json:"vcn_id"`
	TimeCreated    *common.SDKTime   `json:"time_created"`
	FreeformTags   map[string]string `json:"freeform_tags"`
}

// ListInstances retrieves all instances managed by Xanthus
func (o *OCIService) ListInstances(ctx context.Context) ([]OCIInstance, error) {
	// List instances with managed_by=xanthus tag
	listInstancesRequest := core.ListInstancesRequest{
		CompartmentId: &o.tenancyOCID,
		SortBy:        core.ListInstancesSortByTimecreated,
		SortOrder:     core.ListInstancesSortOrderDesc,
	}

	response, err := o.computeClient.ListInstances(ctx, listInstancesRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var instances []OCIInstance
	for _, instance := range response.Items {
		// Filter by managed_by tag
		if managedBy, exists := instance.FreeformTags["managed_by"]; !exists || managedBy != "xanthus" {
			continue
		}

		// Get public IP
		publicIP, err := o.getInstancePublicIP(ctx, *instance.Id)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to get public IP for instance %s: %v\n", *instance.Id, err)
		}

		// Get private IP
		privateIP, err := o.getInstancePrivateIP(ctx, *instance.Id)
		if err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to get private IP for instance %s: %v\n", *instance.Id, err)
		}

		instances = append(instances, OCIInstance{
			ID:               *instance.Id,
			DisplayName:      *instance.DisplayName,
			LifecycleState:   string(instance.LifecycleState),
			AvailabilityDomain: *instance.AvailabilityDomain,
			CompartmentID:    *instance.CompartmentId,
			Shape:            *instance.Shape,
			TimeCreated:      instance.TimeCreated,
			PublicIP:         publicIP,
			PrivateIP:        privateIP,
			FreeformTags:     instance.FreeformTags,
			DefinedTags:      flattenDefinedTags(instance.DefinedTags),
		})
	}

	return instances, nil
}

// GetInstance retrieves details for a specific instance
func (o *OCIService) GetInstance(ctx context.Context, instanceID string) (*OCIInstance, error) {
	getInstanceRequest := core.GetInstanceRequest{
		InstanceId: &instanceID,
	}

	response, err := o.computeClient.GetInstance(ctx, getInstanceRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	instance := response.Instance

	// Get public IP
	publicIP, err := o.getInstancePublicIP(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public IP: %w", err)
	}

	// Get private IP
	privateIP, err := o.getInstancePrivateIP(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get private IP: %w", err)
	}

	return &OCIInstance{
		ID:               *instance.Id,
		DisplayName:      *instance.DisplayName,
		LifecycleState:   string(instance.LifecycleState),
		AvailabilityDomain: *instance.AvailabilityDomain,
		CompartmentID:    *instance.CompartmentId,
		Shape:            *instance.Shape,
		TimeCreated:      instance.TimeCreated,
		PublicIP:         publicIP,
		PrivateIP:        privateIP,
		FreeformTags:     instance.FreeformTags,
		DefinedTags:      flattenDefinedTags(instance.DefinedTags),
	}, nil
}

// getInstancePublicIP retrieves the public IP address of an instance
func (o *OCIService) getInstancePublicIP(ctx context.Context, instanceID string) (string, error) {
	// List VNICs attached to the instance
	listVnicAttachmentsRequest := core.ListVnicAttachmentsRequest{
		CompartmentId: &o.tenancyOCID,
		InstanceId:    &instanceID,
	}

	attachmentResponse, err := o.computeClient.ListVnicAttachments(ctx, listVnicAttachmentsRequest)
	if err != nil {
		return "", fmt.Errorf("failed to list VNIC attachments: %w", err)
	}

	if len(attachmentResponse.Items) == 0 {
		return "", nil
	}

	// Get the first VNIC (primary)
	vnicID := attachmentResponse.Items[0].VnicId
	if vnicID == nil {
		return "", nil
	}

	// Get VNIC details
	getVnicRequest := core.GetVnicRequest{
		VnicId: vnicID,
	}

	vnicResponse, err := o.networkClient.GetVnic(ctx, getVnicRequest)
	if err != nil {
		return "", fmt.Errorf("failed to get VNIC: %w", err)
	}

	if vnicResponse.PublicIp != nil {
		return *vnicResponse.PublicIp, nil
	}

	return "", nil
}

// getInstancePrivateIP retrieves the private IP address of an instance
func (o *OCIService) getInstancePrivateIP(ctx context.Context, instanceID string) (string, error) {
	// List VNICs attached to the instance
	listVnicAttachmentsRequest := core.ListVnicAttachmentsRequest{
		CompartmentId: &o.tenancyOCID,
		InstanceId:    &instanceID,
	}

	attachmentResponse, err := o.computeClient.ListVnicAttachments(ctx, listVnicAttachmentsRequest)
	if err != nil {
		return "", fmt.Errorf("failed to list VNIC attachments: %w", err)
	}

	if len(attachmentResponse.Items) == 0 {
		return "", nil
	}

	// Get the first VNIC (primary)
	vnicID := attachmentResponse.Items[0].VnicId
	if vnicID == nil {
		return "", nil
	}

	// Get VNIC details
	getVnicRequest := core.GetVnicRequest{
		VnicId: vnicID,
	}

	vnicResponse, err := o.networkClient.GetVnic(ctx, getVnicRequest)
	if err != nil {
		return "", fmt.Errorf("failed to get VNIC: %w", err)
	}

	if vnicResponse.PrivateIp != nil {
		return *vnicResponse.PrivateIp, nil
	}

	return "", nil
}

// PowerOffInstance powers off an instance
func (o *OCIService) PowerOffInstance(ctx context.Context, instanceID string) error {
	instanceAction := core.InstanceActionActionStop
	instanceActionRequest := core.InstanceActionRequest{
		InstanceId: &instanceID,
		Action:     instanceAction,
	}

	_, err := o.computeClient.InstanceAction(ctx, instanceActionRequest)
	if err != nil {
		return fmt.Errorf("failed to power off instance: %w", err)
	}

	return nil
}

// PowerOnInstance powers on an instance
func (o *OCIService) PowerOnInstance(ctx context.Context, instanceID string) error {
	instanceAction := core.InstanceActionActionStart
	instanceActionRequest := core.InstanceActionRequest{
		InstanceId: &instanceID,
		Action:     instanceAction,
	}

	_, err := o.computeClient.InstanceAction(ctx, instanceActionRequest)
	if err != nil {
		return fmt.Errorf("failed to power on instance: %w", err)
	}

	return nil
}

// RebootInstance reboots an instance
func (o *OCIService) RebootInstance(ctx context.Context, instanceID string) error {
	instanceAction := core.InstanceActionActionSoftreset
	instanceActionRequest := core.InstanceActionRequest{
		InstanceId: &instanceID,
		Action:     instanceAction,
	}

	_, err := o.computeClient.InstanceAction(ctx, instanceActionRequest)
	if err != nil {
		return fmt.Errorf("failed to reboot instance: %w", err)
	}

	return nil
}

// TerminateInstance terminates an instance
func (o *OCIService) TerminateInstance(ctx context.Context, instanceID string) error {
	terminateInstanceRequest := core.TerminateInstanceRequest{
		InstanceId: &instanceID,
	}

	_, err := o.computeClient.TerminateInstance(ctx, terminateInstanceRequest)
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	return nil
}

// ListAvailabilityDomains lists availability domains in the region
func (o *OCIService) ListAvailabilityDomains(ctx context.Context) ([]identity.AvailabilityDomain, error) {
	listAvailabilityDomainsRequest := identity.ListAvailabilityDomainsRequest{
		CompartmentId: &o.tenancyOCID,
	}

	response, err := o.identityClient.ListAvailabilityDomains(ctx, listAvailabilityDomainsRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list availability domains: %w", err)
	}

	return response.Items, nil
}

// ListComputeShapes lists available compute shapes
func (o *OCIService) ListComputeShapes(ctx context.Context) ([]OCIShape, error) {
	listShapesRequest := core.ListShapesRequest{
		CompartmentId: &o.tenancyOCID,
	}

	response, err := o.computeClient.ListShapes(ctx, listShapesRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list shapes: %w", err)
	}

	var shapes []OCIShape
	for _, shape := range response.Items {
		shapes = append(shapes, OCIShape{
			Shape:                   *shape.Shape,
			ProcessorDescription:    getStringValue(shape.ProcessorDescription),
			Ocpus:                   getFloat32Value(shape.Ocpus),
			MemoryInGBs:             getFloat32Value(shape.MemoryInGBs),
			NetworkingBandwidthInGbps: getFloat32Value(shape.NetworkingBandwidthInGbps),
			MaxVnicAttachments:      getIntValue(shape.MaxVnicAttachments),
			GPUs:                    getIntValue(shape.Gpus),
			LocalDisks:              getIntValue(shape.LocalDisks),
			LocalDisksTotalSizeInGBs: getFloat32Value(shape.LocalDisksTotalSizeInGBs),
			IsLiveMigrationSupported: getBoolValue(shape.IsLiveMigrationSupported),
			IsFlexible:              getBoolValue(shape.IsFlexible),
		})
	}

	return shapes, nil
}

// CreateInstance creates a new OCI compute instance with cloud-init
func (o *OCIService) CreateInstance(ctx context.Context, displayName, shape, imageID, availabilityDomain, subnetID, sshPublicKey, cloudInitScript, timezone string) (*OCIInstance, error) {
	// Encode cloud-init script
	userData := cloudInitScript
	if userData != "" {
		// Replace template variables
		if timezone != "" {
			userData = strings.ReplaceAll(userData, "${TIMEZONE}", timezone)
		} else {
			userData = strings.ReplaceAll(userData, "${TIMEZONE}", "UTC")
		}
		
		// Base64 encode the user data
		userData = base64.StdEncoding.EncodeToString([]byte(userData))
	}

	// Prepare metadata
	metadata := map[string]string{}
	if sshPublicKey != "" {
		metadata["ssh_authorized_keys"] = sshPublicKey
	}
	if userData != "" {
		metadata["user_data"] = userData
	}

	// Create launch instance details
	launchInstanceDetails := core.LaunchInstanceDetails{
		AvailabilityDomain: &availabilityDomain,
		CompartmentId:      &o.tenancyOCID,
		DisplayName:        &displayName,
		ImageId:            &imageID,
		Shape:              &shape,
		CreateVnicDetails: &core.CreateVnicDetails{
			SubnetId:        &subnetID,
			AssignPublicIp:  common.Bool(true),
			DisplayName:     common.String(fmt.Sprintf("%s-vnic", displayName)),
		},
		Metadata: metadata,
		FreeformTags: map[string]string{
			"managed_by": "xanthus",
			"purpose":    "k3s-cluster",
		},
	}

	// Configure shape for flexible shapes
	if strings.Contains(shape, "Flex") {
		launchInstanceDetails.ShapeConfig = &core.LaunchInstanceShapeConfigDetails{
			Ocpus:       common.Float32(1.0),   // 1 OCPU for Always Free (A1.Flex supports up to 4)
			MemoryInGBs: common.Float32(6.0),   // 6GB RAM for Always Free (A1.Flex supports up to 24GB)
		}
	}

	// Launch the instance
	launchInstanceRequest := core.LaunchInstanceRequest{
		LaunchInstanceDetails: launchInstanceDetails,
	}

	response, err := o.computeClient.LaunchInstance(ctx, launchInstanceRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	instance := response.Instance

	// Wait for instance to be running
	instanceID := *instance.Id
	err = o.waitForInstanceState(ctx, instanceID, core.InstanceLifecycleStateRunning, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("instance failed to reach running state: %w", err)
	}

	// Get the instance with IP addresses
	return o.GetInstance(ctx, instanceID)
}

// waitForInstanceState waits for an instance to reach a specific lifecycle state
func (o *OCIService) waitForInstanceState(ctx context.Context, instanceID string, targetState core.InstanceLifecycleStateEnum, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		instance, err := o.GetInstance(ctx, instanceID)
		if err != nil {
			return fmt.Errorf("failed to get instance status: %w", err)
		}

		if instance.LifecycleState == string(targetState) {
			return nil
		}

		// Check for failed states
		if instance.LifecycleState == string(core.InstanceLifecycleStateTerminated) ||
		   instance.LifecycleState == string(core.InstanceLifecycleStateTerminating) {
			return fmt.Errorf("instance reached unexpected state: %s", instance.LifecycleState)
		}

		// Wait before checking again
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(15 * time.Second):
			// Continue loop
		}
	}

	return fmt.Errorf("timeout waiting for instance to reach state %s", targetState)
}

// GetInstanceConsoleConnection gets console connection details for an instance
func (o *OCIService) GetInstanceConsoleConnection(ctx context.Context, instanceID string) (*core.InstanceConsoleConnection, error) {
	// List console connections for the instance
	listConsoleConnectionsRequest := core.ListInstanceConsoleConnectionsRequest{
		CompartmentId: &o.tenancyOCID,
		InstanceId:    &instanceID,
	}

	response, err := o.computeClient.ListInstanceConsoleConnections(ctx, listConsoleConnectionsRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list console connections: %w", err)
	}

	if len(response.Items) > 0 {
		// Return existing connection
		connectionID := response.Items[0].Id
		getConsoleConnectionRequest := core.GetInstanceConsoleConnectionRequest{
			InstanceConsoleConnectionId: connectionID,
		}

		getResponse, err := o.computeClient.GetInstanceConsoleConnection(ctx, getConsoleConnectionRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to get console connection: %w", err)
		}

		return &getResponse.InstanceConsoleConnection, nil
	}

	// Create new console connection
	createConsoleConnectionRequest := core.CreateInstanceConsoleConnectionRequest{
		CreateInstanceConsoleConnectionDetails: core.CreateInstanceConsoleConnectionDetails{
			InstanceId: &instanceID,
		},
	}

	createResponse, err := o.computeClient.CreateInstanceConsoleConnection(ctx, createConsoleConnectionRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create console connection: %w", err)
	}

	return &createResponse.InstanceConsoleConnection, nil
}

// CreateVPSWithK3s creates a complete VPS instance with network setup and K3s installation
func (o *OCIService) CreateVPSWithK3s(ctx context.Context, displayName, shape, sshPublicKey, timezone string) (*OCIInstance, error) {
	// Get the first availability domain
	availabilityDomains, err := o.ListAvailabilityDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list availability domains: %w", err)
	}

	if len(availabilityDomains) == 0 {
		return nil, fmt.Errorf("no availability domains found")
	}

	availabilityDomain := *availabilityDomains[0].Name

	// Set up network infrastructure
	baseName := "xanthus"
	_, subnetID, err := o.ConfigureNetworkResources(ctx, baseName, availabilityDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to configure network resources: %w", err)
	}

	// Get Ubuntu image ID
	imageID, err := o.GetUbuntuImageID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Ubuntu image: %w", err)
	}

	// Use default shape if not provided
	if shape == "" {
		shape = "VM.Standard.A1.Flex" // Always Free eligible ARM64
	}

	// Create the instance with cloud-init
	return o.CreateInstance(ctx, displayName, shape, imageID, availabilityDomain, subnetID, sshPublicKey, ociCloudInitScript, timezone)
}

// DeleteVPSWithCleanup terminates an instance and optionally cleans up network resources
func (o *OCIService) DeleteVPSWithCleanup(ctx context.Context, instanceID string, cleanupNetwork bool) error {
	// Terminate the instance
	err := o.TerminateInstance(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	// If requested, clean up network resources
	// Note: This is a simple implementation - in production you might want more sophisticated cleanup
	if cleanupNetwork {
		// TODO: Implement network cleanup logic
		// This would involve checking if other instances are using the same VCN/subnet
		// and only cleaning up if this was the last instance
		fmt.Printf("Network cleanup requested but not implemented yet\n")
	}

	return nil
}

// ListImages lists available images for instance creation
func (o *OCIService) ListImages(ctx context.Context) ([]core.Image, error) {
	listImagesRequest := core.ListImagesRequest{
		CompartmentId: &o.tenancyOCID,
		SortBy:        core.ListImagesSortByTimecreated,
		SortOrder:     core.ListImagesSortOrderDesc,
	}

	response, err := o.computeClient.ListImages(ctx, listImagesRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return response.Items, nil
}

// GetUbuntuImageID finds the latest Ubuntu 24.04 ARM64 image for A1.Flex
func (o *OCIService) GetUbuntuImageID(ctx context.Context) (string, error) {
	// Use the specific Ubuntu 24.04 ARM64 image OCID for eu-frankfurt-1
	// This image is confirmed compatible with VM.Standard.A1.Flex
	region := "eu-frankfurt-1" // This should match the region in the credentials
	
	if strings.Contains(strings.ToLower(region), "frankfurt") {
		// Return the specific Ubuntu 24.04 ARM64 image OCID for Frankfurt region
		// Canonical-Ubuntu-24.04-aarch64-2025.05.20-0
		return "ocid1.image.oc1.eu-frankfurt-1.aaaaaaaaxhdnngoowpuvwonng4mr2brdemk5wvmompn6ykmohmfuqmwvagjq", nil
	}
	
	// For other regions, fall back to dynamic search
	images, err := o.ListImages(ctx)
	if err != nil {
		return "", err
	}

	// Priority 1: Look for Ubuntu 24.04 ARM64 with specific build 2025.05.20-0
	for _, image := range images {
		if image.DisplayName != nil {
			displayName := strings.ToLower(*image.DisplayName)
			if strings.Contains(displayName, "ubuntu") &&
			   strings.Contains(displayName, "24.04") &&
			   strings.Contains(displayName, "2025.05.20-0") &&
			   (strings.Contains(displayName, "arm64") || strings.Contains(displayName, "aarch64")) {
				return *image.Id, nil
			}
		}
	}

	// Priority 2: Look for any Ubuntu 24.04 ARM64 image
	for _, image := range images {
		if image.DisplayName != nil {
			displayName := strings.ToLower(*image.DisplayName)
			if strings.Contains(displayName, "ubuntu") &&
			   strings.Contains(displayName, "24.04") &&
			   (strings.Contains(displayName, "arm64") || strings.Contains(displayName, "aarch64")) {
				return *image.Id, nil
			}
		}
	}

	// Priority 3: Look for any Ubuntu ARM64 image
	for _, image := range images {
		if image.DisplayName != nil {
			displayName := strings.ToLower(*image.DisplayName)
			if strings.Contains(displayName, "ubuntu") &&
			   (strings.Contains(displayName, "arm64") || strings.Contains(displayName, "aarch64")) {
				return *image.Id, nil
			}
		}
	}

	return "", fmt.Errorf("no compatible Ubuntu ARM64 image found for A1.Flex shape")
}

// SSH Key Management

// CreateSSHKey creates an SSH key pair in OCI (not yet available in OCI Go SDK)
// Note: OCI doesn't have a native SSH key management service like Hetzner
// SSH keys are managed at the instance level during creation
func (o *OCIService) CreateSSHKey(name, publicKey string) error {
	// OCI doesn't have a centralized SSH key management service
	// SSH keys are provided during instance creation in metadata
	// This method exists for interface compatibility but doesn't do anything
	return fmt.Errorf("OCI does not support centralized SSH key management - keys are provided during instance creation")
}

// ValidateSSHKey validates that an SSH public key is in the correct format
func (o *OCIService) ValidateSSHKey(publicKey string) error {
	if publicKey == "" {
		return fmt.Errorf("SSH public key cannot be empty")
	}

	// Basic validation - should start with ssh-rsa, ssh-ed25519, etc.
	if !strings.HasPrefix(publicKey, "ssh-") {
		return fmt.Errorf("invalid SSH public key format - must start with ssh-")
	}

	// Should have at least 3 parts (type, key, comment)
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		return fmt.Errorf("invalid SSH public key format - insufficient parts")
	}

	return nil
}

// FormatSSHKeyForMetadata formats an SSH public key for use in OCI instance metadata
func (o *OCIService) FormatSSHKeyForMetadata(publicKey string) (string, error) {
	if err := o.ValidateSSHKey(publicKey); err != nil {
		return "", err
	}

	// Ensure the key is properly formatted (single line, no extra whitespace)
	return strings.TrimSpace(publicKey), nil
}

// TestConnection tests the OCI connection
func (o *OCIService) TestConnection(ctx context.Context) error {
	// Simple test by listing availability domains
	_, err := o.ListAvailabilityDomains(ctx)
	if err != nil {
		return fmt.Errorf("OCI connection test failed: %w", err)
	}

	return nil
}

// Network automation methods

// CreateOrGetVCN creates a VCN or returns existing one
func (o *OCIService) CreateOrGetVCN(ctx context.Context, displayName string) (*OCIVirtualCloudNetwork, error) {
	// First, try to find existing VCN
	vcn, err := o.findVCNByName(ctx, displayName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for existing VCN: %w", err)
	}

	if vcn != nil {
		return vcn, nil
	}

	// Create new VCN
	cidrBlock := "10.0.0.0/16"
	createVcnRequest := core.CreateVcnRequest{
		CreateVcnDetails: core.CreateVcnDetails{
			CidrBlock:     &cidrBlock,
			CompartmentId: &o.tenancyOCID,
			DisplayName:   &displayName,
			DnsLabel:      common.String("xanthus"),
			FreeformTags: map[string]string{
				"managed_by": "xanthus",
				"purpose":    "k3s-cluster",
			},
		},
	}

	response, err := o.networkClient.CreateVcn(ctx, createVcnRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create VCN: %w", err)
	}

	vcnData := response.Vcn
	return &OCIVirtualCloudNetwork{
		ID:             *vcnData.Id,
		DisplayName:    *vcnData.DisplayName,
		LifecycleState: string(vcnData.LifecycleState),
		CidrBlock:      *vcnData.CidrBlock,
		CompartmentID:  *vcnData.CompartmentId,
		TimeCreated:    vcnData.TimeCreated,
		FreeformTags:   vcnData.FreeformTags,
	}, nil
}

// CreateOrGetSubnet creates a subnet or returns existing one
func (o *OCIService) CreateOrGetSubnet(ctx context.Context, vcnID, displayName, availabilityDomain string) (*OCISubnet, error) {
	// First, try to find existing subnet
	subnet, err := o.findSubnetByName(ctx, vcnID, displayName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for existing subnet: %w", err)
	}

	if subnet != nil {
		return subnet, nil
	}

	// Create new subnet
	cidrBlock := "10.0.1.0/24"
	createSubnetRequest := core.CreateSubnetRequest{
		CreateSubnetDetails: core.CreateSubnetDetails{
			CidrBlock:          &cidrBlock,
			CompartmentId:      &o.tenancyOCID,
			DisplayName:        &displayName,
			VcnId:              &vcnID,
			AvailabilityDomain: &availabilityDomain,
			FreeformTags: map[string]string{
				"managed_by": "xanthus",
				"purpose":    "k3s-cluster",
			},
		},
	}

	response, err := o.networkClient.CreateSubnet(ctx, createSubnetRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet: %w", err)
	}

	subnetData := response.Subnet
	return &OCISubnet{
		ID:             *subnetData.Id,
		DisplayName:    *subnetData.DisplayName,
		LifecycleState: string(subnetData.LifecycleState),
		CidrBlock:      *subnetData.CidrBlock,
		CompartmentID:  *subnetData.CompartmentId,
		VcnID:          *subnetData.VcnId,
		TimeCreated:    subnetData.TimeCreated,
		FreeformTags:   subnetData.FreeformTags,
	}, nil
}

// CreateOrGetInternetGateway creates an internet gateway or returns existing one
func (o *OCIService) CreateOrGetInternetGateway(ctx context.Context, vcnID, displayName string) (*core.InternetGateway, error) {
	// First, try to find existing internet gateway
	igw, err := o.findInternetGatewayByName(ctx, vcnID, displayName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for existing internet gateway: %w", err)
	}

	if igw != nil {
		return igw, nil
	}

	// Create new internet gateway
	createIgwRequest := core.CreateInternetGatewayRequest{
		CreateInternetGatewayDetails: core.CreateInternetGatewayDetails{
			CompartmentId: &o.tenancyOCID,
			DisplayName:   &displayName,
			VcnId:         &vcnID,
			IsEnabled:     common.Bool(true),
			FreeformTags: map[string]string{
				"managed_by": "xanthus",
				"purpose":    "k3s-cluster",
			},
		},
	}

	response, err := o.networkClient.CreateInternetGateway(ctx, createIgwRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create internet gateway: %w", err)
	}

	return &response.InternetGateway, nil
}

// CreateOrGetSecurityList creates a security list or returns existing one
func (o *OCIService) CreateOrGetSecurityList(ctx context.Context, vcnID, displayName string) (*core.SecurityList, error) {
	// First, try to find existing security list
	secList, err := o.findSecurityListByName(ctx, vcnID, displayName)
	if err != nil {
		return nil, fmt.Errorf("failed to search for existing security list: %w", err)
	}

	if secList != nil {
		return secList, nil
	}

	// Create security rules
	egressRules := []core.EgressSecurityRule{
		{
			Protocol:    common.String("all"),
			Destination: common.String("0.0.0.0/0"),
		},
	}

	ingressRules := []core.IngressSecurityRule{
		{
			Protocol: common.String("6"), // TCP
			Source:   common.String("0.0.0.0/0"),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(22),
					Max: common.Int(22),
				},
			},
		},
		{
			Protocol: common.String("6"), // TCP
			Source:   common.String("0.0.0.0/0"),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(443),
					Max: common.Int(443),
				},
			},
		},
		{
			Protocol: common.String("6"), // TCP
			Source:   common.String("0.0.0.0/0"),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(80),
					Max: common.Int(80),
				},
			},
		},
		{
			Protocol: common.String("6"), // TCP
			Source:   common.String("0.0.0.0/0"),
			TcpOptions: &core.TcpOptions{
				DestinationPortRange: &core.PortRange{
					Min: common.Int(6443),
					Max: common.Int(6443),
				},
			},
		},
	}

	// Create new security list
	createSecListRequest := core.CreateSecurityListRequest{
		CreateSecurityListDetails: core.CreateSecurityListDetails{
			CompartmentId:        &o.tenancyOCID,
			DisplayName:          &displayName,
			VcnId:                &vcnID,
			EgressSecurityRules:  egressRules,
			IngressSecurityRules: ingressRules,
			FreeformTags: map[string]string{
				"managed_by": "xanthus",
				"purpose":    "k3s-cluster",
			},
		},
	}

	response, err := o.networkClient.CreateSecurityList(ctx, createSecListRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to create security list: %w", err)
	}

	return &response.SecurityList, nil
}

// ConfigureNetworkResources sets up complete network infrastructure
func (o *OCIService) ConfigureNetworkResources(ctx context.Context, baseName, availabilityDomain string) (string, string, error) {
	// Create VCN
	vcnName := fmt.Sprintf("%s-vcn-%s", baseName, strings.ToLower(o.region))
	vcn, err := o.CreateOrGetVCN(ctx, vcnName)
	if err != nil {
		return "", "", fmt.Errorf("failed to create VCN: %w", err)
	}

	// Create Internet Gateway
	igwName := fmt.Sprintf("%s-igw", baseName)
	igw, err := o.CreateOrGetInternetGateway(ctx, vcn.ID, igwName)
	if err != nil {
		return "", "", fmt.Errorf("failed to create internet gateway: %w", err)
	}

	// Create Security List
	secListName := fmt.Sprintf("%s-security-list", baseName)
	secList, err := o.CreateOrGetSecurityList(ctx, vcn.ID, secListName)
	if err != nil {
		return "", "", fmt.Errorf("failed to create security list: %w", err)
	}

	// Create Subnet
	subnetName := fmt.Sprintf("%s-public-subnet-%s", baseName, strings.ToLower(o.region))
	subnet, err := o.CreateOrGetSubnet(ctx, vcn.ID, subnetName, availabilityDomain)
	if err != nil {
		return "", "", fmt.Errorf("failed to create subnet: %w", err)
	}

	// Update route table to include internet gateway
	err = o.updateRouteTable(ctx, vcn.ID, *igw.Id)
	if err != nil {
		return "", "", fmt.Errorf("failed to update route table: %w", err)
	}

	// Update subnet to use the security list
	err = o.updateSubnetSecurityList(ctx, subnet.ID, *secList.Id)
	if err != nil {
		return "", "", fmt.Errorf("failed to update subnet security list: %w", err)
	}

	return vcn.ID, subnet.ID, nil
}

// Helper methods for finding existing resources

func (o *OCIService) findVCNByName(ctx context.Context, displayName string) (*OCIVirtualCloudNetwork, error) {
	listVcnsRequest := core.ListVcnsRequest{
		CompartmentId: &o.tenancyOCID,
		DisplayName:   &displayName,
	}

	response, err := o.networkClient.ListVcns(ctx, listVcnsRequest)
	if err != nil {
		return nil, err
	}

	for _, vcn := range response.Items {
		if vcn.FreeformTags["managed_by"] == "xanthus" {
			return &OCIVirtualCloudNetwork{
				ID:             *vcn.Id,
				DisplayName:    *vcn.DisplayName,
				LifecycleState: string(vcn.LifecycleState),
				CidrBlock:      *vcn.CidrBlock,
				CompartmentID:  *vcn.CompartmentId,
				TimeCreated:    vcn.TimeCreated,
				FreeformTags:   vcn.FreeformTags,
			}, nil
		}
	}

	return nil, nil
}

func (o *OCIService) findSubnetByName(ctx context.Context, vcnID, displayName string) (*OCISubnet, error) {
	listSubnetsRequest := core.ListSubnetsRequest{
		CompartmentId: &o.tenancyOCID,
		VcnId:         &vcnID,
		DisplayName:   &displayName,
	}

	response, err := o.networkClient.ListSubnets(ctx, listSubnetsRequest)
	if err != nil {
		return nil, err
	}

	for _, subnet := range response.Items {
		if subnet.FreeformTags["managed_by"] == "xanthus" {
			return &OCISubnet{
				ID:             *subnet.Id,
				DisplayName:    *subnet.DisplayName,
				LifecycleState: string(subnet.LifecycleState),
				CidrBlock:      *subnet.CidrBlock,
				CompartmentID:  *subnet.CompartmentId,
				VcnID:          *subnet.VcnId,
				TimeCreated:    subnet.TimeCreated,
				FreeformTags:   subnet.FreeformTags,
			}, nil
		}
	}

	return nil, nil
}

func (o *OCIService) findInternetGatewayByName(ctx context.Context, vcnID, displayName string) (*core.InternetGateway, error) {
	listIgwsRequest := core.ListInternetGatewaysRequest{
		CompartmentId: &o.tenancyOCID,
		VcnId:         &vcnID,
		DisplayName:   &displayName,
	}

	response, err := o.networkClient.ListInternetGateways(ctx, listIgwsRequest)
	if err != nil {
		return nil, err
	}

	for _, igw := range response.Items {
		if igw.FreeformTags["managed_by"] == "xanthus" {
			return &igw, nil
		}
	}

	return nil, nil
}

func (o *OCIService) findSecurityListByName(ctx context.Context, vcnID, displayName string) (*core.SecurityList, error) {
	listSecListsRequest := core.ListSecurityListsRequest{
		CompartmentId: &o.tenancyOCID,
		VcnId:         &vcnID,
		DisplayName:   &displayName,
	}

	response, err := o.networkClient.ListSecurityLists(ctx, listSecListsRequest)
	if err != nil {
		return nil, err
	}

	for _, secList := range response.Items {
		if secList.FreeformTags["managed_by"] == "xanthus" {
			return &secList, nil
		}
	}

	return nil, nil
}

func (o *OCIService) updateRouteTable(ctx context.Context, vcnID, igwID string) error {
	// Get default route table
	listRouteTablesRequest := core.ListRouteTablesRequest{
		CompartmentId: &o.tenancyOCID,
		VcnId:         &vcnID,
	}

	response, err := o.networkClient.ListRouteTables(ctx, listRouteTablesRequest)
	if err != nil {
		return err
	}

	if len(response.Items) == 0 {
		return fmt.Errorf("no route tables found for VCN")
	}

	// Update the default route table
	routeTableID := response.Items[0].Id
	routeRules := []core.RouteRule{
		{
			NetworkEntityId: &igwID,
			Destination:     common.String("0.0.0.0/0"),
		},
	}

	updateRouteTableRequest := core.UpdateRouteTableRequest{
		RtId: routeTableID,
		UpdateRouteTableDetails: core.UpdateRouteTableDetails{
			RouteRules: routeRules,
		},
	}

	_, err = o.networkClient.UpdateRouteTable(ctx, updateRouteTableRequest)
	return err
}

func (o *OCIService) updateSubnetSecurityList(ctx context.Context, subnetID, secListID string) error {
	// Get current subnet
	getSubnetRequest := core.GetSubnetRequest{
		SubnetId: &subnetID,
	}

	_, err := o.networkClient.GetSubnet(ctx, getSubnetRequest)
	if err != nil {
		return err
	}

	// Update with new security list
	securityListIds := []string{secListID}
	updateSubnetRequest := core.UpdateSubnetRequest{
		SubnetId: &subnetID,
		UpdateSubnetDetails: core.UpdateSubnetDetails{
			SecurityListIds: securityListIds,
		},
	}

	_, err = o.networkClient.UpdateSubnet(ctx, updateSubnetRequest)
	return err
}

// Helper functions for handling optional values
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func getFloat32Value(ptr *float32) float32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func getIntValue(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func getBoolValue(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

// flattenDefinedTags converts OCI's nested defined tags to a flat map
func flattenDefinedTags(definedTags map[string]map[string]interface{}) map[string]interface{} {
	flattened := make(map[string]interface{})
	for namespace, tags := range definedTags {
		for key, value := range tags {
			flatKey := fmt.Sprintf("%s.%s", namespace, key)
			flattened[flatKey] = value
		}
	}
	return flattened
}