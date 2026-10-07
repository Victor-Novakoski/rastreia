package server

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/Victor-Novakoski/rastreia/api"
	"github.com/Victor-Novakoski/rastreia/internal/delivery"
	"github.com/Victor-Novakoski/rastreia/internal/push"
)

// TestOpenAPI keeps api/openapi.yaml in step with the router: every route
// served is documented and every documented route is served. It also checks
// that every $ref points to something and that no value is null, which is
// what an unquoted comma in a flow mapping ({ description: a, b }) leaves.
func TestOpenAPI(t *testing.T) {
	var spec map[string]any
	require.NoError(t, yaml.Unmarshal(api.OpenAPI, &spec))

	documented := map[string]bool{}
	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok, "paths")
	for path, item := range paths {
		for method := range item.(map[string]any) {
			if method != "parameters" {
				documented[strings.ToUpper(method)+" "+path] = true
			}
		}
	}

	d := testDeps(Options{})
	d.Live, d.Push = &delivery.LiveHandler{}, &push.Handler{} // only their routes are walked
	h, ok := New(d).(chi.Routes)
	require.True(t, ok)
	served := map[string]bool{}
	err := chi.Walk(h, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		served[method+" "+route] = true
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, slices.Sorted(maps.Keys(served)), slices.Sorted(maps.Keys(documented)))

	for _, problem := range checkSpec(spec, "", spec) {
		t.Error(problem)
	}
}

func checkSpec(root map[string]any, at string, node any) []string {
	var problems []string
	switch n := node.(type) {
	case nil:
		problems = append(problems, fmt.Sprintf("%s: null value (unquoted comma?)", at))
	case map[string]any:
		for k, v := range n {
			if k == "$ref" {
				if !resolves(root, v) {
					problems = append(problems, fmt.Sprintf("%s: $ref %v points nowhere", at, v))
				}
				continue
			}
			problems = append(problems, checkSpec(root, at+"/"+k, v)...)
		}
	case []any:
		for i, v := range n {
			problems = append(problems, checkSpec(root, fmt.Sprintf("%s[%d]", at, i), v)...)
		}
	}
	return problems
}

func resolves(root map[string]any, ref any) bool {
	s, ok := ref.(string)
	if !ok || !strings.HasPrefix(s, "#/") {
		return false
	}
	var node any = root
	for _, part := range strings.Split(strings.TrimPrefix(s, "#/"), "/") {
		m, ok := node.(map[string]any)
		if !ok {
			return false
		}
		if node, ok = m[part]; !ok {
			return false
		}
	}
	return true
}
