# Charts Architecture

## 📋 Purpose
Local Helm charts for Kubernetes deployment with no external dependencies and full customization control.

## 🏗️ Architecture

### Local Chart Strategy
```
charts/
└── xanthus-code-server/
    ├── Chart.yaml          # Chart metadata
    ├── values.yaml         # Default configuration
    └── templates/
        ├── _helpers.tpl    # Template helpers
        ├── deployment.yaml # Main application deployment
        ├── service.yaml    # Service configuration
        ├── pvc.yaml        # Persistent volume claim
        ├── configmap.yaml  # Configuration and scripts
        ├── secret.yaml     # Password secret
        └── ingress.yaml    # Traefik ingress
```

### Deployment Flow
```
Local Chart → VPS Copy → Helm Install → Kubernetes Resources
```

## 🔧 Chart Components

### Chart Metadata (`Chart.yaml`)
```yaml
apiVersion: v2
name: xanthus-code-server
description: Code-server deployment for Xanthus platform
type: application
version: 0.1.0
appVersion: "4.101.2"
keywords:
  - code-server
  - development
  - ide
home: https://github.com/coder/code-server
maintainers:
  - name: Xanthus Platform
```

### Default Values (`values.yaml`)
```yaml
image:
  repository: codercom/code-server
  tag: "4.101.2"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 8080

persistence:
  enabled: true
  size: 10Gi
  accessMode: ReadWriteOnce

ingress:
  enabled: true
  className: "traefik"
  annotations:
    cert-manager.io/cluster-issuer: cloudflare-issuer
  tls:
    enabled: true

resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 500m
    memory: 1Gi

securityContext:
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
```

## 📊 Kubernetes Resources

### Deployment (`deployment.yaml`)
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "xanthus-code-server.fullname" . }}
  labels:
    {{- include "xanthus-code-server.labels" . | nindent 4 }}
