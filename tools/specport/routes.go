package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/server"
)

// routeRecord is one row from chi.Walk: a single (method, pattern) pair.
type routeRecord struct {
	Method  string
	Pattern string
}

// collectMinibluRoutes boots a fresh miniblue router and walks every
// registered route. The router is created via the same constructor used at
// runtime so we see the exact same surface real Terraform/SDK clients hit.
//
// Side-effect-prone env vars are cleared first so booting the router from a
// developer's shell never accidentally writes to ~/.miniblue/state.json or
// connects to a postgres on $DATABASE_URL.
func collectMiniblueRoutes() ([]routeRecord, error) {
	for _, k := range []string{
		"PERSISTENCE",
		"DATABASE_URL",
		"AKS_BACKEND",
		"MINIBLUE_AKS_REAL",
		"SERVICES",
	} {
		_ = os.Unsetenv(k)
	}

	srv := server.New()
	h := srv.Handler()
	mux, ok := h.(chi.Routes)
	if !ok {
		return nil, fmt.Errorf("server.Handler() is %T, expected chi.Routes", h)
	}

	var out []routeRecord
	walk := func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		out = append(out, routeRecord{
			Method:  strings.ToUpper(method),
			Pattern: route,
		})
		return nil
	}
	if err := chi.Walk(mux, walk); err != nil {
		return nil, fmt.Errorf("walk routes: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pattern != out[j].Pattern {
			return out[i].Pattern < out[j].Pattern
		}
		return out[i].Method < out[j].Method
	})
	return out, nil
}

// normalizePattern collapses URL parameter names to "{}", lowercases all
// literal segments, and drops trailing slashes. This is the canonical form
// used to compare a chi route pattern with a swagger path.
//
// Two routes match iff their normalized forms are equal.
func normalizePattern(p string) string {
	var b strings.Builder
	depth := 0
	for _, r := range p {
		switch r {
		case '{':
			if depth == 0 {
				b.WriteByte('{')
			}
			depth++
		case '}':
			depth--
			if depth == 0 {
				b.WriteByte('}')
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	s := strings.ToLower(b.String())
	// Collapse "//" introduced by chi's route grouping.
	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}
	s = strings.TrimRight(s, "/")
	if s == "" {
		s = "/"
	}
	return s
}
