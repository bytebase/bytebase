package oauth2

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/store"
)

// fakeWorkspaceResolver implements workspaceResolver for unit tests so we can
// exercise resolveBoundWorkspace without standing up a real Postgres store.
type fakeWorkspaceResolver struct {
	singleton     string
	singletonErr  error
	findResult    *store.WorkspaceMessage
	findErr       error
	findCallCount int
	lastFind      *store.FindWorkspaceMessage
}

func (r *fakeWorkspaceResolver) GetWorkspaceID(_ context.Context) (string, error) {
	return r.singleton, r.singletonErr
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

	t.Run("self-hosted falls back to singleton when both issued and client workspace are empty", func(t *testing.T) {
		r := &fakeWorkspaceResolver{singleton: "ws-singleton"}
		got, err := resolveBoundWorkspace(ctx, r, false, "", "", "user@example.com")
		require.NoError(t, err)
		require.Equal(t, "ws-singleton", got)
	})

	t.Run("falls back from issued to client workspace before singleton", func(t *testing.T) {
		r := &fakeWorkspaceResolver{singleton: "ws-singleton"}
		got, err := resolveBoundWorkspace(ctx, r, false, "", "ws-client", "user@example.com")
		require.NoError(t, err)
		require.Equal(t, "ws-client", got, "client workspace should win over singleton fallback")
	})

	t.Run("returns error when no workspace is resolvable", func(t *testing.T) {
		r := &fakeWorkspaceResolver{singleton: ""}
		_, err := resolveBoundWorkspace(ctx, r, false, "", "", "user@example.com")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no workspace bound")
		require.NotErrorIs(t, err, errWorkspaceNotMember, "missing workspace is not a membership failure")
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

	t.Run("SaaS singleton-lookup error is wrapped and not membership failure", func(t *testing.T) {
		r := &fakeWorkspaceResolver{singletonErr: pkgerrors.New("db down")}
		_, err := resolveBoundWorkspace(ctx, r, true, "", "", "user@example.com")
		require.Error(t, err)
		require.NotErrorIs(t, err, errWorkspaceNotMember)
		require.Contains(t, err.Error(), "failed to resolve workspace")
	})
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

	tokenStr, err := auth.GenerateOAuth2AccessToken(userEmail, clientID, workspaceID, secret, time.Hour)
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
	// `aud` is serialized as a single-element array by jwt.ClaimStrings.
	require.Equal(t, []any{auth.OAuth2AccessTokenAudience}, claims["aud"])
}
