package command

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/world"
)

// idTokenSource returns the CI job's OIDC token. It is called again on each
// renewal, so a source that can mint a fresh token does.
type idTokenSource func(ctx context.Context) (string, error)

// newIDTokenSource reads BYTEBASE_ID_TOKEN, which a CI such as GitLab fills from
// its own configuration, and otherwise asks GitHub for a token.
func newIDTokenSource(getenv func(string) string, platform world.JobPlatform, audience string, timeout time.Duration) idTokenSource {
	return func(ctx context.Context) (string, error) {
		if token := getenv("BYTEBASE_ID_TOKEN"); token != "" {
			return token, nil
		}
		if platform != world.GitHub {
			return "", errors.New("the job has no ID token: set BYTEBASE_ID_TOKEN from the CI's ID token")
		}
		// A plain client: the --custom-header values are for Bytebase only.
		return fetchGitHubIDToken(ctx, &http.Client{Timeout: timeout}, getenv("ACTIONS_ID_TOKEN_REQUEST_URL"), getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN"), audience)
	}
}

// https://docs.github.com/en/actions/reference/security/oidc#methods-for-requesting-the-oidc-token
func fetchGitHubIDToken(ctx context.Context, httpClient *http.Client, requestURL, requestToken, audience string) (string, error) {
	if requestURL == "" || requestToken == "" {
		return "", errors.New("the job has no ID token: add `permissions: id-token: write` to the job")
	}
	u, err := url.Parse(requestURL)
	if err != nil {
		return "", errors.Wrap(err, "invalid ACTIONS_ID_TOKEN_REQUEST_URL")
	}
	query := u.Query()
	query.Set("audience", audience)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to build the GitHub ID token request")
	}
	req.Header.Set("Authorization", "Bearer "+requestToken)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to request a GitHub ID token")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", errors.Wrap(err, "failed to read the GitHub ID token response")
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("GitHub refused the ID token request: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", errors.Wrap(err, "failed to parse the GitHub ID token response")
	}
	if payload.Value == "" {
		return "", errors.New("GitHub returned no ID token")
	}
	return payload.Value, nil
}
