package tests

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/tests/fake"
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
		dataDir:                   dataDir,
		vcsProviderCreator:        fake.NewGitLab,
		developmentUseV2Scheduler: true,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a MySQL instance.
	mysqlPort := getTestPort()
	stopInstance := mysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer stopInstance()

	mysqlDB, err := sql.Open("mysql", fmt.Sprintf("root@tcp(127.0.0.1:%d)/mysql", mysqlPort))
	a.NoError(err)
	defer mysqlDB.Close()

	_, err = mysqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = mysqlDB.Exec("DROP USER IF EXISTS bytebase")
	a.NoError(err)
	_, err = mysqlDB.Exec("CREATE USER 'bytebase' IDENTIFIED WITH mysql_native_password BY 'bytebase'")
	a.NoError(err)

	_, err = mysqlDB.Exec("GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DELETE, DROP, EVENT, EXECUTE, INDEX, INSERT, PROCESS, REFERENCES, SELECT, SHOW DATABASES, SHOW VIEW, TRIGGER, UPDATE, USAGE, REPLICATION CLIENT, REPLICATION SLAVE, LOCK TABLES, RELOAD ON *.* to bytebase")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	err = ctl.setLicense()
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "mysqlInstance",
			Engine:      v1pb.Engine_MYSQL,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "127.0.0.1", Port: strconv.Itoa(mysqlPort), Username: "bytebase", Password: "bytebase"}},
		},
	})
	a.NoError(err)

	err = ctl.createDatabaseV2(ctx, project, instance, nil /* environment */, databaseName, "", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:      "createTable",
			Content:    []byte(createTable),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
	a.NoError(err)

	// Create sensitive data policy.
	_, err = ctl.orgPolicyServiceClient.CreatePolicy(ctx, &v1pb.CreatePolicyRequest{
		Parent: database.Name,
		Policy: &v1pb.Policy{
			Type: v1pb.PolicyType_MASKING,
			Policy: &v1pb.Policy_MaskingPolicy{
				MaskingPolicy: &v1pb.MaskingPolicy{
					MaskData: []*v1pb.MaskData{
						{
							Table:              tableName,
							Column:             "id",
							SemanticCategoryId: "",
							MaskingLevel:       v1pb.MaskingLevel_FULL,
						},
						{
							Table:              tableName,
							Column:             "author",
							SemanticCategoryId: "",
							MaskingLevel:       v1pb.MaskingLevel_FULL,
						},
					},
				},
			},
		},
	})
	a.NoError(err)

	insertDataSheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:      "insertData",
			Content:    []byte(insertData),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)

	// Insert data into table tech_book.
	err = ctl.changeDatabase(ctx, project, database, insertDataSheet, v1pb.Plan_ChangeDatabaseConfig_DATA)
	a.NoError(err)

	// Query masked data.
	queryResp, err := ctl.sqlServiceClient.Query(ctx, &v1pb.QueryRequest{
		Name: instance.Name, ConnectionDatabase: databaseName, Statement: queryTable,
	})
	a.NoError(err)
	a.Equal(1, len(queryResp.Results))
	diff := cmp.Diff(maskedData, queryResp.Results[0], protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Equal("", diff)

	// Query origin data.
	singleSQLResults, err := ctl.adminQuery(ctx, instance, databaseName, queryTable)
	a.NoError(err)
	a.Len(singleSQLResults, 1)
	result := singleSQLResults[0]
	a.Equal("", result.Error)
	diff = cmp.Diff(originData, result, protocmp.Transform(), protocmp.IgnoreMessages(&durationpb.Duration{}))
	a.Equal("", diff)
}
