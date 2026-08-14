package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// profileStub serves /user and /user/emails with canned bodies, recording
// the Authorization header so tests can assert the token was sent.
type profileStub struct {
	mu   sync.Mutex
	auth []string

	userStatus   int
	userBody     string
	emailsStatus int
	emailsBody   string
	scopes       string
}

func (s *profileStub) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.auth = append(s.auth, r.Header.Get("Authorization"))
		s.mu.Unlock()
		switch r.URL.Path {
		case "/user":
			w.Header().Set("X-OAuth-Scopes", s.scopes)
			w.WriteHeader(s.userStatus)
			_, _ = w.Write([]byte(s.userBody))
		case "/user/emails":
			w.WriteHeader(s.emailsStatus)
			_, _ = w.Write([]byte(s.emailsBody))
		default:
			http.NotFound(w, r)
		}
	}
}

func newStubClient(t *testing.T, s *profileStub) *Client {
	t.Helper()
	ts := httptest.NewServer(s.handler())
	t.Cleanup(ts.Close)
	return New(ts.Client()).WithBaseURL(ts.URL)
}

const (
	userJSON   = `{"login":"octo","name":"Octo Cat","avatar_url":"https://avatars/u/1","html_url":"https://github.com/octo","created_at":"2018-05-01T10:00:00Z","updated_at":"2026-08-01T10:00:00Z"}`
	emailsJSON = `[{"email":"octo@example.com","primary":true,"verified":true},
	              {"email":"noreply@users.noreply.github.com","primary":false,"verified":true}]`
)

func TestFetchProfileSuccess(t *testing.T) {
	s := &profileStub{
		userStatus: http.StatusOK, userBody: userJSON,
		emailsStatus: http.StatusOK, emailsBody: emailsJSON,
		scopes: "repo,user",
	}
	c := newStubClient(t, s)

	p, err := c.FetchProfile(context.Background(), "ghp_secret")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	want := Profile{
		Username: "octo", DisplayName: "Octo Cat", Email: "octo@example.com",
		AvatarURL: "https://avatars/u/1", ProfileURL: "https://github.com/octo",
		Scopes:           "repo,user",
		AccountCreatedAt: "2018-05-01T10:00:00Z", AccountUpdatedAt: "2026-08-01T10:00:00Z",
	}
	if p != want {
		t.Fatalf("profile = %+v, want %+v", p, want)
	}
	for _, a := range s.auth {
		if a != "Bearer ghp_secret" {
			t.Fatalf("Authorization header = %q, want the bearer token", a)
		}
	}
}

func TestFetchProfilePrefersVerifiedPrimaryEmail(t *testing.T) {
	// Two primary entries: the unverified one comes first, the verified one
	// must win.
	body := `[{"email":"old@example.com","primary":true,"verified":false},
	          {"email":"new@example.com","primary":true,"verified":true}]`
	s := &profileStub{userStatus: http.StatusOK, userBody: userJSON, emailsStatus: http.StatusOK, emailsBody: body}
	p, err := newStubClient(t, s).FetchProfile(context.Background(), "t")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if p.Email != "new@example.com" {
		t.Fatalf("email = %q, want the verified primary", p.Email)
	}
}

func TestFetchProfileOptionalFieldsEmpty(t *testing.T) {
	s := &profileStub{
		userStatus:   http.StatusOK,
		userBody:     `{"login":"octo","name":null,"avatar_url":null}`,
		emailsStatus: http.StatusOK,
		emailsBody:   emailsJSON,
	}
	p, err := newStubClient(t, s).FetchProfile(context.Background(), "t")
	if err != nil {
		t.Fatalf("FetchProfile: %v", err)
	}
	if p.DisplayName != "" || p.AvatarURL != "" {
		t.Fatalf("optional fields must fall back to empty: %+v", p)
	}
}

func TestFetchProfileEmailsBestEffort(t *testing.T) {
	// The /user/emails call failing must not fail the connect: identity is
	// complete without the email.
	for name, s := range map[string]*profileStub{
		"server error": {userStatus: http.StatusOK, userBody: userJSON, emailsStatus: http.StatusInternalServerError},
		"malformed":    {userStatus: http.StatusOK, userBody: userJSON, emailsStatus: http.StatusOK, emailsBody: "oops"},
		"empty list":   {userStatus: http.StatusOK, userBody: userJSON, emailsStatus: http.StatusOK, emailsBody: `[]`},
		"no primary":   {userStatus: http.StatusOK, userBody: userJSON, emailsStatus: http.StatusOK, emailsBody: `[{"email":"a@b.c","primary":false}]`},
	} {
		t.Run(name, func(t *testing.T) {
			p, err := newStubClient(t, s).FetchProfile(context.Background(), "t")
			if err != nil {
				t.Fatalf("FetchProfile: %v", err)
			}
			if p.Email != "" || p.Username != "octo" {
				t.Fatalf("profile = %+v, want email empty and username intact", p)
			}
		})
	}
}

func TestFetchProfileRejectsToken(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			s := &profileStub{userStatus: status}
			_, err := newStubClient(t, s).FetchProfile(context.Background(), "t")
			if !errors.Is(err, ErrTokenRejected) {
				t.Fatalf("err = %v, want ErrTokenRejected", err)
			}
		})
	}
	// A token rejected on the emails call fails the connect too.
	s := &profileStub{
		userStatus: http.StatusOK, userBody: userJSON,
		emailsStatus: http.StatusUnauthorized,
	}
	if _, err := newStubClient(t, s).FetchProfile(context.Background(), "t"); !errors.Is(err, ErrTokenRejected) {
		t.Fatalf("emails 401: err = %v, want ErrTokenRejected", err)
	}
}

func TestFetchProfileServerError(t *testing.T) {
	s := &profileStub{userStatus: http.StatusInternalServerError}
	_, err := newStubClient(t, s).FetchProfile(context.Background(), "t")
	if err == nil || errors.Is(err, ErrTokenRejected) || !strings.Contains(err.Error(), "status 500") {
		t.Fatalf("err = %v, want a 500 status error", err)
	}
}

func TestFetchProfileMalformedUserBody(t *testing.T) {
	s := &profileStub{userStatus: http.StatusOK, userBody: "oops"}
	_, err := newStubClient(t, s).FetchProfile(context.Background(), "t")
	if err == nil || !strings.Contains(err.Error(), "unexpected response") {
		t.Fatalf("err = %v, want unexpected-response error", err)
	}
}

func TestFetchProfileNetworkErrorIsSanitized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := ts.URL
	ts.Close()
	_, err := New(ts.Client()).WithBaseURL(url).FetchProfile(context.Background(), "ghp_superSECRET123")
	if err == nil {
		t.Fatal("expected an error against a closed server")
	}
	if strings.Contains(err.Error(), url) || strings.Contains(err.Error(), "ghp_superSECRET123") {
		t.Fatalf("error must not echo the URL or token: %v", err)
	}
	if !strings.Contains(err.Error(), "could not reach github") {
		t.Fatalf("err = %v, want the fixed sanitized message", err)
	}
}

func TestFetchProfileContextDeadline(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(userJSON))
	}))
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := New(ts.Client()).WithBaseURL(ts.URL).FetchProfile(ctx, "t")
	if err == nil {
		t.Fatal("expected a deadline error")
	}
	if !strings.Contains(err.Error(), "could not reach github") {
		t.Fatalf("err = %v, want the fixed sanitized message", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("deadline not honored: took %v", time.Since(start))
	}
}
