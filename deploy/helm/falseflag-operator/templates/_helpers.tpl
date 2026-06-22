{{/*
Expand the name of the chart.
*/}}
{{- define "falseflag.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name.
*/}}
{{- define "falseflag.fullname" -}}
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
Fully qualified component name.
*/}}
{{- define "falseflag.componentName" -}}
{{- $root := index . "root" -}}
{{- $component := index . "component" -}}
{{- printf "%s-%s" (include "falseflag.fullname" $root) $component | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Chart label.
*/}}
{{- define "falseflag.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Standard labels.
*/}}
{{- define "falseflag.labels" -}}
helm.sh/chart: {{ include "falseflag.chart" . }}
app.kubernetes.io/name: {{ include "falseflag.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: falseflag
{{- end }}

{{/*
Component labels.
*/}}
{{- define "falseflag.componentLabels" -}}
{{- $root := index . "root" -}}
{{- $component := index . "component" -}}
{{ include "falseflag.labels" $root }}
app.kubernetes.io/component: {{ $component }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "falseflag.selectorLabels" -}}
{{- $root := index . "root" -}}
{{- $component := index . "component" -}}
app.kubernetes.io/name: {{ include "falseflag.name" $root }}
app.kubernetes.io/instance: {{ $root.Release.Name }}
app.kubernetes.io/component: {{ $component }}
{{- end }}

{{/*
Resolve a component image tag.
*/}}
{{- define "falseflag.imageTag" -}}
{{- $root := index . "root" -}}
{{- $image := index . "image" -}}
{{- default $root.Values.global.imageTag $image.tag }}
{{- end }}

{{/*
Operator service account name.
*/}}
{{- define "falseflag.operatorServiceAccountName" -}}
{{- if .Values.operator.serviceAccount.create }}
{{- default (include "falseflag.componentName" (dict "root" . "component" "operator")) .Values.operator.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.operator.serviceAccount.name }}
{{- end }}
{{- end }}
