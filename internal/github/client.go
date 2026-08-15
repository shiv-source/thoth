package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	profileTimeout = 10 * time.Second
	apiBase        = "https://api.github.com"
)

// ErrTokenRejected marks a 401/403 from api.github.com — the only failure the
// API layer surfaces as a client error (400).
var ErrTokenRejected = errors.New("github rejected the token")

// Profile is the token-free identity fetched from the GitHub API.
type Profile struct {
	Username         string
	DisplayName      string
	Email            string
	AvatarURL        string
	ProfileURL       string
	Scopes           string
	AccountCreatedAt string
	AccountUpdatedAt string
}

// Repository is one repository from /user/repos, reduced to what the
// settings UI needs: the full name to display and the clone URL to use as
// the remote.
type Repository struct {
	FullName string `json:"full_name"`
	CloneURL string `json:"clone_url"`
}

// Client fetches GitHub identity for a token. Errors are fixed messages:
// raw transport errors embed the request URL and must never propagate.
type Client struct {
	hc      *http.Client
	baseURL string
}

func New(hc *http.Client) *Client { return &Client{hc: hc, baseURL: apiBase} }

// WithBaseURL overrides the API base (default api.github.com) — needed for
// GitHub Enterprise hosts and used by tests to point at a stub server.
func (c *Client) WithBaseURL(u string) *Client {
	c.baseURL = u
	return c
}

// FetchProfile resolves the identity behind token: GET /user provides the
// username, display name, and avatar (required); GET /user/emails provides
// the primary verified email (best-effort — /user's own email field is often
// null). Scopes come from the X-OAuth-Scopes header.
func (c *Client) FetchProfile(ctx context.Context, token string) (Profile, error) {
	if c.hc == nil {
		return Profile{}, errors.New("github client not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, profileTimeout)
	defer cancel()

	user, err := c.get(ctx, "/user", token)
	if err != nil {
		return Profile{}, err
	}
	var raw struct {
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.Unmarshal(user.body, &raw); err != nil {
		return Profile{}, errors.New("fetch github profile: github returned an unexpected response")
	}
	p := Profile{
		Username:         raw.Login,
		DisplayName:      raw.Name,
		AvatarURL:        raw.AvatarURL,
		ProfileURL:       raw.HTMLURL,
		Scopes:           user.scopes,
		AccountCreatedAt: raw.CreatedAt,
		AccountUpdatedAt: raw.UpdatedAt,
	}

	emails, err := c.get(ctx, "/user/emails", token)
	switch {
	case errors.Is(err, ErrTokenRejected):
		return Profile{}, err
	case err != nil:
		return p, nil // best-effort: the identity is complete without the email
	}
	p.Email = primaryEmail(emails.body)
	return p, nil
}

// FetchRepos lists the user's repositories, most recently updated first.
// The token's scopes decide which repos appear (private ones need "repo").
func (c *Client) FetchRepos(ctx context.Context, token string) ([]Repository, error) {
	if c.hc == nil {
		return nil, errors.New("github client not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, profileTimeout)
	defer cancel()
	res, err := c.get(ctx, "/user/repos?per_page=100&sort=updated", token)
	if err != nil {
		return nil, err
	}
	var repos []Repository
	if err := json.Unmarshal(res.body, &repos); err != nil {
		return nil, errors.New("fetch github repos: github returned an unexpected response")
	}
	return repos, nil
}

type getResult struct {
	body   []byte
	scopes string
}

// get performs one authenticated GET. Raw transport errors never propagate
// (they embed the request URL); non-2xx returns only the status code.
func (c *Client) get(ctx context.Context, path, token string) (getResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return getResult{}, errors.New("fetch github profile: build request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return getResult{}, errors.New("fetch github profile: could not reach github")
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return getResult{}, fmt.Errorf("fetch github profile: %w", ErrTokenRejected)
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return getResult{}, fmt.Errorf("fetch github profile: github returned status %d", resp.StatusCode)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return getResult{}, errors.New("fetch github profile: could not read response")
	}
	return getResult{body: buf.Bytes(), scopes: resp.Header.Get("X-OAuth-Scopes")}, nil
}

// primaryEmail picks the primary address from /user/emails, preferring a
// verified entry when several are marked primary.
func primaryEmail(body []byte) string {
	var list []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return ""
	}
	var primary, verifiedPrimary string
	for _, e := range list {
		if !e.Primary {
			continue
		}
		if primary == "" {
			primary = e.Email
		}
		if e.Verified && verifiedPrimary == "" {
			verifiedPrimary = e.Email
		}
	}
	if verifiedPrimary != "" {
		return verifiedPrimary
	}
	return primary
}
