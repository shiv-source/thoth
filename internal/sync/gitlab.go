package sync

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
	gitlabTimeout = 10 * time.Second
	gitlabAPIBase = "https://gitlab.com"
)

// gitlabService is the GitLab REST client adapted to the gitService seam
// (GET /api/v4/user for identity, /api/v4/projects for repos). PAT auth via
// the PRIVATE-TOKEN header; a self-hosted instance is reached through the
// provider's base_url override.
type gitlabService struct {
	hc      *http.Client
	baseURL string
}

func newGitlabService(hc *http.Client, baseURL string) *gitlabService {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &gitlabService{hc: hc, baseURL: baseURL}
}

func (s *gitlabService) apiBase() string {
	if s.baseURL != "" {
		return s.baseURL
	}
	return gitlabAPIBase
}

func (s *gitlabService) Profile(ctx context.Context, token string) (Identity, error) {
	body, err := s.get(ctx, "/api/v4/user", token)
	if err != nil {
		return Identity{}, err
	}
	var raw struct {
		Username string `json:"username"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Avatar   string `json:"avatar_url"`
		WebURL   string `json:"web_url"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Identity{}, errors.New("fetch gitlab profile: gitlab returned an unexpected response")
	}
	return Identity{
		Username:    raw.Username,
		DisplayName: raw.Name,
		Email:       raw.Email,
		AvatarURL:   raw.Avatar,
		ProfileURL:  raw.WebURL,
	}, nil
}

func (s *gitlabService) Repos(ctx context.Context, token string) ([]Target, error) {
	body, err := s.get(ctx, "/api/v4/projects?membership=true&per_page=100&order_by=updated_at&sort=desc", token)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		PathWithNamespace string `json:"path_with_namespace"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
		Visibility        string `json:"visibility"`
		Description       string `json:"description"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, errors.New("fetch gitlab projects: gitlab returned an unexpected response")
	}
	out := make([]Target, 0, len(raw))
	for _, p := range raw {
		out = append(out, Target{
			FullName:    p.PathWithNamespace,
			URL:         p.HTTPURLToRepo,
			Private:     p.Visibility == "private",
			Description: p.Description,
		})
	}
	return out, nil
}

// get performs one authenticated GET. Raw transport errors never propagate
// (they embed the request URL); non-2xx returns only the status code.
func (s *gitlabService) get(ctx context.Context, path, token string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, gitlabTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.apiBase()+path, nil)
	if err != nil {
		return nil, errors.New("fetch gitlab: build request")
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := s.hc.Do(req)
	if err != nil {
		return nil, errors.New("fetch gitlab: could not reach gitlab")
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("fetch gitlab: %w", ErrRejected)
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, fmt.Errorf("fetch gitlab: gitlab returned status %d", resp.StatusCode)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, errors.New("fetch gitlab: could not read response")
	}
	return buf.Bytes(), nil
}
