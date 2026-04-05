package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/moabukar/miniblue/internal/services/aci"
	"github.com/moabukar/miniblue/internal/services/acr"
	"github.com/moabukar/miniblue/internal/services/appconfig"
	"github.com/moabukar/miniblue/internal/services/auth"
	"github.com/moabukar/miniblue/internal/services/blob"
	"github.com/moabukar/miniblue/internal/services/cosmosdb"
	"github.com/moabukar/miniblue/internal/services/dbmysql"
	"github.com/moabukar/miniblue/internal/services/dbpostgres"
	"github.com/moabukar/miniblue/internal/services/dns"
	"github.com/moabukar/miniblue/internal/services/eventgrid"
	"github.com/moabukar/miniblue/internal/services/functions"
	"github.com/moabukar/miniblue/internal/services/identity"
	"github.com/moabukar/miniblue/internal/services/keyvault"
	"github.com/moabukar/miniblue/internal/services/metadata"
	"github.com/moabukar/miniblue/internal/services/network"
	"github.com/moabukar/miniblue/internal/services/queue"
	"github.com/moabukar/miniblue/internal/services/redis"
	"github.com/moabukar/miniblue/internal/services/resourcegroups"
	"github.com/moabukar/miniblue/internal/services/sqldb"
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

// SaveState persists the store to disk if file-based persistence is active.
func (s *Server) SaveState() error {
	return s.store.Save()
}

func (s *Server) setupMiddleware() {
	s.router.Use(CORS)
	s.router.Use(StructuredLogger)
	s.router.Use(safeRecover)
	s.router.Use(middleware.RequestID)
	s.router.Use(AzureHeaders)
	s.router.Use(APIVersionCheck)
}

// safeRecover catches panics and returns a 500 error instead of crashing the server.
func safeRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				log.Printf("[PANIC] %s %s: %v", r.Method, r.URL.Path, rvr)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "InternalServerError",
						"message": "An unexpected error occurred. The server has recovered.",
					},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// serviceEnabled checks if a service should be registered based on the SERVICES env var.
// If SERVICES is empty, all services are enabled. Otherwise only listed ones are.
func serviceEnabled(name string, allowed map[string]bool) bool {
	if allowed == nil {
		return true // no filter, enable all
	}
	return allowed[name]
}

// parseServicesFilter reads the SERVICES env var and returns a set of enabled service names.
// Returns nil if SERVICES is empty (meaning all services are enabled).
func parseServicesFilter() map[string]bool {
	env := os.Getenv("SERVICES")
	if env == "" {
		return nil
	}
	parts := strings.Split(env, ",")
	allowed := make(map[string]bool, len(parts))
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name != "" {
			allowed[name] = true
		}
	}
	if len(allowed) > 0 {
		names := make([]string, 0, len(allowed))
		for k := range allowed {
			names = append(names, k)
		}
		log.Printf("SERVICES filter active: %v", names)
	}
	return allowed
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.healthHandler)
	s.router.Get("/metrics", s.metricsHandler)
	s.router.Post("/_miniblue/reset", s.resetHandler)

	allowed := parseServicesFilter()

	// Cloud metadata + auth (always registered -- core infrastructure)
	metadata.NewHandler(s.store).Register(s.router)
	auth.NewHandler(s.store).Register(s.router)

	// Subscriptions + tenants (always registered -- core infrastructure)
	subscriptions.NewHandler(s.store).Register(s.router)

	// Azure services (filterable via SERVICES env var)
	type namedService struct {
		name     string
		register func()
	}
	services := []namedService{
		{"resourcegroups", func() { resourcegroups.NewHandler(s.store).Register(s.router) }},
		{"blob", func() { blob.NewHandler(s.store).Register(s.router) }},
		{"table", func() { table.NewHandler(s.store).Register(s.router) }},
		{"queue", func() { queue.NewHandler(s.store).Register(s.router) }},
		{"keyvault", func() { keyvault.NewHandler(s.store).Register(s.router) }},
		{"cosmosdb", func() { cosmosdb.NewHandler(s.store).Register(s.router) }},
		{"servicebus", func() { servicebus.NewHandler(s.store).Register(s.router) }},
		{"functions", func() { functions.NewHandler(s.store).Register(s.router) }},
		{"network", func() { network.NewHandler(s.store).Register(s.router) }},
		{"dns", func() { dns.NewHandler(s.store).Register(s.router) }},
		{"aci", func() { aci.NewHandler(s.store).Register(s.router) }},
		{"acr", func() { acr.NewHandler(s.store).Register(s.router) }},
		{"eventgrid", func() { eventgrid.NewHandler(s.store).Register(s.router) }},
		{"appconfig", func() { appconfig.NewHandler(s.store).Register(s.router) }},
		{"identity", func() { identity.NewHandler(s.store).Register(s.router) }},
		{"dbpostgres", func() { dbpostgres.NewHandler(s.store).Register(s.router) }},
		{"redis", func() { redis.NewHandler(s.store).Register(s.router) }},
		{"sqldb", func() { sqldb.NewHandler(s.store).Register(s.router) }},
		{"dbmysql", func() { dbmysql.NewHandler(s.store).Register(s.router) }},
	}
	for _, svc := range services {
		if serviceEnabled(svc.name, allowed) {
			svc.register()
		}
	}
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
	if os.Getenv("PERSISTENCE") == "1" {
		return "file"
	}
	if os.Getenv("DATABASE_URL") != "" {
		return "postgres"
	}
	return "memory"
}

func (s *Server) resetHandler(w http.ResponseWriter, r *http.Request) {
	s.store.Reset()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "reset",
		"message": "All state cleared",
	})
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	services := []string{
		"subscriptions", "tenants", "resourcegroups", "blob", "table", "queue", "keyvault",
		"cosmosdb", "servicebus", "functions", "network", "dns",
		"aci", "acr", "eventgrid", "appconfig", "identity", "dbpostgres", "redis",
		"sqldb", "dbmysql",
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "running",
		"services": services,
		"version":  "0.2.0",
	})
}
