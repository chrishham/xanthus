# OCI Automation Implementation Plan

## Overview

This document outlines the implementation plan for automating Oracle Cloud Infrastructure (OCI) deployment in Xanthus, following the proven patterns established by the Hetzner integration.

## Current State Analysis

### Existing OCI Implementation
- **Manual Setup Workflow**: Users manually create OCI instances and register them with Xanthus
- **Provider Support**: OCI recognized as a supported provider with Ubuntu SSH user defaults
- **K3s Integration**: Automatic K3s setup via SSH after manual registration
- **Application Deployment**: Full support for code-server, ArgoCD, and other applications

### Hetzner Automation Pattern
- **Full API Integration**: Complete instance lifecycle management via Hetzner Cloud API
- **Cloud-init Automation**: Automated K3s, Helm, and SSL setup during instance creation
- **Provider Abstraction**: Clean separation between provider-specific logic and common VPS operations
- **Consistent Architecture**: Same REST API endpoints and configuration storage across providers

## Implementation Requirements

### 1. OCI API Integration

**Core Components:**
- **OCI Go SDK**: Official Oracle Cloud SDK for Go
- **Authentication**: API key-based authentication with tenancy/user OCIDs
- **Instance Management**: Create, delete, list, and power control operations
- **Network Resources**: VCN, subnet, security group, and public IP management
- **SSH Key Management**: Key pair creation and registration

**Required Environment Variables:**
```bash
OCI_TENANCY_OCID="ocid1.tenancy.oc1..xxx"
OCI_USER_OCID="ocid1.user.oc1..xxx"
OCI_REGION="us-phoenix-1"
OCI_FINGERPRINT="aa:bb:cc:dd:ee:ff"
OCI_PRIVATE_KEY_PATH="/path/to/oci_api_key.pem"
```

### 2. Service Architecture

**OCIService** (`internal/services/oci.go`):
```go
type OCIService struct {
    client         *core.ComputeClient
    identityClient *identity.IdentityClient
    networkClient  *core.VirtualNetworkClient
    tenancyOCID    string
    region         string
}
```

**Key Methods:**
- `CreateInstance()` - Full instance provisioning with network setup
- `DeleteInstance()` - Complete resource cleanup
- `ListInstances()` - Inventory management and status checking
- `ConfigureNetwork()` - VCN, subnet, and security group automation
- `ManageSSHKeys()` - SSH key lifecycle management
- `PowerActions()` - Start, stop, and reboot operations

### 3. Network Automation

**Auto-Network Configuration:**
- **VCN Creation**: Automatically create Virtual Cloud Network if none exists
- **Subnet Setup**: Public subnet with internet gateway
- **Security Groups**: SSH access (port 22) and custom application ports
- **Public IP**: Automatic assignment and DNS configuration

**Network Naming Convention:**
- VCN: `xanthus-vcn-{region}`
- Subnet: `xanthus-public-subnet-{region}`
- Security List: `xanthus-security-list`

### 4. Instance Configuration

**Default Instance Settings:**
- **Shape**: VM.Standard.E2.1.Micro (Always Free tier eligible)
- **Image**: Ubuntu 22.04 or Ubuntu 24.04 LTS
- **Boot Volume**: 50GB (Always Free tier limit)
- **Network**: Auto-created VCN with public subnet
- **SSH Access**: Xanthus-managed SSH key

**Cost Optimization:**
- Target Always Free tier instances by default
- Provide shape selection for paid instances
- Real-time cost estimation integration

### 5. Cloud-init Integration

**OCI Cloud-init** (`internal/services/oci-cloudinit.yaml`):
```yaml
#cloud-config
timezone: {TIMEZONE}
packages:
  - curl
  - wget
  - git
  - apt-transport-https
  - ca-certificates
  - gnupg
  - jq

write_files:
  - path: /opt/xanthus/setup.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      # OCI-specific setup script
      # K3s installation
      # Helm installation
      # SSL certificate setup
      # KUBECONFIG configuration

runcmd:
  - /opt/xanthus/setup.sh
```

**Setup Script Features:**
- Ubuntu package updates
- K3s installation and verification
- Helm installation
- SSL certificate configuration
- KUBECONFIG environment setup
- Firewall configuration for OCI
- Status tracking and logging

## Implementation Phases

### Phase 1: Core OCI Service (High Priority)

**Deliverables:**
- `internal/services/oci.go` - Core OCI service implementation
- OCI API client with authentication
- Basic instance lifecycle management
- Network automation (VCN, subnet, security groups)
- SSH key management
- Error handling and logging

**Timeline:** 2-3 weeks

### Phase 2: Integration & Automation (Medium Priority)

