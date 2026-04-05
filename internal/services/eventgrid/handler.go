package eventgrid

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

type Topic struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Location   string     `json:"location"`
	Properties TopicProps `json:"properties"`
}

type TopicProps struct {
	ProvisioningState string `json:"provisioningState"`
	Endpoint          string `json:"endpoint"`
}

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics", func(r chi.Router) {
		r.Get("/", h.ListTopics)
		r.Route("/{topicName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateTopic)
			r.Get("/", h.GetTopic)
			r.Delete("/", h.DeleteTopic)
			r.Post("/listKeys", h.ListKeys)
		})
	})
	r.Post("/eventgrid/{topicName}/events", h.PublishEvents)
}

func (h *Handler) topicKey(sub, rg, name string) string {
	return "eg:topic:" + sub + ":" + rg + ":" + name
}

func (h *Handler) CreateOrUpdateTopic(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "topicName")

	var topic Topic
	json.NewDecoder(r.Body).Decode(&topic)
	topic.ID = "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.EventGrid/topics/" + name
	topic.Name = name
	topic.Type = "Microsoft.EventGrid/topics"
	topic.Properties.ProvisioningState = "Succeeded"
	topic.Properties.Endpoint = "https://" + name + ".eastus-1.eventgrid.azure.net/api/events"

	k := h.topicKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, topic)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(topic)
}

func (h *Handler) GetTopic(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "topicName")

	v, ok := h.store.Get(h.topicKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.EventGrid/topics", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "topicName")
	if !h.store.Delete(h.topicKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.EventGrid/topics", name)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListTopics(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("eg:topic:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "topicName")
	if !h.store.Exists(h.topicKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.EventGrid/topics", name)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key1": "miniblue-eventgrid-key1",
		"key2": "miniblue-eventgrid-key2",
	})
}

func (h *Handler) PublishEvents(w http.ResponseWriter, r *http.Request) {
	// Accept and store events
	var events []interface{}
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		azerr.BadRequest(w, "Invalid event payload: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}
