package mssql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestShowplanStatistic(t *testing.T) {
	for _, tc := range []struct {
		name   string
		option *v1pb.QueryOption
		want   string
	}{
		{name: "no option", option: nil, want: "SHOWPLAN_ALL"},
		{name: "empty option", option: &v1pb.QueryOption{}, want: "SHOWPLAN_ALL"},
		{
			name:   "text",
			option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_TEXT},
			want:   "SHOWPLAN_ALL",
		},
		{
			name:   "xml",
			option: &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_XML},
			want:   "SHOWPLAN_XML",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, showplanStatistic(tc.option))
		})
	}
}

func TestQueryConnRecordsLimitedStatement(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, "CREATE TABLE dbo.t (id INT PRIMARY KEY);\nINSERT INTO dbo.t VALUES (1), (2), (3);")

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// SET returns nothing, so the driver pads an empty result in for it.
	results, err := driver.QueryConn(ctx, conn, "SET NOCOUNT OFF;\nSELECT TOP 3 id FROM dbo.t ORDER BY id;", db.QueryContext{
		Limit:                2,
		MaximumSQLResultSize: 1 << 30,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "SET NOCOUNT OFF;", results[0].GetStatement())
	require.Equal(t, "\nSELECT TOP 2 id FROM dbo.t ORDER BY id;", results[1].GetStatement())
	require.Len(t, results[1].GetRows(), 2)
}

