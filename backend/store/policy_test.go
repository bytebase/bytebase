package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGetReviewConfigForDatabasePolicyInheritance(t *testing.T) {
	ctx := context.Background()
	stores := setupPolicyTestStore(ctx, t)

	const (
		workspace   = "default"
		project     = "project-a"
		instance    = "instance-a"
		database    = "database-a"
		environment = "prod"
	)

	reviewConfigRuleTypes := map[string]storepb.SQLReviewRule_Type{}
	for index, id := range []string{"project", "instance", "database", "environment"} {
		ruleType := storepb.SQLReviewRule_Type(index + 1)
		reviewConfigRuleTypes[id] = ruleType
		_, err := stores.CreateReviewConfig(ctx, &store.ReviewConfigMessage{
			ID:        id,
			Workspace: workspace,
			Name:      id,
			Payload: &storepb.ReviewConfigPayload{SqlReviewRules: []*storepb.SQLReviewRule{
				{Type: ruleType},
			}},
		})
		require.NoError(t, err)
	}

	createTagPolicy := func(resourceType storepb.Policy_Resource, resource, reviewConfigID string) *store.PolicyMessage {
		t.Helper()
		policy, err := stores.CreatePolicy(ctx, &store.PolicyMessage{
			Workspace:    workspace,
			ResourceType: resourceType,
			Resource:     resource,
			Type:         storepb.Policy_TAG,
			Payload: fmt.Sprintf(
				`{"tags":{"%s":"%s"}}`,
				common.ReservedTagReviewConfig,
				common.FormatReviewConfig(reviewConfigID),
			),
			Enforce: true,
		})
		require.NoError(t, err)
		return policy
	}

	projectPolicy := createTagPolicy(storepb.Policy_PROJECT, common.FormatProject(project), "project")
	instancePolicy := createTagPolicy(storepb.Policy_INSTANCE, common.FormatProjectInstance(project, instance), "instance")
	databasePolicy := createTagPolicy(storepb.Policy_DATABASE, common.FormatProjectDatabase(project, instance, database), "database")
	environmentPolicy := createTagPolicy(storepb.Policy_ENVIRONMENT, common.FormatEnvironment(environment), "environment")

	environmentID := environment
	db := &store.DatabaseMessage{
		ProjectID:              project,
		InstanceID:             instance,
		DatabaseName:           database,
		EffectiveEnvironmentID: &environmentID,
	}

	assertReviewConfig := func(want string) {
		t.Helper()
		config, err := stores.GetReviewConfigForDatabase(ctx, workspace, db)
		require.NoError(t, err)
		require.Len(t, config.SqlReviewRules, 1)
		require.Equal(t, reviewConfigRuleTypes[want], config.SqlReviewRules[0].Type)
	}

	assertReviewConfig("environment")
	require.NoError(t, stores.DeletePolicy(ctx, environmentPolicy))
	assertReviewConfig("database")
	require.NoError(t, stores.DeletePolicy(ctx, databasePolicy))
	assertReviewConfig("instance")
	require.NoError(t, stores.DeletePolicy(ctx, instancePolicy))
	assertReviewConfig("project")
	require.NoError(t, stores.DeletePolicy(ctx, projectPolicy))
}

func setupPolicyTestStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
		INSERT INTO instance (resource_id, workspace, project) VALUES ('instance-a', 'default', 'project-a');
		INSERT INTO db (instance, name, project) VALUES ('instance-a', 'database-a', 'project-a');
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
