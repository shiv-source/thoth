package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelsEndpoint(t *testing.T) {
	e := New(testDeps(t))
	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Models []struct {
			Value string `json:"value"`
			Label string `json:"label"`
		} `json:"models"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Models) < 2 {
		t.Fatalf("expected several models, got %+v", body.Models)
	}
	if body.Models[0].Value != "" {
		t.Fatalf("first model must be the empty default, got %q", body.Models[0].Value)
	}
}
