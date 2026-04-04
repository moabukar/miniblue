package dns

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/azerr"
	"github.com/moabukar/local-azure/internal/store"
)

type Zone struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Location   string    `json:"location"`
	Properties ZoneProps `json:"properties"`
}

type ZoneProps struct {
	NumberOfRecordSets int      `json:"numberOfRecordSets"`
	NameServers        []string `json:"nameServers"`
}

type RecordSet struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
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

func (h *Handler) recordKey(sub, rg, zone, rtype, name string) string {
	return "dns:record:" + sub + ":" + rg + ":" + zone + ":" + rtype + ":" + name
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
	zone.Properties.NameServers = []string{"ns1-01.azure-dns.com.", "ns2-01.azure-dns.net."}

	k := h.zoneKey(sub, rg, name)
	_, exists := h.store.Get(k)

	// Count records for this zone
	zone.Properties.NumberOfRecordSets = h.store.CountByPrefix("dns:record:" + sub + ":" + rg + ":" + name + ":")

	h.store.Set(k, zone)

	// Auto-create SOA and NS records (Azure always creates these for new zones)
	if !exists {
		soaProps, _ := json.Marshal(map[string]interface{}{
			"TTL": 3600,
			"SOARecord": map[string]interface{}{
				"host":         "ns1-01.azure-dns.com.",
				"email":        "azuredns-hostmaster.microsoft.com",
				"serialNumber": 1,
				"refreshTime":  3600,
				"retryTime":    300,
				"expireTime":   2419200,
				"minimumTTL":   300,
			},
			"fqdn": name + ".",
		})
		h.store.Set(h.recordKey(sub, rg, name, "SOA", "@"), RecordSet{
			ID:         zone.ID + "/SOA/@",
			Name:       "@",
			Type:       "Microsoft.Network/dnsZones/SOA",
			Properties: json.RawMessage(soaProps),
		})

		nsProps, _ := json.Marshal(map[string]interface{}{
			"TTL":       172800,
			"NSRecords": []map[string]string{{"nsdname": "ns1-01.azure-dns.com."}, {"nsdname": "ns2-01.azure-dns.net."}},
			"fqdn":      name + ".",
		})
		h.store.Set(h.recordKey(sub, rg, name, "NS", "@"), RecordSet{
			ID:         zone.ID + "/NS/@",
			Name:       "@",
			Type:       "Microsoft.Network/dnsZones/NS",
			Properties: json.RawMessage(nsProps),
		})

		zone.Properties.NumberOfRecordSets = 2
		h.store.Set(k, zone)
	}

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(zone)
}

func (h *Handler) GetZone(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "zoneName")

	v, ok := h.store.Get(h.zoneKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/dnsZones", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteZone(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "zoneName")

	if !h.store.Delete(h.zoneKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Network/dnsZones", name)
		return
	}
	// Clean up all records in the zone
	h.store.DeleteByPrefix("dns:record:" + sub + ":" + rg + ":" + name + ":")
	w.WriteHeader(http.StatusAccepted)
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

	// Verify parent zone exists
	if !h.store.Exists(h.zoneKey(sub, rg, zoneName)) {
		azerr.NotFound(w, "Microsoft.Network/dnsZones", zoneName)
		return
	}

	// Read raw properties to support any record type
	var raw map[string]json.RawMessage
	json.NewDecoder(r.Body).Decode(&raw)

	rs := RecordSet{
		ID:         "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Network/dnsZones/" + zoneName + "/" + recordType + "/" + recordName,
		Name:       recordName,
		Type:       "Microsoft.Network/dnsZones/" + recordType,
		Properties: raw["properties"],
	}

	k := h.recordKey(sub, rg, zoneName, recordType, recordName)
	_, exists := h.store.Get(k)
	h.store.Set(k, rs)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(rs)
}

func (h *Handler) GetRecordSet(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	zoneName := chi.URLParam(r, "zoneName")
	recordType := chi.URLParam(r, "recordType")
	recordName := chi.URLParam(r, "recordName")

	v, ok := h.store.Get(h.recordKey(sub, rg, zoneName, recordType, recordName))
	if !ok {
		azerr.NotFound(w, "Microsoft.Network/dnsZones/"+recordType, recordName)
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

	if !h.store.Delete(h.recordKey(sub, rg, zoneName, recordType, recordName)) {
		azerr.NotFound(w, "Microsoft.Network/dnsZones/"+recordType, recordName)
		return
	}
	w.WriteHeader(http.StatusOK)
}
