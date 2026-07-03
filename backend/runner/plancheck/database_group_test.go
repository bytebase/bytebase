package plancheck

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

func TestHasDatabaseGroupTarget(t *testing.T) {
	tests := []struct {
		name  string
		specs []*storepb.PlanConfig_Spec
		want  bool
	}{
		{
			name: "direct database target",
			specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"instances/prod/databases/app"},
					},
				},
			}},
		},
		{
			name: "multiple targets are not a database group target",
			specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"projects/project-a/databaseGroups/group", "instances/prod/databases/app"},
					},
				},
			}},
		},
		{
			name: "database group target",
			specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"projects/project-a/databaseGroups/group"},
					},
				},
			}},
			want: true,
		},
		{
			name: "export data group target does not require plan check group expansion",
			specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ExportDataConfig{
					ExportDataConfig: &storepb.PlanConfig_ExportDataConfig{
						Targets: []string{"projects/project-a/databaseGroups/group"},
					},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, HasDatabaseGroupTarget(tt.specs))
		})
	}
}

func TestGetDatabaseGroupForPlanMatchesDatabases(t *testing.T) {
	ctx := context.Background()
	s := setupPlancheckStore(ctx, t)
	setupPlancheckDatabaseGroupFixture(ctx, t, s)

	target := "projects/project-a/databaseGroups/group"
	got, err := GetDatabaseGroupForPlan(ctx, s, &store.PlanMessage{
		ProjectID: "project-a",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{target},
					},
				},
			}},
		},
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, target, got.Name)
	require.Equal(t, []string{common.FormatDatabase("prod", "app")}, databaseNames(got.MatchedDatabases))
}

func setupPlancheckStore(ctx context.Context, t *testing.T) *store.Store {
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
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}

func setupPlancheckDatabaseGroupFixture(ctx context.Context, t *testing.T, s *store.Store) {
	t.Helper()

	_, err := s.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "default",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	for _, databaseName := range []string{"app", "audit"} {
		_, err = s.UpsertDatabase(ctx, &store.DatabaseMessage{
			ProjectID:    "project-a",
			InstanceID:   "prod",
			DatabaseName: databaseName,
			Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
		})
		require.NoError(t, err)
	}
	_, err = s.CreateDatabaseGroup(ctx, &store.DatabaseGroupMessage{
		ProjectID:  "project-a",
		ResourceID: "group",
		Title:      "group",
		Expression: &expr.Expr{Expression: `resource.database_name == "app"`},
	})
	require.NoError(t, err)
}

func databaseNames(databases []*v1pb.DatabaseGroup_Database) []string {
	var names []string
	for _, database := range databases {
		names = append(names, database.Name)
	}
	return names
}
