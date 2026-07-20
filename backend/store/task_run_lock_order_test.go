package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCreatePendingTaskRunsDoesNotDeadlockWithProjectDeletion(t *testing.T) {
	err := runWithConcurrentProjectDeletion(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type) VALUES (102, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
		INSERT INTO task (id, project, plan_id, instance, type) VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
	`, "task", 9901, func(ctx context.Context, s *store.Store) error {
		return s.CreatePendingTaskRuns(ctx, "",
			&store.TaskRunMessage{ProjectID: "project-a", TaskUID: 101},
			&store.TaskRunMessage{ProjectID: "project-a", TaskUID: 102},
		)
	})
	require.NoError(t, err)
}

func TestCreatePendingTaskRunsRejectsDeletedProjectDuringProjectDeletion(t *testing.T) {
	operationErr, deleteErr := runWithCreationBeforeProjectDeletion(t, `
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO task (id, project, plan_id, instance, type) VALUES (101, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
	`, "task", func(ctx context.Context, s *store.Store) error {
		return s.CreatePendingTaskRuns(ctx, "", &store.TaskRunMessage{
			ProjectID: "project-a",
			TaskUID:   101,
		})
	})
	require.NoError(t, deleteErr)
	require.Equal(t, common.NotFound, common.ErrorCode(operationErr))
}
