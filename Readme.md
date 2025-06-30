## config:ssl:csr
● Based on my analysis of the code, config:ssl:csr is a Certificate Signing Request (CSR) 
  configuration stored in Cloudflare KV that contains:

  What config:ssl:csr contains:

  {
    "csr": "-----BEGIN CERTIFICATE REQUEST-----\n...\n-----END CERTIFICATE REQUEST-----",
    "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----",
    "created_at": "2024-01-01T00:00:00Z"
  }

  Purpose and Usage:

  1. SSL Certificate Management: The CSR is used to request SSL certificates from Cloudflare for
   domains managed by Xanthus
  2. SSH Authentication: The private key from the CSR is dual-purposed as the SSH private key
  for connecting to VPS instances
  3. Auto-generated on Login: When a user logs in for the first time, if this config doesn't
  exist, it's automatically generated and stored

  Key Functions:

  - SSL Certificates: The CSR is sent to Cloudflare to obtain origin certificates for domains
  - SSH Access: The same private key is converted to SSH format and used for VPS authentication
  - VPS Creation: When creating VPS instances, the public key derived from this private key is
  added to the server
  - Application Deployment: Used to SSH into VPS instances for Helm deployments and management

  Security Note:

  This is a critical security component - it's essentially the master key for both SSL
  certificates and SSH access to all VPS instances managed by Xanthus. It's generated once per
  user and reused across all their infrastructure.

## config:hetzner:api_key
  ● Based on my analysis of the code, config:hetzner:api_key is the Hetzner Cloud API key stored
  in Cloudflare KV that enables Xanthus to manage VPS instances on Hetzner Cloud.

  What config:hetzner:api_key contains:

  - Encrypted Hetzner API Key: The user's Hetzner Cloud API token, encrypted using the
  Cloudflare token as the encryption key
  - Format: Stored as an encrypted string in KV, decrypted when needed

  Purpose and Usage:

  1. VPS Management: Used to create, delete, power on/off, and reboot VPS instances on Hetzner
  Cloud
  2. Server Information: Fetches server lists, locations, server types, and pricing from Hetzner
   API
  3. Resource Provisioning: Creates servers with specific configurations (CPU, RAM, storage,
  location)

  Key Functions:

  - VPS Lifecycle: Creating and deleting VPS instances
  - Server Operations: Power management, rebooting servers
  - Infrastructure Discovery: Listing available server types, locations, and pricing
  - Network Management: Configuring public IPs and network settings

  Security Features:

  - Encryption: The API key is encrypted using the user's Cloudflare token before storage
  - Validation: Keys are validated against Hetzner API before storage
  - Masking: When displayed in UI, only first 4 and last 4 characters are shown (e.g.,
  "hcap_...xyz8")

  Setup Process:

  1. User enters their Hetzner API key in the setup page
  2. Key is validated by making a test call to Hetzner API
  3. Key is encrypted using the Cloudflare token
  4. Encrypted key is stored in KV as config:hetzner:api_key
  5. Key is decrypted and used for all subsequent Hetzner API calls

  Usage Examples:

  - Creating VPS: POST /vps/create uses this key to provision servers
  - Listing servers: GET /vps/list uses this key to fetch server information
  - Power management: Uses this key for start/stop/reboot operations
  - Getting server options: Uses this key to fetch available locations and server types

  This is essentially the authentication credential that allows Xanthus to act on behalf of the
  user in their Hetzner Cloud account.