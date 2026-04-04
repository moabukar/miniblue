package dns

import (
	"encoding/json"
	"github.com/moabukar/local-azure/internal/azerr"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Zone struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Location   string `json:"location"`
	Properties struct {
		NumberOfRecordSets int `json:"numberOfRecordSets"`
	} `json:"properties"`
}

type RecordSet struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Properties struct {
		TTL     int      `json:"TTL"`
		ARecords []struct {
			IPv4Address string `json:"ipv4Address"`
		} `json:"ARecords,omitempty"`
	} `json:"properties"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsZones", func(r chi.Router) {
		r.Get("/", h.ListZones)
		r.Route("/{zoneName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateZone)
			r.Get("/", h.GetZone)
			r.Delete("/", h.DeleteZone)
			r.Route("/{recordType}/{recordName}", func(r chi.Router) {
				r.Put("/", h.CreateOrUpdateRecordSet)
				r.Get("/", h.GetRecordSet)
				r.Delete("/", h.DeleteRecordSet)
			})
		})
	})
}

func (h *Handler) zoneKey(sub, rg, name string) string {
	return "dns:zone:" + sub + ":" + rg + ":" + name
}

func (h *Handler) CreateOrUpdateZone(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "zoneName")
	
	var zone Zone
	json.NewDecoder(r.Body).Decode(&zone)
	zone.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/dnsZones/" + name
	zone.Name = name
	zone.Type = "Microsoft.Network/dnsZones"
	zone.Location = "global"
	
	h.store.Set(h.zoneKey(sub, rg, name), zone)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(zone)
}

func (h *Handler) GetZone(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "zoneName")
	
	v, ok := h.store.Get(h.zoneKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/dnsZones", "resource")
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteZone(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "zoneName")
	h.store.Delete(h.zoneKey(sub, rg, name))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListZones(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("dns:zone:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) CreateOrUpdateRecordSet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	zoneName := chi.URLParam(r, "zoneName")
	recordType := chi.URLParam(r, "recordType")
	recordName := chi.URLParam(r, "recordName")
	
	var rs RecordSet
	json.NewDecoder(r.Body).Decode(&rs)
	rs.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/dnsZones/" + zoneName + "/" + recordType + "/" + recordName
	rs.Name = recordName
	rs.Type = "Microsoft.Network/dnsZones/" + recordType
	
	key := "dns:record:" + sub + ":" + rg + ":" + zoneName + ":" + recordType + ":" + recordName
	h.store.Set(key, rs)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rs)
}

func (h *Handler) GetRecordSet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	zoneName := chi.URLParam(r, "zoneName")
	recordType := chi.URLParam(r, "recordType")
	recordName := chi.URLParam(r, "recordName")
	
	key := "dns:record:" + sub + ":" + rg + ":" + zoneName + ":" + recordType + ":" + recordName
	v, ok := h.store.Get(key)
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/dnsZones", "resource")
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteRecordSet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	zoneName := chi.URLParam(r, "zoneName")
	recordType := chi.URLParam(r, "recordType")
	recordName := chi.URLParam(r, "recordName")
	
	key := "dns:record:" + sub + ":" + rg + ":" + zoneName + ":" + recordType + ":" + recordName
	h.store.Delete(key)
	w.WriteHeader(http.StatusOK)
}
