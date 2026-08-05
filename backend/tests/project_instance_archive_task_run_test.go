package tests

import (
	"context"
	"database/sql"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

//nolint:tparallel // Subtests mutate one task run through successive lifecycle states.
func TestArchiveInstanceRejectsActiveTaskRuns(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instance := createProjectInstanceTestInstance(
		ctx,
		t,
		ctl,
		&ctl.project.Name,
		"archive-active-task-runs",
		"archive active task runs",
		pg,
	)
	projectID, err := common.GetProjectID(ctl.project.Name)
	a.NoError(err)
	_, instanceID, err := common.GetInstanceResourceName(instance.Name)
	a.NoError(err)

	metadataDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer metadataDB.Close()
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO plan (id, creator, project, name, description)
			VALUES (101, 'creator@example.com', $1, 'Archive precondition plan', '')
	`, projectID)
	a.NoError(err)
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO task (id, project, plan_id, instance, type)
			VALUES (101, $1, 101, $2, 'DATABASE_CREATE')
	`, projectID, instanceID)
	a.NoError(err)
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO task_run (id, project, task_id, attempt, status)
			VALUES (101, $1, 101, 0, 'PENDING')
	`, projectID)
	a.NoError(err)

	for _, status := range []string{"PENDING", "AVAILABLE", "RUNNING"} {
		t.Run(status, func(t *testing.T) {
			_, err := metadataDB.ExecContext(ctx, `
				UPDATE task_run SET status = $1 WHERE project = $2 AND id = 101
			`, status, projectID)
			require.NoError(t, err)

			_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{
				Name: instance.Name,
			}))
			require.Error(t, err)
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

			got, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
			require.NoError(t, err)
			require.Equal(t, v1pb.State_ACTIVE, got.Msg.State)
		})
	}

	_, err = metadataDB.ExecContext(ctx, `
		UPDATE task_run SET status = 'CANCELED' WHERE project = $1 AND id = 101
	`, projectID)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{
		Name: instance.Name,
	}))
	a.NoError(err)
	archived, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	a.Equal(v1pb.State_DELETED, archived.Msg.State)

	// Legacy data may contain an active task run on an already archived
	// instance. Restoration must not make that work runnable without an
	// explicit cancel-and-retry decision from the user.
	_, err = metadataDB.ExecContext(ctx, `
		UPDATE task_run SET status = 'PENDING' WHERE project = $1 AND id = 101
	`, projectID)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{
		Name: instance.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	_, err = ctl.instanceServiceClient.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{
		Name: instance.Name,
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	stillArchived, err := ctl.instanceServiceClient.GetInstance(ctx, connect.NewRequest(&v1pb.GetInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	a.Equal(v1pb.State_DELETED, stillArchived.Msg.State)

	_, err = metadataDB.ExecContext(ctx, `
		UPDATE task_run SET status = 'CANCELED' WHERE project = $1 AND id = 101
	`, projectID)
	a.NoError(err)
	restored, err := ctl.instanceServiceClient.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{
		Name: instance.Name,
	}))
	a.NoError(err)
	a.Equal(v1pb.State_ACTIVE, restored.Msg.State)
}

func TestForceArchiveInstanceRejectsActiveTaskRunsWithoutTransferringDatabases(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	instance := createProjectInstanceTestInstance(
		ctx,
		t,
		ctl,
		nil,
		"force-archive-active-task-runs",
		"force archive active task runs",
		pg,
	)
	projectID, err := common.GetProjectID(ctl.project.Name)
	a.NoError(err)
	_, instanceID, err := common.GetInstanceResourceName(instance.Name)
	a.NoError(err)

	metadataDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer metadataDB.Close()
	var workspaceID string
	a.NoError(metadataDB.QueryRowContext(ctx, `
		SELECT workspace FROM instance WHERE resource_id = $1
	`, instanceID).Scan(&workspaceID))
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO db (instance, name, project) VALUES ($1, 'db-a', $2)
	`, instanceID, projectID)
	a.NoError(err)
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO plan (id, creator, project, name, description)
			VALUES (101, 'creator@example.com', $1, 'Force archive precondition plan', '')
	`, projectID)
	a.NoError(err)
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO task (id, project, plan_id, instance, type)
			VALUES (101, $1, 101, $2, 'DATABASE_CREATE')
	`, projectID, instanceID)
	a.NoError(err)
	_, err = metadataDB.ExecContext(ctx, `
		INSERT INTO task_run (id, project, task_id, attempt, status)
			VALUES (101, $1, 101, 0, 'RUNNING')
	`, projectID)
	a.NoError(err)

	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{
		Name:  instance.Name,
		Force: true,
	}))
	a.Error(err)
	a.Equal(connect.CodeFailedPrecondition, connect.CodeOf(err))
	var databaseProjectID string
	a.NoError(metadataDB.QueryRowContext(ctx, `
		SELECT project FROM db WHERE instance = $1 AND name = 'db-a'
	`, instanceID).Scan(&databaseProjectID))
	a.Equal(projectID, databaseProjectID, "a rejected force archive must not transfer databases")

	_, err = metadataDB.ExecContext(ctx, `
		UPDATE task_run SET status = 'CANCELED' WHERE project = $1 AND id = 101
	`, projectID)
	a.NoError(err)
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{
		Name:  instance.Name,
		Force: true,
	}))
	a.NoError(err)
	a.NoError(metadataDB.QueryRowContext(ctx, `
		SELECT project FROM db WHERE instance = $1 AND name = 'db-a'
	`, instanceID).Scan(&databaseProjectID))
	a.True(common.IsDefaultProject(workspaceID, databaseProjectID))
}
