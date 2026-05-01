# Kubernetes Service (AKS)

miniblue emulates Azure Kubernetes Service via the `Microsoft.ContainerService` ARM endpoints. Two backends are available:

- **stub** (default): cluster create/get/list/delete updates miniblue's in-memory store; `listClusterAdminCredential` returns a syntactically valid kubeconfig pointing at a sentinel host. Sufficient for `terraform plan/apply`, `az aks list`, and IaC iteration.
- **real (k3s in Docker)**: when `AKS_BACKEND=k3s` is set and Docker is reachable, every cluster create launches a `rancher/k3s` container on a dynamic localhost port and `listClusterAdminCredential` returns the real kubeconfig so `kubectl` actually deploys workloads.

Both backends implement the same ARM contract; the choice is invisible to Terraform / Bicep / `az`.

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `PUT` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{name}` | Create or update cluster |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{name}` | Get cluster |
| `DELETE` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters/{name}` | Delete cluster (also tears down the k3s container in real mode) |
| `GET` | `/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/managedClusters` | List clusters in resource group |
| `GET` | `/subscriptions/{sub}/providers/Microsoft.ContainerService/managedClusters` | List clusters in subscription (used by `az aks list`) |
| `GET` | `.../managedClusters/{name}/agentPools` | List node pools |
| `GET` | `.../managedClusters/{name}/agentPools/{poolName}` | Get a specific node pool |
| `POST` | `.../managedClusters/{name}/listClusterAdminCredential` | Returns base64-encoded kubeconfig |
| `POST` | `.../managedClusters/{name}/listClusterUserCredential` | Same shape; same kubeconfig |

Cluster names are validated against Azure's rule (`^[a-zA-Z0-9][-_a-zA-Z0-9]{0,62}$`) and rejected with `400 BadRequest` if invalid.

## Stub mode

Default. No Docker required. Returns realistic ARM responses so Terraform plans, applies, and destroys cleanly. The kubeconfig points at `https://miniblue-aks.invalid:443`; `kubectl` against it will not connect (intended).

```bash
curl -X PUT 'http://localhost:4566/subscriptions/sub1/resourceGroups/myRG/providers/Microsoft.ContainerService/managedClusters/my-cluster' \
  -H 'Content-Type: application/json' \
  -d '{
    "location": "eastus",
    "identity": {"type": "SystemAssigned"},
    "properties": {
      "dnsPrefix": "myprefix",
      "agentPoolProfiles": [
        {"name": "default", "count": 1, "vmSize": "Standard_DS2_v2", "mode": "System"}
      ]
    }
  }'
```

## Real backend

Set `AKS_BACKEND=k3s` (or `MINIBLUE_AKS_REAL=1`) and run miniblue with Docker reachable.

```bash
# Binary install (recommended for local dev)
AKS_BACKEND=k3s ./bin/miniblue

# Or via the `full` Docker image variant (the default scratch image has no
# docker CLI to shell out to; build the `full` target for real backends):
docker build --target=full -t miniblue:full .
docker run -v /var/run/docker.sock:/var/run/docker.sock \
  -p 4566:4566 -p 4567:4567 \
  -e AKS_BACKEND=k3s miniblue:full
```

On startup miniblue warm-pulls the `rancher/k3s` image (~200MB, one-off) so the first cluster create is not blocked. Subsequent creates take ~5 to 10 seconds.

```bash
azlocal aks create --resource-group myRG --name my-cluster --node-count 2
azlocal aks get-credentials --resource-group myRG --name my-cluster
kubectl get nodes
kubectl create deployment hello --image=nginx
```

`azlocal aks get-credentials` defaults to **merging** into `~/.kube/config` (matching real `az aks get-credentials`) so existing contexts (docker-desktop, GKE, real AKS) are preserved. Use `--overwrite-existing` to replace the file or `--file -` to dump to stdout.

## Lifecycle and teardown

In real mode miniblue tracks every k3s container it spawns and cleans them up via several paths:

- **Explicit DELETE** of the cluster, or `azlocal aks delete`, or `terraform destroy`: the k3s container is removed immediately.
- **Resource group cascade delete**: removing the parent resource group also tears down every k3s container backing a cluster in that RG.
- **Graceful miniblue shutdown** (`SIGTERM` / `SIGINT` / `docker stop`): all running k3s containers for known clusters are torn down before miniblue exits, so they do not leak past miniblue's own lifetime.
- **Forceful kill** (`SIGKILL`, OOM, host crash): containers stay running. The next miniblue start with `AKS_BACKEND=k3s` reaps any orphaned `miniblue-aks-*` containers that no longer have a matching cluster in the store.
- **`PERSISTENCE=1`**: surviving k3s containers whose clusters are still in the persisted store are preserved across restarts, and the same kubeconfig keeps working because the host port is part of the persisted backend handle.

If you ever want to manually purge orphans:

```bash
docker ps --filter name=miniblue-aks- -q | xargs -r docker rm -f
```

## Cross-cluster naming

Two clusters with the same short name in different resource groups (e.g. `dev/k1` and `prod/k1`) are kept distinct via a hash suffix on the docker container name (`miniblue-aks-k1-<8charHash>`), so they never clash on the host.

## Real backend constraints

- Requires Docker (or OrbStack / Rancher Desktop) reachable from the miniblue process.
- The default `:latest` Docker image is `FROM scratch` and has no docker CLI; use the `:full` build target (or run miniblue as a host binary) for real backend.
- k3s is API-compatible with vanilla Kubernetes, with a few small differences (Flannel CNI by default, SQLite instead of etcd in single-node mode, Traefik disabled by miniblue for faster boot).
