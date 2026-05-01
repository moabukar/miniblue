# AKS example

Provisions a minimal `azurerm_kubernetes_cluster` against miniblue.

## Prerequisites

```bash
./bin/miniblue
export SSL_CERT_FILE=~/.miniblue/cert.pem
```

## Apply

```bash
terraform init
terraform apply -auto-approve
```

The cluster shows up in `azlocal aks list --resource-group aks-example-rg` and is destroyable via `terraform destroy`.

## Two backends

By default, AKS in miniblue is **stub-only**. The cluster exists in the ARM API so Terraform plans, applies, and destroys cleanly, but `kubectl` against the returned kubeconfig will not connect (no real Kubernetes is running).

To run a **real Kubernetes cluster** behind the AKS resource:

```bash
# Binary (recommended for local dev):
AKS_BACKEND=k3s ./bin/miniblue

# Or via the full Docker image (default `latest` is scratch and has no
# docker CLI; use the `full` target for real backends):
docker build --target=full -t miniblue:full .
docker run -v /var/run/docker.sock:/var/run/docker.sock \
  -p 4566:4566 -p 4567:4567 \
  -e AKS_BACKEND=k3s miniblue:full
```

miniblue starts a `rancher/k3s` container per cluster create (about 5 to 10 seconds, plus a one-off ~200 MB image pull on first run) and returns a working kubeconfig:

```bash
azlocal aks get-credentials --resource-group aks-example-rg --name miniblue-aks
kubectl get nodes
kubectl create deployment hello --image=nginx
```

Requires Docker (or OrbStack / Rancher Desktop) running on the host. There is a short gap of a few seconds between cluster create returning and the node showing as Ready in `kubectl get nodes`.

### Cleanup

`terraform destroy`, `azlocal aks delete`, and resource group cascade delete all remove the underlying k3s container. If miniblue is killed mid-flight, restart it with `AKS_BACKEND=k3s` and any orphaned `miniblue-aks-*` containers will be reaped automatically (the reaper preserves containers referenced by stored AKS clusters when `PERSISTENCE=1`).
