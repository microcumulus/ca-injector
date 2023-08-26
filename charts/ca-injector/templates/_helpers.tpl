{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "ca-injector.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
The components in this chart create additional resources that expand the longest created name strings.
The longest name that gets created adds extra 19 characters, so truncation should be 63-19=44.
*/}}
{{- define "ca-injector.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 44 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 44 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 44 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* Create chart name and version as used by the chart label. */}}
{{- define "ca-injector.chartref" -}}
{{- replace "+" "_" .Chart.Version | printf "%s-%s" .Chart.Name -}}
{{- end }}

{{/* Generate basic labels */}}
{{- define "ca-injector.labels" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: "{{ replace "+" "_" .Chart.AppVersion }}"
app.kubernetes.io/part-of: {{ include "ca-injector.name" . }}
helm.sh/chart: {{ include "ca-injector.chartref" . }}
{{- if .Values.commonLabels}}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{/* Create the name of ca-injector service account to use */}}
{{- define "ca-injector.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "ca-injector.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Allow the release namespace to be overridden for multi-namespace deployments in combined charts
*/}}
{{- define "ca-injector.namespace" -}}
  {{- if .Values.namespaceOverride -}}
    {{- .Values.namespaceOverride -}}
  {{- else -}}
    {{- .Release.Namespace -}}
  {{- end -}}
{{- end -}}

{{/*
To help compatibility with other charts which use global.imagePullSecrets.
Allow either an array of {name: pullSecret} maps (k8s-style), or an array of strings (more common helm-style).
imagePullSecrets:
- name: pullSecret1
- name: pullSecret2

or

imagePullSecrets:
- pullSecret1
- pullSecret2
*/}}
{{- define "ca-injector.imagePullSecrets" -}}
{{- range .Values.imagePullSecrets }}
  {{- if eq (typeOf .) "map[string]interface {}" }}
- {{ toYaml . | trim }}
  {{- else }}
- name: {{ . }}
  {{- end }}
{{- end }}
{{- end -}}
