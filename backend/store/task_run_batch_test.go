package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testpg"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCreatePendingTaskRunsRejectsMultipleProjects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, s, _ := testpg.New(t)

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-a', 'default', 'Project A'),
			('project-b', 'default', 'Project B');
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description) VALUES
			(101, 'creator@example.com', 'project-a', 'Plan A', ''),
			(101, 'creator@example.com', 'project-b', 'Plan B', '');
		INSERT INTO task (id, project, plan_id, instance, type) VALUES
			(101, 'project-a', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE'),
			(101, 'project-b', 101, 'instance-a', 'DATABASE_SCHEMA_UPDATE');
	`)
	require.NoError(t, err)

	err = s.CreatePendingTaskRuns(ctx, "creator@example.com",
		&store.TaskRunMessage{ProjectID: "project-a", TaskUID: 101},
		&store.TaskRunMessage{ProjectID: "project-b", TaskUID: 101},
	)
	require.Equal(t, common.Invalid, common.ErrorCode(err))
	require.ErrorContains(t, err, "same project")

	for _, projectID := range []string{"project-a", "project-b"} {
		taskRuns, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: projectID})
		require.NoError(t, err)
		require.Empty(t, taskRuns)
	}
}
