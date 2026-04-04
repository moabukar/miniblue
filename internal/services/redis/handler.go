package redis

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
	"github.com/moabukar/miniblue/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cache/redis", func(r chi.Router) {
		r.Get("/", h.ListCaches)
		r.Route("/{cacheName}", func(r chi.Router) {
			r.Put("/", h.CreateOrUpdateCache)
			r.Get("/", h.GetCache)
			r.Delete("/", h.DeleteCache)
			r.Post("/listKeys", h.ListKeys)
		})
	})
}

func (h *Handler) cacheKey(sub, rg, name string) string {
	return "redis:" + sub + ":" + rg + ":" + name
}

func redisHostPort() (string, string) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return "localhost", "6379"
	}
	u, err := url.Parse(redisURL)
	if err != nil {
		return "localhost", "6379"
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "6379"
	}
	return host, port
}

func checkRedisConnectivity(host, port string) bool {
	conn, err := net.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func buildCacheResponse(sub, rg, name, host, port string) map[string]interface{} {
	id := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/Microsoft.Cache/redis/" + name
	portNum := 6379
	if port != "" {
		if n, err := strconv.Atoi(port); err == nil {
			portNum = n
		}
	}
	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"type":     "Microsoft.Cache/redis",
		"location": "eastus",
		"properties": map[string]interface{}{
			"provisioningState": "Succeeded",
			"hostName":          host,
			"port":              portNum,
			"sslPort":           6380,
			"redisVersion":      "7.2",
			"sku": map[string]interface{}{
				"name":     "Basic",
				"family":   "C",
				"capacity": 1,
			},
			"enableNonSslPort":  true,
			"publicNetworkAccess": "Enabled",
		},
	}
}

func (h *Handler) CreateOrUpdateCache(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "cacheName")

	host, port := redisHostPort()

	// If REDIS_URL is set, verify connectivity
	if os.Getenv("REDIS_URL") != "" {
		if !checkRedisConnectivity(host, port) {
			azerr.WriteError(w, http.StatusServiceUnavailable, "RedisUnreachable",
				"Cannot connect to Redis at "+net.JoinHostPort(host, port))
			return
		}
	}

	cache := buildCacheResponse(sub, rg, name, host, port)
	k := h.cacheKey(sub, rg, name)
	_, exists := h.store.Get(k)
	h.store.Set(k, cache)

	if exists {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(cache)
}

func (h *Handler) GetCache(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "cacheName")

	v, ok := h.store.Get(h.cacheKey(sub, rg, name))
	if !ok {
		azerr.NotFound(w, "Microsoft.Cache/redis", name)
		return
	}
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) DeleteCache(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "cacheName")

	if !h.store.Delete(h.cacheKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Cache/redis", name)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ListCaches(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	items := h.store.ListByPrefix("redis:" + sub + ":" + rg + ":")
	json.NewEncoder(w).Encode(map[string]interface{}{"value": items})
}

func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	name := chi.URLParam(r, "cacheName")

	if !h.store.Exists(h.cacheKey(sub, rg, name)) {
		azerr.NotFound(w, "Microsoft.Cache/redis", name)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"primaryKey":   "miniblue-redis-key",
		"secondaryKey": "miniblue-redis-key-2",
	})
}
