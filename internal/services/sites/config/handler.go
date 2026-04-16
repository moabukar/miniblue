package config

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/storageauth"
	"github.com/moabukar/miniblue/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(store *store.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/config", func(r chi.Router) {
		r.Get("/web", h.GetConfig)
		r.Put("/web", h.UpdateConfig)
		r.Get("/appSettings", h.GetAppSettings)
		r.Put("/appSettings", h.UpdateAppSettings)
		r.Get("/appSettings/list", h.ListAppSettings)
		r.Post("/appSettings/list", h.ListAppSettings)
		r.Get("/azurestorageaccounts", h.GetAzureStorageAccounts)
		r.Put("/azurestorageaccounts", h.UpdateAzureStorageAccounts)
		r.Post("/azurestorageaccounts/list", h.ListAzureStorageAccounts)
		r.Get("/authsettings", h.GetAuthSettings)
		r.Put("/authsettings", h.UpdateAuthSettings)
		r.Get("/authsettings/list", h.ListAuthSettings)
		r.Put("/authsettings/list", h.UpdateAuthSettings)
		r.Post("/authsettings/list", h.ListAuthSettings)
		r.Get("/authsettingsV2", h.GetAuthSettingsV2)
		r.Put("/authsettingsV2", h.UpdateAuthSettingsV2)
		r.Get("/authsettingsV2/list", h.ListAuthSettingsV2)
		r.Post("/authsettingsV2/list", h.ListAuthSettingsV2)
		r.Get("/backup", h.GetBackupSettings)
		r.Put("/backup", h.UpdateBackupSettings)
		r.Get("/logs", h.GetLogsSettings)
		r.Put("/logs", h.UpdateLogsSettings)
		r.Get("/metadata", h.GetMetadata)
		r.Put("/metadata", h.UpdateMetadata)
		r.Get("/connectionstrings", h.GetConnectionStrings)
		r.Put("/connectionstrings", h.UpdateConnectionStrings)
		r.Get("/connectionStrings/list", h.ListConnectionStrings)
		r.Post("/connectionStrings/list", h.ListConnectionStrings)
		r.Get("/pushsettings", h.GetPushSettings)
		r.Put("/pushsettings", h.UpdatePushSettings)
		r.Get("/slotConfigNames", h.GetSlotConfigNames)
		r.Put("/slotConfigNames", h.UpdateSlotConfigNames)
		r.Get("/publishingcredentials/list", h.ListPublishingCredentials)
		r.Post("/publishingcredentials/list", h.ListPublishingCredentials)
	})
}

func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/config", name)
		return
	}

	site, ok := v.(map[string]interface{})
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/config", name)
		return
	}

	props, _ := site["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	siteConfig, _ := props["siteConfig"].(map[string]interface{})
	if siteConfig == nil {
		siteConfig = map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/web",
		"name":       name + "/config/web",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": siteConfig,
	})
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/config", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	site, ok := v.(map[string]interface{})
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/config", name)
		return
	}

	props, _ := site["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	var siteConfig map[string]interface{}
	if input["properties"] != nil {
		siteConfig, _ = input["properties"].(map[string]interface{})
	}
	if siteConfig == nil {
		siteConfig = map[string]interface{}{}
	}

	props["siteConfig"] = siteConfig
	site["properties"] = props
	h.store.Set(h.siteKey(sub, rg, name), site)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/web",
		"name":       name + "/config/web",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": siteConfig,
	})
}

func (h *Handler) GetAppSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/config", name)
		return
	}

	site, ok := v.(map[string]interface{})
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/config", name)
		return
	}

	props, _ := site["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	appSettings, _ := props["appSettings"].(map[string]interface{})
	if appSettings == nil {
		appSettings = map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/appsettings",
		"name":       name + "/appsettings",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": appSettings,
	})
}

func (h *Handler) ListAppSettings(w http.ResponseWriter, r *http.Request) {
	h.GetAppSettings(w, r)
}

func (h *Handler) UpdateAppSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/config", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	site, ok := v.(map[string]interface{})
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/config", name)
		return
	}

	props, _ := site["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	var appSettings map[string]interface{}
	if input["properties"] != nil {
		appSettings, _ = input["properties"].(map[string]interface{})
	}
	if appSettings == nil {
		appSettings = map[string]interface{}{}
	}

	props["appSettings"] = appSettings
	site["properties"] = props
	h.store.Set(h.siteKey(sub, rg, name), site)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/appsettings",
		"name":       name + "/appsettings",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": appSettings,
	})
}

func (h *Handler) GetAzureStorageAccounts(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	storageAccounts := map[string]interface{}{}
	items := h.store.ListByPrefix(h.azureStorageAccountsKey(sub, rg, name))
	for _, item := range items {
		if sa, ok := item.(map[string]interface{}); ok {
			accountName, _ := sa["accountName"].(string)
			if accountName != "" {
				storageAccounts[accountName] = sa
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/azurestorageaccounts",
		"name":       name + "/azurestorageaccounts",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": map[string]interface{}{"azurestorageaccounts": storageAccounts},
	})
	_ = v
}

func (h *Handler) ListAzureStorageAccounts(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	storageAccounts := map[string]interface{}{}
	items := h.store.ListByPrefix(h.azureStorageAccountsKey(sub, rg, name))
	for _, item := range items {
		if sa, ok := item.(map[string]interface{}); ok {
			accountName, _ := sa["accountName"].(string)
			if accountName != "" {
				storageAccounts[accountName] = sa
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/azurestorageaccounts",
		"name":       name + "/azurestorageaccounts",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": map[string]interface{}{"azurestorageaccounts": storageAccounts},
	})
	_ = v
}

func (h *Handler) UpdateAzureStorageAccounts(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	inputProps, _ := input["properties"].(map[string]interface{})
	inputStorageAccounts, _ := inputProps["azurestorageaccounts"].(map[string]interface{})

	storageAccounts := map[string]interface{}{}
	for accountName, config := range inputStorageAccounts {
		accountConfig, ok := config.(map[string]interface{})
		if !ok {
			continue
		}

		accountKey := storageauth.DeterministicAccountKey(sub, rg, accountName, "1")
		connectionString := fmt.Sprintf(
			"DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net",
			accountName, accountKey,
		)

		storageEntry := map[string]interface{}{
			"value":                connectionString,
			"type":                 accountConfig["type"],
			"accountName":          accountName,
			"accessKey":            accountKey,
			"blobStorageEndpoint":  fmt.Sprintf("https://%s.blob.core.windows.net/", accountName),
			"fileStorageEndpoint":  fmt.Sprintf("https://%s.file.core.windows.net/", accountName),
			"queueStorageEndpoint": fmt.Sprintf("https://%s.queue.core.windows.net/", accountName),
			"tableStorageEndpoint": fmt.Sprintf("https://%s.table.core.windows.net/", accountName),
		}

		if mountPath, ok := accountConfig["mountPath"].(string); ok {
			storageEntry["mountPath"] = mountPath
		}

		storageAccounts[accountName] = storageEntry

		h.store.Set(h.azureStorageAccountsKey(sub, rg, name)+":"+accountName, storageEntry)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/azurestorageaccounts",
		"name":       name + "/azurestorageaccounts",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": map[string]interface{}{"azurestorageaccounts": storageAccounts},
	})
	_ = v
}

func (h *Handler) siteKey(sub, rg, name string) string {
	return "func:" + sub + ":" + rg + ":" + name
}

func (h *Handler) azureStorageAccountsKey(sub, rg, site string) string {
	return "webapp:storage:" + sub + ":" + rg + ":" + site
}

func (h *Handler) authSettingsKey(sub, rg, site string) string {
	return "webapp:auth:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetAuthSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	authSettings, ok := h.store.Get(h.authSettingsKey(sub, rg, name))
	if !ok {
		authSettings = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/authsettings",
			"name": name + "/authsettings",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"enabled":                     false,
				"unauthenticatedClientAction": "RedirectToLoginPage",
				"tokenStoreEnabled":           false,
				"allowedExternalRedirectUrls": []interface{}{},
				"defaultProvider":             "AzureActiveDirectory",
				"tokenRefreshExtensionHours":  0,
				"validateNonce":               true,
				"excludedPaths":               []interface{}{},
				"requireHttps":                true,
				"httpApi":                     map[string]interface{}{"enabled": false},
			},
		}
	}

	json.NewEncoder(w).Encode(authSettings)
	_ = v
}

func (h *Handler) ListAuthSettings(w http.ResponseWriter, r *http.Request) {
	h.GetAuthSettings(w, r)
}

func (h *Handler) UpdateAuthSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"enabled":                     false,
			"unauthenticatedClientAction": "RedirectToLoginPage",
			"tokenStoreEnabled":           false,
			"allowedExternalRedirectUrls": []interface{}{},
			"defaultProvider":             "AzureActiveDirectory",
			"httpApi":                     map[string]interface{}{"enabled": false},
		}
	}

	authSettings := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/authsettings",
		"name":       name + "/authsettings",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.authSettingsKey(sub, rg, name), authSettings)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authSettings)
	_ = v
}

func (h *Handler) backupSettingsKey(sub, rg, site string) string {
	return "webapp:backup:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetBackupSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	backupSettings, ok := h.store.Get(h.backupSettingsKey(sub, rg, name))
	if !ok {
		backupSettings = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/backup",
			"name": name + "/backup",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"enabled":               false,
				"backupSchedule":        map[string]interface{}{"frequencyInterval": 0, "frequencyUnit": "Day"},
				"retentionPeriodInDays": 0,
				"storageAccountUrl":     "",
				"databases":             []interface{}{},
			},
		}
	}

	json.NewEncoder(w).Encode(backupSettings)
	_ = v
}

func (h *Handler) UpdateBackupSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"enabled":               false,
			"backupSchedule":        map[string]interface{}{"frequencyInterval": 0, "frequencyUnit": "Day"},
			"retentionPeriodInDays": 0,
			"databases":             []interface{}{},
		}
	}

	backupSettings := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/backup",
		"name":       name + "/backup",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.backupSettingsKey(sub, rg, name), backupSettings)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(backupSettings)
	_ = v
}

func (h *Handler) logsSettingsKey(sub, rg, site string) string {
	return "webapp:logs:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetLogsSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	logsSettings, ok := h.store.Get(h.logsSettingsKey(sub, rg, name))
	if !ok {
		logsSettings = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/logs",
			"name": name + "/logs",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"applicationLogs": map[string]interface{}{
					"fileSystem":        map[string]interface{}{"level": "Off"},
					"azureTableStorage": map[string]interface{}{"level": "Off"},
					"azureBlobStorage":  map[string]interface{}{"level": "Off"},
				},
				"httpLogs": map[string]interface{}{
					"fileSystem":       map[string]interface{}{"enabled": false},
					"azureBlobStorage": map[string]interface{}{"enabled": false},
				},
				"failedRequestsTracingEnabled": false,
				"detailedErrorMessagesEnabled": false,
			},
		}
	}

	json.NewEncoder(w).Encode(logsSettings)
	_ = v
}

func (h *Handler) UpdateLogsSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"applicationLogs": map[string]interface{}{
				"fileSystem":        map[string]interface{}{"level": "Off"},
				"azureTableStorage": map[string]interface{}{"level": "Off"},
				"azureBlobStorage":  map[string]interface{}{"level": "Off"},
			},
			"httpLogs": map[string]interface{}{
				"fileSystem":       map[string]interface{}{"enabled": false},
				"azureBlobStorage": map[string]interface{}{"enabled": false},
			},
			"failedRequestsTracingEnabled": false,
			"detailedErrorMessagesEnabled": false,
		}
	}

	logsSettings := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/logs",
		"name":       name + "/logs",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.logsSettingsKey(sub, rg, name), logsSettings)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(logsSettings)
	_ = v
}

func (h *Handler) slotConfigNamesKey(sub, rg, site string) string {
	return "webapp:slotconfignames:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetSlotConfigNames(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	slotConfigNames, ok := h.store.Get(h.slotConfigNamesKey(sub, rg, name))
	if !ok {
		slotConfigNames = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/slotConfigNames",
			"name": name + "/slotConfigNames",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"azureStoragePaths":     nil,
				"connectionStringNames": []interface{}{},
				"appSettingNames":       []interface{}{},
				"handlerMappingNames":   []interface{}{},
			},
		}
	}

	json.NewEncoder(w).Encode(slotConfigNames)
	_ = v
}

