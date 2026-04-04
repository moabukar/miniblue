package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/moabukar/miniblue/internal/services/acr"
	"github.com/moabukar/miniblue/internal/services/appconfig"
	"github.com/moabukar/miniblue/internal/services/auth"
	"github.com/moabukar/miniblue/internal/services/blob"
	"github.com/moabukar/miniblue/internal/services/cosmosdb"
	"github.com/moabukar/miniblue/internal/services/dns"
	"github.com/moabukar/miniblue/internal/services/eventgrid"
	"github.com/moabukar/miniblue/internal/services/functions"
	"github.com/moabukar/miniblue/internal/services/identity"
	"github.com/moabukar/miniblue/internal/services/keyvault"
	"github.com/moabukar/miniblue/internal/services/metadata"
	"github.com/moabukar/miniblue/internal/services/network"
	"github.com/moabukar/miniblue/internal/services/queue"
	"github.com/moabukar/miniblue/internal/services/resourcegroups"
	"github.com/moabukar/miniblue/internal/services/servicebus"
	"github.com/moabukar/miniblue/internal/services/subscriptions"
	"github.com/moabukar/miniblue/internal/services/table"
	"github.com/moabukar/miniblue/internal/store"
)

type Server struct {
	router *chi.Mux
	store  *store.Store
}

func New() *Server {
	s := &Server{
		router: chi.NewRouter(),
		store:  store.NewWithBackend(store.NewBackend()),
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) setupMiddleware() {
	s.router.Use(StructuredLogger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RequestID)
	s.router.Use(AzureHeaders)
	s.router.Use(APIVersionCheck)
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.healthHandler)
	s.router.Get("/metrics", s.metricsHandler)

	// Cloud metadata + auth
	metadata.NewHandler(s.store).Register(s.router)
	auth.NewHandler(s.store).Register(s.router)

	// Subscriptions + tenants
	subscriptions.NewHandler(s.store).Register(s.router)

	// Azure services
	resourcegroups.NewHandler(s.store).Register(s.router)
	blob.NewHandler(s.store).Register(s.router)
	table.NewHandler(s.store).Register(s.router)
	queue.NewHandler(s.store).Register(s.router)
	keyvault.NewHandler(s.store).Register(s.router)
	cosmosdb.NewHandler(s.store).Register(s.router)
	servicebus.NewHandler(s.store).Register(s.router)
	functions.NewHandler(s.store).Register(s.router)
	network.NewHandler(s.store).Register(s.router)
	dns.NewHandler(s.store).Register(s.router)
	acr.NewHandler(s.store).Register(s.router)
	eventgrid.NewHandler(s.store).Register(s.router)
	appconfig.NewHandler(s.store).Register(s.router)
	identity.NewHandler(s.store).Register(s.router)
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	m := GetMetrics()
	uptime := time.Since(m.StartTime)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"uptime_seconds":  int(uptime.Seconds()),
		"total_requests":  m.TotalRequests.Load(),
		"total_errors":    m.TotalErrors.Load(),
		"error_rate":      fmt.Sprintf("%.2f%%", errorRate(m)),
		"store_backend":   storeBackendName(s.store),
	})
}

func errorRate(m *Metrics) float64 {
	total := m.TotalRequests.Load()
	if total == 0 {
		return 0
	}
	return float64(m.TotalErrors.Load()) / float64(total) * 100
}

func storeBackendName(st *store.Store) string {
	if os.Getenv("DATABASE_URL") != "" {
		return "postgres"
	}
	return "memory"
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	services := []string{
		"subscriptions", "tenants", "resourcegroups", "blob", "table", "queue", "keyvault",
		"cosmosdb", "servicebus", "functions", "network", "dns",
		"acr", "eventgrid", "appconfig", "identity",
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "running",
		"services": services,
		"version":  "0.1.0",
	})
}
