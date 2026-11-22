package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGetSettingForMCPSessions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const salt = "mcp-test-salt-7f3a"
	// With the cache on, a read that blanked the cached value would show in the
	// console read that follows it.
	db, stores, _ := testcontainer.NewMetadataDBWithCache(t, true)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default');`)
	require.NoError(t, err)
	for _, setting := range []*store.SettingMessage{
		{Name: storepb.SettingName_WORKSPACE_APPROVAL, Value: &storepb.WorkspaceApprovalSetting{Rules: []*storepb.WorkspaceApprovalSetting_Rule{{
			Source:    storepb.WorkspaceApprovalSetting_Rule_CHANGE_DATABASE,
			Condition: &expr.Expr{Expression: "true"},
			Template:  &storepb.ApprovalTemplate{Title: "Fallback Rule", Flow: &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}}},
		}}}},
		{Name: storepb.SettingName_ENVIRONMENT, Value: &storepb.EnvironmentSetting{Environments: []*storepb.EnvironmentSetting_Environment{
			{Id: "prod", Title: "Prod"},
		}}},
		{Name: storepb.SettingName_SEMANTIC_TYPES, Value: &storepb.SemanticTypeSetting{Types: []*storepb.SemanticTypeSetting_SemanticType{
			{Id: "hashed", Title: "Hashed", Algorithm: &storepb.Algorithm{Mask: &storepb.Algorithm_Md5Mask{Md5Mask: &storepb.Algorithm_MD5Mask{Salt: salt}}}},
			{Id: "full", Title: "Full", Algorithm: &storepb.Algorithm{Mask: &storepb.Algorithm_FullMask_{FullMask: &storepb.Algorithm_FullMask{Substitution: "***"}}}},
		}}},
		{Name: storepb.SettingName_DATA_CLASSIFICATION, Value: &storepb.DataClassificationSetting{Configs: []*storepb.DataClassificationSetting_DataClassificationConfig{
			{Id: "default", Title: "Default"},
		}}},
	} {
		setting.Workspace = "default"
		_, err := stores.UpsertSetting(ctx, setting)
		require.NoError(t, err)
	}
	for email, role := range map[string]string{
		"admin@example.com":  "roles/workspaceAdmin",
		"dba@example.com":    "roles/workspaceDBA",
		"member@example.com": "roles/workspaceMember",
	} {
		_, err := stores.PatchIamPolicy(ctx, "default", storepb.Policy_WORKSPACE, common.FormatWorkspace("default"), func(policy *storepb.IamPolicy) error {
			policy.Bindings = append(policy.Bindings, &storepb.Binding{Role: role, Members: []string{common.FormatUserEmail(email)}})
			return nil
		})
		require.NoError(t, err)
	}
	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	service := &SettingService{store: stores, iamManager: iamManager}

	get := func(email string, mcp bool, name v1pb.Setting_SettingName) (*v1pb.Setting, error) {
		authCtx := &common.AuthContext{}
		if mcp {
			authCtx.DelegatedGrant = &common.DelegatedGrant{}
		}
		callerCtx := context.WithValue(ctx, common.AuthContextKey, authCtx)
		callerCtx = context.WithValue(callerCtx, common.WorkspaceIDContextKey, "default")
		callerCtx = context.WithValue(callerCtx, common.UserContextKey, &store.UserMessage{Email: email})
		resp, err := service.GetSetting(callerCtx, connect.NewRequest(&v1pb.GetSettingRequest{Name: common.SettingNamePrefix + name.String()}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}

	t.Run("each reader holds the permission the console asks for", func(t *testing.T) {
		t.Parallel()
		for _, mcp := range []bool{true, false} {
			for _, name := range []v1pb.Setting_SettingName{
				v1pb.Setting_WORKSPACE_APPROVAL, v1pb.Setting_ENVIRONMENT, v1pb.Setting_SEMANTIC_TYPES, v1pb.Setting_DATA_CLASSIFICATION,
			} {
				for _, email := range []string{"admin@example.com", "dba@example.com"} {
					_, err := get(email, mcp, name)
					require.NoError(t, err, "%s reading %v (MCP: %v)", email, name, mcp)
				}
				_, err := get("member@example.com", mcp, name)
				if name == v1pb.Setting_ENVIRONMENT {
					require.NoError(t, err, "a workspace member reads the environments (MCP: %v)", mcp)
					continue
				}
				require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "member reading %v (MCP: %v)", name, mcp)
				require.ErrorContains(t, err, `user does not have permission "bb.settings.get"`)
			}
		}
	})

	t.Run("an MCP session never receives the masking salt", func(t *testing.T) {
		t.Parallel()
		setting, err := get("admin@example.com", true, v1pb.Setting_SEMANTIC_TYPES)
		require.NoError(t, err)
		types := setting.GetValue().GetSemanticType().GetTypes()
		require.Len(t, types, 2)
		require.NotNil(t, types[0].GetAlgorithm().GetMd5Mask(), "the MD5 mask stays visible, without its salt")
		require.Empty(t, types[0].GetAlgorithm().GetMd5Mask().GetSalt())
		require.Equal(t, "***", types[1].GetAlgorithm().GetFullMask().GetSubstitution(), "other masks are not secret")
		serialized, err := protojson.Marshal(setting)
		require.NoError(t, err)
		require.NotContains(t, string(serialized), salt)

		console, err := get("admin@example.com", false, v1pb.Setting_SEMANTIC_TYPES)
		require.NoError(t, err)
		require.Equal(t, salt, console.GetValue().GetSemanticType().GetTypes()[0].GetAlgorithm().GetMd5Mask().GetSalt(),
			"the console's semantic-type editor writes the salt back, so it still reads it")
	})
}
