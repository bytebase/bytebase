package mssql

import (
	"context"
	"testing"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

// TestCountAffectedRows pins what SHOWPLAN_ALL estimates for the statement
// shapes the resource-change extractor hands to it. INSERT ... DEFAULT VALUES
// is counted by the extractor rather than planned, so the parity assertion
// below is what proves skipping the plan does not change the reported number.
// The split layer absorbs inter-statement gaps into the next statement, so a
// sample can reach SHOWPLAN with a leading comment.
func TestCountAffectedRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container := testcontainer.SharedMSSQLContainer(t)

	driver := newSyncTestDatabase(ctx, t, container)
	executeBatches(ctx, t, driver, `
CREATE TABLE dbo.t (id INT IDENTITY(1,1) PRIMARY KEY, c1 INT NULL);
GO

INSERT INTO dbo.t (c1) VALUES (1),(2),(3),(4),(5);
GO

CREATE TABLE dbo.s (id INT PRIMARY KEY, flag BIT NOT NULL);
CREATE TABLE dbo.big (id INT PRIMARY KEY, s_id INT NOT NULL, v INT NOT NULL);
CREATE TABLE dbo.big_archive (id INT PRIMARY KEY, s_id INT NOT NULL, v INT NOT NULL);
GO

-- s flags 10 of its 100 rows and big holds 100 rows for each row of s, so
-- joining big to the flagged rows matches 1000 of its 10000 rows.
INSERT INTO dbo.s SELECT n, CASE WHEN n <= 10 THEN 1 ELSE 0 END FROM (SELECT TOP (100) ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS n FROM sys.all_objects) AS seq;
INSERT INTO dbo.big SELECT n, n % 100 + 1, 0 FROM (SELECT TOP (10000) ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) AS n FROM sys.all_objects AS a CROSS JOIN sys.all_objects AS b) AS seq;
GO

CREATE INDEX big_s_id ON dbo.big (s_id);
CREATE STATISTICS s_flag ON dbo.s (flag);
GO

-- EstimateRows is a cardinality estimate, so it is only stable once the
-- optimizer has statistics. Without them SQL Server falls back to guessing a
-- fixed selectivity for a range predicate and the counts below drift.
UPDATE STATISTICS dbo.t WITH FULLSCAN;
UPDATE STATISTICS dbo.s WITH FULLSCAN;
UPDATE STATISTICS dbo.big WITH FULLSCAN;
GO
`)

	t.Run("insert_default_values_estimates_one_row", func(t *testing.T) {
		t.Parallel()

		rows, err := driver.CountAffectedRows(ctx, `INSERT INTO dbo.t DEFAULT VALUES;`)
		require.NoError(t, err)
		require.EqualValues(t, 1, rows)
	})

	t.Run("leading_comment_does_not_change_estimate", func(t *testing.T) {
		t.Parallel()

		const update = "UPDATE dbo.t SET c1 = 99 WHERE id = 1;"

		withComment, err := driver.CountAffectedRows(ctx, "-- header comment\n"+update)
		require.NoError(t, err)
		bare, err := driver.CountAffectedRows(ctx, update)
		require.NoError(t, err)
		require.Equal(t, bare, withComment)
	})

	t.Run("predicate_narrows_the_estimate", func(t *testing.T) {
		t.Parallel()

		// Seeded ids are 1..5, so the predicate matches 3, 4 and 5.
		rows, err := driver.CountAffectedRows(ctx, `UPDATE dbo.t SET c1 = c1 + 1 WHERE id > 2;`)
		require.NoError(t, err)
		require.EqualValues(t, 3, rows)
	})

	for _, tc := range []struct {
		name      string
		statement string
		// changedRows is how many rows the statement would change. An estimate can miss it,
		// as MERGE estimates 899 of 1000 on SQL Server 2022, so the bounds allow a factor of
		// two, which still excludes the 10, 100, and 10000 rows that the join inputs estimate.
		changedRows int64
	}{
		{
			name:        "delete_with_predicate",
			statement:   `DELETE FROM dbo.big WHERE s_id <= 3;`,
			changedRows: 300,
		},
		{
			name:        "delete_join",
			statement:   `DELETE b FROM dbo.big AS b JOIN dbo.s AS s ON b.s_id = s.id WHERE s.flag = 1;`,
			changedRows: 1000,
		},
		{
			name:        "update_from_join",
			statement:   `UPDATE b SET v = 1 FROM dbo.big AS b JOIN dbo.s AS s ON b.s_id = s.id WHERE s.flag = 1;`,
			changedRows: 1000,
		},
		{
			name:        "delete_where_in_subquery",
			statement:   `DELETE FROM dbo.big WHERE s_id IN (SELECT id FROM dbo.s WHERE flag = 1);`,
			changedRows: 1000,
		},
		{
			name:        "insert_select_with_filter",
			statement:   `INSERT INTO dbo.big_archive SELECT id, s_id, v FROM dbo.big WHERE s_id > 95;`,
			changedRows: 500,
		},
		{
			name:        "insert_select_with_join",
			statement:   `INSERT INTO dbo.big_archive SELECT b.id, b.s_id, b.v FROM dbo.big AS b JOIN dbo.s AS s ON b.s_id = s.id WHERE s.flag = 1;`,
			changedRows: 1000,
		},
		{
			name: "merge",
			statement: `MERGE dbo.big_archive AS tgt
USING (SELECT b.id, b.s_id, b.v FROM dbo.big AS b JOIN dbo.s AS s ON b.s_id = s.id WHERE s.flag = 1) AS src
ON tgt.id = src.id
WHEN NOT MATCHED THEN INSERT (id, s_id, v) VALUES (src.id, src.s_id, src.v);`,
			changedRows: 1000,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rows, err := driver.CountAffectedRows(ctx, tc.statement)
			require.NoError(t, err)
			require.Greater(t, rows, tc.changedRows/2)
			require.LessOrEqual(t, rows, tc.changedRows*2)
		})
	}

	// The seam between the two halves: neither the tsql extractor tests nor the
	// subtests above can catch a sample that the extractor emits but SHOWPLAN
	// rejects, so feed the extractor's own output in rather than a literal.
	t.Run("extractor_sample_is_accepted_by_showplan", func(t *testing.T) {
		t.Parallel()

		const bare = "UPDATE dbo.t SET c1 = c1 + 1 WHERE id > 2;"
		const statement = "-- header comment\n" + bare

		stmts, err := parserbase.ParseStatements(storepb.Engine_MSSQL, statement)
		require.NoError(t, err)
		summary, err := parserbase.ExtractChangedResources(
			storepb.Engine_MSSQL, "resource_change_test", "dbo", nil, parserbase.ExtractASTs(stmts), statement)
		require.NoError(t, err)
		require.Len(t, summary.DMLStatements, 1)

		sampled, err := driver.CountAffectedRows(ctx, summary.DMLStatements[0])
		require.NoError(t, err, "SHOWPLAN must accept the sample the extractor emits")
		want, err := driver.CountAffectedRows(ctx, bare)
		require.NoError(t, err)
		require.Equal(t, want, sampled)
	})
}
