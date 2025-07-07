{{/*
Expand the name of the chart.
*/}}
{{- define "xanthus-platform.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "xanthus-platform.fullname" -}}
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
{{- define "xanthus-platform.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "xanthus-platform.labels" -}}
helm.sh/chart: {{ include "xanthus-platform.chart" . }}
{{ include "xanthus-platform.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "xanthus-platform.selectorLabels" -}}
app.kubernetes.io/name: {{ include "xanthus-platform.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "xanthus-platform.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "xanthus-platform.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the persistent volume claim
*/}}
{{- define "xanthus-platform.pvcName" -}}
{{- printf "%s-data" (include "xanthus-platform.fullname" .) }}
{{- end }}

{{/*
Create the name of the config map
*/}}
{{- define "xanthus-platform.configMapName" -}}
{{- printf "%s-config" (include "xanthus-platform.fullname" .) }}
{{- end }}

{{/*
Create the name of the secret
*/}}
{{- define "xanthus-platform.secretName" -}}
{{- printf "%s-secret" (include "xanthus-platform.fullname" .) }}
{{- end }}