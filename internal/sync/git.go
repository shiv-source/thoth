package sync

import (
	"context"
	"errors"

	agentgit "github.com/shiv-source/thoth/agent/git"
)

// gitService is the identity/target seam for git-kind providers. The shared
// gitDriver serves both GitHub and GitLab through it; push is identical
// (agent/git over HTTPS basic auth).
type gitService interface {
	Profile(ctx context.Context, token string) (Identity, error)
	Repos(ctx context.Context, token string) ([]Target, error)
}

// gitDriver pushes the wiki to a git remote with the stored PAT and commits
// with the stored identity. The flow is the pre-cutover /api/v1/git/setup
// body (internal/api/git.go) generalized: init the repo, point origin at the
// target URL, commit the tree, push.
type gitDriver struct {
	svc gitService
}

func (d *gitDriver) Kind() Kind { return KindGit }

func (d *gitDriver) Fields() []Field {
	return []Field{{Key: "token", Label: "Personal access token", Secret: true, Required: true}}
}

func (d *gitDriver) Verify(ctx context.Context, cfg Config) (Identity, error) {
	token, ok := cfg["token"]
	if !ok || token == "" {
		return Identity{}, errors.New("token is required")
	}
	return d.svc.Profile(ctx, token)
}

func (d *gitDriver) Targets(ctx context.Context, cfg Config) ([]Target, error) {
	token, ok := cfg["token"]
	if !ok || token == "" {
		return nil, errors.New("token is required")
	}
	return d.svc.Repos(ctx, token)
}

// Push commits and pushes the wiki at root to the connection's repo_url.
// Errors are sanitized fixed messages — the raw go-git error may echo the
// remote URL, so only a fixable summary surfaces (the pre-cutover gitFailure
// pattern). The committer is the stored identity.
func (d *gitDriver) Push(_ context.Context, cfg Config, root string, committer Identity) error {
	url, ok := cfg["repo_url"]
	if !ok || url == "" {
		return errors.New("no sync repository selected — pick one in Settings")
	}
	token, ok := cfg["token"]
	if !ok || token == "" {
		return errors.New("no credentials stored — reconnect in Settings")
	}
	if committer.DisplayName == "" && committer.Username == "" {
		return errors.New("no account identity stored — reconnect in Settings")
	}
	repo, err := agentgit.Init(root)
	if err != nil {
		return errors.New("could not initialize a git repository — check that the directory is writable")
	}
	if err := repo.SetRemote(url); err != nil {
		return errors.New("could not update the remote URL — check the URL and that the repository is writable")
	}
	name := committer.DisplayName
	if name == "" {
		name = committer.Username
	}
	email := committer.Email
	if email == "" {
		email = committer.Username + "@users.noreply.local"
	}
	if _, err := repo.CommitAll("chore: sync wiki", agentgit.Identity{Name: name, Email: email}); err != nil {
		return errors.New("could not commit the wiki changes — connect a provider in Settings to commit")
	}
	username := committer.Username
	if username == "" {
		username = "oauth2"
	}
	if err := repo.Push(agentgit.Auth{Username: username, Token: token}); err != nil {
		return errors.New("could not push — check the remote URL, your credentials, and network access")
	}
	return nil
}
