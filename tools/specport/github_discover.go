package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v3"
)

// This file implements discovery against the live Azure/azure-rest-api-specs
// repository using GitHub's Git Trees API (recursive). This keeps discovery
// to ~2 HTTP requests regardless of repo size and makes `-limit` fast.

const (
	azureSpecsOwner = "Azure"
	azureSpecsRepo  = "azure-rest-api-specs"
	azureSpecsRef   = "main"
)

type ghRefResponse struct {
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

type ghTreeResponse struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"` // "blob" or "tree"
	} `json:"tree"`
}

type azureSpecsTree struct {
	Files []string
}

// discoveredService is a single candidate the user can init into a spec yaml.
// DiscoveredID is stable text used for selection: "arm:<path>" or "dp:<path>".
type discoveredService struct {
	DiscoveredID string
	Plane        string // "arm" or "data-plane"
	Org          string
	RPNamespace  string // ARM only
	Service      string // folder name
	StableURL    string // raw URL to a swagger 2.0 JSON file (openapi.json, *.json)
}

func fetchAzureSpecsTree() (*azureSpecsTree, error) {
	// 1) Resolve main SHA
	refURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs/heads/%s",
		azureSpecsOwner, azureSpecsRepo, azureSpecsRef)
	refBody, err := ghGet(refURL)
	if err != nil {
		return nil, err
	}
	var ref ghRefResponse
	if err := json.Unmarshal(refBody, &ref); err != nil {
		return nil, fmt.Errorf("decode %s: %w", refURL, err)
	}
	if ref.Object.SHA == "" {
		return nil, fmt.Errorf("no sha in %s response", refURL)
	}

	// 2) Fetch entire tree recursively (single large response)
	treeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1",
		azureSpecsOwner, azureSpecsRepo, ref.Object.SHA)
	treeBody, err := ghGet(treeURL)
	if err != nil {
		return nil, err
	}
	var tree ghTreeResponse
	if err := json.Unmarshal(treeBody, &tree); err != nil {
		return nil, fmt.Errorf("decode %s: %w", treeURL, err)
	}

	files := make([]string, 0, len(tree.Tree))
	for _, it := range tree.Tree {
		if it.Type != "blob" {
			continue
		}
		// Only consider the main specs folder; this also filters out repo tooling.
		if strings.HasPrefix(it.Path, "specification/") {
			files = append(files, it.Path)
		}
	}
	sort.Strings(files)
	return &azureSpecsTree{Files: files}, nil
}

func discoverFromTree(t *azureSpecsTree) []discoveredService {
	// Build candidates keyed by (plane, org, rpns, service) → best version + best swagger file.
	type key struct {
		plane string
		org   string
		rpns  string
		svc   string
	}
	type best struct {
		version string
		path    string // full repo path to swagger json
	}
	bestBy := map[key]best{}

	// Helper to consider a swagger file and keep the best one.
	consider := func(k key, version string, filePath string) {
		cur, ok := bestBy[k]
		if !ok || version > cur.version {
			bestBy[k] = best{version: version, path: filePath}
			return
		}
		if version < cur.version {
			return
		}
		// Same version: prefer openapi.json, else keep first by lexicographic order.
		curBase := strings.ToLower(pathBase(cur.path))
		newBase := strings.ToLower(pathBase(filePath))
		if newBase == "openapi.json" && curBase != "openapi.json" {
			bestBy[k] = best{version: version, path: filePath}
			return
		}
		if curBase != "openapi.json" && newBase != "openapi.json" && filePath < cur.path {
			bestBy[k] = best{version: version, path: filePath}
		}
	}

	for _, p := range t.Files {
		lp := strings.ToLower(p)
		if !strings.HasSuffix(lp, ".json") {
			continue
		}
		if strings.Contains(lp, "/examples/") {
			continue
		}

		parts := strings.Split(p, "/")
		// ARM: specification/<org>/resource-manager/<RPNS>/<service>/stable/<ver>/<file>.json
		if len(parts) >= 8 && parts[0] == "specification" && parts[2] == "resource-manager" {
			org := parts[1]
			rpns := parts[3]
			svc := parts[4]
			if parts[5] != "stable" {
				continue
			}
			version := parts[6]
			k := key{plane: "arm", org: org, rpns: rpns, svc: svc}
			consider(k, version, p)
			continue
		}

		// Data-plane: specification/<org>/data-plane/<service>/stable/<ver>/<file>.json
		if len(parts) >= 7 && parts[0] == "specification" && parts[2] == "data-plane" {
			org := parts[1]
			svc := parts[3]
			if parts[4] != "stable" {
				continue
			}
			version := parts[5]
			k := key{plane: "data-plane", org: org, svc: svc}
			consider(k, version, p)
			continue
		}
	}

	out := make([]discoveredService, 0, len(bestBy))
	for k, b := range bestBy {
		var id string
		var rpns string
		if k.plane == "arm" {
			id = "arm:" + strings.Join([]string{"specification", k.org, "resource-manager", k.rpns, k.svc}, "/")
			rpns = k.rpns
		} else {
			id = "dp:" + strings.Join([]string{"specification", k.org, "data-plane", k.svc}, "/")
		}
		out = append(out, discoveredService{
			DiscoveredID: id,
			Plane:        k.plane,
			Org:          k.org,
			RPNamespace:  rpns,
			Service:      k.svc,
			StableURL:    rawURL(b.path),
		})
	}
	return out
}

