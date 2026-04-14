package queue

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

type Message struct {
	MessageId      string `json:"messageId"`
	MessageText    string `json:"messageText"`
	InsertionTime  string `json:"insertionTime"`
	ExpirationTime string `json:"expirationTime"`
	PopReceipt     string `json:"popReceipt"`
	DequeueCount   int    `json:"dequeueCount"`
}

type Handler struct {
	store *store.Store
	msgId int
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/queue/{accountName}", func(r chi.Router) {
		r.Get("/", h.ListQueues)
		r.Route("/{queueName}", func(r chi.Router) {
			r.Put("/", h.CreateQueue)
			r.Get("/", h.GetQueue)
			r.Delete("/", h.DeleteQueue)
			r.Route("/messages", func(r chi.Router) {
				r.Post("/", h.SendMessage)
				r.Get("/", h.ReceiveMessages)
				r.Delete("/", h.ClearMessages)
			})
		})
	})
}

func (h *Handler) queueKey(account, name string) string {
	return "queue:meta:" + account + ":" + name
}

func (h *Handler) msgPrefix(account, name string) string {
	return "queue:msg:" + account + ":" + name + ":"
}

func (h *Handler) CreateQueue(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "queueName")

	k := h.queueKey(account, name)
	if h.store.Exists(k) {
		azerr.Conflict(w, "StorageQueue", name)
		return
	}

	meta := map[string]interface{}{
		"name":    name,
		"created": time.Now().UTC().Format(time.RFC3339),
	}
	h.store.Set(k, meta)
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) GetQueue(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "queueName")

	if !h.store.Exists(h.queueKey(account, name)) {
		azerr.NotFound(w, "StorageQueue", name)
		return
	}
	count := h.store.CountByPrefix(h.msgPrefix(account, name))
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":                    name,
		"approximateMessageCount": count,
	})
}

func (h *Handler) ListQueues(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	items := h.store.ListByPrefix(h.queueKey(account, ""))
	json.NewEncoder(w).Encode(map[string]interface{}{"queueUrls": items})
}

func (h *Handler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "queueName")

	if !h.store.Delete(h.queueKey(account, name)) {
		azerr.NotFound(w, "StorageQueue", name)
		return
	}
	h.store.DeleteByPrefix(h.msgPrefix(account, name))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "queueName")

	if !h.store.Exists(h.queueKey(account, name)) {
		azerr.NotFound(w, "StorageQueue", name)
		return
	}

	var body struct {
		MessageText string `json:"messageText"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	h.msgId++
	now := time.Now().UTC()
	msg := Message{
		MessageId:      fmt.Sprintf("%d", h.msgId),
		MessageText:    body.MessageText,
		InsertionTime:  now.Format(time.RFC3339),
		ExpirationTime: now.Add(7 * 24 * time.Hour).Format(time.RFC3339),
		PopReceipt:     fmt.Sprintf("pop-%d", h.msgId),
		DequeueCount:   0,
	}

	h.store.Set(h.msgPrefix(account, name)+msg.MessageId, msg)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

func (h *Handler) ReceiveMessages(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "queueName")

	if !h.store.Exists(h.queueKey(account, name)) {
		azerr.NotFound(w, "StorageQueue", name)
		return
	}

	items := h.store.ListByPrefix(h.msgPrefix(account, name))
	json.NewEncoder(w).Encode(map[string]interface{}{"messages": items})
}

func (h *Handler) ClearMessages(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	name := chi.URLParam(r, "queueName")

	if !h.store.Exists(h.queueKey(account, name)) {
		azerr.NotFound(w, "StorageQueue", name)
		return
	}

	h.store.DeleteByPrefix(h.msgPrefix(account, name))
	w.WriteHeader(http.StatusNoContent)
}
