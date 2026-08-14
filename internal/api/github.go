package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiv-source/thoth/internal/github"
)

// githubIdentity is the token-free view of a stored GitHub connection. The
// token never leaves the store.
type githubIdentity struct {
	Username         string `json:"username"`
	DisplayName      string `json:"display_name"`
	Email            string `json:"email"`
	AvatarURL        string `json:"avatar_url"`
	ProfileURL       string `json:"profile_url"`
	Scopes           string `json:"scopes"`
	AccountCreatedAt string `json:"account_created_at"`
	AccountUpdatedAt string `json:"account_updated_at"`
}

func identityFromAuth(a github.Auth) githubIdentity {
	return githubIdentity{
		Username:         a.Username,
		DisplayName:      a.DisplayName,
		Email:            a.Email,
		AvatarURL:        a.AvatarURL,
		ProfileURL:       a.ProfileURL,
		Scopes:           a.Scopes,
		AccountCreatedAt: a.AccountCreatedAt,
		AccountUpdatedAt: a.AccountUpdatedAt,
	}
}

func connectGitHub(c echo.Context, d Deps) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.Bind(&body); err != nil || body.Token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "token is required"})
	}
	if d.GitHub == nil || d.GitHub.Client == nil || d.GitHub.Repo == nil {
		return internalError(c, d, "github not configured", errors.New("missing github service"))
	}
	profile, err := d.GitHub.Client.FetchProfile(c.Request().Context(), body.Token)
	if err != nil {
		if errors.Is(err, github.ErrTokenRejected) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "github rejected the token"})
		}
		return internalError(c, d, "fetch github profile", err)
	}
	auth := github.Auth{
		Token:            body.Token,
		Username:         profile.Username,
		DisplayName:      profile.DisplayName,
		Email:            profile.Email,
		AvatarURL:        profile.AvatarURL,
		ProfileURL:       profile.ProfileURL,
		Scopes:           profile.Scopes,
		AccountCreatedAt: profile.AccountCreatedAt,
		AccountUpdatedAt: profile.AccountUpdatedAt,
	}
	if err := d.GitHub.Repo.Save(auth); err != nil {
		return internalError(c, d, "save github auth", err)
	}
	return c.JSON(http.StatusOK, identityFromAuth(auth))
}

func getGitHubAuth(c echo.Context, d Deps) error {
	a, _, err := d.GitHub.Repo.Get()
	if err != nil {
		return internalError(c, d, "read github auth", err)
	}
	return c.JSON(http.StatusOK, identityFromAuth(a))
}

func disconnectGitHub(c echo.Context, d Deps) error {
	if err := d.GitHub.Repo.Clear(); err != nil {
		return internalError(c, d, "clear github auth", err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}