func TestQueryConnMatchesResultsToStatements(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	driver := newSyncTestDatabase(ctx, t, testcontainer.SharedMSSQLContainer(t))
	executeBatches(ctx, t, driver, `CREATE TABLE dbo.t (id INT PRIMARY KEY);
INSERT INTO dbo.t VALUES (1), (2), (3);
GO
CREATE PROCEDURE dbo.two_sets AS
BEGIN
	UPDATE dbo.t SET id = id;
	SELECT id FROM dbo.t ORDER BY id;
	SELECT 'second' AS s;
END`)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()
	queryContext := db.QueryContext{MaximumSQLResultSize: 1 << 30}
	describe := func(results []*v1pb.QueryResult) []string {
		var got []string
		for _, r := range results {
			line := fmt.Sprintf("%s %v %d", strings.TrimSpace(r.GetStatement()), r.GetColumnNames(), len(r.GetRows()))
			if r.GetError() != "" {
				line += " error"
			}
			got = append(got, line)
		}
		return got
	}

	// Statements that return nothing keep their places across GO, and the
	// procedure's second result set comes after every statement's result.
	results, err := driver.QueryConn(ctx, conn, `EXEC dbo.two_sets;
DECLARE @x INT;
SELECT @x = id FROM dbo.t;
GO
DECLARE @y INT;
SELECT @y = id FROM dbo.t;
SELECT id FROM dbo.t WHERE id = 3;`, queryContext)
	require.NoError(t, err)
	require.Equal(t, []string{
		"EXEC dbo.two_sets; [id] 3",
		"DECLARE @x INT; [] 0",
		"SELECT @x = id FROM dbo.t; [] 0",
		"DECLARE @y INT; [] 0",
		"SELECT @y = id FROM dbo.t; [] 0",
		"SELECT id FROM dbo.t WHERE id = 3; [id] 1",
		"EXEC dbo.two_sets; [s] 1",
	}, describe(results))

	results, err = driver.QueryConn(ctx, conn, "EXEC dbo.two_sets;", db.QueryContext{Limit: 2, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Equal(t, []string{"EXEC dbo.two_sets; [id] 2", "EXEC dbo.two_sets; [s] 1"}, describe(results))

	// Masking cannot trace the rows of a procedure call or an OUTPUT clause.
	results, err = driver.QueryConn(ctx, conn, "EXEC dbo.two_sets;\nGO\nUPDATE dbo.t SET id = id OUTPUT inserted.id;\nSELECT id FROM dbo.t WHERE id = 1;", db.QueryContext{MaskingEnabled: true, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.Equal(t, []string{
		"EXEC dbo.two_sets; [id] 0 error",
		"UPDATE dbo.t SET id = id OUTPUT inserted.id; [id] 0 error",
		"SELECT id FROM dbo.t WHERE id = 1; [id] 1",
		"EXEC dbo.two_sets; [s] 0 error",
	}, describe(results))
	require.Contains(t, results[0].GetError(), "data masking")

	_, err = driver.QueryConn(ctx, conn, "EXEC dbo.two_sets;\nSELECT 1 AS one;", queryContext)
	require.ErrorContains(t, err, "cannot match the batch's result sets to its statements")

	_, err = driver.QueryConn(ctx, conn, "SET NOEXEC ON;\nSELECT 1 AS one;", queryContext)
	require.EqualError(t, err, "SET NOEXEC is not supported")

	_, err = driver.QueryConn(ctx, conn, "SELECT 1 AS one;\nSELECT 1/0 AS x;", queryContext)
	require.ErrorContains(t, err, "Divide by zero error encountered")

	// A result set cut off at the size limit must not stall the rest of the batch.
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	results, err = driver.QueryConn(timeoutCtx, conn, "SELECT TOP 100 name FROM sys.all_objects;\nSELECT 1 AS one;", db.QueryContext{MaximumSQLResultSize: 1})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Contains(t, results[0].GetError(), "exceeds max allowed output size")
	require.Len(t, results[1].GetRows(), 1)
}

func TestMatchResults(t *testing.T) {
	const (
		rows  = stmtTypeResultSetGenerating | stmtTypeRowCountGenerating
		count = stmtTypeRowCountGenerating
		none  = stmtTypeUnknown
		proc  = stmtTypeProcedure
	)
	// describe names a result by its column, or by its affected rows.
	describe := func(results []*v1pb.QueryResult) []string {
		var got []string
		for _, r := range results {
			name := strings.Join(r.GetColumnNames(), ",")
			if name == "Affected Rows" {
				name = fmt.Sprintf("affected %d", r.GetRows()[0].GetValues()[0].GetInt64Value())
			}
			got = append(got, r.GetStatement()+":"+name)
		}
		return got
	}
	for _, tc := range []struct {
		name      string
		types     []stmtType
		sets      []string
		counts    []int64
		want      []string
		wantExtra []string
		wantErr   bool
	}{
		{
			name:   "each statement gets its own result",
			types:  []stmtType{none, rows, count, rows},
			sets:   []string{"a", "b"},
			counts: []int64{3},
			want:   []string{"s0:", "s1:a", "s2:affected 3", "s3:b"},
		},
		{
			name:      "a procedure call takes every result set",
			types:     []stmtType{none, proc, count},
			sets:      []string{"a", "b", "c"},
			counts:    []int64{4},
			want:      []string{"s0:", "s1:a", "s2:affected 4"},
			wantExtra: []string{"s1:b", "s1:c"},
		},
		{
			name:  "a procedure call that returns nothing",
			types: []stmtType{proc, rows},
			sets:  []string{"a"},
			want:  []string{"s0:", "s1:a"},
		},
		{
			name:   "row counts that cannot be told apart are dropped",
			types:  []stmtType{count, proc, count},
			counts: []int64{1, 2, 3},
			want:   []string{"s0:", "s1:", "s2:"},
		},
		{
			name:    "a procedure call beside another statement that returns rows",
			types:   []stmtType{proc, rows},
			sets:    []string{"a", "b"},
			wantErr: true,
		},
		{
			name:    "two procedure calls",
			types:   []stmtType{proc, proc},
			sets:    []string{"a"},
			wantErr: true,
		},
		{
			name:    "a statement without its result set",
			types:   []stmtType{rows, rows},
			sets:    []string{"a"},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statements := make([]string, len(tc.types))
			for i := range statements {
				statements[i] = fmt.Sprintf("s%d", i)
			}
			var sets []*v1pb.QueryResult
			for _, column := range tc.sets {
				sets = append(sets, &v1pb.QueryResult{ColumnNames: []string{column}})
			}
			results, extra, err := matchResults(statements, tc.types, sets, tc.counts)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, describe(results))
			require.Equal(t, tc.wantExtra, describe(extra))
		})
	}
}
