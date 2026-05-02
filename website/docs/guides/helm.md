# Helm / Kubernetes deployment

Run miniblue inside a Kubernetes cluster (kind, minikube, EKS, AKS, GKE, OrbStack k8s) so other pods in the cluster can target it as a fake Azure endpoint.

## Quick install

```bash
git clone https://github.com/moabukar/miniblue.git
helm install miniblue ./miniblue/charts/miniblue
```

The chart is in-tree under `charts/miniblue`. There is no published chart repo yet.

Verify:

```bash
kubectl rollout status deployment/miniblue
kubectl port-forward svc/miniblue 4566:4566
azlocal health
```

## Common configurations

### Persist state across pod restarts

```yaml
# values.yaml
persistence:
  enabled: true
  storageClass: ""   # use your cluster's default; set explicitly for managed storage classes
  size: 1Gi
```

```bash
helm upgrade --install miniblue ./charts/miniblue -f values.yaml
```

A `PersistentVolumeClaim` is mounted at `/home/miniblue` and miniblue saves its in-memory store to a JSON file there every 30 seconds. Survives pod restarts and rescheduling.

### Use PostgreSQL as the backend

For multi-replica or shared-state setups, point miniblue at a real PostgreSQL.

Inline:

```yaml
databaseUrl: "postgres://user:pass@postgres:5432/miniblue?sslmode=disable"
```

From a Kubernetes secret (recommended):

```bash
kubectl create secret generic miniblue-pg \
  --from-literal=url="postgres://user:pass@postgres:5432/miniblue?sslmode=disable"
```

```yaml
databaseUrlSecret:
  name: miniblue-pg
  key: url
```

### Expose outside the cluster

Default service is `ClusterIP`. Switch to `NodePort` or `LoadBalancer`:

```yaml
service:
  type: LoadBalancer
  httpPort: 4566
  httpsPort: 4567
```

For ingress, the chart does not bundle an Ingress resource yet; create one in your own manifest pointing at `svc/miniblue`.

### Multiple environments via release name

```bash
helm install miniblue-dev ./charts/miniblue
helm install miniblue-staging ./charts/miniblue -f staging.yaml
```

Each release gets its own deployment, service, and PVC keyed by release name.

## Pointing apps at miniblue

Apps inside the same cluster talk to it via the cluster service DNS:

```yaml
env:
  - name: AZURE_RESOURCE_MANAGER
    value: "http://miniblue.default.svc.cluster.local:4566"
```

For the Azure SDKs, the metadata host is the HTTPS endpoint:

```yaml
env:
  - name: ARM_ENDPOINT
    value: "https://miniblue.default.svc.cluster.local:4567"
```

The chart's container ships with miniblue's self-signed cert; trust it inside your app's container or set `AZURE_INSECURE_SKIP_VERIFY=true` for non-prod use.

## Limitations

The chart is intended for **in-cluster development workloads**, not production-like Azure. Real backends that shell out to Docker on the host (ACI, AKS, in-pod) **do not work** in a normal Kubernetes deployment because the miniblue process has no Docker daemon to reach. Specifically:

| Real backend | Works in Helm? |
|---|---|
| `POSTGRES_URL` (real Postgres for `azurerm_postgresql_flexible_server`) | yes (point at a real Postgres reachable from the pod) |
| `REDIS_URL` (real Redis for `azurerm_redis_cache`) | yes (same idea) |
| Real Docker containers for ACI | no (use the binary install on the host) |
| `AKS_BACKEND=k3s` (real k3s clusters per AKS resource) | no (would need privileged DinD; use the binary install for AKS real-mode) |

Stub mode is the default for ACI and AKS, so `terraform apply` against the chart-deployed miniblue still works for these resources; only `kubectl apply` against the returned AKS kubeconfig does not.

## Values reference

| Key | Default | Description |
|---|---|---|
| `replicaCount` | `1` | Use `1` unless you also set `databaseUrl` / `databaseUrlSecret` (the file backend is single-pod) |
| `image.repository` | `moabukar/miniblue` | Image repo |
| `image.tag` | `latest` | Pin to a version (`0.6.0`) for reproducibility |
| `service.type` | `ClusterIP` | `ClusterIP`, `NodePort`, or `LoadBalancer` |
| `service.httpPort` | `4566` | HTTP port |
| `service.httpsPort` | `4567` | HTTPS port |
| `persistence.enabled` | `false` | Mount a PVC at `/home/miniblue` and set `PERSISTENCE=1` |
| `persistence.storageClass` | `""` | Storage class for the PVC |
| `persistence.size` | `1Gi` | PVC size |
| `databaseUrl` | `""` | Inline Postgres connection string |
| `databaseUrlSecret.name` | `""` | Secret name for Postgres URL |
| `databaseUrlSecret.key` | `""` | Secret key holding the URL |
| `env.PORT` | `"4566"` | HTTP port inside the pod |
| `env.TLS_PORT` | `"4567"` | HTTPS port inside the pod |
| `env.LOG_LEVEL` | `"info"` | `debug`, `info`, `warn`, `error` |
| `securityContext` | non-root, read-only fs | Container security context |
| `resources` | 200m CPU / 128Mi mem | Adjust for your workload |

## Uninstall

```bash
helm uninstall miniblue
```

The PVC is preserved by default (Kubernetes' `Reclaim` policy). Delete it explicitly if you want a clean slate:

```bash
kubectl delete pvc miniblue-data
```