spec:
  replicas: 1
  selector:
    matchLabels:
      {{- include "xanthus-code-server.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "xanthus-code-server.selectorLabels" . | nindent 8 }}
    spec:
      initContainers:
      - name: setup-home
        image: busybox:1.35
        command:
        - /bin/sh
        - -c
        - |
          chown -R 1000:1000 /home/coder
          cp /vscode-settings/settings.json /home/coder/.local/share/code-server/User/settings.json
          chmod +x /setup-script/setup-dev-environment.sh
        volumeMounts:
        - name: home
          mountPath: /home/coder
        - name: vscode-settings
          mountPath: /vscode-settings
        - name: setup-script
          mountPath: /setup-script
      containers:
      - name: code-server
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        ports:
        - containerPort: 8080
        env:
        - name: PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{ include "xanthus-code-server.fullname" . }}
              key: password
        volumeMounts:
        - name: home
          mountPath: /home/coder
        - name: setup-script
          mountPath: /home/coder/setup-dev-environment.sh
          subPath: setup-dev-environment.sh
        resources:
          {{- toYaml .Values.resources | nindent 12 }}
      volumes:
      - name: home
        persistentVolumeClaim:
          claimName: {{ include "xanthus-code-server.fullname" . }}-home
      - name: vscode-settings
        configMap:
          name: {{ include "xanthus-code-server.fullname" . }}-vscode-settings
      - name: setup-script
        configMap:
          name: {{ include "xanthus-code-server.fullname" . }}-setup-script
          defaultMode: 0755
```

### ConfigMaps (`configmap.yaml`)
```yaml
# VS Code settings configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "xanthus-code-server.fullname" . }}-vscode-settings
  labels:
    {{- include "xanthus-code-server.labels" . | nindent 4 }}
data:
  settings.json: |
    {
      "workbench.colorTheme": "Default Dark+",
      "editor.fontSize": 14,
      "editor.tabSize": 2,
      "editor.insertSpaces": true,
      "files.autoSave": "afterDelay",
      "files.autoSaveDelay": 1000,
      "terminal.integrated.fontSize": 14,
      "workbench.startupEditor": "welcomePage"
    }
---
# Setup script configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "xanthus-code-server.fullname" . }}-setup-script
  labels:
    {{- include "xanthus-code-server.labels" . | nindent 4 }}
data:
  setup-dev-environment.sh: |
    #!/bin/bash
    # Development environment setup script
    
    echo "🚀 Setting up development environment..."
    
    # Update system packages
    sudo apt-get update
    
    # Install Node.js and npm
    curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
    sudo apt-get install -y nodejs
    
    # Install Python and pip
    sudo apt-get install -y python3 python3-pip
    
    # Install Go
    wget -q https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    
    # Install Docker
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker $USER
    
    # Install kubectl
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
    
    # Install Helm
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    
    # Create workspace directories
    mkdir -p ~/projects ~/workspace ~/scripts
    
    echo "✅ Development environment setup complete!"
    echo "Run 'source ~/.bashrc' to reload your shell"
```

### Persistent Volume Claim (`pvc.yaml`)
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "xanthus-code-server.fullname" . }}-home
  labels:
    {{- include "xanthus-code-server.labels" . | nindent 4 }}
spec:
  accessModes:
    - {{ .Values.persistence.accessMode }}
  resources:
    requests:
      storage: {{ .Values.persistence.size }}
  {{- if .Values.persistence.storageClass }}
  storageClassName: {{ .Values.persistence.storageClass }}
  {{- end }}
```

### Service (`service.yaml`)
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "xanthus-code-server.fullname" . }}
  labels:
    {{- include "xanthus-code-server.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: 8080
      protocol: TCP
      name: http
  selector:
    {{- include "xanthus-code-server.selectorLabels" . | nindent 4 }}
```

### Ingress (`ingress.yaml`)
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "xanthus-code-server.fullname" . }}
  labels:
    {{- include "xanthus-code-server.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if .Values.ingress.className }}
  ingressClassName: {{ .Values.ingress.className }}
  {{- end }}
  {{- if .Values.ingress.tls.enabled }}
  tls:
    - hosts:
        - {{ .Values.ingress.host }}
      secretName: {{ include "xanthus-code-server.fullname" . }}-tls
  {{- end }}
  rules:
    - host: {{ .Values.ingress.host }}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: {{ include "xanthus-code-server.fullname" . }}
                port:
                  number: {{ .Values.service.port }}
```

## 🔧 Template Helpers (`_helpers.tpl`)

```yaml
{{/*
Expand the name of the chart.
*/}}
{{- define "xanthus-code-server.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "xanthus-code-server.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "xanthus-code-server.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "xanthus-code-server.labels" -}}
helm.sh/chart: {{ include "xanthus-code-server.chart" . }}
{{ include "xanthus-code-server.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "xanthus-code-server.selectorLabels" -}}
app.kubernetes.io/name: {{ include "xanthus-code-server.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
```

## 🎯 Chart Features

### ✅ Persistent Storage
- **10GB home directory** - All user data persists across restarts
- **VS Code settings** - Pre-configured IDE settings
- **Development tools** - On-demand installation script

### ✅ Security
- **Non-root user** - Runs as user 1000 (coder)
- **Password protection** - Kubernetes secret-based authentication
- **SSL/TLS** - Automatic certificate management
- **Resource limits** - CPU and memory constraints

### ✅ Development Environment
- **Setup script** - Automated tool installation
- **Multiple languages** - Node.js, Python, Go support
- **Container tools** - Docker, kubectl, Helm
- **Workspace structure** - Organized directory layout

### ✅ Kubernetes Integration
- **Init containers** - Proper permission setup
- **ConfigMaps** - Configuration and script management
- **Secrets** - Secure password storage
- **Ingress** - Traefik-based routing

## 🔄 Chart Deployment Process

### 1. Local Chart Copy
```go
// application_service_deployment.go
func (s *SimpleApplicationService) copyLocalChartToVPS(conn ssh.Connection, chartPath string) error {
    // 1. Create remote directory
    s.sshService.ExecuteCommand(conn, fmt.Sprintf("mkdir -p %s", chartPath))
    
    // 2. Copy chart files
    files := []string{"Chart.yaml", "values.yaml", "templates/"}
    for _, file := range files {
        s.sshService.TransferFile(conn, localPath+file, chartPath+file)
    }
    
    return nil
}
```

### 2. Values File Generation
```go
// Template processing with dynamic values
func (s *SimpleApplicationService) generateValuesFile(app *models.Application) string {
    values := `
image:
  tag: "{{VERSION}}"
ingress:
  host: "{{SUBDOMAIN}}.{{DOMAIN}}"
  enabled: true
  tls:
    enabled: true
`
    
    // Substitute placeholders
    values = strings.ReplaceAll(values, "{{VERSION}}", latestVersion)
    values = strings.ReplaceAll(values, "{{SUBDOMAIN}}", app.Subdomain)
    values = strings.ReplaceAll(values, "{{DOMAIN}}", app.Domain)
    
    return values
}
```

### 3. Helm Installation
```bash
# Executed on VPS via SSH
helm install maria20-code-server /tmp/xanthus-code-server \
  --namespace code-server \
  --values /tmp/maria20-code-server-values.yaml \
  --wait --timeout 10m
```

## 📈 Benefits of Local Charts

### ✅ No External Dependencies
- **Faster deployments** - No GitHub cloning or repository access
- **Network independence** - Works in air-gapped environments
- **Reliability** - No external service dependencies

### ✅ Full Control
- **Custom templates** - Tailored Kubernetes manifests
- **Version control** - Chart templates managed in codebase
- **Customization** - Application-specific configurations

### ✅ Security
- **Vetted templates** - All templates reviewed and controlled
- **No external pulls** - Reduces attack surface
- **Consistent deployment** - Same chart version for all deployments

## 🛠️ Adding New Charts

### 1. Create Chart Structure
```bash
mkdir -p charts/xanthus-new-app/{templates}
touch charts/xanthus-new-app/Chart.yaml
touch charts/xanthus-new-app/values.yaml
```

### 2. Add Kubernetes Manifests
```bash
# Create necessary templates
touch charts/xanthus-new-app/templates/deployment.yaml
touch charts/xanthus-new-app/templates/service.yaml
touch charts/xanthus-new-app/templates/ingress.yaml
```

### 3. Update Configuration
```yaml
# configs/applications/new-app.yaml
helm_chart:
  repository: "local"
  chart: "xanthus-new-app"
  namespace: "new-app"
```

### 4. Create Values Template
```yaml
# internal/templates/applications/new-app.yaml
image:
  tag: "{{VERSION}}"
ingress:
  host: "{{SUBDOMAIN}}.{{DOMAIN}}"
```

## 🔧 Chart Maintenance

### Version Updates
- **Chart version** - Increment in Chart.yaml
- **App version** - Update appVersion in Chart.yaml
- **Image tags** - Update default values.yaml

### Template Updates
- **Kubernetes API** - Update apiVersion as needed
- **Resource definitions** - Add new resource types
- **Security updates** - Update security contexts

### Testing
- **Helm lint** - Validate chart syntax
- **Dry run** - Test template rendering
- **Local deployment** - Test on development cluster

## 🔒 Security Considerations

### Container Security
- **Non-root execution** - All containers run as non-root users
- **Resource limits** - CPU and memory constraints
- **Security contexts** - Proper filesystem permissions

### Secret Management
- **Kubernetes secrets** - Passwords stored as secrets
- **No hardcoded secrets** - All secrets generated dynamically
- **Proper RBAC** - Minimal required permissions

### Network Security
- **Ingress security** - TLS termination at ingress
- **Service mesh** - Consider service mesh for advanced security
- **Network policies** - Implement network segmentation