func rawURL(repoPath string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", azureSpecsOwner, azureSpecsRepo, azureSpecsRef, repoPath)
}

func pathBase(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

func cmdDiscover(args []string) error {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	orgFilter := fs.String("org", "", "limit to one Azure org under specification/ (e.g. keyvault)")
	plane := fs.String("plane", "all", "arm | data-plane | all")
	limit := fs.Int("limit", 200, "max services to print (0 = no limit)")
	format := fs.String("format", "table", "output format: table | tsv")
	cachePath := fs.String("cache", defaultDiscoverCachePath(), "path to discovery cache JSON file")
	maxAge := fs.Duration("max-age", 24*time.Hour, "max cache age before re-fetching from GitHub")
	noCache := fs.Bool("no-cache", false, "do not read or write cache; always query GitHub")
	if err := fs.Parse(reorderFlagArgs(args)); err != nil {
		return err
	}

	var wantARM, wantDP bool
	switch strings.ToLower(*plane) {
	case "arm":
		wantARM, wantDP = true, false
	case "data-plane", "dataplane", "dp":
		wantARM, wantDP = false, true
	case "all":
		wantARM, wantDP = true, true
	default:
		return fmt.Errorf("invalid -plane %q (want arm|data-plane|all)", *plane)
	}

	var all []discoveredService
	if !*noCache {
		if cached, ok := loadDiscoverCache(*cachePath, *maxAge); ok {
			all = cached
		}
	}
	if all == nil {
		// Cache miss (or disabled): fetch the tree once and build results locally.
		tree, err := fetchAzureSpecsTree()
		if err != nil {
			return err
		}
		all = discoverFromTree(tree)
		if !*noCache {
			_ = saveDiscoverCache(*cachePath, all)
		}
	}
	all = filterDiscovered(all, wantARM, wantDP, *orgFilter)

	sort.Slice(all, func(i, j int) bool {
		if all[i].Plane != all[j].Plane {
			return all[i].Plane < all[j].Plane
		}
		if all[i].Org != all[j].Org {
			return all[i].Org < all[j].Org
		}
		if all[i].RPNamespace != all[j].RPNamespace {
			return all[i].RPNamespace < all[j].RPNamespace
		}
		return all[i].Service < all[j].Service
	})

	if *limit > 0 && len(all) > *limit {
		all = all[:*limit]
	}
	switch strings.ToLower(*format) {
	case "tsv":
		for _, s := range all {
			// discovered_id plane org rpns service stable_url
			fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n",
				s.DiscoveredID, s.Plane, s.Org, emptyDash(s.RPNamespace), s.Service, s.StableURL)
		}
	case "table":
		// Column-aligned table; still copy/paste-friendly.
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "discovered_id\tplane\torg\trp_namespace\tservice\tstable_url")
		fmt.Fprintln(w, "------------\t-----\t---\t------------\t-------\t---------")
		for _, s := range all {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				s.DiscoveredID, s.Plane, s.Org, emptyDash(s.RPNamespace), s.Service, s.StableURL)
		}
		_ = w.Flush()
	default:
		return fmt.Errorf("invalid -format %q (want table|tsv)", *format)
	}
	return nil
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func defaultDiscoverCachePath() string {
	return filepath.Join("tools", "specport", ".cache", "azure-rest-api-specs.discovery.json")
}

type discoverCacheFile struct {
	GeneratedAt string              `json:"generatedAt"`
	Services    []discoveredService `json:"services"`
}

func loadDiscoverCache(path string, maxAge time.Duration) ([]discoveredService, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if maxAge > 0 && time.Since(info.ModTime()) > maxAge {
		return nil, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var f discoverCacheFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, false
	}
	if len(f.Services) == 0 {
		return nil, false
	}
	return f.Services, true
}

