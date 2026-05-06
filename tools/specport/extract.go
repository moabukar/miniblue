package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cmdList prints the configured services.
func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	specDir := fs.String("spec-dir", defaultSpecDir, "directory holding service config files")
	if err := fs.Parse(args); err != nil {
		return err
	}
	names, err := listServiceConfigs(*specDir)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Fprintf(os.Stderr, "no service configs found in %s\n", *specDir)
		return nil
	}
	for _, n := range names {
		fmt.Println(n)
	}
	return nil
}

// cmdExtract fetches every spec source for a service and writes the
// checklist with all rows marked TODO. Use diff afterwards to populate
// IMPLEMENTED / MISSING / EXTRA.
func cmdExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	specDir := fs.String("spec-dir", defaultSpecDir, "directory holding service config files")
	outDir := fs.String("out-dir", defaultOutDir, "directory where checklists are written")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("extract takes exactly one service name (got %d args)", fs.NArg())
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
	return writeChecklist(*outDir, cl)
}

const (
	defaultSpecDir = "tools/specport/specs"
	defaultOutDir  = "tools/specport/checklists"
)

// buildChecklist downloads every source listed in cfg and returns a
// checklist with all entries marked TODO. Diff replaces TODO with the
// real status afterwards.
func buildChecklist(cfg *serviceConfig) (*checklist, error) {
	cl := &checklist{
		Service:     cfg.Service,
		DisplayName: cfg.DisplayName,
		RPNamespace: cfg.RPNamespace,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	for _, src := range cfg.Sources {
		body, err := fetchSpec(src.URL)
		if err != nil {
			return nil, fmt.Errorf("source %q: %w", src.Name, err)
		}
		ops, err := loadSwaggerOperations(body)
		if err != nil {
			return nil, fmt.Errorf("source %q: %w", src.Name, err)
		}
		section := checklistSection{
			Name:               src.Name,
			Plane:              src.Plane,
			Description:        src.Description,
			URL:                src.URL,
			MinibluePathPrefix: src.MinibluePathPrefix,
			Entries:            make([]checklistEntry, 0, len(ops)),
		}
		for _, op := range ops {
			section.Entries = append(section.Entries, checklistEntry{
				Method:        op.Method,
				SpecPath:      op.Path,
				MinibluePath:  applyPrefix(src.MinibluePathPrefix, op.Path),
				OperationID:   op.OperationID,
				LRO:           op.LRO,
				Pageable:      op.Pageable,
				Summary:       op.Summary,
				Status:        statusTodo,
			})
		}
		cl.Sections = append(cl.Sections, section)
	}
	return cl, nil
}

// applyPrefix joins a miniblue prefix with a swagger path. The swagger
// path is always rooted at "/", so the prefix is just stripped of any
// trailing slash before concatenation.
func applyPrefix(prefix, specPath string) string {
	if prefix == "" {
		return specPath
	}
	prefix = strings.TrimRight(prefix, "/")
	if !strings.HasPrefix(specPath, "/") {
		specPath = "/" + specPath
	}
	return prefix + specPath
}

// fetchSpec retrieves a swagger document from a URL or local file. Local
// paths help when iterating offline against a vendored copy of the
// azure-rest-api-specs tree.
func fetchSpec(src string) ([]byte, error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return httpGet(src)
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "miniblue-specport/1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
