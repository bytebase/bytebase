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

// TestMCPAdministersTheMaskingRulesAfterAConfirmation drives administer_workspace's
// masking entry over a live session: call_api is refused, the first save on a
// workspace with no masking rules creates the policy through allowMissing, and
// a second change over the stored rules is stored as confirmed.
func TestMCPAdministersTheMaskingRulesAfterAConfirmation(t *testing.T) {
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

	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{Name: "workspaces/-"}))
	a.NoError(err)
	policyName := workspace.Msg.GetName() + "/policies/masking_rule"
	stored := func() ([]*v1pb.MaskingRulePolicy_MaskingRule, bool) {
		policy, err := ctl.orgPolicyServiceClient.GetPolicy(ctx, connect.NewRequest(&v1pb.GetPolicyRequest{Name: policyName}))
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil, false
		}
		a.NoError(err)
		return policy.Msg.GetMaskingRulePolicy().GetRules(), true
	}
	body := func(allowMissing bool, rules ...*v1pb.MaskingRulePolicy_MaskingRule) map[string]any {
		encoded, err := protojson.Marshal(&v1pb.UpdatePolicyRequest{
			Policy: &v1pb.Policy{Name: policyName, Type: v1pb.PolicyType_MASKING_RULE,
				Policy: &v1pb.Policy_MaskingRulePolicy{MaskingRulePolicy: &v1pb.MaskingRulePolicy{Rules: rules}}},
			AllowMissing: allowMissing,
		})
		a.NoError(err)
		var decoded map[string]any
		a.NoError(json.Unmarshal(encoded, &decoded))
		return decoded
	}
	rule := func(id, title, condition, semanticType string) *v1pb.MaskingRulePolicy_MaskingRule {
		return &v1pb.MaskingRulePolicy_MaskingRule{Id: id, Condition: &expr.Expr{Expression: condition, Title: title}, SemanticType: semanticType}
	}
	administer := func(body map[string]any) (bool, string) {
		return callToolOnSession(ctx, t, session, "administer_workspace", map[string]any{
			"operationId": "OrgPolicyService/UpdatePolicy",
			"body":        body,
		})
	}

	_, found := stored()
	a.False(found, "a new workspace stores no masking rules")
	prod := rule("6f1d2c3b-4a5e-4f60-8b7c-9d0e1f2a3b4c", "Personal data in prod",
		`resource.environment_id == "prod" && resource.column_name in ["birth_date", "gender", "amount"]`, "bb.default")

	direct := callAPIOnSession(ctx, t, session, "OrgPolicyService/UpdatePolicy", body(true, prod))
	a.Equal(http.StatusForbidden, direct.Status, direct.Error)
	a.Contains(direct.Error, "administer_workspace")
	_, found = stored()
	a.False(found, "call_api changed nothing")

	failed, text := administer(body(true, prod))
	a.False(failed, text)
	a.Contains(text, "Applied")
	a.Equal(int32(1), asked.Load())
	rules, found := stored()
	a.True(found, "the first save creates the policy")
	a.Len(rules, 1)
	a.Equal("bb.default", rules[0].GetSemanticType())

	test := rule("0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d", "Test", `resource.environment_id == "test"`, "bb.default-partial")
	failed, text = administer(body(false, append(rules, test)...))
	a.False(failed, text)
	a.Contains(text, "Applied")
	a.Equal(int32(2), asked.Load())
	rules, _ = stored()
	a.Len(rules, 2)
	a.Equal(`resource.environment_id == "test"`, rules[1].GetCondition().GetExpression())
}
