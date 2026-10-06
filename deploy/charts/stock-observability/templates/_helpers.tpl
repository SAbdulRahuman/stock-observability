{{/* Usage: include "so.fullname" (list $ "component") */}}
{{- define "so.fullname" -}}
{{- printf "%s%s" (index . 0).Values.namePrefix (index . 1) | trunc 63 | trimSuffix "-" -}}
{{- end }}

{{- define "so.selectorLabels" -}}
app.kubernetes.io/name: {{ index . 1 }}
app.kubernetes.io/instance: {{ (index . 0).Release.Name }}
{{- end }}

{{- define "so.labels" -}}
{{- $root := index . 0 -}}
helm.sh/chart: {{ printf "%s-%s" $root.Chart.Name $root.Chart.Version }}
app.kubernetes.io/managed-by: {{ $root.Release.Service }}
app.kubernetes.io/part-of: stock-observability
app.kubernetes.io/version: {{ $root.Chart.AppVersion | quote }}
{{ include "so.selectorLabels" . }}
{{- end }}

{{- define "so.externalURL" -}}
{{- trimSuffix "/" .Values.externalURL -}}
{{- end }}

{{- define "so.image" -}}
{{- printf "%s:%s" .repository (toString .tag) -}}
{{- end }}

{{- define "so.podSecurityContext" -}}
runAsNonRoot: true
fsGroupChangePolicy: OnRootMismatch
seccompProfile:
  type: RuntimeDefault
{{- if not .Values.openshift }}
runAsUser: {{ .Values.podSecurityContext.runAsUser }}
fsGroup: {{ .Values.podSecurityContext.fsGroup }}
{{- end }}
{{- end }}

{{- define "so.containerSecurityContext" -}}
allowPrivilegeEscalation: false
capabilities:
  drop: [ALL]
{{- end }}

{{- define "so.podSpecCommon" -}}
automountServiceAccountToken: false
{{- with .Values.imagePullSecrets }}
imagePullSecrets:
  {{- toYaml . | nindent 2 }}
{{- end }}
securityContext:
  {{- include "so.podSecurityContext" . | nindent 2 }}
{{- end }}

{{/* Usage: include "so.pvc" (list $ "component" .Values.x.persistence) */}}
{{- define "so.pvc" -}}
{{- $root := index . 0 -}}
{{- $c := index . 1 -}}
{{- $p := index . 2 -}}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "so.fullname" (list $root $c) }}-data
  labels:
    {{- include "so.labels" (list $root $c) | nindent 4 }}
  annotations:
    helm.sh/resource-policy: keep
spec:
  accessModes: [ReadWriteOnce]
  {{- with $root.Values.storageClassName }}
  storageClassName: {{ . }}
  {{- end }}
  resources:
    requests:
      storage: {{ $p.size }}
{{- end }}
