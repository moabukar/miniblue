package queue

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/local-azure/internal/store"
)

type Message struct {
	MessageId       string `json:"messageId"`
	MessageText     string `json:"messageText"`
	InsertionTime   string `json:"insertionTime"`
	ExpirationTime  string `json:"expirationTime"`
	PopReceipt      string `json:"popReceipt"`
	DequeueCount    int    `json:"dequeueCount"`
}

type Handler struct {
	store *store.Store
	msgId int
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/queue/{accountName}/{queueName}", func(r chi.Router) {
		r.Put("/", h.CreateQueue)
		r.Delete("/", h.DeleteQueue)
		r.Route("/messages", func(r chi.Router) {
			r.Post("/", h.SendMessage)
			r.Get("/", h.ReceiveMessages)
			r.Delete("/", h.ClearMessages)
		})
	})
}

func (h *Handler) CreateQueue(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	queue := chi.URLParam(r, "queueName")
	
	var body struct {
		MessageText string `json:"messageText"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	
	h.msgId++
	msg := Message{
		MessageId:      fmt.Sprintf("%d", h.msgId),
		MessageText:    body.MessageText,
		InsertionTime:  time.Now().UTC().Format(time.RFC3339),
		ExpirationTime: time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339),
		DequeueCount:   0,
	}
	
	key := "queue:" + account + ":" + queue + ":" + msg.MessageId
	h.store.Set(key, msg)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

func (h *Handler) ReceiveMessages(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "accountName")
	queue := chi.URLParam(r, "queueName")
	items := h.store.ListByPrefix("queue:" + account + ":" + queue + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"messages": items})
}

func (h *Handler) ClearMessages(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
