package api

import (
	"context"
	"errors"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/gitutil"
)

// gitStepTimeout bounds every git command; a hung push must not wedge the
// request forever.
const gitStepTimeout = 15 * time.Second

// gitSetup pushes the wiki to a git remote: initializes a repository when the
// wiki is not one yet, points origin at url, commits the current tree, and
// pushes. The wiki path is read live from d.Wiki.Root() — the settings
// callback mutates it on wiki-path changes, so this always targets the
// current root.
func gitSetup(c echo.Context, d Deps) error {
	var body struct {
		URL string `json:"url"`
	}
	if err := c.Bind(&body); err != nil || body.URL == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "url is required"})
	}
	root := d.Wiki.Root()
	if err := gitutil.Init(root); err != nil {
		return gitFailure(c, d, err)
	}
	if err := gitSetRemote(root, body.URL); err != nil {
		return gitFailure(c, d, err)
	}
	if _, err := gitCommitAll(root); err != nil {
		return gitFailure(c, d, err)
	}
	if err := gitPush(root); err != nil {
		return gitFailure(c, d, err)
	}
	if d.Store != nil {
		_ = d.Settings.SetSyncResult(true, "")
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

// gitFailure reports a failed step with a sanitized hint: the raw error may
// echo credentials or the remote URL, so only a fixable summary goes out.
// The summary (not the raw error) is what gets recorded in the sync state.
func gitFailure(c echo.Context, d Deps, err error) error {
	if d.Store != nil {
		_ = d.Settings.SetSyncResult(false, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
}

func gitCmd(ctx context.Context, root string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// gitSetRemote points origin at url, adding the remote when it does not exist
// yet (`set-url` fails for a missing remote on older git versions).
func gitSetRemote(root, url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitStepTimeout)
	defer cancel()
	if _, err := gitCmd(ctx, root, "remote", "get-url", "origin"); err != nil {
		ctx, cancel = context.WithTimeout(context.Background(), gitStepTimeout)
		defer cancel()
		if _, err := gitCmd(ctx, root, "remote", "add", "origin", url); err != nil {
			return errors.New("could not add the remote — check the URL and that git is installed")
		}
		return nil
	}
	ctx, cancel = context.WithTimeout(context.Background(), gitStepTimeout)
	defer cancel()
	if _, err := gitCmd(ctx, root, "remote", "set-url", "origin", url); err != nil {
		return errors.New("could not update the remote URL — check the URL and that git is installed")
	}
	return nil
}

// gitCommitAll stages everything and commits; an empty tree ("nothing to
// commit") is not an error, just nothing to push.
func gitCommitAll(root string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitStepTimeout)
	defer cancel()
	if _, err := gitCmd(ctx, root, "add", "-A"); err != nil {
		return false, errors.New("could not stage the wiki files — check that the wiki directory is writable")
	}
	ctx, cancel = context.WithTimeout(context.Background(), gitStepTimeout)
	defer cancel()
	out, err := gitCmd(ctx, root, "commit", "-m", "chore: sync wiki")
	if err != nil {
		if strings.Contains(out, "nothing to commit") {
			return false, nil
		}
		return false, errors.New("could not commit the wiki changes — set your git identity (user.name / user.email) or commit manually")
	}
	return true, nil
}

// gitPush pushes the current branch and sets the upstream tracking.
func gitPush(root string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitStepTimeout)
	defer cancel()
	if _, err := gitCmd(ctx, root, "push", "-u", "origin", "HEAD"); err != nil {
		return errors.New("could not push — check the remote URL, your credentials, and network access")
	}
	return nil
}