func saveDiscoverCache(path string, services []discoveredService) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f := discoverCacheFile{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Services:    services,
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func filterDiscovered(in []discoveredService, wantARM, wantDP bool, orgFilter string) []discoveredService {
	out := make([]discoveredService, 0, len(in))
	for _, s := range in {
		if orgFilter != "" && s.Org != orgFilter {
			continue
		}
		if s.Plane == "arm" && !wantARM {
			continue
		}
		if s.Plane == "data-plane" && !wantDP {
			continue
		}
		out = append(out, s)
	}
	return out
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	specDir := fs.String("spec-dir", defaultSpecDir, "directory holding service config files")
	from := fs.String("from", "", "discovered_id from `specport discover` output")
	fromURL := fs.String("url", "", "raw swagger JSON URL (alternative to --from; useful when rate-limited)")
	fromPlane := fs.String("plane", "", "arm | data-plane (required with --url)")
	fromRPNS := fs.String("rp-namespace", "", "ARM Resource Provider Namespace (optional; only with --url)")
	displayName := fs.String("display-name", "", "override display_name in generated YAML")
	routeFilter := fs.String("route-filter", "", "comma-separated extra route filter substrings (optional)")
	minibluePrefix := fs.String("miniblue-prefix", "", "miniblue path prefix for data-plane sources (optional)")
	cachePath := fs.String("cache", defaultDiscoverCachePath(), "path to discovery cache JSON file (used to avoid GitHub API calls)")
	if err := fs.Parse(reorderFlagArgs(args)); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("init takes exactly one slug argument (got %d args)", fs.NArg())
	}
	slug := fs.Arg(0)

	var ds discoveredService
	switch {
	case *from != "":
		var err error
		ds, err = resolveDiscoveredWithCache(*from, *cachePath)
		if err != nil {
			return err
		}
	case *fromURL != "":
		if *fromPlane == "" {
			return fmt.Errorf("--plane is required when using --url")
		}
		p := strings.ToLower(*fromPlane)
		if p != "arm" && p != "data-plane" {
			return fmt.Errorf("invalid --plane %q (want arm|data-plane)", *fromPlane)
		}
		ds = discoveredService{
			DiscoveredID: "url:" + *fromURL,
			Plane:        p,
			RPNamespace:  *fromRPNS,
			Service:      slug,
			StableURL:    *fromURL,
		}
	default:
		return fmt.Errorf("init requires either --from <discovered_id> or --url <raw-swagger-json-url>")
	}

	cfg := serviceConfig{
		Service:     slug,
		DisplayName: firstNonEmpty(*displayName, titleCase(slug)),
		RPNamespace: ds.RPNamespace,
		Sources: []specSource{
			{
				Name:        defaultSourceName(ds),
				Plane:       ds.Plane,
				Description: defaultDescription(ds),
				URL:         ds.StableURL,
			},
		},
	}
	if ds.Plane == "data-plane" {
		cfg.Sources[0].MinibluePathPrefix = *minibluePrefix
	}

	filters := []string{}
	if ds.Plane == "arm" && ds.RPNamespace != "" {
		filters = append(filters, "/providers/"+ds.RPNamespace)
	}
	if ds.Plane == "data-plane" && cfg.Sources[0].MinibluePathPrefix != "" {
		filters = append(filters, cfg.Sources[0].MinibluePathPrefix)
	}
	if *routeFilter != "" {
		for _, p := range strings.Split(*routeFilter, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				filters = append(filters, p)
			}
		}
	}
	if len(filters) > 0 {
		cfg.MatchRouteFilters = filters
	}

	if err := os.MkdirAll(*specDir, 0o755); err != nil {
		return err
	}
	outPath := fmt.Sprintf("%s/%s.yaml", strings.TrimRight(*specDir, "/"), slug)
	b, err := yamlMarshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, b, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", outPath)
	return nil
}

func defaultSourceName(ds discoveredService) string {
	if ds.Plane == "arm" {
		return "arm-management"
	}
	return "data-plane"
}

func defaultDescription(ds discoveredService) string {
	if ds.Plane == "arm" && ds.RPNamespace != "" {
		return "ARM management plane for " + ds.RPNamespace + " (" + ds.Service + ")."
	}
	return ds.Plane + " surface (" + ds.Service + ")."
}

func titleCase(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}

// resolveDiscoveredWithCache resolves a discovered id into a stable swagger URL.
// It prefers the discovery cache to avoid GitHub API rate limits; if missing,
// it falls back to live GitHub queries (may require GITHUB_TOKEN).
func resolveDiscoveredWithCache(id string, cachePath string) (discoveredService, error) {
	if services, ok := loadDiscoverCache(cachePath, 0); ok {
		for _, s := range services {
			if s.DiscoveredID == id {
				return s, nil
			}
		}
	}
	// Cache miss: fetch tree and resolve via local discovery.
	tree, err := fetchAzureSpecsTree()
	if err != nil {
		return discoveredService{}, err
	}
	all := discoverFromTree(tree)
	for _, s := range all {
		if s.DiscoveredID == id {
			return s, nil
		}
	}
	return discoveredService{}, fmt.Errorf("unknown discovered_id %q (not found in tree)", id)
}

func ghGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 45 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "miniblue-specport/1")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET %s: status %d (%s)", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(resp.Body)
}

// yamlMarshal keeps YAML generation localized to this file so we don't need to
// thread yaml as a dependency through the discover/init logic.
func yamlMarshal(v interface{}) ([]byte, error) {
	out, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	// Ensure a trailing newline (helps keep diffs stable).
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	return out, nil
}

