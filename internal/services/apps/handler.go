package apps

import (
	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/store"
)

const (
	APIVersion = "2024-03-01"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	h.RegisterManagedEnvironments(r)
	h.RegisterContainerApps(r)
	h.RegisterJobs(r)
}
