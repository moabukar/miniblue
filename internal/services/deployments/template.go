package deployments

import (
	"regexp"
	"strings"
)

// flattenParameters takes the deployment's "parameters" object (which has
// shape {"name": {"value": ...}}) and returns {"name": value}.
func flattenParameters(raw interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	m, _ := raw.(map[string]interface{})
	for k, v := range m {
		entry, _ := v.(map[string]interface{})
		if entry == nil {
			continue
		}
		if val, ok := entry["value"]; ok {
			out[k] = val
		}
	}
	return out
}

// mergeParameterDefaults merges template.parameters defaultValue settings
// with the deployment's explicit parameter values. Explicit wins.
func mergeParameterDefaults(tplParams map[string]interface{}, supplied map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range tplParams {
		entry, _ := v.(map[string]interface{})
		if entry == nil {
			continue
		}
		if def, ok := entry["defaultValue"]; ok {
			out[k] = def
		}
	}
	for k, v := range supplied {
		out[k] = v
	}
	return out
}

// resolveExpressions walks a parsed JSON value and substitutes ARM template
// expressions of the form "[parameters('x')]" and "[variables('x')]" with
// their concrete values. Other expression functions (concat, resourceId,
// reference, copyIndex, etc.) are passed through unchanged in Phase 1; the
// template author should pre-resolve them or use literals.
func resolveExpressions(v interface{}, params, vars map[string]interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return resolveString(t, params, vars)
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, vv := range t {
			out[k] = resolveExpressions(vv, params, vars)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = resolveExpressions(item, params, vars)
		}
		return out
	default:
		return v
	}
}

var (
	// Whole-string parameter/variable references. Evaluated to the underlying
	// value (which may itself be any JSON type).
	wholeParam = regexp.MustCompile(`^\[parameters\('([^']+)'\)\]$`)
	wholeVar   = regexp.MustCompile(`^\[variables\('([^']+)'\)\]$`)
	// Embedded references inside a larger string (rare in well-formed templates
	// but Bicep can emit them via `${}` interpolation that lowers to concat).
	// We do NOT try to handle concat() here; we only inline whole-string forms.
)

func resolveString(s string, params, vars map[string]interface{}) interface{} {
	if m := wholeParam.FindStringSubmatch(s); m != nil {
		if v, ok := params[m[1]]; ok {
			return v
		}
		return s
	}
	if m := wholeVar.FindStringSubmatch(s); m != nil {
		if v, ok := vars[m[1]]; ok {
			return v
		}
		return s
	}
	// Pass through any other [...] expressions unchanged. A WARNING-style
	// trace could be added here if we wanted to flag unsupported functions.
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s
	}
	return s
}
