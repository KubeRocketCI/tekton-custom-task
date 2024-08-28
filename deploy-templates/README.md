# tekton-custom-task

![Version: 0.0.1](https://img.shields.io/badge/Version-0.0.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.1](https://img.shields.io/badge/AppVersion-0.0.1-informational?style=flat-square)

A Helm chart for KubeRocketCI tekton-custom-task

**Homepage:** <https://github.com/KubeRocketCI/tekton-custom-task>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/epam-delivery-platform> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://git.epam.com/KubeRocketCI/tekton-custom-task>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| annotations | object | `{}` |  |
| image.repository | string | `"KubeRocketCI/tekton-custom-task"` | tekton-custom-task Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/KubeRocketCI/tekton-custom-task) |
| image.tag | string | `nil` | tekton-custom-task Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/KubeRocketCI/tekton-custom-task/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| name | string | `"tekton-custom-task"` | component name |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |
