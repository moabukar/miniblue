package sites

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

func (h *Handler) slotKey(sub, rg, site, slot string) string {
	return "slot:" + sub + ":" + rg + ":" + site + ":" + slot
}

func (h *Handler) ListSlots(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "name")
	items := h.store.ListByPrefix(h.slotKey(sub, rg, name, ""))
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) buildSlotResponse(sub, rg, siteName, slotName string, input map[string]interface{}) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Web/sites/" + siteName + "/slots/" + slotName

	props, _ := input["properties"].(map[string]interface{})
	if props == nil {
		props = map[string]interface{}{}
	}

	location, _ := input["location"].(string)
	if location == "" {
		location = "eastus"
	}

	tags, _ := input["tags"].(map[string]interface{})
	if tags == nil {
		tags = map[string]interface{}{}
	}

	return map[string]interface{}{
		"id":       id,
		"name":     siteName + "/" + slotName,
		"type":     "Microsoft.Web/sites/slots",
		"kind":     input["kind"],
		"location": location,
		"tags":     tags,
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"defaultHostName":   slotName + "-" + siteName + ".azurewebsites.net",
			"hostNames":         []interface{}{slotName + "-" + siteName + ".azurewebsites.net"},
			"state":             "Running",
			"enabled":           true,
			"serverFarmId":      props["serverFarmId"],
		},
	}
}

func (h *Handler) CreateOrUpdateSlot(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	siteName := chi.URLParam(r, "name")
	slotName := chi.URLParam(r, "slotName")

	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	slot := h.buildSlotResponse(sub, rg, siteName, slotName, input)
	k := h.slotKey(sub, rg, siteName, slotName)
	_, exists := h.store.Get(k)
	h.store.Set(k, slot)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(slot)
}

func (h *Handler) GetSlot(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	siteName := chi.URLParam(r, "name")
	slotName := chi.URLParam(r, "slotName")

	v, ok := h.store.Get(h.slotKey(sub, rg, siteName, slotName))
	if !ok {
		azerr.NotFound(w, "Microsoft.Web/sites/slots", slotName)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteSlot(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	siteName := chi.URLParam(r, "name")
	slotName := chi.URLParam(r, "slotName")

	if !h.store.Delete(h.slotKey(sub, rg, siteName, slotName)) {
		azerr.NotFound(w, "Microsoft.Web/sites/slots", slotName)
		return
	}
	w.WriteHeader(http.StatusOK)
}
