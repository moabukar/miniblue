package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/moabukar/local-azure/internal/services/appconfig"
	"github.com/moabukar/local-azure/internal/services/blob"
	"github.com/moabukar/local-azure/internal/services/cosmosdb"
	"github.com/moabukar/local-azure/internal/services/dns"
	"github.com/moabukar/local-azure/internal/services/eventgrid"
	"github.com/moabukar/local-azure/internal/services/functions"
	"github.com/moabukar/local-azure/internal/services/identity"
	"github.com/moabukar/local-azure/internal/services/keyvault"
	"github.com/moabukar/local-azure/internal/services/network"
	"github.com/moabukar/local-azure/internal/services/queue"
	"github.com/moabukar/local-azure/internal/services/acr"
	"github.com/moabukar/local-azure/internal/services/resourcegroups"
	"github.com/moabukar/local-azure/internal/services/servicebus"
	"github.com/moabukar/local-azure/internal/services/table"
	"github.com/moabukar/local-azure/internal/store"
)

type Server struct {
	router *chi.Mux
	store  *store.Store
}

func New() *Server {
	s := &Server{
		router: chi.NewRouter(),
		store:  store.New(),
	}
	s.setupMiddleware()
	s.setupRoutes()
	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RequestID)
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("x-ms-version", "2023-11-03")
			next.ServeHTTP(w, r)
		})
	})
}

func (s *Server) setupRoutes() {
	s.router.Get("/health", s.healthHandler)
	
	// Register all services
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

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	services := []string{
		"resourcegroups", "blob", "table", "queue", "keyvault",
		"cosmosdb", "servicebus", "functions", "network", "dns",
		"acr", "eventgrid", "appconfig", "identity",
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "running",
		"services": services,
	})
}

func (s *Server) Run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4566"
	}
	log.Printf("local-azure starting on :%s", port)
	return http.ListenAndServe(":"+port, s.router)
}