**Deliverables:**
- Update `internal/services/provider_resolver.go` to support OCI automation
- Extend `internal/handlers/vps/vps_lifecycle.go` with OCI endpoints
- Create `internal/services/oci-cloudinit.yaml` configuration
- Update VPS creation handlers
- Integration testing

**Timeline:** 1-2 weeks

### Phase 3: UI Enhancement (Low Priority)

**Deliverables:**
- Update VPS creation wizard to support OCI automation
- Add OCI-specific configuration options (region, shape selection)
- Real-time status monitoring
- Cost estimation integration
- Error handling improvements

**Timeline:** 1 week

## User Experience

### Simplest Approach: Environment Variables

**Setup Process:**
1. **OCI API Setup**: User creates API key in OCI console
2. **Environment Configuration**: Set OCI credentials via environment variables
3. **One-Click Creation**: Same VPS creation flow as Hetzner
4. **Automatic Setup**: Cloud-init handles K3s, Helm, and SSL configuration

**Benefits:**
- ✅ Consistent with Hetzner pattern (`HETZNER_TOKEN`)
- ✅ Secure credential management
- ✅ No UI complexity for initial implementation
- ✅ Standard OCI authentication method

### Alternative Approaches

**Configuration File:**
- Support `~/.oci/config` standard OCI configuration
- Automatic credential discovery
- Multiple profile support

**In-App Configuration:**
- Store OCI credentials in Cloudflare KV (encrypted)
- Web-based credential management
- Team sharing capabilities

## Technical Considerations

### Security
- **API Key Security**: Secure storage and transmission of OCI credentials
- **Network Security**: Proper security group configuration
- **SSH Key Management**: Secure key generation and storage
- **Encryption**: End-to-end encryption for sensitive data

### Performance
- **API Rate Limits**: Respect OCI API rate limits
- **Caching**: Cache instance and network information
- **Parallel Operations**: Concurrent API calls where possible
- **Background Processing**: Non-blocking operations for long-running tasks

### Error Handling
- **Validation**: Comprehensive input validation
- **Graceful Degradation**: Fallback mechanisms for partial failures
- **Cleanup**: Automatic resource cleanup on failures
- **User Feedback**: Clear error messages and recovery instructions

### Monitoring
- **Logging**: Comprehensive logging for debugging
- **Metrics**: Performance and usage metrics
- **Alerting**: Error detection and notification
- **Status Tracking**: Real-time status monitoring

## Testing Strategy

### Unit Tests
- OCI service method testing
- Network configuration validation
- Error handling scenarios
- Authentication testing

### Integration Tests
- End-to-end instance creation
- Network setup validation
- Cloud-init execution testing
- Provider resolution testing

### E2E Tests
- Complete VPS lifecycle testing
- Application deployment on OCI instances
- Cost tracking validation
- UI workflow testing

## Success Metrics

### Technical Metrics
- **Instance Creation Time**: < 5 minutes from request to ready
- **API Response Time**: < 2 seconds for common operations
- **Success Rate**: > 95% for instance creation
- **Error Recovery**: < 10% manual intervention required

### User Experience Metrics
- **Setup Time**: < 15 minutes for initial OCI configuration
- **Learning Curve**: Same UI/UX as Hetzner integration
- **Support Tickets**: < 5% increase in support volume
- **User Satisfaction**: > 90% positive feedback

## Risk Mitigation

### Technical Risks
- **OCI API Changes**: Monitor OCI SDK updates and deprecations
- **Network Complexity**: Comprehensive testing of network configurations
- **Cost Overruns**: Clear cost warnings and Always Free tier defaults
- **Security Vulnerabilities**: Regular security audits and updates

### User Adoption Risks
- **Complexity**: Keep initial implementation simple
- **Documentation**: Comprehensive setup and troubleshooting guides
- **Support**: Dedicated support for OCI-specific issues
- **Migration**: Smooth transition from manual to automated workflow

## Future Enhancements

### Advanced Features
- **Multi-Region Support**: Deploy across multiple OCI regions
- **Auto-Scaling**: Dynamic instance scaling based on demand
- **Backup Integration**: Automated backup and disaster recovery
- **Monitoring Integration**: OCI monitoring and alerting

### Enterprise Features
- **Compartment Management**: Multi-tenancy support
- **Identity Integration**: OCI Identity and Access Management
- **Compliance**: Security and compliance reporting
- **Cost Management**: Advanced cost allocation and budgeting

## Conclusion

This implementation plan provides a comprehensive approach to automating OCI deployment in Xanthus, leveraging the proven patterns from the Hetzner integration while addressing OCI-specific requirements. The phased approach ensures minimal disruption to existing functionality while delivering immediate value to users.

The focus on the Always Free tier and environment variable configuration provides the simplest possible user experience while maintaining the security and reliability standards established by the existing Hetzner automation.