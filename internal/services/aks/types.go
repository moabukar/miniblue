package aks

import (
	"time"
)

// buildClusterResponse constructs the ARM JSON for a managedCluster from the
// PUT request body. Unset fields get sensible defaults so a minimum-viable
// `azurerm_kubernetes_cluster` block round-trips cleanly.
func buildClusterResponse(sub, rg, name string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerService/managedClusters/" + name

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	tags, _ := input["tags"].(map[string]interface{})

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	kubeVersion, _ := props["kubernetesVersion"].(string)
	if kubeVersion == "" {
		kubeVersion = "1.30.0"
	}

	dnsPrefix, _ := props["dnsPrefix"].(string)
	if dnsPrefix == "" {
		dnsPrefix = name
	}

	// Default agent pool – mirror the input if provided, else create a minimum.
	agentPools := normalizeAgentPools(props["agentPoolProfiles"], name)

	identity, _ := input["identity"].(map[string]interface{})
	if identity == nil {
		identity = map[string]interface{}{
			"type": "SystemAssigned",
			"principalId": "00000000-0000-0000-0000-000000000003",
			"tenantId":    "00000000-0000-0000-0000-000000000001",
		}
	}

	servicePrincipal, _ := props["servicePrincipalProfile"].(map[string]interface{})
	if servicePrincipal == nil {
		servicePrincipal = map[string]interface{}{"clientId": "msi"}
	}

	networkProfile, _ := props["networkProfile"].(map[string]interface{})
	if networkProfile == nil {
		networkProfile = map[string]interface{}{
			"networkPlugin":   "kubenet",
			"loadBalancerSku": "standard",
			"serviceCidr":     "10.0.0.0/16",
			"dnsServiceIP":    "10.0.0.10",
			"podCidr":         "10.244.0.0/16",
		}
	}

	resp := map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.ContainerService/managedClusters",
		"location": location,
		"identity": identity,
		"properties": map[string]interface{}{
			"provisioningState":       "Succeeded",
			"powerState":              map[string]interface{}{"code": "Running"},
			"kubernetesVersion":       kubeVersion,
			"currentKubernetesVersion": kubeVersion,
			"dnsPrefix":               dnsPrefix,
			"fqdn":                    dnsPrefix + ".hcp." + location + ".azmk8s.io",
			"agentPoolProfiles":       agentPools,
			"servicePrincipalProfile": servicePrincipal,
			"networkProfile":          networkProfile,
			"nodeResourceGroup":       "MC_" + rg + "_" + name + "_" + location,
			"enableRBAC":              true,
			"maxAgentPools":           10,
			"createdAt":               time.Now().UTC().Format(time.RFC3339),
		},
	}
	if tags != nil {
		resp["tags"] = tags
	}
	return resp
}

// normalizeAgentPools fills in defaults for missing fields so listings work
// uniformly whether or not the caller specified a pool.
func normalizeAgentPools(raw interface{}, clusterName string) []interface{} {
	pools, _ := raw.([]interface{})
	if len(pools) == 0 {
		return []interface{}{
			map[string]interface{}{
				"name":              "default",
				"count":             1,
				"vmSize":            "Standard_DS2_v2",
				"osType":            "Linux",
				"osDiskSizeGB":      30,
				"mode":              "System",
				"orchestratorVersion": "1.30.0",
				"provisioningState": "Succeeded",
				"powerState":        map[string]interface{}{"code": "Running"},
				"type":              "VirtualMachineScaleSets",
			},
		}
	}
	out := make([]interface{}, 0, len(pools))
	for _, p := range pools {
		pm, _ := p.(map[string]interface{})
		if pm == nil {
			continue
		}
		if _, ok := pm["count"]; !ok {
			pm["count"] = 1
		}
		if _, ok := pm["vmSize"]; !ok {
			pm["vmSize"] = "Standard_DS2_v2"
		}
		if _, ok := pm["osType"]; !ok {
			pm["osType"] = "Linux"
		}
		if _, ok := pm["mode"]; !ok {
			pm["mode"] = "System"
		}
		if _, ok := pm["provisioningState"]; !ok {
			pm["provisioningState"] = "Succeeded"
		}
		if _, ok := pm["powerState"]; !ok {
			pm["powerState"] = map[string]interface{}{"code": "Running"}
		}
		out = append(out, pm)
	}
	return out
}

// agentPoolsFromCluster returns each agentPoolProfile wrapped as a full
// Microsoft.ContainerService/managedClusters/agentPools sub-resource.
func agentPoolsFromCluster(cluster interface{}, sub, rg, clusterName string) []interface{} {
	cm, _ := cluster.(map[string]interface{})
	if cm == nil {
		return nil
	}
	props, _ := cm["properties"].(map[string]interface{})
	if props == nil {
		return nil
	}
	pools, _ := props["agentPoolProfiles"].([]interface{})
	out := make([]interface{}, 0, len(pools))
	for _, p := range pools {
		pm, _ := p.(map[string]interface{})
		if pm == nil {
			continue
		}
		poolName, _ := pm["name"].(string)
		out = append(out, map[string]interface{}{
			"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ContainerService/managedClusters/" + clusterName + "/agentPools/" + poolName,
			"name":       poolName,
			"type":       "Microsoft.ContainerService/managedClusters/agentPools",
			"properties": pm,
		})
	}
	return out
}

// stripInternalFields returns a copy of the cluster response with fields
// prefixed by "_miniblue_" removed (callers should never see internal handles).
func stripInternalFields(v interface{}) interface{} {
	cm, _ := v.(map[string]interface{})
	if cm == nil {
		return v
	}
	out := make(map[string]interface{}, len(cm))
	for k, vv := range cm {
		if len(k) >= 9 && k[:9] == "_miniblue" {
			continue
		}
		out[k] = vv
	}
	return out
}

func stripInternalFieldsList(items []interface{}) []interface{} {
	out := make([]interface{}, 0, len(items))
	for _, it := range items {
		out = append(out, stripInternalFields(it))
	}
	return out
}
