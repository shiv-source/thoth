package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	agentgit "github.com/shiv-source/thoth/agent/git"
)

// gitSetup pushes the wiki to a git remote: initializes a repository when the
// wiki is not one yet, points origin at url, commits the current tree, and
// pushes. The wiki path is read live from d.Wiki.Root() — the settings
// callback mutates it on wiki-path changes, so this always targets the
// current root.
//
// The pure-Go backend replaces the git binary: no os/exec anywhere. Push
// credentials and the committer identity come from the stored GitHub
// connection (the github_auth row), so a connected account is required.
func gitSetup(c echo.Context, d Deps) error {
	var body struct {
		URL string `json:"url"`
	}
	if err := c.Bind(&body); err != nil || body.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
	}
	root := d.Wiki.Root()
	repo, err := agentgit.Init(root)
	if err != nil {
		return gitFailure(c, d, errors.New("could not initialize a git repository — check that the directory is writable"))
	}
	id, auth, err := gitCredentials(d)
	if err != nil {
		return gitFailure(c, d, err)
	}
	if err := repo.SetRemote(body.URL); err != nil {
		return gitFailure(c, d, errors.New("could not update the remote URL — check the URL and that the repository is writable"))
	}
	if _, err := repo.CommitAll("chore: sync wiki", id); err != nil {
		return gitFailure(c, d, errors.New("could not commit the wiki changes — connect a GitHub account in Settings to commit"))
	}
	if err := repo.Push(auth); err != nil {
		return gitFailure(c, d, errors.New("could not push — check the remote URL, your credentials, and network access"))
	}
	if d.Store != nil {
		_ = d.Settings.SetSyncResult(true, "")
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// gitCredentials returns the committer identity and push auth from the stored
// GitHub connection. The sync requires the token (BasicAuth), so a missing
// connection is a clean, sanitized error.
func gitCredentials(d Deps) (agentgit.Identity, agentgit.Auth, error) {
	unconnected := errors.New("no GitHub connection — connect one in Settings to sync")
	if d.GitHub == nil || d.GitHub.Repo == nil {
		return agentgit.Identity{}, agentgit.Auth{}, unconnected
	}
	a, ok, err := d.GitHub.Repo.Get()
	if err != nil || !ok {
		return agentgit.Identity{}, agentgit.Auth{}, unconnected
	}
	name := a.DisplayName
	if name == "" {
		name = a.Username
	}
	return agentgit.Identity{Name: name, Email: a.Email},
		agentgit.Auth{Username: a.Username, Token: a.Token}, nil
}

// gitFailure reports a failed step with a sanitized hint: the raw go-git error
// may echo credentials or the remote URL, so only a fixable summary goes out.
// The summary (not the raw error) is what gets recorded in the sync state.
func gitFailure(c echo.Context, d Deps, err error) error {
	if d.Store != nil {
		_ = d.Settings.SetSyncResult(false, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
}
