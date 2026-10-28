package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSearchDatabases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name, deleted) VALUES
			('project-a', 'default', 'Project A', FALSE),
			('project-b', 'default', 'Project B', FALSE),
			('project-archived', 'default', 'Archived', TRUE);
		INSERT INTO instance (resource_id, workspace) VALUES ('prod', 'default'), ('staging', 'default');
		INSERT INTO db (instance, name, project) VALUES
			('prod', 'a1', 'project-a'),
			('prod', 'a2', 'project-a'),
			('prod', 'b1', 'project-b'),
			('staging', 'a1', 'project-b'),
			('prod', 'archived1', 'project-archived');
	`)
	require.NoError(t, err)
	for role, permissions := range map[string][]permission.Permission{
		"database-lister": {permission.DatabasesList},
		"database-getter": {permission.ProjectsGet, permission.DatabasesGet},
	} {
		granted := map[permission.Permission]bool{}
		for _, p := range permissions {
			granted[p] = true
		}
		_, err := stores.CreateRole(ctx, &store.RoleMessage{ResourceID: role, Workspace: "default", Name: role, Permissions: granted})
		require.NoError(t, err)
	}
	grant := func(resourceType storepb.Policy_Resource, resource, email, role string) {
		_, err := stores.PatchIamPolicy(ctx, "default", resourceType, resource, func(policy *storepb.IamPolicy) error {
			policy.Bindings = append(policy.Bindings, &storepb.Binding{Role: role, Members: []string{common.FormatUserEmail(email)}})
			return nil
		})
		require.NoError(t, err)
	}
	grant(storepb.Policy_WORKSPACE, common.FormatWorkspace("default"), "admin@example.com", "roles/workspaceAdmin")
	grant(storepb.Policy_PROJECT, common.FormatProject("project-a"), "dev@example.com", "roles/projectDeveloper")
	grant(storepb.Policy_PROJECT, common.FormatProject("project-b"), "lister@example.com", "roles/database-lister")
	grant(storepb.Policy_PROJECT, common.FormatProject("project-b"), "getter@example.com", "roles/database-getter")

	iamManager, err := iam.NewManager(stores, nil, false)
	require.NoError(t, err)
	service := &DatabaseService{store: stores, iamManager: iamManager, licenseService: newInstanceServiceTestLicenseService(t, stores)}
	search := func(email, filter string, pageSize int32) ([]string, error) {
		callerCtx := context.WithValue(ctx, common.WorkspaceIDContextKey, "default")
		callerCtx = context.WithValue(callerCtx, common.UserContextKey, &store.UserMessage{Email: email})
		var names []string
		pageToken := ""
		for {
			resp, err := service.SearchDatabases(callerCtx, connect.NewRequest(&v1pb.SearchDatabasesRequest{
				Filter:    filter,
				PageSize:  pageSize,
				PageToken: pageToken,
			}))
			if err != nil {
				return nil, err
			}
			for _, database := range resp.Msg.Databases {
				names = append(names, database.Name)
			}
			if resp.Msg.NextPageToken == "" {
				return names, nil
			}
			pageToken = resp.Msg.NextPageToken
		}
	}
	found := func(email, filter string, pageSize int32) []string {
		names, err := search(email, filter, pageSize)
		require.NoError(t, err)
		return names
	}

	projectA := []string{"instances/prod/databases/a1", "instances/prod/databases/a2"}
	projectB := []string{"instances/prod/databases/b1", "instances/staging/databases/a1"}
	require.Empty(t, found("nobody@example.com", "", 100), "a caller who can list no project finds no database")
	require.Equal(t, projectA, found("dev@example.com", "", 1), "staging/a1 shares a name but sits in a project dev cannot list")
	require.Equal(t, projectB, found("lister@example.com", "", 100), "bb.databases.list alone is the permission the search checks")
	require.Empty(t, found("getter@example.com", "", 100), "bb.projects.get and bb.databases.get do not list a project")
	require.Equal(t, projectB, found("admin@example.com", `project == "projects/project-b"`, 100))
	require.Empty(t, found("dev@example.com", `project == "projects/project-b"`, 100), "a filter cannot reach a project the caller cannot list")
	require.Equal(t, append(projectA, projectB...), found("admin@example.com", "", 1),
		"pages cross projects, and an archived project lists nothing, as projects/{archived} does")
	_, err = search("dev@example.com", "name ==", 100)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestShouldDiffSchemaViaSDL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		engine storepb.Engine
		req    *v1pb.DiffSchemaRequest
		want   bool
	}{
		{
			name:   "postgres raw schema text uses SDL",
			engine: storepb.Engine_POSTGRES,
			req: &v1pb.DiffSchemaRequest{
				Target: &v1pb.DiffSchemaRequest_Schema{Schema: "CREATE TABLE t(id int);"},
			},
			want: true,
		},
		{
			name:   "postgres empty raw schema text uses SDL",
			engine: storepb.Engine_POSTGRES,
			req: &v1pb.DiffSchemaRequest{
				Target: &v1pb.DiffSchemaRequest_Schema{Schema: ""},
			},
			want: true,
		},
		{
			name:   "postgres changelog target uses metadata diff",
			engine: storepb.Engine_POSTGRES,
			req: &v1pb.DiffSchemaRequest{
				Target: &v1pb.DiffSchemaRequest_Changelog{Changelog: "instances/prod/databases/app/changelogs/123"},
			},
			want: false,
		},
		{
			name:   "cockroach raw schema text uses SDL",
			engine: storepb.Engine_COCKROACHDB,
			req: &v1pb.DiffSchemaRequest{
				Target: &v1pb.DiffSchemaRequest_Schema{Schema: "CREATE TABLE t(id int);"},
			},
			want: true,
		},
		{
			name:   "mysql raw schema text uses SDL",
			engine: storepb.Engine_MYSQL,
			req: &v1pb.DiffSchemaRequest{
				Target: &v1pb.DiffSchemaRequest_Schema{Schema: "CREATE TABLE t(id int);"},
			},
			want: true,
		},
		{
			name:   "mysql changelog target uses metadata diff",
			engine: storepb.Engine_MYSQL,
			req: &v1pb.DiffSchemaRequest{
				Target: &v1pb.DiffSchemaRequest_Changelog{Changelog: "instances/prod/databases/app/changelogs/123"},
			},
			want: false,
		},
		{
			// The gate keys off the REAL engine: MariaDB is excluded from the SDL path even
			// though it aliases to MySQL for parsing (canonicalizing it as MySQL 8.0 would emit
			// utf8mb4_0900_ai_ci, which MariaDB lacks).
			name:   "mariadb raw schema text does NOT use SDL",
			engine: storepb.Engine_MARIADB,
			req: &v1pb.DiffSchemaRequest{
				Target: &v1pb.DiffSchemaRequest_Schema{Schema: "CREATE TABLE t(id int);"},
			},
			want: false,
		},
		{
			// OceanBase is excluded from every SDL path pending a live oracle, despite aliasing
			// to MySQL for parsing.
			name:   "oceanbase raw schema text does NOT use SDL",
			engine: storepb.Engine_OCEANBASE,
			req: &v1pb.DiffSchemaRequest{
				Target: &v1pb.DiffSchemaRequest_Schema{Schema: "CREATE TABLE t(id int);"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, shouldDiffSchemaViaSDL(tt.engine, tt.req))
		})
	}
}

func TestFormatDatabaseResourceName(t *testing.T) {
	t.Parallel()
	database := &store.DatabaseMessage{InstanceID: "instance-a", DatabaseName: "app"}
	require.Equal(t, "instances/instance-a/databases/app", formatDatabaseResourceName(&store.InstanceMessage{ResourceID: "instance-a"}, database))
	projectID := "project-a"
	require.Equal(t, "projects/project-a/instances/instance-a/databases/app", formatDatabaseResourceName(&store.InstanceMessage{
		ResourceID: "instance-a",
		ProjectID:  &projectID,
	}, database))
}

func TestListDatabaseFilter(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		input    string
		wantSQL  string
		wantArgs []any
		error    *connect.Error
	}{
		{
			input:    `environment == "environments/test"`,
			wantSQL:  `(COALESCE(db.environment, instance.environment) = $1)`,
			wantArgs: []any{"test"},
		},
		{
			input: `environment == "test"`,
			error: connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid environment filter %q", "test")),
		},
		{
			input:    `project == "projects/sample"`,
			wantSQL:  `(db.project = $1)`,
			wantArgs: []any{"sample"},
		},
		{
			input:    `name.contains("Employee")`,
			wantSQL:  `(LOWER(db.name) LIKE $1 ESCAPE '\')`,
			wantArgs: []any{"%employee%"},
		},
		{
			input:    `table.contains("user")`,
			wantSQL:  "(EXISTS (\n\t\t\t\t\t\tSELECT 1\n\t\t\t\t\t\tFROM json_array_elements(ds.metadata->'schemas') AS s,\n\t\t\t\t\t\t \t json_array_elements(s->'tables') AS t\n\t\t\t\t\t\tWHERE t->>'name' LIKE $1 ESCAPE '\\'))",
			wantArgs: []any{"%user%"},
		},
		{
			input: `name.matches("Employee")`,
			error: connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function matches")),
		},
		{
			input:    `engine in ["MYSQL", "POSTGRES"]`,
			wantSQL:  `(instance.metadata->>'engine' = ANY($1))`,
			wantArgs: []any{[]any{v1pb.Engine_MYSQL.String(), v1pb.Engine_POSTGRES.String()}},
		},
		{
			input:    `labels.region == "asia" && labels.tenant == "bytebase"`,
			wantSQL:  `((db.metadata->'labels'->>$1::text = $2 AND db.metadata->'labels'->>$3::text = $4))`,
			wantArgs: []any{"region", "asia", "tenant", "bytebase"},
		},
		{
			input:    `(labels.region == "asia" || labels.tenant == "bytebase") && exclude_unassigned == true`,
			wantSQL:  `(((db.metadata->'labels'->>$1::text = $2 OR db.metadata->'labels'->>$3::text = $4) AND db.project != $5 AND db.project != 'default'))`,
			wantArgs: []any{"region", "asia", "tenant", "bytebase", common.DefaultProjectID("test-workspace")},
		},
		{
			input:    `labels.region in ["asia", "europe"] && labels.tenant == "bytebase"`,
			wantSQL:  `((db.metadata->'labels'->>$1::text = ANY($2) AND db.metadata->'labels'->>$3::text = $4))`,
			wantArgs: []any{"region", []any{"asia", "europe"}, "tenant", "bytebase"},
		},
	}

	for _, tc := range testCases {
		filterQ, err := store.GetListDatabaseFilter("test-workspace", tc.input)
		if tc.error != nil {
			require.Error(t, err)
			require.Equal(t, tc.error.Message(), err.Error())
		} else {
			require.NoError(t, err)
			sql, args, err := filterQ.ToSQL()
			require.NoError(t, err)
			require.Equal(t, tc.wantSQL, sql)
			require.Equal(t, tc.wantArgs, args)
		}
	}
}

