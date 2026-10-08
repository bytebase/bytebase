package tests

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// setMCPCapability sets the workspace MCP capability ceiling via the settings API.
func (ctl *controller) setMCPCapability(ctx context.Context, capability v1pb.MCPSetting_Capability) error {
	return ctl.updateMCPCapability(ctx, capability, false)
}

func (ctl *controller) updateMCPCapability(ctx context.Context, capability v1pb.MCPSetting_Capability, validateOnly bool) error {
	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		ValidateOnly: validateOnly,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_MCP.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_Mcp{
					Mcp: &v1pb.MCPSetting{Capability: capability},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"value.mcp.capability"},
		},
	}))
	return err
}

func (ctl *controller) getMCPSetting(ctx context.Context) (*v1pb.MCPSetting, error) {
	resp, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
		Name: "settings/" + v1pb.Setting_MCP.String(),
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.Value.GetMcp(), nil
}

func (ctl *controller) getMCPCapability(ctx context.Context) (v1pb.MCPSetting_Capability, error) {
	setting, err := ctl.getMCPSetting(ctx)
	if err != nil {
		return v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, err
	}
	return setting.GetCapability(), nil
}

// currentWorkspaceID returns the id the setting rows are keyed by. setting has
// PRIMARY KEY (workspace, name), so a raw predicate naming only the name reaches
// every workspace in the metadata database, not this test's own.
func currentWorkspaceID(ctx context.Context, t *testing.T, ctl *controller) string {
	t.Helper()
	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	require.NoError(t, err)
	return strings.TrimPrefix(workspace.Msg.Name, "workspaces/")
}

// TestMCPCapabilitySettingRoundTrip verifies the workspace MCP capability
// ceiling round-trips through the v1 settings API for every defined value, and
// that an explicit write of UNSPECIFIED is rejected. Workspace creation stores
// a concrete capability.
func TestMCPCapabilitySettingRoundTrip(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	// A fresh workspace starts with the safe, immediately useful ceiling.
	capability, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, capability)

	for _, want := range []v1pb.MCPSetting_Capability{
		v1pb.MCPSetting_DISABLED,
		v1pb.MCPSetting_READ_ONLY,
		v1pb.MCPSetting_READ_WRITE,
	} {
		a.NoError(ctl.setMCPCapability(ctx, want), want.String())
		got, err := ctl.getMCPCapability(ctx)
		a.NoError(err, want.String())
		a.Equal(want, got, want.String())
	}

	// Explicit UNSPECIFIED and unknown enum numbers — including the reserved
	// number 2 (was METADATA_ONLY) — are rejected, and the stored value is left
	// untouched.
	for _, invalid := range []v1pb.MCPSetting_Capability{
		v1pb.MCPSetting_CAPABILITY_UNSPECIFIED,
		v1pb.MCPSetting_Capability(2),
		v1pb.MCPSetting_Capability(99),
	} {
		err = ctl.setMCPCapability(ctx, invalid)
		a.Error(err, invalid.String())
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err), invalid.String())
	}
	got, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_WRITE, got)

	// A validate-only update must not leak into served state: the store caches
	// the setting object, so an in-place mutation would flip the live /mcp gate
	// without persisting anything.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_DISABLED))
	a.NoError(ctl.updateMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE, true))
	got, err = ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_DISABLED, got)
}

// TestMCPSettingExistsWithTheWorkspace pins that workspace creation writes the
// safe default for new workspaces.
func TestMCPSettingExistsWithTheWorkspace(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	got, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
		Name: "settings/" + v1pb.Setting_MCP.String(),
	}))
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, got.Msg.Value.GetMcp().GetCapability())

	list, err := ctl.settingServiceClient.ListSettings(ctx, connect.NewRequest(&v1pb.ListSettingsRequest{}))
	a.NoError(err)
	var listed bool
	for _, setting := range list.Msg.Settings {
		if setting.Name == "settings/"+v1pb.Setting_MCP.String() {
			listed = true
		}
	}
	a.True(listed, "a list-based client must be able to discover the setting it can read")

	// No allow_missing: the default update semantics have to work on a resource
	// the API just said exists.
	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_MCP.String(),
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Mcp{
				Mcp: &v1pb.MCPSetting{Capability: v1pb.MCPSetting_READ_ONLY},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value.mcp.capability"}},
	}))
	a.NoError(err, "reading the resource then failing to patch it is two answers to one question")

	stored, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, stored)
}

