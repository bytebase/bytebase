package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestSyncerForPostgreSQL(t *testing.T) {
	const (
		databaseName = "test_sync_postgresql_schema_db"
		createSchema = `
		CREATE SCHEMA schema1;
		CREATE TABLE schema1.trd (
			"A" int DEFAULT NULL,
			"B" int DEFAULT NULL,
			c int DEFAULT NULL,
			UNIQUE ("A","B",c)
		  );
		  CREATE TABLE "TFK" (
			a int DEFAULT NULL,
			b int DEFAULT NULL,
			c int DEFAULT NULL,
			CONSTRAINT tfk_ibfk_1 FOREIGN KEY (a, b, c) REFERENCES schema1.trd ("A", "B", c)
		  );
		CREATE VIEW "VW" AS SELECT * FROM "TFK";
		`
	)
	wantDatabaseMetadata := &v1pb.DatabaseMetadata{
		Name:         "instances/instance-syncer-postgres/databases/test_sync_postgresql_schema_db/metadata",
		Owner:        "bytebase",
		CharacterSet: "UTF8",
		Collation:    "en_US.UTF-8",
		Schemas: []*v1pb.SchemaMetadata{
			{
				Name:    "public",
				Owner:   "pg_database_owner",
				Comment: "standard public schema",
				Tables: []*v1pb.TableMetadata{
					{
						Name:  "TFK",
						Owner: "bytebase",
						Columns: []*v1pb.ColumnMetadata{
							{
								Name:     "a",
								Position: 1,
								Nullable: true,
								Type:     "integer",
							},
							{
								Name:     "b",
								Position: 2,
								Nullable: true,
								Type:     "integer",
							},
							{
								Name:     "c",
								Position: 3,
								Nullable: true,
								Type:     "integer",
							},
						},
						ForeignKeys: []*v1pb.ForeignKeyMetadata{
							{
								Name:              "tfk_ibfk_1",
								Columns:           []string{"a", "b", "c"},
								ReferencedSchema:  "schema1",
								ReferencedTable:   "trd",
								ReferencedColumns: []string{"A", "B", "c"},
								OnDelete:          "NO ACTION",
								OnUpdate:          "NO ACTION",
								MatchType:         "SIMPLE",
							},
						},
					},
				},
				Views: []*v1pb.ViewMetadata{
					{
						Name: "VW",
						Definition: strings.Join([]string{
							` SELECT a,`,
							`    b,`,
							`    c`,
							`   FROM public."TFK";`},
							"\n"),
						Columns: []*v1pb.ColumnMetadata{
							{
								Name:     "a",
								Position: 1,
								Nullable: true,
								Type:     "integer",
							},
							{
								Name:     "b",
								Position: 2,
								Nullable: true,
								Type:     "integer",
							},
							{
								Name:     "c",
								Position: 3,
								Nullable: true,
								Type:     "integer",
							},
						},
						DependencyColumns: []*v1pb.DependencyColumn{
							{
								Schema: "public",
								Table:  "TFK",
								Column: "a",
							},
							{
								Schema: "public",
								Table:  "TFK",
								Column: "b",
							},
							{
								Schema: "public",
								Table:  "TFK",
								Column: "c",
							},
						},
					},
				},
			},
			{
				Name:  "schema1",
				Owner: "bytebase",
				Tables: []*v1pb.TableMetadata{
					{
						Name:  "trd",
						Owner: "bytebase",
						Columns: []*v1pb.ColumnMetadata{
							{
								Name:     "A",
								Position: 1,
								Nullable: true,
								Type:     "integer",
							},
							{
								Name:     "B",
								Position: 2,
								Nullable: true,
								Type:     "integer",
							},
							{
								Name:     "c",
								Position: 3,
								Nullable: true,
								Type:     "integer",
							},
						},
						Indexes: []*v1pb.IndexMetadata{
							{
								Name:            "trd_A_B_c_key",
								Expressions:     []string{`"A"`, `"B"`, "c"},
								Descending:      []bool{false, false, false},
								OpclassNames:    []string{"int4_ops", "int4_ops", "int4_ops"},
								OpclassDefaults: []bool{true, true, true},
								Type:            "btree",
								Unique:          true,
								Definition:      `CREATE UNIQUE INDEX "trd_A_B_c_key" ON schema1.trd USING btree ("A", "B", c);`,
								IsConstraint:    true,
							},
						},
						IndexSize: 8192,
					},
				},
			},
		},
	}

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	pgDB := pgContainer.db
	err = pgDB.Ping()
	a.NoError(err)

	_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)
	_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
	a.NoError(err)
	_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: "instance-syncer-postgres",
		Instance: &v1pb.Instance{
			Title:       "pgInstance",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, databaseName, "bytebase")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte(createSchema),
		},
	}))
	a.NoError(err)
	sheet := sheetResp.Msg

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, false)
	a.NoError(err)

	latestSchemaMetadataResp, err := ctl.databaseServiceClient.GetDatabaseMetadata(ctx, connect.NewRequest(&v1pb.GetDatabaseMetadataRequest{
		Name: fmt.Sprintf("%s/metadata", database.Name),
	}))
	a.NoError(err)
	latestSchemaMetadata := latestSchemaMetadataResp.Msg

	diff := cmp.Diff(wantDatabaseMetadata, latestSchemaMetadata, protocmp.Transform())
	a.Empty(diff)
}
