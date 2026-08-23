package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetConversationFound(t *testing.T) {
	d := testDeps(t)
	e := New(d)

	// seed one conversation with a message
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader([]byte(`{"title":"hello world"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("seed POST status %d: %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id, err := d.Store.CreateConversation("another")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Store.AddMessage(id, "user", "hi"); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+id, nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Conversation struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"conversation"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Conversation.ID != id || body.Conversation.Title != "another" {
		t.Fatalf("unexpected conversation: %+v", body.Conversation)
	}
	if len(body.Messages) != 1 || body.Messages[0].Content != "hi" {
		t.Fatalf("unexpected messages: %+v", body.Messages)
	}
}

func TestGetConversationNotFound(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/nope", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateConversationRejectsEmptyTitle(t *testing.T) {
	d := testDeps(t)
	e := New(d)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader([]byte(`{"title":""}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty title, got %d", rec.Code)
	}
}

func TestConversationsStoreError(t *testing.T) {
	d := testDeps(t)
	if err := d.Store.Close(); err != nil {
		t.Fatal(err)
	}
	e := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("list on closed store: expected 500, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader([]byte(`{"title":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("create on closed store: expected 500, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/conversations/abc", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("get on closed store: expected 500, got %d", rec.Code)
	}
}
