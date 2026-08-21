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

func (ctl *controller) getMCPCapability(ctx context.Context) (v1pb.MCPSetting_Capability, error) {
	resp, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
		Name: "settings/" + v1pb.Setting_MCP.String(),
	}))
	if err != nil {
		return v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, err
	}
	return resp.Msg.Value.GetMcp().GetCapability(), nil
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
// that an explicit write of UNSPECIFIED is rejected — absent has defined
// resolver semantics (it resolves to READ_WRITE), so writing "unspecified"
// is a caller bug.
func TestMCPCapabilitySettingRoundTrip(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// A fresh workspace has no explicit ceiling.
	capability, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, capability)

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

	// Saving the other field of the MCP setting must not revert a ceiling
	// written out-of-band (emergency SQL): the update path merges the named
	// mask paths onto the stored value and writes the whole message back, so
	// merging onto a stale cached base would silently clear the kill switch.
	// The cache is warm from the reads above and holds DISABLED; flip the
	// stored value behind the server's back, then update the other field.
	workspaceID := currentWorkspaceID(ctx, t, ctl)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{capability}', '"READ_ONLY"')
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID)
	a.NoError(err)
	a.NoError(ctl.setIgnoreMaskingExemptions(ctx, true))
	got, err = ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, got,
		"the sibling update must merge onto the stored setting, not a stale cached one")
	ignoring, err := ctl.getIgnoreMaskingExemptions(ctx)
	a.NoError(err)
	a.True(ignoring, "and the field it named must actually be written")
}

// TestMCPSettingIsPresentBeforeItIsConfigured is the answer to "does
// settings/MCP exist?" on a workspace that never touched it. Get, List and
// Update have to agree: a client that reads the resource and then cannot patch
// it with default update semantics has been told two different things.
func TestMCPSettingIsPresentBeforeItIsConfigured(t *testing.T) {
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
	a.NoError(err, "an unconfigured MCP setting reads as its zero value, not 404")
	a.Equal(v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, got.Msg.Value.GetMcp().GetCapability())

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

// TestMCPUnreadableCeilingSurvivesAToggleOnlyUpdate is the write-path half of
// the fail-closed rule. GetMCPSettingsUncached refuses a stored capability this
// build cannot read; the merge would erase it, because the unmarshaler drops an
// enum name it does not know and marshalling omits the zero enum — so the row
// would come back with no capability key and the next read would resolve it to
// READ_WRITE. Saving the masking toggle would reopen MCP.
//
// Reachable on a rolling upgrade, not only from a typo: a newer replica can
// write a capability name this build has never heard of.
func TestMCPUnreadableCeilingSurvivesAToggleOnlyUpdate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_DISABLED))

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
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	a.Contains(err.Error(), "value.mcp.capability", "the refusal must name the way out")

	var stored string
	a.NoError(db.QueryRowContext(ctx, `
		SELECT value ->> 'capability' FROM setting
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID).Scan(&stored))
	a.Equal("READ_ONLYY", stored, "the unreadable ceiling is still there, so enforcement still fails closed")

	// The dry run reads the same raw row the locked merge does, so it cannot
	// report success on a request the real write refuses.
	dryRun := ctl.updateMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY, true)
	a.NoError(dryRun, "a dry run that repairs the capability is fine")

	// The same request that sets a capability repairs the row.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY))
	got, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, got)
	a.NoError(ctl.setIgnoreMaskingExemptions(ctx, true), "and the toggle saves once the ceiling is readable")
}

// TestMCPUnknownFieldSurvivesAPartialUpdate is the mixed-version half of the
// same rule. An older replica parses the row with the shared unmarshaler, which
// discards what it cannot name, and re-marshalling writes the row back without
// it — so a partial update on the old replica would delete configuration the
// new one wrote. Reads stay lenient on purpose (one field from a newer release
// must not disable MCP); the write refuses.
func TestMCPUnknownFieldSurvivesAPartialUpdate(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY))

	workspaceID := currentWorkspaceID(ctx, t, ctl)
	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	result, err := db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{rulesFromANewerRelease}', '[{"x":1}]')
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID)
	a.NoError(err)
	affected, err := result.RowsAffected()
	a.NoError(err)
	a.Equal(int64(1), affected, "the MCP setting row must exist for this test to mean anything")

	// The read is unaffected: an unknown field must not disable MCP.
	capability, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, capability)

	err = ctl.setIgnoreMaskingExemptions(ctx, true)
	a.Error(err, "a partial write must not delete a field a newer replica wrote")
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	a.Contains(err.Error(), "rulesFromANewerRelease", "the refusal must name what would be lost")

	var present bool
	a.NoError(db.QueryRowContext(ctx, `
		SELECT jsonb_exists(value, 'rulesFromANewerRelease') FROM setting
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID).Scan(&present))
	a.True(present, "and the field is still there")
}
