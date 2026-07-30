package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestOrgPolicyServiceListsInstanceAndDatabaseTagPolicies(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	stores := setupOrgPolicyServiceTestStore(ctx, t)
	service := NewOrgPolicyService(stores, nil, nil)

	for _, test := range []struct {
		name             string
		resourceType     storepb.Policy_Resource
		wantResourceType v1pb.PolicyResourceType
		parent           string
	}{
		{
			name:             "instance",
			resourceType:     storepb.Policy_INSTANCE,
			wantResourceType: v1pb.PolicyResourceType_INSTANCE,
			parent:           "projects/project-a/instances/instance-a",
		},
		{
			name:             "database",
			resourceType:     storepb.Policy_DATABASE,
			wantResourceType: v1pb.PolicyResourceType_DATABASE,
			parent:           "projects/project-a/instances/instance-a/databases/database-a",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := stores.CreatePolicy(ctx, &store.PolicyMessage{
				Workspace:    "default",
				ResourceType: test.resourceType,
				Resource:     test.parent,
				Type:         storepb.Policy_TAG,
				Payload:      `{"tags":{}}`,
				Enforce:      true,
			})
			require.NoError(t, err)

			response, err := service.ListPolicies(ctx, connect.NewRequest(&v1pb.ListPoliciesRequest{
				Parent: test.parent,
			}))
			require.NoError(t, err)
			require.Len(t, response.Msg.Policies, 1)
			require.Equal(t, fmt.Sprintf("%s/%stag", test.parent, common.PolicyNamePrefix), response.Msg.Policies[0].Name)
			require.Equal(t, test.wantResourceType, response.Msg.Policies[0].ResourceType)
		})
	}
}

func setupOrgPolicyServiceTestStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })
	return stores
}
