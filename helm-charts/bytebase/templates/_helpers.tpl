{{/*
Allow the release namespace to be overridden for multi-namespace deployments in combined charts
*/}}
{{- define "bytebase.namespace" -}}
  {{- if .Values.namespaceOverride -}}
    {{- .Values.namespaceOverride -}}
  {{- else -}}
    {{- .Release.Namespace -}}
  {{- end -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "bytebase.labels" -}}
{{ include "bytebase.selectorLabels" . }}
app.kubernetes.io/version: {{ .Values.bytebase.version}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "bytebase.selectorLabels" -}}
app: bytebase
{{- end }}

{{/*
Create the Bytebase image reference
*/}}
{{- define "bytebase.image" -}}
{{- $image := printf "bytebase/bytebase:%s" .Values.bytebase.version -}}
{{- if .Values.bytebase.registryMirrorHost -}}
{{- $image = printf "%s/bytebase/bytebase:%s" (trimSuffix "/" .Values.bytebase.registryMirrorHost) .Values.bytebase.version -}}
{{- end -}}
{{- $digest := .Values.bytebase.digest -}}
{{- with .Values.global -}}
  {{- with .azure -}}
    {{- with .images -}}
      {{- with .bytebase -}}
        {{- $image = printf "%s/%s:%s" .registry .image .tag -}}
        {{- $digest = .digest | default "" -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if $digest -}}
{{- printf "%s@%s" $image $digest -}}
{{- else -}}
{{- $image -}}
{{- end -}}
{{- end -}}

{{/*
Create the BusyBox image reference
*/}}
{{- define "bytebase.busyboxImage" -}}
{{- $image := "busybox" -}}
{{- if .Values.bytebase.registryMirrorHost -}}
{{- $image = printf "%s/busybox" (trimSuffix "/" .Values.bytebase.registryMirrorHost) -}}
{{- end -}}
{{- if .Values.bytebase.busyboxDigest -}}
{{- printf "%s@%s" $image .Values.bytebase.busyboxDigest -}}
{{- else -}}
{{- $image -}}
{{- end -}}
{{- end -}}

{{/*
Create the name of the general service account
*/}}
{{- define "bytebase.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default ("bytebase") .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}
