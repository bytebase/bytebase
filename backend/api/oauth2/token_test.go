package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/store"
)

// fakeWorkspaceResolver implements workspaceResolver for unit tests so we can
// exercise resolveBoundWorkspace without standing up a real Postgres store.
type fakeWorkspaceResolver struct {
	findResult    *store.WorkspaceMessage
	findErr       error
	findCallCount int
	lastFind      *store.FindWorkspaceMessage
}

func (r *fakeWorkspaceResolver) FindWorkspace(_ context.Context, find *store.FindWorkspaceMessage) (*store.WorkspaceMessage, error) {
	r.findCallCount++
	r.lastFind = find
	return r.findResult, r.findErr
}

func TestResolveBoundWorkspace(t *testing.T) {
	ctx := context.Background()

	t.Run("self-hosted skips membership check and returns issued workspace", func(t *testing.T) {
		r := &fakeWorkspaceResolver{}
		got, err := resolveBoundWorkspace(ctx, r, false, "ws-issued", "", "user@example.com")
		require.NoError(t, err)
		require.Equal(t, "ws-issued", got)
		require.Zero(t, r.findCallCount, "self-hosted must not call FindWorkspace")
	})

	t.Run("returns invalid grant when neither issued nor client workspace is bound", func(t *testing.T) {
		r := &fakeWorkspaceResolver{}
		_, err := resolveBoundWorkspace(ctx, r, false, "", "", "user@example.com")
		require.ErrorIs(t, err, errWorkspaceNotBound)
		require.Zero(t, r.findCallCount)
	})

	t.Run("falls back from issued to legacy client workspace", func(t *testing.T) {
		r := &fakeWorkspaceResolver{}
		got, err := resolveBoundWorkspace(ctx, r, false, "", "ws-client", "user@example.com")
		require.NoError(t, err)
		require.Equal(t, "ws-client", got)
	})

	t.Run("SaaS member returns workspace", func(t *testing.T) {
		r := &fakeWorkspaceResolver{findResult: &store.WorkspaceMessage{ResourceID: "ws-issued"}}
		got, err := resolveBoundWorkspace(ctx, r, true, "ws-issued", "", "user@example.com")
		require.NoError(t, err)
		require.Equal(t, "ws-issued", got)
		require.Equal(t, 1, r.findCallCount)
		require.NotNil(t, r.lastFind.WorkspaceID)
		require.Equal(t, "ws-issued", *r.lastFind.WorkspaceID)
		require.Equal(t, "user@example.com", r.lastFind.Email)
	})

	t.Run("SaaS non-member returns errWorkspaceNotMember sentinel", func(t *testing.T) {
		r := &fakeWorkspaceResolver{findResult: nil}
		_, err := resolveBoundWorkspace(ctx, r, true, "ws-issued", "", "user@example.com")
		require.Error(t, err)
		require.ErrorIs(t, err, errWorkspaceNotMember,
			"caller relies on errors.Is(errWorkspaceNotMember) to map this to invalid_grant 400")
	})

	t.Run("SaaS FindWorkspace internal error is not membership failure", func(t *testing.T) {
		r := &fakeWorkspaceResolver{findErr: errors.New("db unreachable")}
		_, err := resolveBoundWorkspace(ctx, r, true, "ws-issued", "", "user@example.com")
		require.Error(t, err)
		require.NotErrorIs(t, err, errWorkspaceNotMember,
			"internal errors must not be misclassified as membership failure (would 400 instead of 500)")
	})
}

func TestWorkspaceResolutionErrorForUnboundGrant(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodPost, "/api/oauth2/token", nil), rec)

	require.NoError(t, workspaceResolutionError(c, errWorkspaceNotBound))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "invalid_grant", body["error"])
}

// TestIssueTokensPlacesWorkspaceInJWT verifies the in-memory propagation of
// the workspace argument through GenerateOAuth2AccessToken into the JWT
// workspace_id claim. This is the last hop of the consent → token binding:
//
//	auth code (workspace col) → handler reads it via resolveBoundWorkspace
//	  → s.issueTokens(c, client, userEmail, workspaceID)
//	  → auth.GenerateOAuth2AccessToken(... , workspaceID, ...)
//	  → JWT.workspace_id claim
//
// The store-side round-trip is exercised by store_test.TestOAuth2WorkspaceBinding.
func TestIssueTokensPlacesWorkspaceInJWT(t *testing.T) {
	const secret = "test-secret"
	const userEmail = "demo@example.com"
	const clientID = "client-xyz"
	const workspaceID = "ws-consent-bound"

	tokenStr, err := auth.GenerateOAuth2AccessToken(userEmail, clientID, workspaceID, testResource, "", secret, time.Hour)
	require.NoError(t, err)

	// Decode the token (signature-verified) and assert the workspace_id
	// claim equals what we passed in. Anything else means the consent-time
	// binding got dropped on the way to the wire.
	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.Equal(t, workspaceID, claims["workspace_id"])
	require.Equal(t, userEmail, claims["sub"])
	require.Equal(t, clientID, claims["client_id"])
	// `aud` is serialized as a single-element array by jwt.ClaimStrings. Since
	// P1a PR 3 it carries the grant's stored canonical resource URI.
	require.Equal(t, []any{testResource}, claims["aud"])
}
