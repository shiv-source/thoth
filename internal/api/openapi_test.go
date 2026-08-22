package api

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// openAPISpec mirrors the parts of the embedded OpenAPI document the tests
// assert against: the version, the info block, and the paths map.
type openAPISpec struct {
	OpenAPI string `json:"openapi"`
	Info    struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	} `json:"info"`
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

func parseOpenAPISpec(t *testing.T) openAPISpec {
	t.Helper()
	sub, err := docsSub()
	if err != nil {
		t.Fatalf("docsSub: %v", err)
	}
	specBytes, err := fs.ReadFile(sub, openAPISpecPath)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec openAPISpec
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		t.Fatalf("openapi.json does not parse: %v", err)
	}
	return spec
}

// TestSwaggerRoutePresence guards against spec drift: every route registered
// in newServer must be documented in the embedded OpenAPI spec. A new endpoint
// without a spec entry fails here, so the reference cannot go stale.
func TestSwaggerRoutePresence(t *testing.T) {
	deps := testDeps(t)
	deps.ServeAPIDocs = true
	e, _ := newServer(deps)

	spec := parseOpenAPISpec(t)
	for _, r := range e.Routes() {
		path := r.Path
		// The SPA wildcard and the API reference viewer are not REST
		// endpoints — skip them. /swagger.json is documented in the spec.
		if path == "/*" || path == "/api/docs" || strings.HasPrefix(path, "/api/docs/swagger-ui/") {
			continue
		}
		// Echo path params are :id; OpenAPI uses {id}.
		openPath := echoPathToOpenAPI(path)
		method := strings.ToLower(r.Method)
		ops, ok := spec.Paths[openPath]
		if !ok {
			t.Errorf("route %s %s is not documented in openapi.json", r.Method, path)
			continue
		}
		if _, ok := ops[method]; !ok {
			t.Errorf("route %s %s has no %s operation in openapi.json", r.Method, path, method)
		}
	}
}

func echoPathToOpenAPI(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}

// TestSwaggerSpecWellFormed asserts the embedded spec is a valid OpenAPI 3.x
// document: the version, info block, and a non-empty paths map.
func TestSwaggerSpecWellFormed(t *testing.T) {
	spec := parseOpenAPISpec(t)
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		t.Fatalf("openapi version = %q, want 3.x", spec.OpenAPI)
	}
	if spec.Info.Title == "" || spec.Info.Version == "" {
		t.Fatalf("info block incomplete: %+v", spec.Info)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("spec has no paths")
	}
}

// TestSwaggerRouteGated asserts the API reference routes are registered only
// when ServeAPIDocs is true; when false they are absent entirely (not 404
// stubs), so normal serve never exposes them.
func TestSwaggerRouteGated(t *testing.T) {
	routes := []string{"/swagger.json", "/api/docs", "/api/docs/swagger-ui/*"}
	tests := []struct {
		name      string
		serveDocs bool
		wantRoute bool
	}{
		{"normal serve hides the docs", false, false},
		{"serve --dev exposes the docs", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := testDeps(t)
			deps.ServeAPIDocs = tt.serveDocs
			e, _ := newServer(deps)
			registered := map[string]bool{}
			for _, r := range e.Routes() {
				registered[r.Path] = true
			}
			for _, path := range routes {
				if registered[path] != tt.wantRoute {
					t.Fatalf("route %s registered = %v, want %v", path, registered[path], tt.wantRoute)
				}
			}
		})
	}
}

// TestSwaggerRouteServesSpec asserts the routes, when present, serve the
// embedded OpenAPI document and the viewer.
func TestSwaggerRouteServesSpec(t *testing.T) {
	deps := testDeps(t)
	deps.ServeAPIDocs = true
	e, _ := newServer(deps)

	for _, tt := range []struct {
		path       string
		wantStatus int
		wantCT     string
	}{
		{"/swagger.json", http.StatusOK, "application/json"},
		{"/api/docs", http.StatusOK, "text/html"},
		{"/api/docs/swagger-ui/swagger-ui.css", http.StatusOK, "text/css"},
		{"/api/docs/swagger-ui/swagger-ui-bundle.js", http.StatusOK, "application/javascript"},
		{"/api/docs/swagger-ui/NOTICE", http.StatusOK, "application/octet-stream"},
		{"/api/docs/swagger-ui/LICENSE", http.StatusOK, "application/octet-stream"},
		{"/api/docs/swagger-ui/nope.js", http.StatusNotFound, ""},
		{"/api/docs/swagger-ui/../openapi.json", http.StatusNotFound, ""},
	} {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCT != "" && !strings.HasPrefix(rec.Header().Get("Content-Type"), tt.wantCT) {
				t.Fatalf("content-type = %q, want prefix %q", rec.Header().Get("Content-Type"), tt.wantCT)
			}
			if tt.wantStatus == http.StatusOK && tt.path == "/swagger.json" && !json.Valid(rec.Body.Bytes()) {
				t.Fatalf("spec body is not valid JSON: %s", io.LimitReader(rec.Body, 64))
			}
		})
	}
}

// TestSwaggerViewerAssetsResolve guards against a viewer page whose asset
// references 404 — the relative-path bug where /api/docs (no trailing slash)
// resolved ./swagger-ui/... to /api/swagger-ui/.... Every src/href the page
// emits must be servable by the docs routes.
func TestSwaggerViewerAssetsResolve(t *testing.T) {
	deps := testDeps(t)
	deps.ServeAPIDocs = true
	e, _ := newServer(deps)

	indexReq := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
	indexRec := httptest.NewRecorder()
	e.ServeHTTP(indexRec, indexReq)
	if indexRec.Code != http.StatusOK {
		t.Fatalf("viewer status %d", indexRec.Code)
	}
	html := indexRec.Body.String()
	refs := append(regexp.MustCompile(`src="([^"]+)"`).FindAllStringSubmatch(html, -1),
		regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(html, -1)...)
	checked := 0
	for _, m := range refs {
		u := m[1]
		if strings.HasPrefix(u, "data:") || strings.HasPrefix(u, "#") {
			continue
		}
		req := httptest.NewRequest(http.MethodGet, u, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("viewer asset %s -> %d, want 200", u, rec.Code)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("viewer page references no resolvable assets")
	}
}
