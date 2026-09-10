# az-storage-mock

Cobra CLI using Azure `armstorage/v4` and `armstorage.NewClientFactory`.

Examples:

```bash
az-storage-mock storage container create --resource-group myRG --name documents --account-name myaccount
az-storage-mock storage container list --resource-group myRG --account-name myaccount
az-storage-mock storage container update --resource-group myRG --name documents --account-name myaccount
az-storage-mock storage container show --resource-group myRG --name documents --account-name myaccount
az-storage-mock storage container exists --resource-group myRG --name documents --account-name myaccount
az-storage-mock storage container delete --resource-group myRG --name documents --account-name myaccount
```

`--resource-group` is a persistent flag inherited by all container commands.
