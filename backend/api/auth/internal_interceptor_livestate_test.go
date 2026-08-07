package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

const (
	liveWorkspace = "ws-live"
	liveUserEmail = "live@example.com"
	liveClientID  = "client-A"
	liveResource  = "https://bb.example.com/mcp"
)

// procedureRequest overrides a request's Spec so the interceptor sees a real
// registered procedure — GetAuthContext resolves it through the proto registry,
// which connect.NewRequest alone leaves empty.
type procedureRequest struct {
	connect.AnyRequest
	procedure string
}

func (r *procedureRequest) Spec() connect.Spec {
	return connect.Spec{Procedure: r.procedure}
}

func newProcedureRequest(t *testing.T, bearer string) *procedureRequest {
	t.Helper()
	req := connect.NewRequest(&emptypb.Empty{})
	req.Header().Set("Authorization", "Bearer "+bearer)
	return &procedureRequest{AnyRequest: req, procedure: "/bytebase.v1.SQLService/Query"}
}

// newLiveStore boots a migrated store with one workspace and one member, so
// resolvePrincipal has real state to re-resolve against.
func newLiveStore(t *testing.T) *store.Store {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO workspace (resource_id) VALUES ('%s');
		INSERT INTO principal (name, email, password_hash) VALUES ('live', '%s', 'unused');
	`, liveWorkspace, liveUserEmail))
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	st, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)

	_, err = st.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
		Workspace: liveWorkspace,
		Member:    common.FormatUserEmail(liveUserEmail),
		Roles:     []string{"roles/workspaceMember"},
	})
	require.NoError(t, err)
	return st
}

// TestInternalInterceptorPopulatesDelegatedGrant pins the AuthContext contract
// P1b keys on: every internal-chain request carries the verified credential's
// grant state — scope, resource, client, correlation — verbatim, and the two
// empty-scope states stay distinguishable (see common.DelegatedGrant). The
// public chain carries none.
func TestInternalInterceptorPopulatesDelegatedGrant(t *testing.T) {
	st := newLiveStore(t)
	in := NewInternalMCPAuthInterceptor(st, testSecret, &config.Profile{})

	capture := func(t *testing.T, interceptor connect.Interceptor, bearer string) *common.AuthContext {
		t.Helper()
		var captured *common.AuthContext
		next := func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			captured, _ = common.GetAuthContextFromContext(ctx)
			return nil, nil
		}
		_, err := interceptor.WrapUnary(next)(context.Background(), newProcedureRequest(t, bearer))
		require.NoError(t, err)
		require.NotNil(t, captured)
		return captured
	}

	rows := []struct {
		name string
		cred DelegatedMCPCredential
	}{
		{
			name: "a consented grant travels verbatim",
			cred: DelegatedMCPCredential{
				Principal:     liveUserEmail,
				WorkspaceID:   liveWorkspace,
				ClientID:      liveClientID,
				CorrelationID: "corr-1",
				Scope:         "mcp:read-only",
				Resource:      liveResource,
			},
		},
		{
			name: "legacy pre-grant session: scope and resource both empty",
			cred: DelegatedMCPCredential{
				Principal:     liveUserEmail,
				WorkspaceID:   liveWorkspace,
				CorrelationID: "corr-2",
			},
		},
		{
			// A grant that recorded no scope: a scope-less consent (permanent
			// population) or a PR-3-era mint during a rolling upgrade
			// (transient, and its grant DID record a scope). The populated
			// resource is what keeps it distinguishable from the genuinely
			// pre-grant row above — collapsing the two could widen a consented
			// read-only session to full legacy semantics.
			name: "grant-backed token: resource present, scope empty",
			cred: DelegatedMCPCredential{
				Principal:     liveUserEmail,
				WorkspaceID:   liveWorkspace,
				ClientID:      liveClientID,
				CorrelationID: "corr-3",
				Resource:      liveResource,
			},
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			token, err := GenerateInternalMCPToken(row.cred, testSecret)
			require.NoError(t, err)

			captured := capture(t, in, token)
			require.NotNil(t, captured.DelegatedGrant,
				"every internal-chain request must carry its delegated grant state")
			require.Equal(t, row.cred.Scope, captured.DelegatedGrant.Scope)
			require.Equal(t, row.cred.Resource, captured.DelegatedGrant.Resource)
			require.Equal(t, row.cred.ClientID, captured.DelegatedGrant.ClientID)
			require.Equal(t, row.cred.CorrelationID, captured.DelegatedGrant.CorrelationID)
		})
	}

	t.Run("the public chain carries no delegated grant", func(t *testing.T) {
		webToken, err := GenerateAccessToken(liveUserEmail, liveWorkspace, testSecret, time.Hour)
		require.NoError(t, err)

		pub := New(st, testSecret, nil, nil, &config.Profile{})
		captured := capture(t, pub, webToken)
		require.Nil(t, captured.DelegatedGrant,
			"a public-chain request must leave the delegated grant zero-valued")
	})
}

// TestInternalChainMembershipRevocationTakesEffectNextRequest pins the
// property the delegated credential rests on: it carries identity and grant
// state only, so authorization-relevant state — here workspace membership — is
// re-resolved against the store on EVERY internal request. Revoking membership
// must bite on the very next request with the SAME still-valid credential: no
// re-consent, no token expiry involved. If a refactor ever starts trusting the
// credential for authorization state, this goes red.
func TestInternalChainMembershipRevocationTakesEffectNextRequest(t *testing.T) {
	ctx := context.Background()
	st := newLiveStore(t)
	in := NewInternalMCPAuthInterceptor(st, testSecret, &config.Profile{})

	token, err := GenerateInternalMCPToken(DelegatedMCPCredential{
		Principal:     liveUserEmail,
		WorkspaceID:   liveWorkspace,
		ClientID:      liveClientID,
		CorrelationID: "corr-live",
		Scope:         "mcp:read-only",
		Resource:      liveResource,
	}, testSecret)
	require.NoError(t, err)

	call := func() (reached bool, err error) {
		next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			reached = true
			return nil, nil
		}
		_, err = in.WrapUnary(next)(context.Background(), newProcedureRequest(t, token))
		return reached, err
	}

	reached, err := call()
	require.NoError(t, err)
	require.True(t, reached)

	setRoles := func(roles []string) {
		_, err := st.PatchWorkspaceIamPolicy(ctx, &store.PatchIamPolicyMessage{
			Workspace: liveWorkspace,
			Member:    common.FormatUserEmail(liveUserEmail),
			Roles:     roles,
		})
		require.NoError(t, err)
	}

	setRoles(nil) // revoke the membership entirely
	reached, err = call()
	require.Error(t, err, "the same credential must stop working the moment membership is revoked")
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.False(t, reached, "no handler may run for a revoked member")

	// Live in both directions, not a one-way latch: restoring membership
	// restores service with the unchanged credential.
	setRoles([]string{"roles/workspaceMember"})
	reached, err = call()
	require.NoError(t, err)
	require.True(t, reached)
}
