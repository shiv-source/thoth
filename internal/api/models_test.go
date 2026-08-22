package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/shiv-source/thoth/internal/settings"
)

// modelBody decodes a model JSON body (group item or create/update response).
type modelBody struct {
	ID       int64  `json:"id"`
	Value    string `json:"value"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Provider string `json:"provider"`
}

// groupBody decodes one provider group of GET /api/v1/models.
type groupBody struct {
	Provider string      `json:"provider"`
	Models   []modelBody `json:"models"`
}

func doModelsRequest(t *testing.T, e http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func decodeGroups(t *testing.T, rec *httptest.ResponseRecorder) []groupBody {
	t.Helper()
	var body struct {
		Groups []groupBody `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode groups: %v (body %s)", err, rec.Body.String())
	}
	return body.Groups
}

func TestModelsListFromDB(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateModel("my-model", "My Model", "strongest", "Vendor"); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doModelsRequest(t, e, http.MethodGet, "/api/v1/models", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	groups := decodeGroups(t, rec)
	if len(groups) != 1 || groups[0].Provider != "Vendor" || len(groups[0].Models) != 1 {
		t.Fatalf("groups mismatch: %+v", groups)
	}
	got := groups[0].Models[0]
	if got.ID == 0 || got.Value != "my-model" || got.Name != "My Model" ||
		got.Tag != "strongest" || got.Provider != "Vendor" {
		t.Fatalf("model mismatch: %+v", got)
	}
}

func TestModelsGroupsSortedByProvider(t *testing.T) {
	d := testDeps(t)
	// Insert out of order, with mixed case, so sorting is observable.
	for _, m := range []struct{ value, provider string }{
		{"a", "DeepSeek"},
		{"b", "anthropic"},
		{"c", "Google"},
		{"d", "anthropic"},
	} {
		if _, err := d.Store.CreateModel(m.value, m.value, "", m.provider); err != nil {
			t.Fatal(err)
		}
	}
	e := New(d)
	groups := decodeGroups(t, doModelsRequest(t, e, http.MethodGet, "/api/v1/models", nil))
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %+v", groups)
	}
	// Case-insensitive A→Z: anthropic, DeepSeek, Google.
	if groups[0].Provider != "anthropic" || groups[1].Provider != "DeepSeek" || groups[2].Provider != "Google" {
		t.Fatalf("providers not sorted A→Z: %+v", groups)
	}
	// Models within a group keep insertion order.
	if len(groups[0].Models) != 2 || groups[0].Models[0].Value != "b" || groups[0].Models[1].Value != "d" {
		t.Fatalf("anthropic group order: %+v", groups[0].Models)
	}
}

func TestModelsCreate(t *testing.T) {
	e := New(testDeps(t))
	rec := doModelsRequest(t, e, http.MethodPost, "/api/v1/models", map[string]string{
		"value": "new-model", "name": "New Model", "tag": "test", "provider": "Vendor",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var created modelBody
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.ID == 0 || created.Value != "new-model" || created.Name != "New Model" || created.Tag != "test" {
		t.Fatalf("created mismatch: %+v", created)
	}
}

func TestModelsCreateValidation(t *testing.T) {
	e := New(testDeps(t))
	for _, body := range []map[string]string{
		{},
		{"value": "only-value"},
		{"name": "only-name"},
	} {
		rec := doModelsRequest(t, e, http.MethodPost, "/api/v1/models", body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %v: status %d, want 400", body, rec.Code)
		}
	}
}

func TestModelsCreateDuplicate(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateModel("dup", "First", "", ""); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doModelsRequest(t, e, http.MethodPost, "/api/v1/models", map[string]string{
		"value": "dup", "name": "Second",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, want 409: %s", rec.Code, rec.Body.String())
	}
}

func TestModelsUpdate(t *testing.T) {
	d := testDeps(t)
	m, err := d.Store.CreateModel("old-value", "Old", "before", "Vendor")
	if err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doModelsRequest(t, e, http.MethodPut, "/api/v1/models/"+strconv.FormatInt(m.ID, 10), map[string]string{
		"value": "new-value", "name": "New", "tag": "after", "provider": "Other",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	groups := decodeGroups(t, doModelsRequest(t, e, http.MethodGet, "/api/v1/models", nil))
	if len(groups) != 1 || len(groups[0].Models) != 1 {
		t.Fatalf("update mismatch: %+v", groups)
	}
	got := groups[0].Models[0]
	if got.Value != "new-value" || got.Name != "New" || got.Tag != "after" || got.Provider != "Other" {
		t.Fatalf("update mismatch: %+v", got)
	}
}

func TestModelsUpdateNotFound(t *testing.T) {
	e := New(testDeps(t))
	rec := doModelsRequest(t, e, http.MethodPut, "/api/v1/models/404", map[string]string{
		"value": "x", "name": "X",
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestModelsUpdateDuplicate(t *testing.T) {
	d := testDeps(t)
	if _, err := d.Store.CreateModel("taken", "Taken", "", ""); err != nil {
		t.Fatal(err)
	}
	m, err := d.Store.CreateModel("free", "Free", "", "")
	if err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doModelsRequest(t, e, http.MethodPut, "/api/v1/models/"+strconv.FormatInt(m.ID, 10), map[string]string{
		"value": "taken", "name": "Free",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d, want 409", rec.Code)
	}
}

func TestModelsUpdateRenamesSelectedModel(t *testing.T) {
	d := testDeps(t)
	m, err := d.Store.CreateModel("old-value", "Old", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Settings.SetSetting(settings.KeyModel, "old-value"); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doModelsRequest(t, e, http.MethodPut, "/api/v1/models/"+strconv.FormatInt(m.ID, 10), map[string]string{
		"value": "new-value", "name": "New",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	got, _, err := d.Settings.Setting(settings.KeyModel)
	if err != nil || got != "new-value" {
		t.Fatalf("model setting = %q/%v, want new-value", got, err)
	}
}

func TestModelsDelete(t *testing.T) {
	d := testDeps(t)
	m, err := d.Store.CreateModel("doomed", "Doomed", "", "")
	if err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doModelsRequest(t, e, http.MethodDelete, "/api/v1/models/"+strconv.FormatInt(m.ID, 10), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if groups := decodeGroups(t, doModelsRequest(t, e, http.MethodGet, "/api/v1/models", nil)); len(groups) != 0 {
		t.Fatalf("model survived delete: %+v", groups)
	}
}

func TestModelsDeleteNotFound(t *testing.T) {
	e := New(testDeps(t))
	rec := doModelsRequest(t, e, http.MethodDelete, "/api/v1/models/404", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestModelsDeleteClearsSelectedModel(t *testing.T) {
	d := testDeps(t)
	m, err := d.Store.CreateModel("doomed", "Doomed", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Settings.SetSetting(settings.KeyModel, "doomed"); err != nil {
		t.Fatal(err)
	}
	e := New(d)
	rec := doModelsRequest(t, e, http.MethodDelete, "/api/v1/models/"+strconv.FormatInt(m.ID, 10), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	got, _, err := d.Settings.Setting(settings.KeyModel)
	if err != nil || got != "" {
		t.Fatalf("model setting = %q/%v, want cleared", got, err)
	}
}
