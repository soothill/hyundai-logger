# Hyundai Logger Helm Chart

A Helm chart for deploying Hyundai Logger to Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- InfluxDB 2.0+ (can be deployed separately or using this chart's dependency)

## Installing the Chart

```bash
# Add the repository (if published)
helm repo add hyundai-logger https://charts.example.com

# Install the chart
helm install my-logger hyundai-logger/hyundai-logger

# Or install from local directory
helm install my-logger ./deployments/helm/hyundai-logger
```

## Configuration

The following table lists the configurable parameters of the Hyundai Logger chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `hyundai-logger` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `config.username` | Hyundai Bluelink username | `""` |
| `config.password` | Hyundai Bluelink password | `""` |
| `config.pin` | Hyundai Bluelink PIN | `""` |
| `config.region` | Region (na, eu, kr, etc.) | `na` |
| `config.pollInterval` | Polling interval in minutes | `5` |
| `influxdb.enabled` | Deploy InfluxDB as dependency | `true` |
| `influxdb.url` | InfluxDB URL (if external) | `""` |
| `influxdb.token` | InfluxDB auth token | `""` |
| `influxdb.org` | InfluxDB organization | `hyundai` |
| `influxdb.bucket` | InfluxDB bucket | `vehicle-data` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `100m` |
| `resources.requests.memory` | Memory request | `128Mi` |

## Using with External InfluxDB

```bash
helm install my-logger ./hyundai-logger \
  --set influxdb.enabled=false \
  --set influxdb.url=https://influxdb.example.com \
  --set influxdb.token=your-token \
  --set config.username=your-email@example.com \
  --set config.password=your-password
```

## Using Secrets

For production, use Kubernetes secrets:

```bash
kubectl create secret generic hyundai-credentials \
  --from-literal=username=your-email@example.com \
  --from-literal=password=your-password \
  --from-literal=pin=1234

kubectl create secret generic influxdb-credentials \
  --from-literal=token=your-influxdb-token

helm install my-logger ./hyundai-logger \
  --set config.existingSecret=hyundai-credentials \
  --set influxdb.existingSecret=influxdb-credentials
```

## Uninstalling the Chart

```bash
helm uninstall my-logger
```
