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
	// ARM-style paths: used by Azure SDKs to enumerate and manage resources
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceBus/namespaces", func(r chi.Router) {
		r.Get("/", h.ListNamespaces)
		r.Route("/{namespaceName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateNamespace)
			r.Get("/", h.GetNamespace)
			r.Delete("/", h.DeleteNamespace)
			r.Route("/queues", func(r chi.Router) {
				r.Get("/", h.ListQueuesARM)
				r.Route("/{queueName}", func(r chi.Router) {
					r.Put("/", h.CreateQueueARM)
					r.Get("/", h.GetQueueARM)
					r.Delete("/", h.DeleteQueueARM)
				})
			})
			r.Route("/topics", func(r chi.Router) {
				r.Get("/", h.ListTopicsARM)
				r.Route("/{topicName}", func(r chi.Router) {
					r.Put("/", h.CreateTopicARM)
					r.Get("/", h.GetTopicARM)
					r.Delete("/", h.DeleteTopicARM)
				})
			})
		})
	})

	// Data-plane paths: used for messaging operations (send/receive)
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

// --- ARM namespace handlers ---

func (h *Handler) namespaceKey(sub, rg, name string) string {
	return "sb:namespace:" + sub + ":" + rg + ":" + name
}

func (h *Handler) buildNamespaceResponse(sub, rg, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":       "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ServiceBus/namespaces/" + name,
		"name":     name,
		"type":     "Microsoft.ServiceBus/namespaces",
		"location": "eastus",
		"sku": map[string]interface{}{
			"name": "Standard",
			"tier": "Standard",
		},
		"properties": map[string]interface{}{
			"provisioningState":       "Succeeded",
			"status":                  "Active",
			"serviceBusEndpoint":      "https://" + name + ".servicebus.windows.net:443/",
			"metricId":                sub + ":" + name,
			"createdAt":               "2026-01-01T00:00:00Z",
			"updatedAt":               "2026-01-01T00:00:00Z",
			"disableLocalAuth":        false,
			"zoneRedundant":           false,
		},
	}
}

func (h *Handler) CreateOrUpdateNamespace(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "namespaceName")

	k := h.namespaceKey(sub, rg, name)
	_, exists := h.store.Get(k)

	ns := h.buildNamespaceResponse(sub, rg, name)
	h.store.Set(k, ns)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(ns)
}

func (h *Handler) GetNamespace(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "namespaceName")

	v, ok := h.store.Get(h.namespaceKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteNamespace(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "namespaceName")

	if !h.store.Delete(h.namespaceKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces", name)
		return
	}
	h.store.DeleteByPrefix("sb:queue:" + name + ":")
	h.store.DeleteByPrefix("sb:topic:" + name + ":")
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) ListNamespaces(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("sb:namespace:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) armQueueKey(sub, rg, ns, name string) string {
	return "sb:armqueue:" + sub + ":" + rg + ":" + ns + ":" + name
}

func (h *Handler) buildARMQueueResponse(sub, rg, ns, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ServiceBus/namespaces/" + ns + "/queues/" + name,
		"name": name,
		"type": "Microsoft.ServiceBus/namespaces/queues",
		"properties": map[string]interface{}{
			"status":                  "Active",
			"messageCount":            0,
			"maxDeliveryCount":        10,
			"defaultMessageTimeToLive": "P14D",
			"lockDuration":            "PT1M",
			"createdAt":               "2026-01-01T00:00:00Z",
		},
	}
}

func (h *Handler) CreateQueueARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	ns := chi.URLParam(r, "namespaceName")
	name := chi.URLParam(r, "queueName")

	k := h.armQueueKey(sub, rg, ns, name)
	_, exists := h.store.Get(k)

	q := h.buildARMQueueResponse(sub, rg, ns, name)
	h.store.Set(k, q)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(q)
}

func (h *Handler) GetQueueARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	ns := chi.URLParam(r, "namespaceName")
	name := chi.URLParam(r, "queueName")

	v, ok := h.store.Get(h.armQueueKey(sub, rg, ns, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces/queues", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteQueueARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	ns := chi.URLParam(r, "namespaceName")
	name := chi.URLParam(r, "queueName")

	if !h.store.Delete(h.armQueueKey(sub, rg, ns, name)) {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces/queues", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListQueuesARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	ns := chi.URLParam(r, "namespaceName")
	items := h.store.ListByPrefix("sb:armqueue:" + sub + ":" + rg + ":" + ns + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) armTopicKey(sub, rg, ns, name string) string {
	return "sb:armtopic:" + sub + ":" + rg + ":" + ns + ":" + name
}

func (h *Handler) CreateTopicARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	ns := chi.URLParam(r, "namespaceName")
	name := chi.URLParam(r, "topicName")

	k := h.armTopicKey(sub, rg, ns, name)
	_, exists := h.store.Get(k)

	t := map[string]interface{}{
		"id":   "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.ServiceBus/namespaces/" + ns + "/topics/" + name,
		"name": name,
		"type": "Microsoft.ServiceBus/namespaces/topics",
		"properties": map[string]interface{}{
			"status":                  "Active",
			"defaultMessageTimeToLive": "P14D",
			"createdAt":               "2026-01-01T00:00:00Z",
		},
	}
	h.store.Set(k, t)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(t)
}

func (h *Handler) GetTopicARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	ns := chi.URLParam(r, "namespaceName")
	name := chi.URLParam(r, "topicName")

	v, ok := h.store.Get(h.armTopicKey(sub, rg, ns, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces/topics", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteTopicARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	ns := chi.URLParam(r, "namespaceName")
	name := chi.URLParam(r, "topicName")

	if !h.store.Delete(h.armTopicKey(sub, rg, ns, name)) {
		azerr.NotFound(w, "Microsoft.ServiceBus/namespaces/topics", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListTopicsARM(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	ns := chi.URLParam(r, "namespaceName")
	items := h.store.ListByPrefix("sb:armtopic:" + sub + ":" + rg + ":" + ns + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

// --- Data-plane topic handlers (legacy) ---

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
