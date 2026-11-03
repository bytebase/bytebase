package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

func TestConfirmedApprovalRulesReplaceOnlyTheRulesShown(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default');`)
	require.NoError(t, err)
	_, err = stores.PatchIamPolicy(ctx, "default", storepb.Policy_WORKSPACE, common.FormatWorkspace("default"), func(policy *storepb.IamPolicy) error {
		policy.Bindings = append(policy.Bindings, &storepb.Binding{Role: "roles/workspaceAdmin", Members: []string{common.FormatUserEmail("admin@example.com")}})
		return nil
	})
	require.NoError(t, err)
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	service := &SettingService{store: stores, profile: &config.Profile{}, licenseService: newEnterpriseLicenseService(ctx, t, stores, "default"), iamManager: iamManager}

	rule := func(title, expression string) *storepb.WorkspaceApprovalSetting_Rule {
		return &storepb.WorkspaceApprovalSetting_Rule{
			Source:    storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
			Condition: &expr.Expr{Expression: expression},
			Template:  &storepb.ApprovalTemplate{Title: title, Flow: &storepb.ApprovalFlow{Roles: []string{"roles/workspaceDBA"}}},
		}
	}
	save := func(rules ...*storepb.WorkspaceApprovalSetting_Rule) {
		t.Helper()
		_, err := stores.UpsertSetting(ctx, &store.SettingMessage{Name: storepb.SettingName_WORKSPACE_APPROVAL, Workspace: "default", Value: &storepb.WorkspaceApprovalSetting{Rules: rules}})
		require.NoError(t, err)
	}
	shownDigest := func() []byte {
		t.Helper()
		setting, err := stores.GetSetting(ctx, "default", storepb.SettingName_WORKSPACE_APPROVAL)
		require.NoError(t, err)
		shown, err := convertToSettingMessage(setting)
		require.NoError(t, err)
		return auth.MCPValueDigest(shown.GetValue().GetWorkspaceApproval())
	}
	stored := func() *storepb.WorkspaceApprovalSetting {
		t.Helper()
		setting, err := stores.GetWorkspaceApprovalSetting(ctx, "default")
		require.NoError(t, err)
		return setting
	}

	callerCtx := context.WithValue(ctx, common.WorkspaceIDContextKey, "default")
	callerCtx = context.WithValue(callerCtx, common.UserContextKey, &store.UserMessage{Email: "admin@example.com"})
	request := &v1pb.UpdateSettingRequest{Setting: &v1pb.Setting{
		Name: common.SettingNamePrefix + v1pb.Setting_WORKSPACE_APPROVAL.String(),
		Value: &v1pb.SettingValue{Value: &v1pb.SettingValue_WorkspaceApproval{WorkspaceApproval: &v1pb.WorkspaceApprovalSetting{Rules: []*v1pb.WorkspaceApprovalSetting_Rule{{
			Source:    v1pb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
			Condition: &expr.Expr{Expression: `resource.environment_id == "prod"`},
			Template:  &v1pb.ApprovalTemplate{Title: "Prod", Flow: &v1pb.ApprovalFlow{Roles: []string{"roles/workspaceDBA"}}},
		}}}}},
	}}
	confirm := func(digest []byte) error {
		confirmedCtx := auth.WithMCPConfirmedDecision(callerCtx, auth.MCPConfirmedDecision{
			Procedure:    v1connect.SettingServiceUpdateSettingProcedure,
			Resource:     request.Setting.Name,
			StoredDigest: digest,
		})
		_, err := service.UpdateSetting(confirmedCtx, connect.NewRequest(request))
		return err
	}

	save(rule("Test", `resource.environment_id == "test"`))
	asked := shownDigest()
	// Another admin edits the rules after the user was asked.
	edited := []*storepb.WorkspaceApprovalSetting_Rule{rule("Test", `resource.environment_id == "test"`), rule("Staging", `resource.environment_id == "staging"`)}
	save(edited...)

	err = confirm(asked)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), err)
	require.True(t, proto.Equal(&storepb.WorkspaceApprovalSetting{Rules: edited}, stored()), "the other admin's edit stands")

	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(confirm(nil)), "a confirmation with no digest matches no rules")

	require.NoError(t, confirm(shownDigest()))
	require.Len(t, stored().GetRules(), 1)
	require.Equal(t, `resource.environment_id == "prod"`, stored().GetRules()[0].GetCondition().GetExpression())
}