func (h *Handler) UpdateSlotConfigNames(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"azureStoragePaths":     nil,
			"connectionStringNames": []interface{}{},
			"appSettingNames":       []interface{}{},
			"handlerMappingNames":   []interface{}{},
		}
	}

	slotConfigNames := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/slotConfigNames",
		"name":       name + "/slotConfigNames",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.slotConfigNamesKey(sub, rg, name), slotConfigNames)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(slotConfigNames)
	_ = v
}

func (h *Handler) metadataKey(sub, rg, site string) string {
	return "webapp:metadata:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	metadata, ok := h.store.Get(h.metadataKey(sub, rg, name))
	if !ok {
		metadata = map[string]interface{}{
			"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/metadata",
			"name":       name + "/metadata",
			"type":       "Microsoft.Web/sites/config",
			"kind":       nil,
			"properties": map[string]interface{}{},
		}
	}

	json.NewEncoder(w).Encode(metadata)
	_ = v
}

func (h *Handler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	metadata := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/metadata",
		"name":       name + "/metadata",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.metadataKey(sub, rg, name), metadata)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metadata)
	_ = v
}

func (h *Handler) connectionStringsKey(sub, rg, site string) string {
	return "webapp:connectionstrings:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetConnectionStrings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	connectionStrings := map[string]interface{}{}
	items := h.store.ListByPrefix(h.connectionStringsKey(sub, rg, name))
	for _, item := range items {
		if cs, ok := item.(map[string]interface{}); ok {
			csName, _ := cs["name"].(string)
			if csName != "" {
				connectionStrings[csName] = cs
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/connectionstrings",
		"name":       name + "/connectionstrings",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": map[string]interface{}{"connectionStrings": connectionStrings},
	})
	_ = v
}

func (h *Handler) ListConnectionStrings(w http.ResponseWriter, r *http.Request) {
	h.GetConnectionStrings(w, r)
}

func (h *Handler) UpdateConnectionStrings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	inputConnectionStrings, _ := props["connectionStrings"].(map[string]interface{})

	connectionStrings := map[string]interface{}{}
	for connName, connConfig := range inputConnectionStrings {
		connInfo, ok := connConfig.(map[string]interface{})
		if !ok {
			continue
		}
		connectionStrings[connName] = map[string]interface{}{
			"type":  connInfo["type"],
			"value": connInfo["value"],
		}
		h.store.Set(h.connectionStringsKey(sub, rg, name)+":"+connName, connectionStrings[connName])
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/connectionstrings",
		"name":       name + "/connectionstrings",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": map[string]interface{}{"connectionStrings": connectionStrings},
	})
	_ = v
}

func (h *Handler) pushSettingsKey(sub, rg, site string) string {
	return "webapp:pushsettings:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetPushSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	pushSettings, ok := h.store.Get(h.pushSettingsKey(sub, rg, name))
	if !ok {
		pushSettings = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/pushsettings",
			"name": name + "/pushsettings",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"isPushEnabled":     false,
				"dynamicTagsJson":   "",
				"tagWhitelistJson":  "",
				"tagsRequiringAuth": "",
			},
		}
	}

	json.NewEncoder(w).Encode(pushSettings)
	_ = v
}

func (h *Handler) UpdatePushSettings(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"isPushEnabled":     false,
			"dynamicTagsJson":   "",
			"tagWhitelistJson":  "",
			"tagsRequiringAuth": "",
		}
	}

	pushSettings := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/pushsettings",
		"name":       name + "/pushsettings",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.pushSettingsKey(sub, rg, name), pushSettings)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pushSettings)
	_ = v
}

func (h *Handler) authSettingsV2Key(sub, rg, site string) string {
	return "webapp:authv2:" + sub + ":" + rg + ":" + site
}

