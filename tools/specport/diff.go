package main

import (
	"flag"
	"fmt"
	"strings"
)

// cmdDiff runs cmdExtract logic, then boots miniblue's router and rewrites
// each row's status to IMPLEMENTED/MISSING and appends EXTRA rows for
// in-scope chi routes that no spec operation covers.
func cmdDiff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	specDir := fs.String("spec-dir", defaultSpecDir, "directory holding service config files")
	outDir := fs.String("out-dir", defaultOutDir, "directory where checklists are written")
	failOnMissing := fs.Bool("fail-on-missing", false, "exit non-zero if any spec operation is MISSING")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("diff takes exactly one service name (got %d args)", fs.NArg())
	}
	service := fs.Arg(0)
	cfg, err := loadServiceConfig(*specDir, service)
	if err != nil {
		return err
	}

	cl, err := buildChecklist(cfg)
	if err != nil {
		return err
	}

	routes, err := collectMiniblueRoutes()
	if err != nil {
		return err
	}
	matchAndAnnotate(cl, cfg, routes)

	if err := writeChecklist(*outDir, cl); err != nil {
		return err
	}

	if *failOnMissing && cl.Summary != nil && cl.Summary.Missing > 0 {
		return fmt.Errorf("%d MISSING operation(s); fail-on-missing is set", cl.Summary.Missing)
	}
	return nil
}

// matchAndAnnotate fills in IMPLEMENTED / MISSING / EXTRA on the checklist.
// The matching strategy is purely structural:
//   - Each spec entry's MinibluePath is normalized via normalizePattern.
//   - Each chi route is normalized the same way.
//   - A spec entry is IMPLEMENTED iff some chi route has the same method
//     and normalized pattern.
//   - A chi route is EXTRA iff its pattern matches one of the service's
//     filters but no spec entry matched it.
//
// EXTRA detection is opt-in via cfg.MatchRouteFilters; without filters the
// tool would flag every route in the entire miniblue server as EXTRA.
func matchAndAnnotate(cl *checklist, cfg *serviceConfig, routes []routeRecord) {
	type routeKey struct {
		Method      string
		NormPattern string
	}
	routeIndex := make(map[routeKey]string, len(routes))
	for _, r := range routes {
		k := routeKey{Method: r.Method, NormPattern: normalizePattern(r.Pattern)}
		// Keep the first inserted pattern for stable diffs.
		if _, ok := routeIndex[k]; !ok {
			routeIndex[k] = r.Pattern
		}
	}

	usedRoutes := make(map[routeKey]bool, len(routes))
	for i := range cl.Sections {
		sec := &cl.Sections[i]
		for j := range sec.Entries {
			e := &sec.Entries[j]
			k := routeKey{Method: e.Method, NormPattern: normalizePattern(e.MinibluePath)}
			if pattern, ok := routeIndex[k]; ok {
				e.Status = statusImplemented
				e.MatchedRoute = pattern
				usedRoutes[k] = true
			} else {
				e.Status = statusMissing
			}
		}
	}

	if len(cfg.MatchRouteFilters) == 0 {
		return
	}
	filters := make([]string, 0, len(cfg.MatchRouteFilters))
	for _, f := range cfg.MatchRouteFilters {
		filters = append(filters, normalizePattern(f))
	}

	// Bucket extra routes into the section whose prefix they match best.
	// In practice, ARM routes go into the ARM section and data-plane
	// routes go into the data-plane section. If nothing matches we drop
	// them into the first section to avoid losing them silently.
	for _, r := range routes {
		k := routeKey{Method: r.Method, NormPattern: normalizePattern(r.Pattern)}
		if usedRoutes[k] {
			continue
		}
		norm := k.NormPattern
		inScope := false
		for _, f := range filters {
			if strings.Contains(norm, f) {
				inScope = true
				break
			}
		}
		if !inScope {
			continue
		}
		dst := pickSectionForRoute(cl, norm)
		dst.ExtraRoutes = append(dst.ExtraRoutes, extraRoute(r))
	}
}

// pickSectionForRoute returns the section whose MinibluePathPrefix is the
// longest case-insensitive prefix of norm. Falls back to the first section.
func pickSectionForRoute(cl *checklist, norm string) *checklistSection {
	var best *checklistSection
	bestLen := -1
	for i := range cl.Sections {
		sec := &cl.Sections[i]
		var prefix string
		if sec.MinibluePathPrefix != "" {
			prefix = normalizePattern(sec.MinibluePathPrefix)
		} else {
			// ARM sections have no explicit prefix; match by RP namespace.
			prefix = "/providers/" + strings.ToLower(cl.RPNamespace)
		}
		if strings.Contains(norm, prefix) && len(prefix) > bestLen {
			best = sec
			bestLen = len(prefix)
		}
	}
	if best == nil && len(cl.Sections) > 0 {
		best = &cl.Sections[0]
	}
	return best
}