func TestMCPMissingRowUsesGenericUpdateSemantics(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspaceID := currentWorkspaceID(ctx, t, ctl)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	_, err = db.ExecContext(ctx, `
		DELETE FROM setting WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID)
	a.NoError(err)

	_, err = ctl.getMCPSetting(ctx)
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_MCP.String(),
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Mcp{
				Mcp: &v1pb.MCPSetting{Capability: v1pb.MCPSetting_READ_ONLY},
			}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"value.mcp.capability"}},
	}))
	a.Error(err)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err))

	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name:  "settings/" + v1pb.Setting_MCP.String(),
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Mcp{Mcp: &v1pb.MCPSetting{}}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{},
	}))
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err),
		"an update that names no capability must not create a row")
	_, err = ctl.getMCPSetting(ctx)
	a.Equal(connect.CodeNotFound, connect.CodeOf(err), "the refused update left no row behind")

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY),
		"allow_missing must create the MCP row")
	stored, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, stored)
}

// TestMCPRepairIgnoresARetiredKey pins why the retired ignoreMaskingExemptions
// key needs no migration. The store's unmarshaler discards a key this build
// does not define, and a save marshals what it read, so the key neither fails
// the read nor survives the next write.
func TestMCPRepairIgnoresARetiredKey(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspaceID := currentWorkspaceID(ctx, t, ctl)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()

	_, err = db.ExecContext(ctx, `
		INSERT INTO setting (name, workspace, value)
		VALUES ('MCP', $1, '{"capability": "READ_ONLYY", "ignoreMaskingExemptions": true}')
		ON CONFLICT (name, workspace) DO UPDATE SET value = EXCLUDED.value;
	`, workspaceID)
	a.NoError(err)

	_, err = ctl.getMCPSetting(ctx)
	a.NoError(err, "the retired key must not fail the read")

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY),
		"the retired key must not block repairing the ceiling")
	repaired, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, repaired)

	var retired bool
	a.NoError(db.QueryRowContext(ctx, `
		SELECT jsonb_exists(value, 'ignoreMaskingExemptions') FROM setting
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID).Scan(&retired))
	a.False(retired, "the save rewrote the row without the key")
}

