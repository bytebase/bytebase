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

-- EstimateRows is a cardinality estimate, so it is only stable once the
-- optimizer has statistics. Without them SQL Server falls back to guessing a
-- fixed selectivity for a range predicate and the counts below drift.
UPDATE STATISTICS dbo.t WITH FULLSCAN;
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
		require.Len(t, summary.SampleDMLS, 1)

		sampled, err := driver.CountAffectedRows(ctx, summary.SampleDMLS[0])
		require.NoError(t, err, "SHOWPLAN must accept the sample the extractor emits")
		want, err := driver.CountAffectedRows(ctx, bare)
		require.NoError(t, err)
		require.Equal(t, want, sampled)
	})
}
