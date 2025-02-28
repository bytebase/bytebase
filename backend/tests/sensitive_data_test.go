package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	maskedData = &v1pb.QueryResult{
		ColumnNames:     []string{"id", "name", "author"},
		ColumnTypeNames: []string{"INT", "VARCHAR", "VARCHAR"},
		Masked:          []bool{true, false, true},
		Sensitive:       []bool{true, false, true},
		Rows: []*v1pb.QueryRow{
			{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "bytebase"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}},
				},
			},
			{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "PostgreSQL 14 Internals"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}},
				},
			},
			{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "Designing Data-Intensive Applications"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "******"}},
				},
			},
		},
		Statement: "SELECT * FROM tech_book",
		RowsCount: 3,
	}
	originData = &v1pb.QueryResult{
		ColumnNames:     []string{"id", "name", "author"},
		ColumnTypeNames: []string{"INT", "VARCHAR", "VARCHAR"},
		Rows: []*v1pb.QueryRow{
			{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "bytebase"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "bber"}},
				},
			},
			{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_Int64Value{Int64Value: 2}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "PostgreSQL 14 Internals"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "Egor Rogov"}},
				},
			},
			{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_Int64Value{Int64Value: 3}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "Designing Data-Intensive Applications"}},
					{Kind: &v1pb.RowValue_StringValue{StringValue: "Martin Kleppmann"}},
				},
			},
		},
		Statement: "SELECT * FROM tech_book",
		RowsCount: 3,
	}
)

func TestSensitiveData(t *testing.T) {
	const (
		databaseName = "sensitive_data"
		tableName    = "tech_book"
		createTable  = `
			CREATE TABLE tech_book(
				id int primary key,
				name varchar(220),
				author varchar(220)
			);
		`
		insertData = `
			INSERT INTO tech_book VALUES
				(1, 'bytebase', 'bber'),
				(2, 'PostgreSQL 14 Internals', 'Egor Rogov'),
				(3, 'Designing Data-Intensive Applications', 'Martin Kleppmann');
		`
		queryTable = `SELECT * FROM tech_book`
	)
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir: dataDir,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	_, err = ctl.settingServiceClient.UpdateSetting(ctx, &v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/bb.workspace.semantic-types",
			Value: &v1pb.Value{
				Value: &v1pb.Value_SemanticTypeSettingValue{
					SemanticTypeSettingValue: &v1pb.SemanticTypeSetting{
						Types: []*v1pb.SemanticTypeSetting_SemanticType{
							{
								Id:    "default",
								Title: "Default",
								Algorithm: &v1pb.Algorithm{
									Mask: &v1pb.Algorithm_FullMask_{FullMask: &v1pb.Algorithm_FullMask{Substitution: "******"}},
								},
							},
						},
					},
				},
			},
		},
		AllowMissing: true,
	})
	a.NoError(err)

	mysqlContainer, err := getMySQLContainer(ctx)
	a.NoError(err)

	defer func() {
		mysqlContainer.db.Close()
		err := mysqlContainer.container.Terminate(ctx)
		a.NoError(err)
	}()

	mysqlDB := mysqlContainer.db
	_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
	a.NoError(err)
	_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
	a.NoError(err)

	_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase")
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: mysqlContainer.host, Port: mysqlContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	})
	a.NoError(err)

	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	// Validate query syntax error.
	_, err = ctl.sqlServiceClient.Query(ctx, &v1pb.QueryRequest{
		Name:         database.Name,
		Statement:    "SELECT hello TO world;",
		DataSourceId: "admin",
	})
	a.Error(err)
	// TODO(d): deprecate the details with diagonose check. And the error is not reached anyway.
	/*
		st := status.Convert(err)
		a.Len(st.Details(), 1)
		report, ok := st.Details()[0].(*v1pb.PlanCheckRun_Result_SqlReviewReport)
		a.True(ok)
		a.Equal(int32(1), report.Line)
		a.Equal(int32(13), report.Column)
		a.Equal("Syntax error at line 1:13 \nrelated text: SELECT hello TO", report.Detail)
	*/

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "createTable",
			Content: []byte(createTable),
		},
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
	a.NoError(err)

	// Create sensitive data in the database config.
	_, err = ctl.databaseCatalogServiceClient.UpdateDatabaseCatalog(ctx, &v1pb.UpdateDatabaseCatalogRequest{
		Catalog: &v1pb.DatabaseCatalog{
			Name: fmt.Sprintf("%s/catalog", database.Name),
			Schemas: []*v1pb.SchemaCatalog{
				{
					Name: "",
					Tables: []*v1pb.TableCatalog{
						{
							Name: tableName,
							Kind: &v1pb.TableCatalog_Columns_{
								Columns: &v1pb.TableCatalog_Columns{
									Columns: []*v1pb.ColumnCatalog{
										{
											Name:         "id",
											SemanticType: "default",
										},
										{
											Name:         "author",
											SemanticType: "default",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	insertDataSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "insertData",
			Content: []byte(insertData),
		},
	})
	a.NoError(err)

	// Insert data into table tech_book.
	err = ctl.changeDatabase(ctx, ctl.project, database, insertDataSheet, v1pb.Plan_ChangeDatabaseConfig_DATA)
	a.NoError(err)

	// Query masked data.
	queryResp, err := ctl.sqlServiceClient.Query(ctx, &v1pb.QueryRequest{
		Name:         database.Name,
		Statement:    queryTable,
		DataSourceId: "admin",
	})
	a.NoError(err)
	a.Equal(1, len(queryResp.Results))
	diff := cmp.Diff(maskedData, queryResp.Results[0], protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Empty(diff)

	// Query origin data.
	singleSQLResults, err := ctl.adminQuery(ctx, database, queryTable)
	a.NoError(err)
	a.Len(singleSQLResults, 1)
	result := singleSQLResults[0]
	a.Equal("", result.Error)
	diff = cmp.Diff(originData, result, protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Empty(diff)
}