// TestMCPMissingCapabilityRefusesPartialUpdate pins that an update naming no
// capability is refused before it writes, so it cannot erase a ceiling this
// build cannot read, such as a tier a newer release wrote.
func TestMCPMissingCapabilityRefusesPartialUpdate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspaceID := currentWorkspaceID(ctx, t, ctl)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{capability}', '"READ_ONLYY"')
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID)
	a.NoError(err)

	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name:  "settings/" + v1pb.Setting_MCP.String(),
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_Mcp{Mcp: &v1pb.MCPSetting{}}},
		},
		UpdateMask: &fieldmaskpb.FieldMask{},
	}))
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
	a.ErrorContains(err, "capability must be specified")

	var stored string
	a.NoError(db.QueryRowContext(ctx, `
		SELECT value ->> 'capability' FROM setting
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID).Scan(&stored))
	a.Equal("READ_ONLYY", stored, "the unreadable ceiling is still there, so enforcement still fails closed")
}

// TestMCPCapabilityBitesTheNextRequest is the live-state pin for the ceiling
// gate. The gate reads the MCP setting straight from the database on every
// request, so a changed ceiling binds the next request of a session that is
// already open: no re-consent, no reconnect, no restart.
//
// The writes are direct SQL. An in-process UpdateSetting refreshes the setting
// cache, so a cached read would pass too; an out-of-band edit is the state only
// an uncached read answers correctly. READ_ONLY rather than DISABLED: the /mcp
// connection gate refuses DISABLED on a read of its own before the ceiling gate
// runs.
//
// The RED state: point the gate at the cached Store.GetSetting instead of
// GetMCPSettingsUncached and the out-of-band READ_ONLY stops biting.
func TestMCPCapabilityBitesTheNextRequest(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspaceID := currentWorkspaceID(ctx, t, ctl)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	writeCeiling := func(capability v1pb.MCPSetting_Capability) {
		result, err := db.ExecContext(ctx, `
			UPDATE setting SET value = jsonb_set(value, '{capability}', to_jsonb($2::text))
			WHERE workspace = $1 AND name = 'MCP';
		`, workspaceID, capability.String())
		a.NoError(err)
		affected, err := result.RowsAffected()
		a.NoError(err)
		a.Equal(int64(1), affected, "the MCP setting row must exist for the out-of-band write to mean anything")
	}

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	session := openMCPSession(ctx, t, ctl, token)
	defer session.Close()
	createSheet := func() mcpCallResult {
		return callAPIOnSession(ctx, t, session, "SheetService/CreateSheet",
			sheetBody(ctl.project.Name, "SELECT 1;"))
	}

	served := createSheet()
	a.Equal(http.StatusOK, served.Status, "READ_WRITE serves a WRITE method: %s", served.Error)

	writeCeiling(v1pb.MCPSetting_READ_ONLY)
	refused := createSheet()
	a.Equal(http.StatusForbidden, refused.Status,
		"the ceiling must bite the very next request of the session already open")
	a.Contains(refused.Error, "capability ceiling is READ_ONLY",
		"the refusal must be the ceiling gate's")

	writeCeiling(v1pb.MCPSetting_READ_WRITE)
	served = createSheet()
	a.Equal(http.StatusOK, served.Status,
		"live in both directions, on the unchanged session: %s", served.Error)
}

// TestMCPSessionCannotTurnMCPOff drives the ceiling write from an OAuth-minted
// session, the credential an agent holds; TestMCPCannotRewriteItsOwnCeiling
// makes the same refusal on a session opened with the console bearer.
// UpdateSetting is FORBIDDEN to MCP sessions, so the class gate refuses it
// ahead of the handler whatever the payload, and the principal is a workspace
// admin who could otherwise write this setting.
//
// The stored value is the half that matters: a guard that refused after the
// write would leave the setting changed while reporting a denial.
func TestMCPSessionCannotTurnMCPOff(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))

	token, _ := mintMCPOAuthToken(t, ctl, ctl.authInterceptor.token)
	session := openMCPSession(ctx, t, ctl, token)
	defer session.Close()

	out := callAPIOnSession(ctx, t, session, "SettingService/UpdateSetting", map[string]any{
		"allowMissing": true,
		"setting": map[string]any{
			"name": "settings/" + v1pb.Setting_MCP.String(),
			"value": map[string]any{
				"mcp": map[string]any{"capability": "DISABLED"},
			},
		},
		"updateMask": "value.mcp.capability",
	})
	t.Logf("MCP UpdateSetting{capability: DISABLED} → status=%d error=%q", out.Status, out.Error)

	a.Equal(http.StatusForbidden, out.Status,
		"an MCP session must not be able to rewrite the workspace settings")
	a.Contains(out.Error, "not available to MCP sessions",
		"the refusal must come from the FORBIDDEN gate, not from a permission check")

	stored, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_WRITE, stored, "the refusal must land before the write")

	a.Empty(mcpAuditRows(ctx, t, ctl, workspace.Msg.Name, "/bytebase.v1.SettingService/UpdateSetting"),
		"a gate refusal is never stored")
}
