// Minimal Bicep example deployable against miniblue.
//
//   az deployment group create \
//     --resource-group bicep-rg \
//     --template-file main.bicep \
//     --parameters saName=mystorage clusterName=mycluster
//
// Bicep transpiles to ARM JSON which miniblue's
// Microsoft.Resources/deployments endpoint applies by dispatching each
// resource to its existing handler.

param location string = 'eastus'
param saName string
param clusterName string

resource sa 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: saName
  location: location
  kind: 'StorageV2'
  sku: {
    name: 'Standard_LRS'
  }
  properties: {}
}

resource aks 'Microsoft.ContainerService/managedClusters@2023-09-01' = {
  name: clusterName
  location: location
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    dnsPrefix: 'bicep'
  }
}