func (h *Handler) GetAuthSettingsV2(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	authSettingsV2, ok := h.store.Get(h.authSettingsV2Key(sub, rg, name))
	if !ok {
		authSettingsV2 = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/authsettingsV2",
			"name": name + "/authsettingsV2",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"globalValidation": map[string]interface{}{
					"requireAuthentication":       false,
					"unauthenticatedClientAction": "AllowAnonymous",
					"excludedPaths":               []interface{}{},
				},
				"httpSettings": map[string]interface{}{
					"requireHttps": true,
					"routes":       map[string]interface{}{"apiPrefix": ""},
					"forwardProxy": map[string]interface{}{"convention": "NoProxy"},
				},
				"identityProviders": map[string]interface{}{
					"azureActiveDirectory": map[string]interface{}{"enabled": false},
					"facebook":             map[string]interface{}{"enabled": false},
					"gitHub":               map[string]interface{}{"enabled": false},
					"google":               map[string]interface{}{"enabled": false},
					"twitter":              map[string]interface{}{"enabled": false},
					"apple":                map[string]interface{}{"enabled": false},
				},
				"login": map[string]interface{}{
					"routes": map[string]interface{}{"logoutEndpoint": ""},
					"tokenStore": map[string]interface{}{
						"enabled": false,
					},
					"cookieExpiration": map[string]interface{}{
						"convention": "FixedTime",
					},
				},
				"platform": map[string]interface{}{
					"enabled":        false,
					"runtimeVersion": "",
				},
			},
		}
	}

	json.NewEncoder(w).Encode(authSettingsV2)
	_ = v
}

func (h *Handler) ListAuthSettingsV2(w http.ResponseWriter, r *http.Request) {
	h.GetAuthSettingsV2(w, r)
}

func (h *Handler) UpdateAuthSettingsV2(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{
			"globalValidation": map[string]interface{}{
				"requireAuthentication":       false,
				"unauthenticatedClientAction": "AllowAnonymous",
				"excludedPaths":               []interface{}{},
			},
			"httpSettings": map[string]interface{}{
				"requireHttps": true,
				"routes":       map[string]interface{}{"apiPrefix": ""},
				"forwardProxy": map[string]interface{}{"convention": "NoProxy"},
			},
			"identityProviders": map[string]interface{}{
				"azureActiveDirectory": map[string]interface{}{"enabled": false},
				"facebook":             map[string]interface{}{"enabled": false},
				"gitHub":               map[string]interface{}{"enabled": false},
				"google":               map[string]interface{}{"enabled": false},
				"twitter":              map[string]interface{}{"enabled": false},
				"apple":                map[string]interface{}{"enabled": false},
			},
			"login": map[string]interface{}{
				"routes":           map[string]interface{}{"logoutEndpoint": ""},
				"tokenStore":       map[string]interface{}{"enabled": false},
				"cookieExpiration": map[string]interface{}{"convention": "FixedTime"},
			},
			"platform": map[string]interface{}{
				"enabled":        false,
				"runtimeVersion": "",
			},
		}
	}

	authSettingsV2 := map[string]interface{}{
		"id":         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/authsettingsV2",
		"name":       name + "/authsettingsV2",
		"type":       "Microsoft.Web/sites/config",
		"kind":       nil,
		"properties": props,
	}

	h.store.Set(h.authSettingsV2Key(sub, rg, name), authSettingsV2)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authSettingsV2)
	_ = v
}

func (h *Handler) publishingCredentialsKey(sub, rg, site string) string {
	return "webapp:publishingcredentials:" + sub + ":" + rg + ":" + site
}

func (h *Handler) ListPublishingCredentials(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")

	v, ok := h.store.Get(h.siteKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites", name)
		return
	}

	publishingCredentials, ok := h.store.Get(h.publishingCredentialsKey(sub, rg, name))
	if !ok {
		publishingCredentials = map[string]interface{}{
			"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + name + "/config/publishingcredentials",
			"name": name + "/publishingcredentials",
			"type": "Microsoft.Web/sites/config",
			"kind": nil,
			"properties": map[string]interface{}{
				"publishingUsername":          "$" + name,
				"publishingPassword":          "",
				"passwordHash":                nil,
				"passwordSalt":                nil,
				"activeDirectoryUserTenantId": nil,
				"activeDirectoryUserId":       nil,
			},
		}
	}

	json.NewEncoder(w).Encode(publishingCredentials)
	_ = v
}
