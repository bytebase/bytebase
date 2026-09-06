package v1

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// mcpSettingRead is one answer from the MCP settings reader.
type mcpSettingRead struct {
	setting *storepb.MCPSetting
	err     error
}

func (r mcpSettingRead) GetMCPSettingsUncached(context.Context, string) (*storepb.MCPSetting, error) {
	return r.setting, r.err
}

// TestActuatorMCPSetting pins the three decisions in what actuator info
// discloses about the MCP policy: the masking toggle travels with the ceiling,
// an unreadable row leaves the setting absent instead of failing the shared
// bootstrap response, and the resolution the gate already made for this
// request wins over a second read.
func TestActuatorMCPSetting(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		read mcpSettingRead
		ctx  context.Context
		want *v1pb.MCPSetting
	}{
		{
			// Every masking state is written in terms of this, so withholding it
			// leaves the table unusable.
			name: "the toggle reaches the response",
			read: mcpSettingRead{setting: &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_ONLY, IgnoreMaskingExemptions: true}},
			ctx:  ctx,
			want: &v1pb.MCPSetting{Capability: v1pb.MCPSetting_READ_ONLY, IgnoreMaskingExemptions: true},
		},
		{
			// Actuator info is a shared bootstrap response, so it remains
			// available and leaves the optional setting absent; the policy page
			// then withholds editing rather than showing a guessed ceiling.
			name: "a row that does not unmarshal leaves the setting absent",
			read: mcpSettingRead{err: errors.New("failed to unmarshal setting MCP")},
			ctx:  ctx,
			want: nil,
		},
		{
			// The stored ceiling says DISABLED; the gate admitted this request
			// under READ_ONLY. Answering from the store would report a ceiling
			// the request was not admitted under.
			name: "the gate's resolution wins over a second read",
			read: mcpSettingRead{setting: &storepb.MCPSetting{Capability: storepb.MCPSetting_DISABLED}},
			ctx:  withMCPSettings(ctx, &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_ONLY, IgnoreMaskingExemptions: true}),
			want: &v1pb.MCPSetting{Capability: v1pb.MCPSetting_READ_ONLY, IgnoreMaskingExemptions: true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := actuatorMCPSetting(tc.ctx, tc.read, "ws-test")
			if tc.want == nil {
				require.Nil(t, got)
				return
			}
			require.Equal(t, tc.want.Capability, got.GetCapability())
			require.Equal(t, tc.want.IgnoreMaskingExemptions, got.GetIgnoreMaskingExemptions())
		})
	}
}
