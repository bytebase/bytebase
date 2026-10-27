package tests

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

var (
	maskedData = &v1pb.QueryResult{
		ColumnNames:     []string{"id", "name", "author"},
		ColumnTypeNames: []string{"INT4", "VARCHAR", "VARCHAR"},
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
		ColumnTypeNames: []string{"INT4", "VARCHAR", "VARCHAR"},
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
	databaseName := uniqueDB("sensitive_data")
	const (
		tableName   = "tech_book"
		createTable = `
			CREATE TABLE tech_book(
				id int primary key,
				name varchar(220),
				author varchar(220)
			);
			CREATE TABLE ghost();
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
	ctl, ctx := startWorkspace(ctx, t)

	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
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

	pgContainer := sharedPgTarget(t)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			SyncDatabases: &v1pb.SyncDatabases{},
			Title:         "pgInstance",
			Engine:        v1pb.Engine_POSTGRES,
			Environment:   new("environments/prod"),
			Activation:    true,
			DataSources:   []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.GetHost(), Port: pgContainer.GetPort(), Username: "postgres", Password: "root-password", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, databaseName, "postgres")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	// Validate query syntax error - with new ACL system, syntax errors are returned in query results
	syntaxErrorResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT hello TO world;",
	}))
	a.NoError(err)
	a.Equal(1, len(syntaxErrorResp.Msg.Results))
	a.NotEmpty(syntaxErrorResp.Msg.Results[0].Error)
	a.Contains(strings.ToLower(syntaxErrorResp.Msg.Results[0].Error), "syntax error")
	// Check the detailed_error field for syntax_error with position
	syntaxErr := syntaxErrorResp.Msg.Results[0].GetSyntaxError()
	a.NotNil(syntaxErr)
	a.NotNil(syntaxErr.StartPosition)
	a.Equal(int32(1), syntaxErr.StartPosition.Line)
	// Column 13 is where the pg parser puts "TO" in "SELECT hello TO world;".
	// The MySQL parser reported 14 for the same statement; the column is a
	// per-parser convention, not behavior this test is about.
	a.Equal(int32(13), syntaxErr.StartPosition.Column)

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
					// Postgres has schemas; the MySQL arm of this test left it empty.
					Name: "public",
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
		Name:      database.Name,
		Statement: queryTable,
	}))
	a.NoError(err)
	a.Equal(1, len(queryResp.Msg.Results))

	// Build expected masked data dynamically with the correct instance name
	// Extract instance ID from instance.Name (which is in format "instances/instance-id")
	instanceParts := strings.Split(instance.Name, "/")
	instanceID := instanceParts[len(instanceParts)-1]

	expectedMaskedData := &v1pb.QueryResult{
		ColumnNames:     []string{"id", "name", "author"},
		ColumnTypeNames: []string{"INT4", "VARCHAR", "VARCHAR"},
		Masked: []*v1pb.MaskingReason{
			{
				SemanticTypeId:    "default",
				Algorithm:         "Full mask",
				Context:           fmt.Sprintf("Column-level semantic type: %s.%s.public.%s.%s", instanceID, databaseName, tableName, "id"),
				SemanticTypeTitle: "Default",
			},
			nil,
			{
				SemanticTypeId:    "default",
				Algorithm:         "Full mask",
				Context:           fmt.Sprintf("Column-level semantic type: %s.%s.public.%s.%s", instanceID, databaseName, tableName, "author"),
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

	// Query with a type-incompatible cast on a masked column.
	// The database error message would normally contain the actual column value,
	// but the masking pipeline should redact it.
	errorLeakResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT author::uuid FROM tech_book WHERE name = 'bytebase'",
	}))
	a.NoError(err)
	a.Equal(1, len(errorLeakResp.Msg.Results))
	a.NotEmpty(errorLeakResp.Msg.Results[0].Error)
	// The error should be redacted — must not contain the actual column value.
	a.NotContains(errorLeakResp.Msg.Results[0].Error, "bber")
	a.Contains(errorLeakResp.Msg.Results[0].Error, "data masking policies")

	// Same query on a non-masked column should preserve the original error.
	nonMaskedErrorResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT name::uuid FROM tech_book WHERE name = 'bytebase'",
	}))
	a.NoError(err)
	a.Equal(1, len(nonMaskedErrorResp.Msg.Results))
	a.NotEmpty(nonMaskedErrorResp.Msg.Results[0].Error)
	// The error should NOT be redacted — name is not masked.
	a.NotContains(nonMaskedErrorResp.Msg.Results[0].Error, "data masking policies")

	const refusedPredicate = "Using sensitive columns in query conditions is not allowed"
	for _, statement := range []string{
		"SELECT name FROM tech_book WHERE author = 'bber'",
		"SELECT b.name FROM tech_book b JOIN tech_book a ON a.id = b.id AND ascii(substr(a.author, 1, 1)) > 70",
		"SELECT name FROM tech_book WHERE EXISTS (SELECT 1 FROM tech_book t WHERE t.author LIKE 'E%')",
		"EXPLAIN ANALYZE SELECT name FROM tech_book WHERE author LIKE 'E%'",
		"SELECT count(*) FILTER (WHERE author LIKE 'E%') FROM tech_book",
		"SELECT count(*) FROM tech_book NATURAL JOIN (SELECT 'bber'::varchar AS author) x",
		// Running this would fail on the cast, so the refusal proves it never ran.
		"SELECT name FROM tech_book WHERE author::int = 1",
	} {
		resp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
			Name:      database.Name,
			Statement: statement,
		}))
		a.NoError(err, statement)
		a.Len(resp.Msg.Results, 1, statement)
		a.True(strings.HasPrefix(resp.Msg.Results[0].Error, refusedPredicate), "%s: %s", statement, resp.Msg.Results[0].Error)
		a.Empty(resp.Msg.Results[0].Rows, statement)
	}

	unmaskedPredicateResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT name FROM tech_book WHERE name = 'bytebase'",
	}))
	a.NoError(err)
	a.Len(unmaskedPredicateResp.Msg.Results, 1)
	a.Empty(unmaskedPredicateResp.Msg.Results[0].Error)
	a.Len(unmaskedPredicateResp.Msg.Results[0].Rows, 1)

	for _, statement := range []string{
		"SELECT a FROM (SELECT author AS a FROM tech_book UNION ALL SELECT name FROM tech_book) s",
		"WITH c AS (SELECT author FROM tech_book UNION SELECT name FROM tech_book) SELECT author FROM c",
		"SELECT s.x FROM (SELECT row_number() OVER (ORDER BY name) AS rn, author AS x FROM tech_book) s",
	} {
		resp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
			Name:      database.Name,
			Statement: statement,
		}))
		a.NoError(err, statement)
		a.Len(resp.Msg.Results, 1, statement)
		a.Empty(resp.Msg.Results[0].Error, statement)
		a.NotEmpty(resp.Msg.Results[0].Rows, statement)
		for _, row := range resp.Msg.Results[0].Rows {
			a.NotContains([]string{"bber", "Egor Rogov", "Martin Kleppmann"}, row.Values[0].GetStringValue(), statement)
		}
	}

	batchResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT name FROM tech_book; SELECT name FROM tech_book WHERE author = 'bber';",
	}))
	a.NoError(err)
	a.Len(batchResp.Msg.Results, 2)
	a.Empty(batchResp.Msg.Results[0].Error)
	a.Len(batchResp.Msg.Results[0].Rows, 3)
	a.Equal(fmt.Sprintf("%s: %s.public.%s.author", refusedPredicate, databaseName, tableName), batchResp.Msg.Results[1].Error)

	// Created directly on the target, after the last schema sync, so the synced
	// schema does not know the function or how many columns it returns.
	targetDB, err := sql.Open("pgx", fmt.Sprintf("postgresql://postgres:root-password@%s:%s/%s?sslmode=disable", pgContainer.GetHost(), pgContainer.GetPort(), databaseName))
	a.NoError(err)
	defer targetDB.Close()
	_, err = targetDB.ExecContext(ctx, `CREATE FUNCTION unsynced_rows() RETURNS TABLE(a int, b int) LANGUAGE sql AS 'SELECT 1, 2'`)
	a.NoError(err)

	// The span lists one placeholder column for the function, so tech_book's
	// columns would shift onto the wrong maskers and author past the last one.
	misalignedResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT * FROM unsynced_rows() f CROSS JOIN tech_book",
	}))
	a.NoError(err)
	a.Len(misalignedResp.Msg.Results, 1)
	a.Contains(misalignedResp.Msg.Results[0].Error, "masking cannot be applied: the query returned 5 columns but Bytebase resolved 4")
	a.Empty(misalignedResp.Msg.Results[0].Rows)

	unmaskedResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT * FROM unsynced_rows()",
	}))
	a.NoError(err)
	a.Len(unmaskedResp.Msg.Results, 1)
	a.Equal([]string{"a", "b"}, unmaskedResp.Msg.Results[0].ColumnNames)
	a.Len(unmaskedResp.Msg.Results[0].Rows, 1)

	// ghost synced with no columns. A column added outside Bytebase is masked by
	// name once it syncs, so the condition on it must be refused before the
	// query runs: a run would take the 30 seconds pg_sleep asks for. The query
	// syncs the database, so it comes after the unsynced_rows checks.
	_, err = targetDB.ExecContext(ctx, "ALTER TABLE ghost ADD COLUMN ghost_secret text; INSERT INTO ghost VALUES ('x')")
	a.NoError(err)
	workspace, err := ctl.workspaceServiceClient.GetWorkspace(ctx, connect.NewRequest(&v1pb.GetWorkspaceRequest{Name: "workspaces/-"}))
	a.NoError(err)
	_, err = ctl.orgPolicyServiceClient.UpdatePolicy(ctx, connect.NewRequest(&v1pb.UpdatePolicyRequest{
		Policy: &v1pb.Policy{
			Name: workspace.Msg.Name + "/policies/masking_rule",
			Type: v1pb.PolicyType_MASKING_RULE,
			Policy: &v1pb.Policy_MaskingRulePolicy{
				MaskingRulePolicy: &v1pb.MaskingRulePolicy{
					Rules: []*v1pb.MaskingRulePolicy_MaskingRule{{
						Id:           "6b1e7c2a-3f4d-4e5a-9b8c-0d1e2f3a4b5c",
						Condition:    &expr.Expr{Expression: `resource.column_name == "ghost_secret"`},
						SemanticType: "default",
					}},
				},
			},
		},
		AllowMissing: true,
		UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"masking_rule_policy"}},
	}))
	a.NoError(err)
	start := time.Now()
	ghostResp, err := ctl.sqlServiceClient.Query(ctx, connect.NewRequest(&v1pb.QueryRequest{
		Name:      database.Name,
		Statement: "SELECT 1 FROM ghost WHERE ghost_secret = 'x' AND pg_sleep(30) IS NOT NULL",
	}))
	a.NoError(err)
	a.Less(time.Since(start), 20*time.Second)
	a.Len(ghostResp.Msg.Results, 1)
	a.True(strings.HasPrefix(ghostResp.Msg.Results[0].Error, refusedPredicate), ghostResp.Msg.Results[0].Error)
}
