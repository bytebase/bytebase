package auth

import (
	"context"
	"time"

	"connectrpc.com/connect"
	errs "github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/store"
)

// InternalMCPAuthInterceptor guards the private in-memory transport that
// carries /mcp tool traffic to the internal API handler chain. It accepts ONLY
// the delegated credential minted at the /mcp boundary: public bearer tokens
// are refused here, and the public chain — whose keyfunc only knows kid "v1" —
// symmetrically refuses the internal credential.
//
// Identity and grant state ride the credential; authorization does not. The
// principal is re-resolved against the store and workspace membership on every
// request, and the downstream ACL interceptor re-resolves RBAC live, exactly
// as for a public request.
//
// Unlike the public interceptor, allow_without_credential RPCs get no
// authentication skip: every internal request carries a freshly minted
// credential, so an invalid one is a bug and fails closed.
type InternalMCPAuthInterceptor struct {
	store   *store.Store
	secret  string
	profile *config.Profile
}

// NewInternalMCPAuthInterceptor returns the auth interceptor for the internal
// MCP handler chain.
func NewInternalMCPAuthInterceptor(store *store.Store, secret string, profile *config.Profile) *InternalMCPAuthInterceptor {
	return &InternalMCPAuthInterceptor{
		store:   store,
		secret:  secret,
		profile: profile,
	}
}

// WrapUnary implements the ConnectRPC interceptor interface for unary RPCs.
func (in *InternalMCPAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		tokenStr, err := GetTokenFromHeaders(req.Header())
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		cred, err := VerifyInternalMCPToken(tokenStr, in.secret)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		authContext, err := getAuthContext(req.Spec().Procedure)
		if err != nil {
			return nil, err
		}
		// The credential's grant state travels verbatim into the AuthContext —
		// this is the contract P1b's enforcement keys on (see
		// common.DelegatedGrant for the empty-state semantics). Grant state
		// only: the principal is re-resolved below, and RBAC downstream.
		authContext.DelegatedGrant = &common.DelegatedGrant{
			Scope:         cred.Scope,
			Resource:      cred.Resource,
			ClientID:      cred.ClientID,
			CorrelationID: cred.CorrelationID,
		}
		ctx = context.WithValue(ctx, common.AuthContextKey, authContext)

		user, err := resolvePrincipal(ctx, in.store, in.profile, cred.Principal, cred.WorkspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		in.profile.LastActiveTS.Store(time.Now().Unix())
		ctx = context.WithValue(ctx, common.UserContextKey, user)
		ctx = context.WithValue(ctx, common.WorkspaceIDContextKey, cred.WorkspaceID)
		return next(ctx, req)
	}
}

// WrapStreamingClient implements the ConnectRPC interceptor interface for streaming clients.
func (*InternalMCPAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return next(ctx, spec)
	}
}

// WrapStreamingHandler rejects streaming outright: no MCP tool streams, so a
// streaming call on the internal transport is a bug and fails closed.
func (*InternalMCPAuthInterceptor) WrapStreamingHandler(connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(_ context.Context, _ connect.StreamingHandlerConn) error {
		return connect.NewError(connect.CodeUnimplemented, errs.New("streaming is not supported on the internal MCP transport"))
	}
}
