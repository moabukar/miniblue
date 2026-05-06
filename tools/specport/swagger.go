package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// swaggerDoc is a deliberately minimal Swagger 2.0 reader. Azure ships its
// REST specs as Swagger 2.0 (the file is sometimes named openapi.json but
// the document still says "swagger": "2.0"), so a strict OpenAPI 3 parser
// would refuse to load it. We only need a handful of fields to extract
// (method, path, operationId, LRO flag) — everything else is ignored to
// keep the reader resilient against schema drift across api-versions.
type swaggerDoc struct {
	Swagger  string                     `json:"swagger"`
	Info     swaggerInfo                `json:"info"`
	Host     string                     `json:"host"`
	BasePath string                     `json:"basePath"`
	// Paths is read as raw JSON because each path-item object can contain
	// non-method keys (parameters, x-ms-extensions) that we filter out
	// before decoding.
	Paths map[string]json.RawMessage `json:"paths"`
}

type swaggerInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// operationSpec captures only the per-operation fields specport cares about.
// Strings ($ref, parameter shapes, response schemas, …) are intentionally
// dropped: extracting routes does not require resolving any $ref.
type operationSpec struct {
	OperationID string `json:"operationId"`
	Description string `json:"description"`
	Summary     string `json:"summary"`
	// LongRunning is the standard Azure x-ms-long-running-operation flag.
	LongRunning bool `json:"x-ms-long-running-operation"`
	// Pageable is non-nil when the operation returns a paged collection.
	Pageable *pageableExtension `json:"x-ms-pageable,omitempty"`
}

type pageableExtension struct {
	NextLinkName string `json:"nextLinkName"`
}

// validHTTPMethods is the set of swagger keys we treat as operations.
// Other keys at the path-item level (parameters, x-ms-*) are skipped.
var validHTTPMethods = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"patch": true, "head": true, "options": true,
}

// operation is the per-row data the rest of specport works with.
// (Output and matching do not need any swagger schema details.)
type operation struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	OperationID string `json:"operationId"`
	LRO         bool   `json:"lro"`
	Pageable    bool   `json:"pageable"`
	Summary     string `json:"summary,omitempty"`
}

// loadSwaggerOperations downloads (or reads from disk) one swagger document
// and returns its operations. If src starts with http:// or https:// the
// document is fetched, otherwise it is treated as a local file path. This
// makes the tool usable both online and inside an air-gapped vendor cache.
func loadSwaggerOperations(rawJSON []byte) ([]operation, error) {
	var doc swaggerDoc
	if err := json.Unmarshal(rawJSON, &doc); err != nil {
		return nil, fmt.Errorf("decode swagger: %w", err)
	}
	if doc.Swagger == "" {
		// Some emitted docs name the file openapi.json but still declare
		// "swagger": "2.0". A missing swagger field probably means an
		// OpenAPI 3 document — refuse explicitly so we surface the gap
		// rather than silently emit zero operations.
		return nil, fmt.Errorf("document has no top-level \"swagger\" field; only Swagger 2.0 is supported (got info.title=%q version=%q)", doc.Info.Title, doc.Info.Version)
	}

	out := make([]operation, 0, 64)
	for path, raw := range doc.Paths {
		var pathItem map[string]json.RawMessage
		if err := json.Unmarshal(raw, &pathItem); err != nil {
			return nil, fmt.Errorf("path %q: %w", path, err)
		}
		for method, opRaw := range pathItem {
			lower := strings.ToLower(method)
			if !validHTTPMethods[lower] {
				continue
			}
			var op operationSpec
			if err := json.Unmarshal(opRaw, &op); err != nil {
				return nil, fmt.Errorf("%s %s: %w", strings.ToUpper(method), path, err)
			}
			out = append(out, operation{
				Method:      strings.ToUpper(lower),
				Path:        path,
				OperationID: op.OperationID,
				LRO:         op.LongRunning,
				Pageable:    op.Pageable != nil,
				Summary:     firstNonEmpty(op.Summary, op.Description),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
