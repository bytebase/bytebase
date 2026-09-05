package v1

import (
	"context"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testpg"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestGetMCPInfoHandler drives the RPC against a real store, across the ways a
// stored row resolves: served, disabled, a name nothing resolves, a number
// nothing serves, and a row that does not unmarshal. capability is the whole
// answer now, and the consent page derives its three states from it alone, so
// this is the backend receipt for what the page may disclose.
func TestGetMCPInfoHandler(t *testing.T) {
	ctx := context.Background()
	db, stores, _ := testpg.New(t)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	service := NewWorkspaceService(stores, nil, &config.Profile{}, licenseService, nil)

	const workspaceID = "mcp-info-handler"
	_, err = stores.CreateWorkspace(ctx, &store.WorkspaceMessage{
		ResourceID: workspaceID,
		Payload:    &storepb.WorkspacePayload{Title: "MCP info handler"},
	}, "admin@example.com")
	require.NoError(t, err)

	get := func(ctx context.Context) (*v1pb.MCPInfo, error) {
		resp, err := service.GetMCPInfo(ctx, connect.NewRequest(&v1pb.GetMCPInfoRequest{}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
	workspaceCtx := func() context.Context {
		return context.WithValue(ctx, common.WorkspaceIDContextKey, workspaceID)
	}
	setCeiling := func(t *testing.T, value string) {
		t.Helper()
		_, err := db.ExecContext(ctx, `
			INSERT INTO setting (name, workspace, value) VALUES ('MCP', $1, $2)
			ON CONFLICT (workspace, name) DO UPDATE SET value = EXCLUDED.value`,
			workspaceID, value)
		require.NoError(t, err)
	}

	t.Run("a workspace with the default MCP policy", func(t *testing.T) {
		info, err := get(workspaceCtx())
		require.NoError(t, err)
		require.Equal(t, "workspaces/"+workspaceID, info.Workspace)
		require.Equal(t, v1pb.MCPSetting_READ_ONLY, info.Capability,
			"a new workspace is seeded READ_ONLY; only a workspace with no row at all falls back to READ_WRITE")
		require.False(t, info.IgnoreMaskingExemptions)
	})

	t.Run("the toggle reaches the response", func(t *testing.T) {
		setCeiling(t, `{"capability":"READ_ONLY","ignoreMaskingExemptions":true}`)
		info, err := get(workspaceCtx())
		require.NoError(t, err)
		require.Equal(t, v1pb.MCPSetting_READ_ONLY, info.Capability)
		require.True(t, info.IgnoreMaskingExemptions,
			"every masking state is written in terms of this, so withholding it leaves the table unusable")
	})

	t.Run("a disabled workspace is still answered", func(t *testing.T) {
		setCeiling(t, `{"capability":"DISABLED"}`)
		info, err := get(workspaceCtx())
		require.NoError(t, err)
		require.Equal(t, v1pb.MCPSetting_DISABLED, info.Capability)
	})

	// The two subtests below are BOT-106. Both ceilings refuse every MCP
	// connection and both used to refuse this call whole, which left the admin
	// repairing the row with no reading of what is stored, and the consent page
	// with no policy to disclose.
	t.Run("a ceiling this build cannot read is described, not refused", func(t *testing.T) {
		setCeiling(t, `{"capability":"READ_ONLYY"}`)
		info, err := get(workspaceCtx())
		require.NoError(t, err, "the ceiling is described, not refused")
		require.Equal(t, v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, info.Capability,
			"a ceiling nobody can resolve must not arrive as a mode, least of all the permissive one")
		require.False(t, info.IgnoreMaskingExemptions,
			"no MCP request runs under this ceiling, so the toggle carries neither the row's value nor a decision")
	})

	// The subtest above covers a mistyped enum NAME, which unmarshals away to
	// unset and is therefore describable. A wrong-TYPED value is different: it
	// fails the whole unmarshal, so the generic setting read errors and there is
	// no ceiling in the row to describe. The last two rows are why that is a
	// property of the row and not of the capability — the capability is
	// readable and a sibling field is not, and one failed unmarshal takes the
	// whole row with it.
	t.Run("a row that does not unmarshal is refused, not described", func(t *testing.T) {
		for _, row := range []string{
			`{"capability":{}}`,
			`{"capability":true}`,
			`{"capability":1.5}`,
			`{"capability":[]}`,
			`{"capability":"READ_ONLY","ignoreMaskingExemptions":[]}`,
			`{"capability":"READ_ONLY","ignoreMaskingExemptions":"yes"}`,
		} {
			setCeiling(t, row)
			_, err := get(workspaceCtx())
			require.Error(t, err, "%s: no ceiling in the row to describe", row)
			require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err), row)
		}
	})

	t.Run("a ceiling this build does not serve arrives as the stored number", func(t *testing.T) {
		// The reserved 2, or a value a newer release wrote. It parses, so the
		// number is the answer: a client that has no name for it is exactly the
		// client that must not disclose it, and the number is what tells it so.
		setCeiling(t, `{"capability":2}`)
		info, err := get(workspaceCtx())
		require.NoError(t, err)
		require.NotEqual(t, v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, info.Capability,
			"it parsed; the number is the answer, not a fallback to unresolvable")
		require.Equal(t, v1pb.MCPSetting_Capability(2), info.Capability)
	})

	t.Run("the gate's resolution wins over a second read", func(t *testing.T) {
		// The stored ceiling still says DISABLED below; the gate admitted this
		// request under READ_ONLY. Answering from the store would report a
		// ceiling the request was not admitted under.
		setCeiling(t, `{"capability":"DISABLED"}`)
		stamped := withMCPSettings(workspaceCtx(), &storepb.MCPSetting{
			Capability:              storepb.MCPSetting_READ_ONLY,
			IgnoreMaskingExemptions: true,
		})
		info, err := get(stamped)
		require.NoError(t, err)
		require.Equal(t, v1pb.MCPSetting_READ_ONLY, info.Capability)
		require.True(t, info.IgnoreMaskingExemptions)
	})

	t.Run("no workspace on the request", func(t *testing.T) {
		_, err := get(ctx)
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
