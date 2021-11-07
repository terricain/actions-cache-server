{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "acs.fullname" -}}
{{-   if .Values.fullnameOverride -}}
{{-     .Values.server.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{-   else -}}
{{-     $name := default .Chart.Name .Values.nameOverride -}}
{{-     if contains $name .Release.Name -}}
{{-       printf "%s-%s" .Release.Name .Values.name | trunc 63 | trimSuffix "-" -}}
{{-     else -}}
{{-       printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{-     end -}}
{{-   end -}}
{{- end -}}

{{- define "acs.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "acs.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "acs.common.metaLabels" -}}
chart: {{ template "acs.chart" . }}
heritage: {{ .Release.Service }}
{{- end -}}

{{- define "acs.common.matchLabels" -}}
app: {{ template "acs.name" . }}
release: {{ .Release.Name }}
{{- end -}}

{{- define "acs.matchLabels" -}}
component: {{ .Values.name | quote }}
{{ include "acs.common.matchLabels" . }}
{{- end -}}

{{- define "acs.labels" -}}
{{ include "acs.matchLabels" . }}
{{ include "acs.common.metaLabels" . }}
{{- end -}}

{{/*
Define the namespace template if set with forceNamespace or .Release.Namespace is set
*/}}
{{- define "acs.namespace" -}}
{{-   if .Values.forceNamespace -}}
{{      printf "namespace: %s" .Values.forceNamespace }}
{{-   else -}}
{{      printf "namespace: %s" .Release.Namespace }}
{{-   end -}}
{{- end -}}