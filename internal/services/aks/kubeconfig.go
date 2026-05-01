package aks

import "fmt"

// stubKubeconfig returns a syntactically-valid kubeconfig pointing at a
// sentinel host. `kubectl` will not connect (intended) but Terraform / Bicep
// / `az aks get-credentials` parse it cleanly.
func stubKubeconfig(clusterName string) []byte {
	const stubCert = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JTUlOSUJMVUVTVFVCQ0VSVElGSUNBVEUKLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQ=="
	const stubToken = "miniblue-stub-token-not-a-real-secret"
	return []byte(fmt.Sprintf(`apiVersion: v1
kind: Config
current-context: %[1]s
clusters:
- name: %[1]s
  cluster:
    server: https://miniblue-aks.invalid:443
    certificate-authority-data: %[2]s
contexts:
- name: %[1]s
  context:
    cluster: %[1]s
    user: clusterUser_miniblue_%[1]s
users:
- name: clusterUser_miniblue_%[1]s
  user:
    token: %[3]s
`, clusterName, stubCert, stubToken))
}
