package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// serviceConfig describes one miniblue service whose Azure REST spec we want
// to track. One config file per service lives under tools/specport/specs/.
type serviceConfig struct {
	// Service is the short slug used on the CLI and as the checklist
	// filename (e.g. "keyvault").
	Service string `yaml:"service"`

	// DisplayName is the human-readable name shown in the checklist header
	// (e.g. "Key Vault").
	DisplayName string `yaml:"display_name"`

	// RPNamespace is the ARM Resource Provider Namespace
	// (e.g. "Microsoft.KeyVault"). Used to filter which chi routes count
	// as "in scope" for this service when computing EXTRA markers.
	RPNamespace string `yaml:"rp_namespace"`

	// Sources is the list of Azure spec files to fetch and parse.
	Sources []specSource `yaml:"sources"`

	// MatchRouteFilters is an optional list of substrings (matched
	// case-insensitively against the chi route pattern) that determine
	// which routes are "in scope" for EXTRA detection. If empty, only
	// MISSING and IMPLEMENTED are reported.
	MatchRouteFilters []string `yaml:"match_route_filters,omitempty"`
}

// specSource is one Azure spec file (one swagger 2.0 JSON document).
// Multiple sources are allowed when a service has both an ARM and a
// data-plane surface (e.g. Key Vault).
type specSource struct {
	// Name is a short identifier for this source, shown in checklist
	// section headings (e.g. "arm-management", "data-plane-secrets").
	Name string `yaml:"name"`

	// Plane is "arm" or "data-plane". Used purely for grouping output.
	Plane string `yaml:"plane"`

	// Description is rendered as a one-liner under the section heading.
	Description string `yaml:"description,omitempty"`

	// URL is the raw JSON URL of the swagger 2.0 spec.
	URL string `yaml:"url"`

	// MinibluePathPrefix is prepended to every spec path before structural
	// matching. Required for data-plane specs whose original paths are
	// rooted at a vanity host (e.g. "{vaultBaseUrl}/secrets") that
	// miniblue routes under "/keyvault/{vaultName}/secrets".
	MinibluePathPrefix string `yaml:"miniblue_path_prefix,omitempty"`
}

func loadServiceConfig(specDir, service string) (*serviceConfig, error) {
	path := filepath.Join(specDir, service+".yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var c serviceConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Service == "" {
		return nil, fmt.Errorf("%s: service field is required", path)
	}
	if c.Service != service {
		return nil, fmt.Errorf("%s: service field %q does not match filename", path, c.Service)
	}
	if len(c.Sources) == 0 {
		return nil, fmt.Errorf("%s: at least one source is required", path)
	}
	for i, s := range c.Sources {
		if s.URL == "" {
			return nil, fmt.Errorf("%s: sources[%d].url is required", path, i)
		}
		if s.Name == "" {
			return nil, fmt.Errorf("%s: sources[%d].name is required", path, i)
		}
		switch s.Plane {
		case "arm", "data-plane":
		default:
			return nil, fmt.Errorf("%s: sources[%d].plane must be \"arm\" or \"data-plane\" (got %q)", path, i, s.Plane)
		}
	}
	return &c, nil
}

// listServiceConfigs returns the slugs of every *.yaml file in specDir,
// sorted, suitable for display in `specport list`.
func listServiceConfigs(specDir string) ([]string, error) {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", specDir, err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		out = append(out, strings.TrimSuffix(name, ".yaml"))
	}
	sort.Strings(out)
	return out, nil
}
