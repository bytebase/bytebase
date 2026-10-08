package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestPermissionDeniedErrorMarksTheRequest(t *testing.T) {
	t.Parallel()

	marked := false
	ctx := withSetPermissionDenied(context.Background(), func() { marked = true })
	err := permissionDeniedError(ctx, errors.New("user does not have permission"))

	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.ErrorContains(t, err, "user does not have permission")
	require.True(t, marked, "the refusal must reach the audit interceptor")

	// A handler built outside the interceptor chain, and every test that calls
	// one, reaches this with no setter registered.
	require.Equal(t, connect.CodePermissionDenied,
		connect.CodeOf(permissionDeniedError(context.Background(), errors.New("no setter"))))
}
