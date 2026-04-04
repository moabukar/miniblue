package servicebus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Message struct {
	MessageId string `json:"messageId"`
	Body      string `json:"body"`
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
		r.Route("/queues/{queueName}", func(r chi.Router) {
			r.Put("/", h.CreateQueue)
			r.Delete("/", h.DeleteQueue)
			r.Post("/messages", h.SendMessage)
			r.Get("/messages/head", h.ReceiveMessage)
		})
		r.Route("/topics/{topicName}", func(r chi.Router) {
			r.Put("/", h.CreateTopic)
			r.Delete("/", h.DeleteTopic)
			r.Post("/messages", h.PublishMessage)
		})
	})
}

func (h *Handler) CreateQueue(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) CreateTopic(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeleteTopic(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	queue := chi.URLParam(r, "queueName")
	
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
	
	key := "sb:" + ns + ":queue:" + queue + ":" + msg.MessageId
	h.store.Set(key, msg)
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) ReceiveMessage(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	queue := chi.URLParam(r, "queueName")
	items := h.store.ListByPrefix("sb:" + ns + ":queue:" + queue + ":")
	if len(items) > 0 {
		json.NewEncoder(w).Encode(items[0])
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
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
	
	key := "sb:" + ns + ":topic:" + topic + ":" + msg.MessageId
	h.store.Set(key, msg)
	w.WriteHeader(http.StatusCreated)
}
