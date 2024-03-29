# ca-injector

![Version: 0.0.1](https://img.shields.io/badge/Version-0.0.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v1.0.3](https://img.shields.io/badge/AppVersion-v1.0.3-informational?style=flat-square)

A kubernetes MutatingAdmissionWebhook to inject certificate bundles into pods based on annotations

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"andrewstuart/ca-injector"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| patch.enabled | bool | `true` |  |
| patch.image.pullPolicy | string | `"IfNotPresent"` |  |
| patch.image.repository | string | `"registry.k8s.io/ingress-nginx/kube-webhook-certgen"` |  |
| patch.image.tag | string | `"v1.3.0"` |  |
| patch.nodeSelector | object | `{}` |  |
| patch.podAnnotations | object | `{}` |  |
| patch.priorityClassName | string | `""` |  |
| patch.resources | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podSecurityContext | object | `{}` |  |
| replicaCount | int | `1` |  |
| resources | object | `{}` |  |
| securityContext | object | `{}` |  |
| service.port | int | `80` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.11.0](https://github.com/norwoodj/helm-docs/releases/v1.11.0)
