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

// TestMCPUnreadableCeilingIsVisibleToTheAdmin is the read-path half of the same
// rule, and the defect BOT-100 records. Enforcement inspects the row's own
// capability key and refuses; the settings API parses with the shared
// unmarshaler, which drops an enum name it does not know, so the same row read
// two ways said READ_WRITE — the most permissive ceiling — over a workspace
// refusing every MCP connection.
//
// The second-order effect is the one an admin feels: with the page already
// showing Read-write, picking Read-write changes nothing, so the row could not
// be repaired to READ_WRITE in one save at all.
func TestMCPUnreadableCeilingIsVisibleToTheAdmin(t *testing.T) {
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
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{capability}', '"READ_ONLYY"')
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID)
	a.NoError(err)

	setting, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.True(setting.GetCapabilityUnreadable(),
		"the admin has to be told, or the page describes a ceiling nobody is enforcing")
	a.Equal(v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, setting.GetCapability(),
		"and the parsed capability stays unset, so no reader mistakes it for a choice")

	// Every token the store treats as unreadable, not just the mistyped name.
	// Each of these unmarshals away to unset without erroring, so a reader that
	// only looked at the parsed capability would report the permissive default
	// over a workspace refusing every connection.
	for _, token := range []string{`null`, `""`, `"CAPABILITY_UNSPECIFIED"`} {
		_, err = db.ExecContext(ctx, `
			UPDATE setting SET value = jsonb_set(value, '{capability}', $2::jsonb)
			WHERE workspace = $1 AND name = 'MCP';
		`, workspaceID, token)
		a.NoError(err, token)

		setting, err = ctl.getMCPSetting(ctx)
		a.NoError(err, token)
		a.True(setting.GetCapabilityUnreadable(),
			"%s is a ceiling the store refuses; reporting it as readable is the defect this field ends", token)
		a.Equal(v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, setting.GetCapability(), token)
	}

	// A wrong-TYPED token is a different state, and deliberately not a
	// repairable one. It fails the whole unmarshal rather than just the enum,
	// so there is no ceiling in the row to describe and no supported writer
	// that produces one — the read says so plainly instead of offering a fix
	// it would need a raw-replacement path to deliver.
	for _, token := range []string{`{}`, `true`, `1.5`, `[]`} {
		_, err = db.ExecContext(ctx, `
			UPDATE setting SET value = jsonb_set(value, '{capability}', $2::jsonb)
			WHERE workspace = $1 AND name = 'MCP';
		`, workspaceID, token)
		a.NoError(err, token)

		_, err = ctl.getMCPSetting(ctx)
		a.Error(err, "%s carries no ceiling to report, so the read fails rather than inventing one", token)
	}

	// Back to the mistyped name for the repair below.
	_, err = db.ExecContext(ctx, `
		UPDATE setting SET value = jsonb_set(value, '{capability}', '"READ_ONLYY"')
		WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID)
	a.NoError(err)

	// The same row through the list surface: a client that discovers the
	// setting there and then patches it must not be told a different story.
	list, err := ctl.settingServiceClient.ListSettings(ctx, connect.NewRequest(&v1pb.ListSettingsRequest{}))
	a.NoError(err)
	var listed *v1pb.MCPSetting
	for _, entry := range list.Msg.Settings {
		if entry.Name == "settings/"+v1pb.Setting_MCP.String() {
			listed = entry.Value.GetMcp()
		}
	}
	a.NotNil(listed)
	a.True(listed.GetCapabilityUnreadable())

	// One save repairs it, to the most permissive ceiling included — the pick
	// the old shape could not express.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))
	repaired, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_WRITE, repaired.GetCapability())
	a.False(repaired.GetCapabilityUnreadable(), "a readable row carries no unreadable value")

	// A workspace that never configured MCP is not the same state and must not
	// be reported as one: it resolves READ_WRITE and nothing is wrong with it.
	_, err = db.ExecContext(ctx, `
		DELETE FROM setting WHERE workspace = $1 AND name = 'MCP';
	`, workspaceID)
	a.NoError(err)
	unconfigured, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.False(unconfigured.GetCapabilityUnreadable())
	a.Equal(v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, unconfigured.GetCapability())
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

	// The toggle is readable even though the ceiling is not, so the page must
	// show it rather than reporting the workspace as not ignoring exemptions.
	setting, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.True(setting.GetCapabilityUnreadable(), "the ceiling is the unreadable part")
	a.True(setting.GetIgnoreMaskingExemptions(),
		"the toggle is readable even though the ceiling is not")

	// Repairing the ceiling must not disarm the toggle.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_ONLY))
	repaired, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.Equal(v1pb.MCPSetting_READ_ONLY, repaired.GetCapability())
	a.True(repaired.GetIgnoreMaskingExemptions(),
		"repairing the ceiling must not return exemptions to MCP sessions")
	a.False(repaired.GetCapabilityUnreadable())
}

// TestMCPToggleOnlyRowIsNotUnreadable pins the other direction. A first edit
// that sets only the masking toggle leaves a row with no capability key at all,
// which is the ordinary never-configured state resolving READ_WRITE — not a row
// an admin has to repair.
func TestMCPToggleOnlyRowIsNotUnreadable(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	a.NoError(ctl.setIgnoreMaskingExemptions(ctx, true))

	setting, err := ctl.getMCPSetting(ctx)
	a.NoError(err)
	a.True(setting.GetIgnoreMaskingExemptions())
	a.False(setting.GetCapabilityUnreadable(),
		"no capability key is never-configured, not unreadable; claiming otherwise "+
			"tells the admin every MCP connection is refused when none is")
	a.Equal(v1pb.MCPSetting_CAPABILITY_UNSPECIFIED, setting.GetCapability())
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

// TestMCPRefusedRepairAuditsTheCeilingThatCausedIt pins the before-image on a
// refusal, which nothing else covers.
//
// updateMCPSetting decides everything against the LOCKED row — the guard reads
// raw for exactly that reason — so the audit row for a refusal has to describe
// that row too. It used to record only the pre-lock snapshot taken before
// UpdateSetting dispatched, because the locked capture happened after the point
// where a refusal returns.
//
// This covers the deterministic half. The stale case needs another replica to
// write between the pre-lock read and the lock, and there is no injection point
// for that in this package, so the fix is structural: the capture is now the
// first statement in the callback and no refusal can return ahead of it.
func TestMCPRefusedRepairAuditsTheCeilingThatCausedIt(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{
		Name: "workspaces/-",
	}))
	a.NoError(err)
	workspaceID := currentWorkspaceID(ctx, t, ctl)

	db, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer db.Close()
	_, err = db.ExecContext(ctx, `
		INSERT INTO setting (name, workspace, value) VALUES ('MCP', $1, '{"capability": "READ_ONLYY"}')
		ON CONFLICT (name, workspace) DO UPDATE SET value = EXCLUDED.value;
	`, workspaceID)
	a.NoError(err)

	// Refused: the toggle alone would marshal the row back without the ceiling.
	err = ctl.setIgnoreMaskingExemptions(ctx, true)
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))

	resp, err := ctl.auditLogServiceClient.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
		Parent:  workspace.Msg.Name,
		Filter:  `method == "/bytebase.v1.SettingService/UpdateSetting"`,
		OrderBy: "create_time desc",
	}))
	a.NoError(err)
	a.NotEmpty(resp.Msg.AuditLogs, "a refused MCP write still belongs in the audit log")

	before := resp.Msg.AuditLogs[0].ServiceData
	a.NotNil(before, "the refusal has to say which ceiling it was protecting")
	setting := &v1pb.Setting{}
	a.NoError(before.UnmarshalTo(setting))
	a.True(setting.GetValue().GetMcp().GetCapabilityUnreadable(),
		"the before-image must show the unreadable ceiling that caused the refusal")
}
