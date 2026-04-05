package servicebus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

type Queue struct {
	Name       string    `json:"name"`
	Properties QueueProps `json:"properties"`
}

type QueueProps struct {
	Status                string `json:"status"`
	MessageCount          int    `json:"messageCount"`
	MaxDeliveryCount      int    `json:"maxDeliveryCount"`
	DefaultMessageTTL     string `json:"defaultMessageTimeToLive"`
	LockDuration          string `json:"lockDuration"`
	CreatedAt             string `json:"createdAt"`
}

type Message struct {
	MessageId    string `json:"messageId"`
	Body         string `json:"body"`
	EnqueuedTime string `json:"enqueuedTime"`
}

type Handler struct {
	store *store.Store
	msgId int
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/servicebus/{namespace}", func(r chi.Router) {
		r.Route("/queues", func(r chi.Router) {
			r.Get("/", h.ListQueues)
			r.Route("/{queueName}", func(r chi.Router) {
				r.Put("/", h.CreateQueue)
				r.Get("/", h.GetQueue)
				r.Delete("/", h.DeleteQueue)
				r.Post("/messages", h.SendMessage)
				r.Get("/messages/head", h.ReceiveMessage)
			})
		})
		r.Route("/topics", func(r chi.Router) {
			r.Route("/{topicName}", func(r chi.Router) {
				r.Put("/", h.CreateTopic)
				r.Delete("/", h.DeleteTopic)
				r.Post("/messages", h.PublishMessage)
			})
		})
	})
}

func (h *Handler) queueKey(ns, name string) string {
	return "sb:queue:" + ns + ":" + name
}

func (h *Handler) msgPrefix(ns, name string) string {
	return "sb:msg:" + ns + ":" + name + ":"
}

func (h *Handler) CreateQueue(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "queueName")

	k := h.queueKey(ns, name)
	if h.store.Exists(k) {
		azerr.Conflict(w, "Microsoft.ServiceBus/namespaces/queues", name)
		return
	}

	q := Queue{
		Name: name,
		Properties: QueueProps{
			Status:                "Active",
			MessageCount:          0,
			MaxDeliveryCount:      10,
			DefaultMessageTTL:     "P14D",
			LockDuration:          "PT1M",
			CreatedAt:             time.Now().UTC().Format(time.RFC3339),
		},
	}

	h.store.Set(k, q)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(q)
}

func (h *Handler) GetQueue(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "queueName")

	v, ok := h.store.Get(h.queueKey(ns, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces/queues", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "queueName")

	if !h.store.Delete(h.queueKey(ns, name)) {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces/queues", name)
		return
	}
	h.store.DeleteByPrefix(h.msgPrefix(ns, name))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListQueues(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	items := h.store.ListByPrefix("sb:queue:" + ns + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	queue := chi.URLParam(r, "queueName")

	if !h.store.Exists(h.queueKey(ns, queue)) {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces/queues", queue)
		return
	}

	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		azerr.BadRequest(w, "Invalid message body: "+err.Error())
		return
	}

	h.msgId++
	msg := Message{
		MessageId:    fmt.Sprintf("%010d", h.msgId),
		Body:         body.Body,
		EnqueuedTime: time.Now().UTC().Format(time.RFC3339),
	}

	h.store.Set(h.msgPrefix(ns, queue)+msg.MessageId, msg)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

func (h *Handler) ReceiveMessage(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	queue := chi.URLParam(r, "queueName")

	if !h.store.Exists(h.queueKey(ns, queue)) {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces/queues", queue)
		return
	}

	// Get messages sorted by key (zero-padded IDs ensure FIFO order)
	prefix := h.msgPrefix(ns, queue)
	keys := h.store.ListKeysByPrefix(prefix)
	if len(keys) > 0 {
		sort.Strings(keys)
		if v, ok := h.store.Get(keys[0]); ok {
			json.NewEncoder(w).Encode(v)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name": chi.URLParam(r, "topicName"),
		"properties": map[string]interface{}{"status": "Active"},
	})
}

func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) PublishMessage(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	topic := chi.URLParam(r, "topicName")

	var body struct {
		Body string `json:"body"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	h.msgId++
	msg := Message{
		MessageId:    fmt.Sprintf("%d", h.msgId),
		Body:         body.Body,
		EnqueuedTime: time.Now().UTC().Format(time.RFC3339),
	}

	key := "sb:topic:" + ns + ":" + topic + ":" + msg.MessageId
	h.store.Set(key, msg)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}
