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

func TestPlanServiceListPlansHidesIssueLessDatabaseDrafts(t *testing.T) {
	ctx := context.Background()
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	changeDraft, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "change draft",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: "change",
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)
	createDraft, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "create draft",
		Config: &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: "create",
			Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{},
			},
		}}},
	}, "creator@example.com")
	require.NoError(t, err)

	response, err := service.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
		Parent:   "projects/project-a",
		PageSize: 100,
	}))
	require.NoError(t, err)
	var got []string
	for _, plan := range response.Msg.Plans {
		got = append(got, plan.Title)
	}
	require.Empty(t, got)
	for _, plan := range []*store.PlanMessage{changeDraft, createDraft} {
		gotPlan, err := service.GetPlan(ctx, connect.NewRequest(&v1pb.GetPlanRequest{
			Name: fmt.Sprintf("projects/project-a/plans/%d", plan.UID),
		}))
		require.NoError(t, err)
		require.Equal(t, plan.Name, gotPlan.Msg.Title)
	}
}

func TestPlanServiceCreatePlanRejectsMixedDatabaseSpecs(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	ctx = context.WithValue(ctx, common.UserContextKey, &store.UserMessage{
		Email: "creator@example.com",
		Name:  "creator",
	})
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	_, err := service.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: "projects/project-a",
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{
				{
					Id: "create",
					Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
						CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{},
					},
				},
				{
					Id: "change",
					Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{},
					},
				},
			},
		},
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.ErrorContains(t, err, "each plan must contain only one type")
}

func setupPlanServiceTestStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}
