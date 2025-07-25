package services

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/chrishham/xanthus/internal/models"
)

// generateValuesFile generates a Helm values file using template-based approach
func (s *SimpleApplicationService) generateValuesFile(predefinedApp *models.PredefinedApplication, subdomain, domain, releaseName, timezone string) (string, error) {
	// Check if a values template is specified in the configuration
	if predefinedApp.HelmChart.ValuesTemplate != "" {
		return s.generateFromTemplate(predefinedApp, subdomain, domain, releaseName, timezone)
	}

	// Fallback to minimal values if no template is specified
	return s.generateMinimalValues(predefinedApp, subdomain, domain, releaseName, timezone)
}

// generateFromTemplate generates values from a template file with placeholder substitution
func (s *SimpleApplicationService) generateFromTemplate(predefinedApp *models.PredefinedApplication, subdomain, domain, releaseName, timezone string) (string, error) {
	templatePath := fmt.Sprintf("internal/templates/applications/%s", predefinedApp.HelmChart.ValuesTemplate)

	// Read the template file - use embedded FS if available
	var templateContent []byte
	var err error

	if s.embedFS != nil {
		templateContent, err = fs.ReadFile(*s.embedFS, templatePath)
		if err != nil {
			return "", fmt.Errorf("failed to read embedded template file %s: %w", templatePath, err)
		}
	} else {
		// Fallback to filesystem
		templateContent, err = os.ReadFile(templatePath)
		if err != nil {
			return "", fmt.Errorf("failed to read template file %s: %w", templatePath, err)
		}
	}

	// Use UTC as default timezone if none specified
	if timezone == "" {
		timezone = "UTC"
	}

	// Prepare placeholder values
	placeholders := map[string]string{
		"VERSION":      predefinedApp.Version,
		"SUBDOMAIN":    subdomain,
		"DOMAIN":       domain,
		"RELEASE_NAME": releaseName,
		"TIMEZONE":     timezone,
	}

	// Add any additional placeholders from the configuration
	// These placeholders can contain Go template syntax like {{.Version}} that needs to be resolved
	templateData := map[string]string{
		"Version":   predefinedApp.Version,
		"Subdomain": subdomain,
		"Domain":    domain,
	}

	for key, templateValue := range predefinedApp.HelmChart.Placeholders {
		// Resolve Go template syntax in the placeholder value
		resolvedValue := templateValue
		for templateKey, actualValue := range templateData {
			resolvedValue = strings.ReplaceAll(resolvedValue, fmt.Sprintf("{{.%s}}", templateKey), actualValue)
		}
		placeholders[key] = resolvedValue
	}

	// Replace placeholders in the template
	content := string(templateContent)
	for placeholder, value := range placeholders {
		content = strings.ReplaceAll(content, fmt.Sprintf("{{%s}}", placeholder), value)
	}

	return content, nil
}

// generateMinimalValues generates minimal values when no template is available
func (s *SimpleApplicationService) generateMinimalValues(predefinedApp *models.PredefinedApplication, subdomain, domain, releaseName, timezone string) (string, error) {
	// Use UTC as default timezone if none specified
	if timezone == "" {
		timezone = "UTC"
	}

	// Generate basic ingress configuration for any application with timezone support
	return fmt.Sprintf(`
# Minimal values generated for %s
ingress:
  enabled: true
  className: "traefik"
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
  hosts:
    - host: %s.%s
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: %s-tls
      hosts:
        - %s.%s

# Application version
image:
  tag: "%s"

# Universal timezone configuration
env:
  - name: TZ
    value: "%s"

# Pod-level timezone configuration for containers that support it
podSpec:
  env:
    - name: TZ
      value: "%s"
`, predefinedApp.ID, subdomain, domain, releaseName, subdomain, domain, predefinedApp.Version, timezone, timezone), nil
}
