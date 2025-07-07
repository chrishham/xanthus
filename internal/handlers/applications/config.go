package applications

import "time"

// ApplicationConfig holds configuration for application handlers
type ApplicationConfig struct {
	// HTTP request timeout
	RequestTimeout time.Duration

	// SSH connection timeout
	SSHTimeout time.Duration

	// Helm operation timeout
	HelmTimeout time.Duration

	// Password validation settings
	MinPasswordLength int

	// Supported application types
	SupportedAppTypes []string

	// Default namespaces for different app types
	DefaultNamespaces map[string]string

	// Version check limits
	MaxVersionsToCheck int
}

// DefaultConfig returns the default configuration
func DefaultConfig() *ApplicationConfig {
	return &ApplicationConfig{
		RequestTimeout:    30 * time.Second,
		SSHTimeout:        10 * time.Second,
		HelmTimeout:       300 * time.Second, // 5 minutes
		MinPasswordLength: 8,
		SupportedAppTypes: []string{"code-server", "argocd"},
		DefaultNamespaces: map[string]string{
			"code-server": "code-server",
			"argocd":      "argocd",
		},
		MaxVersionsToCheck: 50,
	}
}

// ApplicationConstants holds constant values used throughout the application handlers
type ApplicationConstants struct {
	// KV storage prefixes
	AppPrefix      string
	PasswordPrefix string
	VPSPrefix      string

	// Secret names
	SSHKeySecret    string
	TLSSecretSuffix string

	// Release name patterns
	ReleaseNameFormat string

	// Helm chart repositories
	ArgoCDRepo     string
	CodeServerRepo string

	// Default Docker image tags
	DefaultCodeServerTag string
	DefaultArgoCDTag     string
}

// Constants returns the application constants
func Constants() *ApplicationConstants {
	return &ApplicationConstants{
		AppPrefix:      "app:",
		PasswordPrefix: ":password",
		VPSPrefix:      "vps:",

		SSHKeySecret:    "config:ssl:csr",
		TLSSecretSuffix: "-tls",

		ReleaseNameFormat: "%s-%s", // subdomain-appid

		ArgoCDRepo:     "oci://ghcr.io/argoproj/argo-helm/argo-cd",
		CodeServerRepo: "https://github.com/coder/code-server-helm.git",

		DefaultCodeServerTag: "latest",
		DefaultArgoCDTag:     "8.1.2",
	}
}

// ApplicationStatus represents the possible states of an application
type ApplicationStatus string

const (
	StatusPending   ApplicationStatus = "pending"
	StatusDeploying ApplicationStatus = "deploying"
	StatusDeployed  ApplicationStatus = "deployed"
	StatusFailed    ApplicationStatus = "failed"
	StatusUpdating  ApplicationStatus = "updating"
	StatusDeleting  ApplicationStatus = "deleting"
)

// ApplicationType represents supported application types
type ApplicationType string

const (
	TypeCodeServer ApplicationType = "code-server"
	TypeArgoCD     ApplicationType = "argocd"
	TypeXanthus    ApplicationType = "xanthus"
)

// IsValidType checks if the application type is supported
func (at ApplicationType) IsValid() bool {
	switch at {
	case TypeCodeServer, TypeArgoCD, TypeXanthus:
		return true
	default:
		return false
	}
}

// ErrorMessages contains common error messages
var ErrorMessages = struct {
	Unauthorized           string
	InvalidRequestData     string
	ApplicationNotFound    string
	InvalidApplicationType string
	PasswordTooShort       string
	VPSConnectionFailed    string
	HelmOperationFailed    string
	KVStorageFailed        string
}{
	Unauthorized:           "Unauthorized",
	InvalidRequestData:     "Invalid request data",
	ApplicationNotFound:    "Application not found",
	InvalidApplicationType: "Invalid application type",
	PasswordTooShort:       "Password must be at least 8 characters long",
	VPSConnectionFailed:    "Failed to connect to VPS",
	HelmOperationFailed:    "Helm operation failed",
	KVStorageFailed:        "Failed to access KV storage",
}

// SuccessMessages contains common success messages
var SuccessMessages = struct {
	ApplicationCreated string
	ApplicationUpdated string
	ApplicationDeleted string
	PasswordUpdated    string
}{
	ApplicationCreated: "Application created successfully",
	ApplicationUpdated: "Application updated successfully",
	ApplicationDeleted: "Application deleted successfully",
	PasswordUpdated:    "Password updated successfully",
}
