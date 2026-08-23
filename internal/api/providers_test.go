package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/shiv-source/thoth/internal/settings"
)

// providerBody decodes a provider JSON body.
type providerBody struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	BaseURL    string `json:"base_url"`
	HasAPIKey  bool   `json:"has_api_key"`
	APIKey     string `json:"api_key"`
	ModelCount int    `json:"model_count"`
}

func decodeProviders(t *testing.T, rec *httptest.ResponseRecorder) []providerBody {
	t.Helper()
	var body struct {
		Providers []providerBody `json:"providers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode providers: %v (body %s)", err, rec.Body.String())
	}
	return body.Providers
}

func doProvidersReq(t *testing.T, e http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return doReq(t, e, method, path, body)
}

func TestProvidersListSortedWithCounts(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateProvider("Zeta", "", ""); err != nil {
		t.Fatal(err)
	}
	alpha, err := d.Store.CreateProvider("alpha", "https://alpha.example", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Store.CreateModel("a1", "A1", "", alpha.ID); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	providers := decodeProviders(t, doProvidersReq(t, e, http.MethodGet, "/api/v1/providers", ""))
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %+v", providers)
	}
	// Case-insensitive A→Z: alpha, Zeta.
	if providers[0].Name != "alpha" || providers[1].Name != "Zeta" {
		t.Fatalf("not sorted A→Z: %+v", providers)
	}
	if providers[0].BaseURL != "https://alpha.example" || providers[0].ModelCount != 1 {
		t.Fatalf("alpha mismatch: %+v", providers[0])
	}
	if providers[1].ModelCount != 0 {
		t.Fatalf("Zeta should have 0 models: %+v", providers[1])
	}
}

func TestProvidersCreate(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	rec := doProvidersReq(t, e, http.MethodPost, "/api/v1/providers",
		`{"name":"DeepSeek","base_url":"https://api.deepseek.com","api_key":"ds-secret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var created providerBody
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.Name != "DeepSeek" || created.BaseURL != "https://api.deepseek.com" || !created.HasAPIKey {
		t.Fatalf("created mismatch: %+v", created)
	}
}

func TestProvidersCreateValidation(t *testing.T) {
	e := New(testDeps(t))
	for _, body := range []string{`{}`, `{"name":""}`} {
		if rec := doProvidersReq(t, e, http.MethodPost, "/api/v1/providers", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: status %d, want 400", body, rec.Code)
		}
	}
}

func TestProvidersCreateDuplicate(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateProvider("DeepSeek", "", ""); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doProvidersReq(t, e, http.MethodPost, "/api/v1/providers", `{"name":"DeepSeek"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestProvidersUpdate(t *testing.T) {
	d := testDeps(t)
	p, err := d.Store.CreateProvider("DeepSeek", "https://api.deepseek.com", "ds-secret")
	if err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doProvidersReq(t, e, http.MethodPut, "/api/v1/providers/"+strconv.FormatInt(p.ID, 10),
		`{"name":"DeepSeek AI","base_url":"https://api.deepseek.ai","api_key":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var updated providerBody
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "DeepSeek AI" || updated.BaseURL != "https://api.deepseek.ai" || !updated.HasAPIKey {
		t.Fatalf("updated mismatch: %+v", updated)
	}
	// The empty api_key left the stored key untouched; the base_url cleared
	// back to the default endpoint.
	rec = doProvidersReq(t, e, http.MethodPut, "/api/v1/providers/"+strconv.FormatInt(p.ID, 10),
		`{"name":"DeepSeek AI","base_url":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("clear status %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.BaseURL != "" || !updated.HasAPIKey {
		t.Fatalf("clear mismatch: %+v", updated)
	}
	got, err := d.Store.Provider(p.ID)
	if err != nil || got.APIKey != "ds-secret" {
		t.Fatalf("stored key lost on empty PUT: %+v / %v", got, err)
	}
}

func TestProvidersUpdateErrors(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateProvider("Taken", "", ""); err != nil {
		t.Fatal(err)
	}
	p, err := d.Store.CreateProvider("Free", "", "")
	if err != nil {
		t.Fatal(err)
	}
	e := New(d)
	// Empty name.
	if rec := doProvidersReq(t, e, http.MethodPut, "/api/v1/providers/"+strconv.FormatInt(p.ID, 10),
		`{"name":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty name: %d, want 400", rec.Code)
	}
	// Unknown id.
	if rec := doProvidersReq(t, e, http.MethodPut, "/api/v1/providers/404", `{"name":"X"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown id: %d, want 404", rec.Code)
	}
	// Name collision.
	if rec := doProvidersReq(t, e, http.MethodPut, "/api/v1/providers/"+strconv.FormatInt(p.ID, 10),
		`{"name":"Taken"}`); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate name: %d, want 409", rec.Code)
	}
}

func TestProvidersKeyNeverEchoed(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateProvider("OpenAI", "", "sk-openai-secret"); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doProvidersReq(t, e, http.MethodGet, "/api/v1/providers", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "sk-openai-secret") {
		t.Fatalf("GET /api/v1/providers echoed the api key: %s", rec.Body.String())
	}
}

func TestProvidersDelete(t *testing.T) {
	d := testDeps(t)
	p, err := d.Store.CreateProvider("Doomed", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Store.CreateModel("m1", "M1", "", p.ID); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doProvidersReq(t, e, http.MethodDelete, "/api/v1/providers/"+strconv.FormatInt(p.ID, 10), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if providers := decodeProviders(t, doProvidersReq(t, e, http.MethodGet, "/api/v1/providers", "")); len(providers) != 0 {
		t.Fatalf("provider survived delete: %+v", providers)
	}
	models, err := d.Store.ListModels()
	if err != nil || len(models) != 0 {
		t.Fatalf("models not cascaded: %v %+v", err, models)
	}
}

func TestProvidersDeleteClearsSelectedModel(t *testing.T) {
	d := testDeps(t)
	p, err := d.Store.CreateProvider("Doomed", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Store.CreateModel("doomed-model", "Doomed", "", p.ID); err != nil {
		t.Fatal(err)
	}
	if err := d.Settings.SetSetting(settings.KeyModel, "doomed-model"); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doProvidersReq(t, e, http.MethodDelete, "/api/v1/providers/"+strconv.FormatInt(p.ID, 10), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	got, _, err := d.Settings.Setting(settings.KeyModel)
	if err != nil || got != "" {
		t.Fatalf("model setting = %q/%v, want cleared", got, err)
	}
}

func TestProvidersDeleteNotFound(t *testing.T) {
	e := New(testDeps(t))
	if rec := doProvidersReq(t, e, http.MethodDelete, "/api/v1/providers/404", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}