func TestGetDatabaseMetadataFilter(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		input        string
		wantSchema   *string
		wantTable    *string
		wantWildcard bool
		errContains  string
	}{
		{
			name:         "table contains",
			input:        `table.contains("user")`,
			wantTable:    ptrValue("user"),
			wantWildcard: true,
		},
		{
			name:         "schema and table contains",
			input:        `schema == "public" && table.contains("user")`,
			wantSchema:   ptrValue("public"),
			wantTable:    ptrValue("user"),
			wantWildcard: true,
		},
		{
			name:        "table matches unsupported",
			input:       `table.matches("user")`,
			errContains: "unexpected function matches",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filter, err := getDatabaseMetadataFilter(tc.input)
			if tc.errContains != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errContains)
				return
			}

			require.NoError(t, err)
			if tc.wantSchema != nil {
				require.NotNil(t, filter.schema)
				require.Equal(t, *tc.wantSchema, *filter.schema)
			}
			if tc.wantTable != nil {
				require.NotNil(t, filter.table)
				require.Equal(t, *tc.wantTable, filter.table.name)
				require.Equal(t, tc.wantWildcard, filter.table.wildcard)
			}
		})
	}
}

func TestGetDatabaseMetadataFilterRedactsSyntaxError(t *testing.T) {
	t.Parallel()
	_, err := getDatabaseMetadataFilter(`table == "sensitive SQL" &&`)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.ErrorContains(t, err, "invalid filter expression")
	require.NotContains(t, err.Error(), "sensitive SQL")
}

