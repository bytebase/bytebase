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

func TestPlanServiceListPlansHidesMalformedUIPlans(t *testing.T) {
	ctx := context.Background()
	stores := setupPlanServiceTestStore(ctx, t)
	service := NewPlanService(stores, nil, nil, nil, nil)

	createPlan := func(name string, config *storepb.PlanConfig) *store.PlanMessage {
		t.Helper()
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: "project-a",
			Name:      name,
			Config:    config,
		}, "creator@example.com")
		require.NoError(t, err)
		return plan
	}
	changeConfig := func(id, release string) *storepb.PlanConfig {
		return &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: id,
			Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{Release: release},
			},
		}}}
	}
	createConfig := func(id string) *storepb.PlanConfig {
		return &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
			Id: id,
			Config: &storepb.PlanConfig_Spec_CreateDatabaseConfig{
				CreateDatabaseConfig: &storepb.PlanConfig_CreateDatabaseConfig{},
			},
		}}}
	}

	createPlan("malformed change", changeConfig("malformed-change", ""))
	createPlan("malformed create", createConfig("malformed-create"))
	createPlan("malformed mixed", &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{
		createConfig("mixed-create").Specs[0],
		changeConfig("mixed-change", "").Specs[0],
	}})
	oldMalformed := createPlan("old malformed", changeConfig("old", ""))
	_, err := stores.GetDB().ExecContext(ctx, `
		UPDATE plan SET created_at = CURRENT_TIMESTAMP - INTERVAL '31 days'
		WHERE project = $1 AND id = $2`, oldMalformed.ProjectID, oldMalformed.UID)
	require.NoError(t, err)
	gitOps := createPlan("GitOps", changeConfig("gitops", "projects/project-a/releases/release-a"))
	export := createPlan("export", &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
		Id: "export",
		Config: &storepb.PlanConfig_Spec_ExportDataConfig{
			ExportDataConfig: &storepb.PlanConfig_ExportDataConfig{},
		},
	}}})
	deleted := createPlan("deleted", changeConfig("deleted", ""))
	_, err = stores.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID: deleted.UID, ProjectID: deleted.ProjectID, Deleted: new(true),
	})
	require.NoError(t, err)
	linked := createPlan("linked", changeConfig("linked", ""))
	_, err = stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID: linked.ProjectID, CreatorEmail: "creator@example.com", PlanUID: &linked.UID,
		Title: "linked issue", Type: storepb.Issue_DATABASE_CHANGE, Payload: &storepb.Issue{},
	})
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
	require.ElementsMatch(t, []string{gitOps.Name, export.Name, deleted.Name, linked.Name}, got)

	gotMalformed, err := service.GetPlan(ctx, connect.NewRequest(&v1pb.GetPlanRequest{
		Name: fmt.Sprintf("projects/project-a/plans/%d", oldMalformed.UID),
	}))
	require.NoError(t, err)
	require.Equal(t, oldMalformed.Name, gotMalformed.Msg.Title)
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
