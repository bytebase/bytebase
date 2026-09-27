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
app.kubernetes.io/version: {{ .Values.bytebase.version | default "" | toString | splitList "@" | first | trunc 63 | quote }}
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
{{- with .Values.bytebase.digest -}}
{{- fail "bytebase.digest is no longer supported; append the digest to bytebase.version" -}}
{{- end -}}
{{- with .Values.bytebase.busyboxDigest -}}
{{- fail "bytebase.busyboxDigest is no longer supported; append the digest to bytebase.busyboxVersion" -}}
{{- end -}}
{{- $version := .Values.bytebase.version | default "" | toString -}}
{{- $image := printf "bytebase/bytebase:%s" $version -}}
{{- if .Values.bytebase.registryMirrorHost -}}
{{- $image = printf "%s/bytebase/bytebase:%s" (trimSuffix "/" .Values.bytebase.registryMirrorHost) $version -}}
{{- end -}}
{{- with .Values.global -}}
  {{- with .azure -}}
    {{- with .images -}}
      {{- with .bytebase -}}
        {{- with .digest -}}
          {{- fail "global.azure.images.bytebase.digest is no longer supported; append the digest to global.azure.images.bytebase.tag" -}}
        {{- end -}}
        {{- $image = printf "%s/%s:%s" .registry .image .tag -}}
      {{- end -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- $image -}}
{{- end -}}

{{/*
Create the BusyBox image reference
*/}}
{{- define "bytebase.busyboxImage" -}}
{{- $image := "busybox" -}}
{{- if .Values.bytebase.registryMirrorHost -}}
{{- $image = printf "%s/busybox" (trimSuffix "/" .Values.bytebase.registryMirrorHost) -}}
{{- end -}}
{{- with .Values.bytebase.busyboxVersion | default "" | toString -}}
{{- $image = printf "%s:%s" $image . -}}
{{- end -}}
{{- $image -}}
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
