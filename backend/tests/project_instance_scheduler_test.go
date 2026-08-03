package tests

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestProjectInstancePlanCheckSchedulingFollowsProjectLifecycle(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pg, err := provisionPgInstance(ctx, t)
	a.NoError(err)
	const databaseID = "bot36_scheduler_database"
	createPgDatabase(t, pg, databaseID)

	instance := createProjectInstanceTestInstance(ctx, t, ctl, &ctl.project.Name, "bot36-scheduler-instance", "scheduler instance", pg)
	_, err = ctl.instanceServiceClient.SyncInstance(ctx, connect.NewRequest(&v1pb.SyncInstanceRequest{Name: instance.Name}))
	a.NoError(err)
	databaseName := fmt.Sprintf("%s/databases/%s", instance.Name, databaseID)
	_, err = ctl.databaseServiceClient.SyncDatabase(ctx, connect.NewRequest(&v1pb.SyncDatabaseRequest{Name: databaseName}))
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
	}))
	a.NoError(err)
	plan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: ctl.project.Name,
		Plan: &v1pb.Plan{Specs: []*v1pb.Plan_Spec{{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
				Targets: []string{databaseName},
				Sheet:   sheet.Msg.Name,
			}},
		}}},
	}))
	a.NoError(err)
	projectID, planID, err := common.GetProjectIDPlanID(plan.Msg.Name)
	a.NoError(err)
	planCheckName := plan.Msg.Name + "/planCheckRun"
	a.Eventually(func() bool {
		response, err := ctl.planServiceClient.GetPlanCheckRun(ctx, connect.NewRequest(&v1pb.GetPlanCheckRunRequest{Name: planCheckName}))
		return err == nil && (response.Msg.Status == v1pb.PlanCheckRun_DONE || response.Msg.Status == v1pb.PlanCheckRun_FAILED)
	}, 30*time.Second, 100*time.Millisecond, "initial plan check should finish before testing the archive gate")

	_, err = ctl.projectServiceClient.DeleteProject(ctx, connect.NewRequest(&v1pb.DeleteProjectRequest{Name: ctl.project.Name}))
	a.NoError(err)

	metadataDB, err := sql.Open("pgx", ctl.profile.PgURL)
	a.NoError(err)
	defer metadataDB.Close()
	result, err := metadataDB.ExecContext(ctx, `
		UPDATE plan_check_run
		SET status = 'AVAILABLE', updated_at = now()
		WHERE project = $1 AND plan_id = $2
	`, projectID, planID)
	a.NoError(err)
	rowsAffected, err := result.RowsAffected()
	a.NoError(err)
	a.EqualValues(1, rowsAffected)

	time.Sleep(6 * time.Second)
	var status string
	a.NoError(metadataDB.QueryRowContext(ctx, `
		SELECT status
		FROM plan_check_run
		WHERE project = $1 AND plan_id = $2
	`, projectID, planID).Scan(&status))
	a.Equal("AVAILABLE", status, "an archived project's plan check must not be claimed")

	_, err = ctl.projectServiceClient.UndeleteProject(ctx, connect.NewRequest(&v1pb.UndeleteProjectRequest{Name: ctl.project.Name}))
	a.NoError(err)
	_, err = ctl.planServiceClient.RunPlanChecks(ctx, connect.NewRequest(&v1pb.RunPlanChecksRequest{Name: plan.Msg.Name}))
	a.NoError(err)
	a.Eventually(func() bool {
		response, err := ctl.planServiceClient.GetPlanCheckRun(ctx, connect.NewRequest(&v1pb.GetPlanCheckRunRequest{Name: planCheckName}))
		return err == nil && (response.Msg.Status == v1pb.PlanCheckRun_DONE || response.Msg.Status == v1pb.PlanCheckRun_FAILED)
	}, 30*time.Second, 100*time.Millisecond, "restoring the project should allow plan check scheduling to resume")
}
