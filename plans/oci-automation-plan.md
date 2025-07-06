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

**Single Authentication Token:**
```bash
OCI_AUTH_TOKEN="eyJ0ZW5hbmN5IjoiY2lkMS50ZW5hbmN5Lm9jMS4uYWFhYSIsInVzZXIiOiJvY2lkMS51c2VyLm9jMS4uYmJiYiIsInJlZ2lvbiI6InVzLXBob2VuaXgtMSIsImZpbmdlcnByaW50IjoiYWE6YmI6Y2M6ZGQ6ZWU6ZmYiLCJwcml2YXRlX2tleSI6Ii0tLS0tQkVHSU4gUFJJVkFURSBLRVktLS0tLVxuTUlJRXZBSUJBREFOQmdrcWhraUc5dzBCQVFFRkFBU0NCS2t3Z2dTbEFnRUFBb0lCQVFDNklcbi4uLlxuLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLVxuIn0="
```

**Token Structure (Base64 encoded JSON):**
```json
{
  "tenancy": "ocid1.tenancy.oc1..aaaa",
  "user": "ocid1.user.oc1..bbbb", 
  "region": "us-phoenix-1",
  "fingerprint": "aa:bb:cc:dd:ee:ff",
  "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQC6I\n...\n-----END PRIVATE KEY-----\n"
}
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

### Phase 3: UI Enhancement (Medium Priority)

**Deliverables:**
- **OCI Token Generator UI**: Built-in token generator in VPS creation wizard
- **OCI Credentials Instructions**: Step-by-step guidance for OCI Console setup
- **Token Validation**: OCI auth token validation similar to Hetzner API key
- **Region/Shape Selection**: OCI-specific configuration options
- **Error Handling**: User-friendly error messages and recovery instructions

**UI Components:**
- Token generator form with validation
- Instructions panel with OCI Console navigation
- Region dropdown with Always Free tier regions
- Shape selection with cost estimates
- Progress indicators and loading states

**Timeline:** 1-2 weeks

## User Experience

### Single OCI Auth Token Approach

**Setup Process:**
1. **OCI Console Setup**: User creates API key in OCI console (Identity & Security → Users → API Keys)
2. **Token Generation**: User enters OCI credentials in Xanthus UI token generator
3. **Token Validation**: System validates OCI auth token like Hetzner API key
4. **One-Click Creation**: Same VPS creation flow as Hetzner
5. **Automatic Setup**: Cloud-init handles K3s, Helm, and SSL configuration

**UI Implementation:**

**Step 2: OCI Auth Token Configuration**
- **Instructions Panel**: Clear guidance on where to get OCI credentials
- **Built-in Token Generator**: 
  - Form fields for Tenancy OCID, User OCID, Region, Fingerprint, OCI API Private Key
  - Generate button creates single base64-encoded auth token
  - No external tools or scripts needed
- **Token Input**: Option to paste pre-generated token
- **Validation**: Same validation flow as Hetzner API key

**Where Users Get OCI Credentials:**
1. **OCI Console** → Identity & Security → Users → [User] → API Keys
2. **Generate OCI API Key Pair** (RSA key for OCI API authentication, not SSH)
3. **Copy Configuration Info** from OCI Console after key upload
4. **Use Xanthus Token Generator** to create single auth token

**Note**: SSH access is handled automatically by Xanthus (same as Hetzner) - users only need OCI API credentials.

**Token Generator Features:**
```html
<!-- Built into VPS creation wizard Step 2 -->
<div class="token-generator">
  <h4>🔧 Generate OCI Auth Token</h4>
  <input placeholder="Tenancy OCID: ocid1.tenancy.oc1..aaaa..." />
  <input placeholder="User OCID: ocid1.user.oc1..bbbb..." />
  <select>Region selection</select>
  <input placeholder="Key Fingerprint: aa:bb:cc:dd..." />
  <textarea placeholder="OCI API Private Key content"></textarea>
  <button onclick="generateToken()">Generate Auth Token</button>
</div>
```

**Benefits:**
- ✅ **Same UX as Hetzner**: Single token input and validation
- ✅ **No File Management**: Everything handled in browser/memory
- ✅ **Built-in Generator**: No external tools or scripts required
- ✅ **Secure**: Token encrypted in Cloudflare KV like Hetzner key
- ✅ **User-Friendly**: Step-by-step guidance with clear instructions
- ✅ **Portable**: Generated token works across environments

**User Complexity:**
- **First-time Setup**: ~10-15 minutes (same as Hetzner)
- **Token Generation**: ~2 minutes using built-in generator
- **Subsequent Usage**: Automatic (token stored in KV)

### Implementation Components

**Frontend (`internal/utils/oci.go`):**
```go
type OCICredentials struct {
    Tenancy     string `json:"tenancy"`
    User        string `json:"user"`
    Region      string `json:"region"`
    Fingerprint string `json:"fingerprint"`
    PrivateKey  string `json:"private_key"`
}

func DecodeOCIAuthToken(token string) (*OCICredentials, error)
func GetOCIAuthToken(token, accountID string) (string, error)
```

**Backend Handler:**
```go
func (h *BaseHandler) getOCIAuthToken(c *gin.Context, token, accountID string) (*OCICredentials, bool)
func (h *BaseHandler) validateOCIToken() // Similar to validateHetznerKey()
```

**JavaScript (vps-creation-wizard.js):**
```javascript
generateOCIToken()     // Creates base64 token from form inputs
validateOCIToken()     // Validates token via API like Hetzner
isOCICredsComplete()   // Form validation
```

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
- **Setup Time**: < 15 minutes for initial OCI configuration (same as Hetzner)
- **Token Generation**: < 2 minutes using built-in generator
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

**Key Innovation - Single Auth Token Approach:**
The plan addresses the complexity of OCI's multi-credential authentication by implementing a single `OCI_AUTH_TOKEN` similar to Hetzner's `HETZNER_TOKEN`. This provides users with the exact same experience across both providers while handling OCI's complexity behind the scenes.

**Built-in Token Generator:**
Users can generate their OCI auth token directly in the Xanthus UI without external tools or scripts. The token generator guides users through collecting OCI credentials from the Oracle Console and creates a single base64-encoded token for authentication.

**No File Management:**
Unlike traditional OCI CLI tools that require config files, Xanthus handles everything in memory and encrypted KV storage, maintaining the security and simplicity that users expect from cloud automation tools.

The focus on the Always Free tier, built-in token generation, and consistent UX provides the simplest possible user experience while maintaining the security and reliability standards established by the existing Hetzner automation.