func ptrValue[T any](v T) *T {
	return &v
}

// TestResolveDiffSchemaTargetSDL pins the X11 fix: the SDL target is selected by ONEOF
// PRESENCE, not string emptiness — an intentionally empty schema text is a legal target
// meaning "empty schema" (the diff previews dropping everything), while a request with
// no target at all still errors.
func TestResolveDiffSchemaTargetSDL(t *testing.T) {
	s := &DatabaseService{}
	ctx := t.Context()

	t.Run("empty_schema_text_is_a_valid_target", func(t *testing.T) {
		got, err := s.resolveDiffSchemaTargetSDL(ctx, &v1pb.DiffSchemaRequest{
			Target: &v1pb.DiffSchemaRequest_Schema{Schema: ""},
		}, storepb.Engine_MYSQL)
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("non_empty_schema_text_passed_through", func(t *testing.T) {
		got, err := s.resolveDiffSchemaTargetSDL(ctx, &v1pb.DiffSchemaRequest{
			Target: &v1pb.DiffSchemaRequest_Schema{Schema: "CREATE TABLE t(id int);"},
		}, storepb.Engine_POSTGRES)
		require.NoError(t, err)
		require.Equal(t, "CREATE TABLE t(id int);", got)
	})

	t.Run("no_target_at_all_errors", func(t *testing.T) {
		_, err := s.resolveDiffSchemaTargetSDL(ctx, &v1pb.DiffSchemaRequest{}, storepb.Engine_MYSQL)
		require.ErrorContains(t, err, "target must be either schema text or changelog")
	})
}

// TestCheckDiffSchemaTargetProject pins the gate DiffSchema puts on a changelog
// target. The ACL interceptor authorizes request.name only, so without it a
// changelog under another project's database would hand over that project's
// schema. A foreign changelog is indistinguishable from one that does not
// exist, or the error itself answers the question.
func TestCheckDiffSchemaTargetProject(t *testing.T) {
	t.Parallel()
	source := &store.DatabaseMessage{ProjectID: "project-a", InstanceID: "shared-instance", DatabaseName: "app-a"}

	require.NoError(t, checkDiffSchemaTargetProject(source.ProjectID, source, "instances/shared-instance/databases/app-a/changelogs/1"))

	foreign := checkDiffSchemaTargetProject(source.ProjectID,
		&store.DatabaseMessage{ProjectID: "project-b", InstanceID: "shared-instance", DatabaseName: "app-b"},
		"instances/shared-instance/databases/app-b/changelogs/2")
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(foreign))
	require.NotContains(t, foreign.Error(), "project-b")

	missing := checkDiffSchemaTargetProject(source.ProjectID, nil, "instances/shared-instance/databases/no-such-db/changelogs/1")
	require.Equal(t, connect.CodeOf(foreign), connect.CodeOf(missing))
}
