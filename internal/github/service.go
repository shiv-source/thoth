package github

// Service bundles the GitHub client and the auth repo for API wiring.
type Service struct {
	Client *Client
	Repo   *Repo
}
