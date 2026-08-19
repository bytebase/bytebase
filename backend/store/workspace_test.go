package store_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestListWorkspacesByEmailEvaluatesBindingConditions(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

	const email = "member@example.com"
	activeCondition := `request.time < timestamp("2099-01-01T00:00:00Z")`
	expiredCondition := `request.time < timestamp("2000-01-01T00:00:00Z")`
	tests := []struct {
		workspace  string
		memberType string
		condition  string
		want       bool
	}{
		{workspace: "active-direct", memberType: "direct", condition: activeCondition, want: true},
		{workspace: "expired-direct", memberType: "direct", condition: expiredCondition},
		{workspace: "active-group", memberType: "group", condition: activeCondition, want: true},
		{workspace: "expired-group", memberType: "group", condition: expiredCondition},
		{workspace: "active-all-users", memberType: "allUsers", condition: activeCondition, want: true},
		{workspace: "expired-all-users", memberType: "allUsers", condition: expiredCondition},
	}

	var want []string
	for _, test := range tests {
		_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ($1)`, test.workspace)
		require.NoError(t, err)

		member := common.FormatUserEmail(email)
		switch test.memberType {
		case "group":
			group, err := stores.CreateGroup(ctx, &store.GroupMessage{
				Workspace: test.workspace,
				Email:     test.workspace + "@example.com",
				Title:     test.workspace,
				Payload: &storepb.GroupPayload{Members: []*storepb.GroupMember{{
					Member: common.FormatUserEmail(email),
					Role:   storepb.GroupMember_MEMBER,
				}}},
			})
			require.NoError(t, err)
			member = common.FormatGroupEmail(group.Email)
		case "allUsers":
			member = common.AllUsers
		default:
			require.Equal(t, "direct", test.memberType)
		}

		payload, err := protojson.Marshal(&storepb.IamPolicy{Bindings: []*storepb.Binding{{
			Role:      "roles/workspaceMember",
			Members:   []string{member},
			Condition: &expr.Expr{Expression: test.condition},
		}}})
		require.NoError(t, err)
		_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
			Workspace:         test.workspace,
			Resource:          common.FormatWorkspace(test.workspace),
			ResourceType:      storepb.Policy_WORKSPACE,
			Payload:           string(payload),
			Type:              storepb.Policy_IAM,
			InheritFromParent: false,
			Enforce:           true,
		})
		require.NoError(t, err)
		if test.want {
			want = append(want, test.workspace)
		}
	}

	workspaces, err := stores.ListWorkspacesByEmail(ctx, &store.FindWorkspaceMessage{
		Email:          email,
		IncludeAllUser: true,
	})
	require.NoError(t, err)
	got := make([]string, 0, len(workspaces))
	for _, workspace := range workspaces {
		got = append(got, workspace.ResourceID)
	}
	slices.Sort(got)
	slices.Sort(want)
	require.Equal(t, want, got)

	for _, test := range tests {
		workspace, err := stores.FindWorkspace(ctx, &store.FindWorkspaceMessage{
			WorkspaceID:    &test.workspace,
			Email:          email,
			IncludeAllUser: true,
		})
		require.NoError(t, err)
		if test.want {
			require.NotNil(t, workspace)
		} else {
			require.Nil(t, workspace)
		}
	}
}
