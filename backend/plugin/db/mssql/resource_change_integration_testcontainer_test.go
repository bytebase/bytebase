package mssql

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tsql" // register MSSQL statement parser
)

// TestResourceChangeIntegrationMSSQL verifies the resource_change.go migration
// against a real SQL Server. It targets three claims made during review:
//  1. INSERT ... DEFAULT VALUES: our new code skips EXPLAIN and directly
//     counts 1 row. Proving SHOWPLAN would also return 1 confirms the final
//     affected-rows number is unchanged.
//  2. A statement whose text carries a leading SQL comment (from the
//     split-layer's gap-absorption behavior) must still be accepted by
//     SHOWPLAN_ALL so the downstream estimate is unaffected.
//  3. DROP INDEX ... ON t registers t in ChangedResources (omni 51d40fb
//     exposed the target table; migration now consumes it).
//
//nolint:tparallel
func TestResourceChangeIntegrationMSSQL(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestMSSQLContainer(ctx, t)
	defer container.Close(ctx)

	host := container.GetHost()
	portInt, err := strconv.Atoi(container.GetPort())
	require.NoError(t, err)

	dbName := fmt.Sprintf("resource_change_%d", time.Now().UnixNano())
	driver := openMSSQL(ctx, t, host, portInt, "master")
	_, err = driver.Execute(ctx, fmt.Sprintf("CREATE DATABASE [%s]", dbName), db.ExecuteOptions{CreateDatabase: true})
	require.NoError(t, err)
	driver.Close(ctx)

	defer func() {
		cleanup := openMSSQL(ctx, t, host, portInt, "master")
		defer cleanup.Close(ctx)
		_, _ = cleanup.Execute(ctx, fmt.Sprintf("ALTER DATABASE [%s] SET SINGLE_USER WITH ROLLBACK IMMEDIATE", dbName), db.ExecuteOptions{CreateDatabase: true})
		_, _ = cleanup.Execute(ctx, fmt.Sprintf("DROP DATABASE [%s]", dbName), db.ExecuteOptions{CreateDatabase: true})
	}()

	driver = openMSSQL(ctx, t, host, portInt, dbName)
	defer driver.Close(ctx)
	mssqlDriver, ok := driver.(*Driver)
	require.True(t, ok, "expected *Driver")

	// Seed a small table with 5 rows so UPDATE/DELETE have something to plan against.
	setup := []string{
		`CREATE TABLE dbo.t (id INT IDENTITY(1,1) PRIMARY KEY, c1 INT NULL)`,
		`INSERT INTO dbo.t (c1) VALUES (1),(2),(3),(4),(5)`,
		`CREATE INDEX idx_c1 ON dbo.t(c1)`,
	}
	for _, s := range setup {
		_, err := driver.Execute(ctx, s, db.ExecuteOptions{})
		require.NoError(t, err, "setup: %s", s)
	}

	t.Run("INSERT_DEFAULT_VALUES_parity", func(t *testing.T) {
		sql := `INSERT INTO dbo.t DEFAULT VALUES;`

		// Path 1 (new code): extractor must route this into InsertCount with no sample.
		summary := summarize(t, sql)
		require.Equal(t, 1, summary.InsertCount, "DEFAULT VALUES should be counted as 1 known insert")
		require.Equal(t, 0, summary.DMLCount, "DEFAULT VALUES must not be sent to EXPLAIN")
		require.Empty(t, summary.SampleDMLS, "no sample text should be emitted for DEFAULT VALUES")

		// Path 2 (proof of parity): if we HAD sent it through SHOWPLAN, what would it return?
		// Matches the old ANTLR code's dmlCount+sample path. Must be 1 so the final affected-rows
		// number is identical between old and new.
		showplanRows, err := mssqlDriver.CountAffectedRows(ctx, sql)
		require.NoError(t, err)
		require.EqualValues(t, 1, showplanRows, "SHOWPLAN_ALL must also estimate DEFAULT VALUES as 1 row")
	})

	t.Run("UPDATE_with_leading_comment_showplan_ok", func(t *testing.T) {
		// New code's SampleDMLS may contain a leading comment because
		// split.go absorbs inter-statement gaps into the next statement.
		// Assert that SQL Server's SHOWPLAN_ALL accepts such text verbatim
		// and returns a usable EstimateRows, so the downstream number is
		// not affected by the cosmetic regression.
		sampleAsSentToShowplan := "-- header comment\nUPDATE dbo.t SET c1 = 99 WHERE id = 1;"
		rows, err := mssqlDriver.CountAffectedRows(ctx, sampleAsSentToShowplan)
		require.NoError(t, err, "SHOWPLAN should accept leading comments")
		require.EqualValues(t, 1, rows, "UPDATE of id=1 should estimate 1 row regardless of comment prefix")

		// Control: same UPDATE with no comment prefix should yield the same estimate.
		rows2, err := mssqlDriver.CountAffectedRows(ctx, "UPDATE dbo.t SET c1 = 99 WHERE id = 1;")
		require.NoError(t, err)
		require.Equal(t, rows, rows2, "leading comment must not change the estimate")
	})

	t.Run("INSERT_VALUES_multi_row_insertCount", func(t *testing.T) {
		sql := `INSERT INTO dbo.t (c1) VALUES (10),(20),(30);`
		summary := summarize(t, sql)
		require.Equal(t, 3, summary.InsertCount)
		require.Equal(t, 0, summary.DMLCount)
		require.Empty(t, summary.SampleDMLS)
	})

	t.Run("DROP_INDEX_tracks_target_table", func(t *testing.T) {
		sql := `DROP INDEX idx_c1 ON dbo.t;`
		summary := summarize(t, sql)

		proto := summary.ChangedResources.Build()
		found := false
		for _, database := range proto.Databases {
			for _, schema := range database.Schemas {
				for _, tbl := range schema.Tables {
					if schema.Name == "dbo" && tbl.Name == "t" {
						found = true
					}
				}
			}
		}
		require.True(t, found, "DROP INDEX ... ON dbo.t should register dbo.t as a changed table (omni 51d40fb)")

		// Sanity: executing the DROP INDEX against the seeded DB should succeed,
		// proving the SQL we parsed is actually runnable.
		_, err := driver.Execute(ctx, sql, db.ExecuteOptions{})
		require.NoError(t, err)
	})

	t.Run("UPDATE_goes_through_explain", func(t *testing.T) {
		sql := `UPDATE dbo.t SET c1 = c1 + 1 WHERE id > 2;`
		summary := summarize(t, sql)
		require.Equal(t, 0, summary.InsertCount)
		require.Equal(t, 1, summary.DMLCount)
		require.Len(t, summary.SampleDMLS, 1)
		require.Equal(t, strings.TrimSpace(sql), summary.SampleDMLS[0])

		rows, err := mssqlDriver.CountAffectedRows(ctx, summary.SampleDMLS[0])
		require.NoError(t, err)
		// Seeded 5 rows with id 1..5; predicate matches ids 3,4,5 → 3 rows.
		require.EqualValues(t, 3, rows)
	})
}

func summarize(t *testing.T, sql string) *parserbase.ChangeSummary {
	t.Helper()
	stmts, err := parserbase.ParseStatements(storepb.Engine_MSSQL, sql)
	require.NoError(t, err)
	asts := parserbase.ExtractASTs(stmts)
	summary, err := parserbase.ExtractChangedResources(storepb.Engine_MSSQL, "resource_change_test", "dbo", nil, asts, sql)
	require.NoError(t, err)
	return summary
}

func openMSSQL(ctx context.Context, t *testing.T, host string, port int, database string) db.Driver {
	t.Helper()
	driverInstance := &Driver{}
	cfg := db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Type:     storepb.DataSourceType_ADMIN,
			Username: "sa",
			Host:     host,
			Port:     strconv.Itoa(port),
			Database: database,
		},
		Password: "Test123!",
		ConnectionContext: db.ConnectionContext{
			DatabaseName: database,
		},
	}
	driver, err := driverInstance.Open(ctx, storepb.Engine_MSSQL, cfg)
	require.NoError(t, err)
	return driver
}
