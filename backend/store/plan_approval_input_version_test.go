package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestUpdatePlanBumpsApprovalInputVersionOnlyWhenRequested(t *testing.T) {
	ctx := context.Background()
	s := setupPlanApprovalInputVersionStore(ctx, t)

	created, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{Id: "spec-a"}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	require.EqualValues(t, 0, created.Config.GetApprovalInputVersion())

	config := &storepb.PlanConfig{
		ApprovalInputVersion: 99,
		Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-b"}},
	}
	updated, err := s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:                      created.UID,
		ProjectID:                created.ProjectID,
		Config:                   config,
		BumpApprovalInputVersion: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, updated.Config.GetApprovalInputVersion())

	description := "description-only"
	updated, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:         created.UID,
		ProjectID:   created.ProjectID,
		Description: &description,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, updated.Config.GetApprovalInputVersion())

	config = &storepb.PlanConfig{
		ApprovalInputVersion: 99,
		Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-c"}},
	}
	updated, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:                      created.UID,
		ProjectID:                created.ProjectID,
		Config:                   config,
		BumpApprovalInputVersion: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, updated.Config.GetApprovalInputVersion())

	config = &storepb.PlanConfig{
		ApprovalInputVersion: 7,
		Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-d"}},
	}
	updated, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       created.UID,
		ProjectID: created.ProjectID,
		Config:    config,
	})
	require.NoError(t, err)
	require.EqualValues(t, 7, updated.Config.GetApprovalInputVersion())

	config = &storepb.PlanConfig{
		ApprovalInputVersion: 99,
		Specs:                []*storepb.PlanConfig_Spec{{Id: "spec-e"}},
	}
	updated, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:                      created.UID,
		ProjectID:                created.ProjectID,
		Config:                   config,
		BumpApprovalInputVersion: true,
	})
	require.NoError(t, err)
	require.EqualValues(t, 8, updated.Config.GetApprovalInputVersion())
}

func setupPlanApprovalInputVersionStore(ctx context.Context, t *testing.T) *store.Store {
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
