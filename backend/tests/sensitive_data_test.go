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
	"google.golang.org/protobuf/types/known/durationpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

var (
	maskedData = &v1pb.QueryResult{
		ColumnNames:     []string{"id", "name", "author"},
		ColumnTypeNames: []string{"INT", "VARCHAR", "VARCHAR"},
		Masked: []*v1pb.MaskingReason{
			{SemanticTypeId: "default", Algorithm: "Full mask"},
			nil,
			{SemanticTypeId: "default", Algorithm: "Full mask"},
		},
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
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	_, err = ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		Setting: &v1pb.Setting{
			Name: "settings/" + v1pb.Setting_SEMANTIC_TYPES.String(),
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_SemanticType{
					SemanticType: &v1pb.SemanticTypeSetting{
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
	}))
	a.NoError(err)

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
		InstanceId: generateRandomString("instance"),
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

	// Validate query syntax error - with new ACL system, syntax errors are returned in query results
	syntaxErrorResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:         database.Name,
		Statement:    "SELECT hello TO world;",
		DataSourceId: "admin",
	}))
	a.NoError(err)
	a.Equal(1, len(syntaxErrorResp.Msg.Results))
	a.NotEmpty(syntaxErrorResp.Msg.Results[0].Error)
	a.Contains(syntaxErrorResp.Msg.Results[0].Error, "Syntax error")
	// Check the detailed_error field for syntax_error with position
	syntaxErr := syntaxErrorResp.Msg.Results[0].GetSyntaxError()
	a.NotNil(syntaxErr)
	a.NotNil(syntaxErr.StartPosition)
	a.Equal(int32(1), syntaxErr.StartPosition.Line)
	a.Equal(int32(14), syntaxErr.StartPosition.Column)

	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte(createTable),
		},
	}))
	a.NoError(err)
	sheet := sheetResp.Msg

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, false)
	a.NoError(err)

	// Create sensitive data in the database config.
	_, err = ctl.databaseCatalogServiceClient.UpdateDatabaseCatalog(ctx, connect.NewRequest(&v1pb.UpdateDatabaseCatalogRequest{
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
	}))
	a.NoError(err)

	insertDataSheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte(insertData),
		},
	}))
	a.NoError(err)
	insertDataSheet := insertDataSheetResp.Msg

	// Insert data into table tech_book.
	err = ctl.changeDatabase(ctx, ctl.project, database, insertDataSheet, false)
	a.NoError(err)

	// Query masked data.
	queryResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:         database.Name,
		Statement:    queryTable,
		DataSourceId: "admin",
	}))
	a.NoError(err)
	a.Equal(1, len(queryResp.Msg.Results))

	// Build expected masked data dynamically with the correct instance name
	// Extract instance ID from instance.Name (which is in format "instances/instance-id")
	instanceParts := strings.Split(instance.Name, "/")
	instanceID := instanceParts[len(instanceParts)-1]

	expectedMaskedData := &v1pb.QueryResult{
		ColumnNames:     []string{"id", "name", "author"},
		ColumnTypeNames: []string{"INT", "VARCHAR", "VARCHAR"},
		Masked: []*v1pb.MaskingReason{
			{
				SemanticTypeId:    "default",
				Algorithm:         "Full mask",
				Context:           fmt.Sprintf("Column-level semantic type: %s.%s.%s.%s", instanceID, databaseName, tableName, "id"),
				SemanticTypeTitle: "Default",
			},
			nil,
			{
				SemanticTypeId:    "default",
				Algorithm:         "Full mask",
				Context:           fmt.Sprintf("Column-level semantic type: %s.%s.%s.%s", instanceID, databaseName, tableName, "author"),
				SemanticTypeTitle: "Default",
			},
		},
		Rows:      maskedData.Rows,
		Statement: "SELECT * FROM tech_book",
		RowsCount: 3,
	}

	diff := cmp.Diff(expectedMaskedData, queryResp.Msg.Results[0], protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
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
