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
AKS_BACKEND=k3s ./bin/miniblue
```

> **Run miniblue as a binary, not via the published Docker image.** The image is `FROM scratch` and has no `docker` CLI to shell out to. Use `brew install miniblue` or download from the releases page. Same constraint applies to ACI's real backend.

When set, miniblue starts a `rancher/k3s` container per cluster create (about 5 to 10 seconds) and returns a working kubeconfig:

```bash
azlocal aks get-credentials --resource-group aks-example-rg --name miniblue-aks
kubectl get nodes
kubectl create deployment hello --image=nginx
```

Requires Docker (or OrbStack / Rancher Desktop) running on the host. The first cluster create downloads `rancher/k3s` (about 200 MB), subsequent creates reuse the cached image. There is a short gap of a few seconds between cluster create returning and the node showing as Ready in `kubectl get nodes`.

### Cleanup note for real backend

`terraform destroy` and `azlocal aks delete` both remove the underlying k3s container. Cascade-deleting the resource group via `azlocal group delete` only clears miniblue's ARM state; if you ran in real-backend mode and skip the explicit delete, prune orphaned containers with:

```bash
docker ps --filter name=miniblue-aks- -q | xargs -r docker rm -f
```
