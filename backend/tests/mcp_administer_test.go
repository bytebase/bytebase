package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestMCPAdministersTheApprovalRulesAfterAConfirmation drives
// administer_workspace over a live session: call_api is refused, the tool asks
// once per change, and the rules the user confirmed are stored. The second
// change runs over a stored rule with no approvers, so the rules the tool shows
// and the rules the handler locks must agree for that shape too.
func TestMCPAdministersTheApprovalRulesAfterAConfirmation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl, ctx := startWorkspace(ctx, t)
	a.NoError(ctl.setMCPCapability(ctx, v1pb.MCPSetting_READ_WRITE))

	var asked atomic.Int32
	session := openMCPSessionWithOptions(ctx, t, ctl, ctl.authInterceptor.token, &mcp.ClientOptions{
		ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			asked.Add(1)
			return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"choice": "apply"}}, nil
		},
	})
	defer session.Close()

	const approvalName = "settings/WORKSPACE_APPROVAL"
	stored := func() []*v1pb.WorkspaceApprovalSetting_Rule {
		setting, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{Name: approvalName}))
		a.NoError(err)
		return setting.Msg.GetValue().GetWorkspaceApproval().GetRules()
	}
	body := func(rules []*v1pb.WorkspaceApprovalSetting_Rule) map[string]any {
		encoded, err := protojson.Marshal(&v1pb.UpdateSettingRequest{Setting: &v1pb.Setting{
			Name:  approvalName,
			Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_WorkspaceApproval{WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{Rules: rules}}},
		}})
		a.NoError(err)
		var decoded map[string]any
		a.NoError(json.Unmarshal(encoded, &decoded))
		return decoded
	}
	rule := func(title, condition string, roles ...string) *v1pb.WorkspaceApprovalSetting_Rule {
		return &v1pb.WorkspaceApprovalSetting_Rule{
			Source:    v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
			Condition: &expr.Expr{Expression: condition},
			Template:  &v1pb.ApprovalTemplate{Title: title, Flow: &v1pb.ApprovalFlow{Roles: roles}},
		}
	}
	administer := func(rules []*v1pb.WorkspaceApprovalSetting_Rule) (bool, string) {
		return callToolOnSession(ctx, t, session, "administer_workspace", map[string]any{
			"operationId": "SettingService/UpdateSetting",
			"body":        body(rules),
		})
	}

	seeded := stored()
	a.NotEmpty(seeded, "a new workspace stores the Fallback Rule")
	withTest := append(append([]*v1pb.WorkspaceApprovalSetting_Rule{}, seeded...), rule("Test", `resource.environment_id == "test"`))

	direct := callAPIOnSession(ctx, t, session, "SettingService/UpdateSetting", body(withTest))
	a.Equal(http.StatusForbidden, direct.Status, direct.Error)
	a.Contains(direct.Error, "administer_workspace")
	a.Len(stored(), len(seeded), "call_api changed nothing")

	failed, text := administer(withTest)
	a.False(failed, text)
	a.Contains(text, "Applied")
	a.Equal(int32(1), asked.Load())
	a.Len(stored(), len(withTest))

	withProd := append(stored(), rule("Prod", `resource.environment_id == "prod"`, "roles/workspaceDBA"))
	failed, text = administer(withProd)
	a.False(failed, text)
	a.Contains(text, "Applied")
	a.Equal(int32(2), asked.Load())
	after := stored()
	a.Len(after, len(withProd))
	a.Equal(`resource.environment_id == "prod"`, after[len(after)-1].GetCondition().GetExpression())
}
