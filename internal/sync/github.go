package sync

import (
	"context"
	"errors"

	"github.com/shiv-source/thoth/internal/github"
)

// ErrRejected marks credentials the provider rejected — the only failure the
// API layer surfaces as a client error (400); everything else is internal.
var ErrRejected = errors.New("provider rejected the credentials")

// githubService adapts the internal/github REST client to the gitService seam
// so the shared git driver can serve both GitHub and GitLab.
type githubService struct {
	client *github.Client
}

func (s *githubService) Profile(ctx context.Context, token string) (Identity, error) {
	p, err := s.client.FetchProfile(ctx, token)
	if errors.Is(err, github.ErrTokenRejected) {
		return Identity{}, ErrRejected
	}
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Username:    p.Username,
		DisplayName: p.DisplayName,
		Email:       p.Email,
		AvatarURL:   p.AvatarURL,
		ProfileURL:  p.ProfileURL,
		Scopes:      p.Scopes,
	}, nil
}

func (s *githubService) Repos(ctx context.Context, token string) ([]Target, error) {
	repos, err := s.client.FetchRepos(ctx, token)
	if errors.Is(err, github.ErrTokenRejected) {
		return nil, ErrRejected
	}
	if err != nil {
		return nil, err
	}
	out := make([]Target, 0, len(repos))
	for _, r := range repos {
		out = append(out, Target{
			FullName:    r.FullName,
			URL:         r.CloneURL,
			Private:     r.Private,
			Description: r.Description,
		})
	}
	return out, nil
}
