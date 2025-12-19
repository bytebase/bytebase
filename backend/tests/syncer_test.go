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

	"github.com/bytebase/bytebase/backend/common"
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
			Environment: stringPtr("environments/prod"),
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

func TestSyncerForMySQL(t *testing.T) {
	const (
		databaseName = "test_sync_mysql_schema_db"
		createSchema = `
		CREATE TABLE trd (
			a int DEFAULT NULL,
			b int DEFAULT NULL,
			c int DEFAULT NULL,
			UNIQUE KEY a (a,b,c)
		);
		CREATE TABLE t1 (
			id int PRIMARY KEY
		);
		CREATE TABLE tfk (
			a int DEFAULT NULL,
			b int DEFAULT NULL,
			c int DEFAULT NULL,
			KEY a (a,b,c),
			CONSTRAINT tfk_ibfk_1 FOREIGN KEY (a, b, c) REFERENCES trd (a, b, c),
			CONSTRAINT tfk_ibfk_2 FOREIGN KEY (a) REFERENCES t1 (id)
		);
		`
		expectedSchema = `{
			"name":"instances/instance-syncer-mysql/databases/test_sync_mysql_schema_db/metadata",
			"schemas":[
			   {
				  "tables":[
					 {
						"name":"t1",
						"columns":[
						   {
							  "name":"id",
							  "position":1,
							  "type":"int"
						   }
						],
						"indexes":[
						   {
							  "name":"PRIMARY",
							  "expressions":[
								 "id"
							  ],
							  "keyLength": [
								 "-1"
							  ],
							  "descending": [
							  	false
							  ],
							  "type":"BTREE",
							  "unique":true,
							  "primary":true,
							  "visible":true
						   }
						],
						"charset": "utf8mb4",
						"engine":"InnoDB",
						"collation":"utf8mb4_general_ci",
						"dataSize":"16384"
					 },
					 {
						"name":"tfk",
						"columns":[
						   {
							  "name":"a",
							  "position":1,
							  "nullable":true,
							  "type":"int",
							  "default": "NULL",
						      "has_default": true
						   },
						   {
							  "name":"b",
							  "position":2,
							  "nullable":true,
							  "type":"int",
							  "default": "NULL",
						      "has_default": true
						   },
						   {
							  "name":"c",
							  "position":3,
							  "nullable":true,
							  "type":"int",
							  "default": "NULL",
						      "has_default": true
						   }
						],
						"indexes":[
						   {
							  "name":"a",
							  "expressions":[
								 "a",
								 "b",
								 "c"
							  ],
							  "keyLength": [
								-1,
								-1,
								-1
							  ],
							  "descending": [
							  	false,
								false,
								false
							  ],
							  "type":"BTREE",
							  "visible":true
						   }
						],
						"engine":"InnoDB",
						"charset": "utf8mb4",
						"collation":"utf8mb4_general_ci",
						"dataSize":"16384",
						"indexSize":"16384",
						"foreignKeys":[
						   {
							  "name":"tfk_ibfk_1",
							  "columns":[
								 "a",
								 "b",
								 "c"
							  ],
							  "referencedTable":"trd",
							  "referencedColumns":[
								 "a",
								 "b",
								 "c"
							  ],
							  "onDelete":"NO ACTION",
							  "onUpdate":"NO ACTION",
							  "matchType":"NONE"
						   },
						   {
							  "name":"tfk_ibfk_2",
							  "columns":[
								 "a"
							  ],
							  "referencedTable":"t1",
							  "referencedColumns":[
								 "id"
							  ],
							  "onDelete":"NO ACTION",
							  "onUpdate":"NO ACTION",
							  "matchType":"NONE"
						   }
						]
					 },
					 {
						"name":"trd",
						"columns":[
						   {
							  "name":"a",
							  "position":1,
							  "nullable":true,
							  "type":"int",
							  "default": "NULL",
						      "has_default": true
						   },
						   {
							  "name":"b",
							  "position":2,
							  "nullable":true,
							  "type":"int",
							  "default": "NULL",
						      "has_default": true
						   },
						   {
							  "name":"c",
							  "position":3,
							  "nullable":true,
							  "type":"int",
							  "default": "NULL",
						      "has_default": true
						   }
						],
						"indexes":[
						   {
							  "name":"a",
							  "expressions":[
								 "a",
								 "b",
								 "c"
							  ],
							  "keyLength": [
								-1,
								-1,
								-1
							  ],
							  "descending": [
							  	false,
								false,
								false
							  ],
							  "type":"BTREE",
							  "unique":true,
							  "visible":true
						   }
						],
						"engine":"InnoDB",
						"charset": "utf8mb4",
						"collation":"utf8mb4_general_ci",
						"dataSize":"16384",
						"indexSize":"16384"
					 }
				  ]
			   }
			],
			"characterSet":"utf8mb4",
			"collation":"utf8mb4_general_ci"
		 }`
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	mysqlContainer, err := getMySQLContainer(ctx)
	defer func() {
		mysqlContainer.Close(ctx)
	}()
	a.NoError(err)

	mysqlDB := mysqlContainer.db
	_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
	a.NoError(err)
	_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
	a.NoError(err)

	_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase")
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: "instance-syncer-mysql",
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: mysqlContainer.host, Port: mysqlContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, databaseName, "")
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

	var expectedSchemaMetadata v1pb.DatabaseMetadata
	err = common.ProtojsonUnmarshaler.Unmarshal([]byte(expectedSchema), &expectedSchemaMetadata)
	a.NoError(err)

	diff := cmp.Diff(&expectedSchemaMetadata, latestSchemaMetadata, protocmp.Transform())
	a.Empty(diff)
}
