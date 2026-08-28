package tests

import (
	"context"
	"database/sql"
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
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

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
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

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

// TestMCPUnrecognizedCeilingSurvivesAToggleOnlyUpdate is the write-path half of
// the fail-closed rule. The unmarshaler resolves an unknown enum name to
// UNSPECIFIED, and marshalling would omit that zero enum. The partial update
// therefore requires the request to set a recognized capability rather than
// silently erasing the stored value.
func TestMCPUnrecognizedCeilingSurvivesAToggleOnlyUpdate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	workspaceID := currentWorkspaceID(ctx, t, ctl)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	result, err := db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{capability}', '"READ_ONLYY"')
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID)
	a.NoError(err)
	affected, err := result.RowsAffected()
	a.NoError(err)
	a.Equal(int64(1), affected, "the MCP setting row must exist for this test to mean anything")

	err = ctl.setIgnoreMaskingExemptions(ctx, true)
	a.Error(err, "saving the toggle must not erase a ceiling this build cannot read")
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))

	var stored string
	a.NoError(db.QueryRowContext(ctx, `
		SELECT value ->> 'capability' FROM setting
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID).Scan(&stored))
	a.Equal("READ_ONLYY", stored, "the unreadable ceiling is still there, so enforcement still fails closed")

	// A dry run can repair the parsed capability state without persisting it.
	dryRun := ctl.updateMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY, true)
	a.NoError(dryRun, "a dry run that repairs the capability is fine")

	// The same request that sets a capability repairs the row.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY))
	got, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, got)
	a.NoError(ctl.setIgnoreMaskingExemptions(ctx, true), "and the toggle saves once the ceiling is readable")
}

func TestMCPMissingRowUsesGenericUpdateSemantics(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

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

	a.NoError(ctl.setIgnoreMaskingExemptions(ctx, true),
		"allow_missing must create an MCP row from the READ_WRITE compatibility fallback")
	setting, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_WRITE, setting.GetCapability())
	a.True(setting.GetIgnoreMaskingExemptions())
}

// TestMCPRepairKeepsTheMaskingToggle pins that fixing the ceiling does not
// disarm the masking toggle. Repairing a policy must change the one field the
// request names; handing users' unmasking exemptions back to MCP sessions as a
// side effect of a ceiling fix would be a silent loosening.
func TestMCPRepairKeepsTheMaskingToggle(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

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

	// Generic reads still expose the fields this build understands.
	setting, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.True(setting.GetIgnoreMaskingExemptions(),
		"the toggle is readable even though the ceiling is not")

	// Repairing the ceiling must not disarm the toggle.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY))
	repaired, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, repaired.GetCapability())
	a.True(repaired.GetIgnoreMaskingExemptions(),
		"repairing the ceiling must not return exemptions to MCP sessions")
}

// TestMCPMissingCapabilityRefusesPartialUpdate pins that a partial update cannot
// make an invalid row look permissive.
func TestMCPMissingCapabilityRefusesPartialUpdate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	workspaceID := currentWorkspaceID(ctx, t, ctl)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = '{}' WHERE workspace = $1 AND name = 'MCP'
	`, workspaceID)
	a.NoError(err)

	err = ctl.setIgnoreMaskingExemptions(ctx, true)
	a.Error(err)
	a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err))
}
