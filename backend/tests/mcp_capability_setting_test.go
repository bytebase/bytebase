package tests

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// setMCPCapability sets the workspace MCP capability ceiling via the settings API.
func (ctl *controller) setMCPCapability(ctx context.Context, capability v1pb.WorkspaceProfileSetting_MCPCapability) error {
	return ctl.updateMCPCapability(ctx, capability, false)
}

func (ctl *controller) updateMCPCapability(ctx context.Context, capability v1pb.WorkspaceProfileSetting_MCPCapability, validateOnly bool) error {
	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		ValidateOnly: validateOnly,
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_WORKSPACE_PROFILE.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceProfile{
					WorkspaceProfile: &v1pb.WorkspaceProfileSetting{
						McpCapability: capability,
					},
				},
			},
		},
		UpdateMask: &fieldmaskpb.FieldMask{
			Paths: []string{"value.workspace_profile.mcp_capability"},
		},
	}))
	return err
}

func (ctl *controller) getMCPCapability(ctx context.Context) (v1pb.WorkspaceProfileSetting_MCPCapability, error) {
	resp, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
		Name: "settings/" + v1pb.Setting_WORKSPACE_PROFILE.String(),
	}))
	if err != nil {
		return v1pb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED, err
	}
	return resp.Msg.Value.GetWorkspaceProfile().GetMcpCapability(), nil
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
	a.Equal(v1pb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED, capability)

	for _, want := range []v1pb.WorkspaceProfileSetting_MCPCapability{
		v1pb.WorkspaceProfileSetting_DISABLED,
		v1pb.WorkspaceProfileSetting_READ_ONLY,
		v1pb.WorkspaceProfileSetting_READ_WRITE,
	} {
		a.NoError(ctl.setMCPCapability(ctx, want), want.String())
		got, err := ctl.getMCPCapability(ctx)
		a.NoError(err, want.String())
		a.Equal(want, got, want.String())
	}

	// Explicit UNSPECIFIED and unknown enum numbers — including the reserved
	// number 2 (was METADATA_ONLY) — are rejected, and the stored value is left
	// untouched.
	for _, invalid := range []v1pb.WorkspaceProfileSetting_MCPCapability{
		v1pb.WorkspaceProfileSetting_MCP_CAPABILITY_UNSPECIFIED,
		v1pb.WorkspaceProfileSetting_MCPCapability(2),
		v1pb.WorkspaceProfileSetting_MCPCapability(99),
	} {
		err = ctl.setMCPCapability(ctx, invalid)
		a.Error(err, invalid.String())
		a.Equal(connect.CodeInvalidArgument, connect.CodeOf(err), invalid.String())
	}
	got, err := ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.WorkspaceProfileSetting_READ_WRITE, got)

	// A validate-only update must not leak into served state: the store caches
	// the profile object, so an in-place mutation would flip the live /mcp gate
	// without persisting anything.
	a.NoError(ctl.setMCPCapability(ctx, v1pb.WorkspaceProfileSetting_DISABLED))
	a.NoError(ctl.updateMCPCapability(ctx, v1pb.WorkspaceProfileSetting_READ_WRITE, true))
	got, err = ctl.getMCPCapability(ctx)
	a.NoError(err)
	a.Equal(v1pb.WorkspaceProfileSetting_DISABLED, got)
}
