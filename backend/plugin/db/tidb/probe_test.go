package tidb

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"

	// The review check parses with Bytebase's TiDB parser and runs the engine's
	// registered advisors, neither of which this package otherwise imports.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

// TestDryRunRuleProbesThroughTheAdapter is the advisor seam's acceptance test:
// STATEMENT_DML_DRY_RUN runs its EXPLAIN through the driver's Probe, so a
// statement the server refuses produces advice and one it accepts produces none.
// Before the seam, a driver without a *sql.DB answered zero queries and zero
// advice for every statement, which read as a pass.
func TestDryRunRuleProbesThroughTheAdapter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.t (id INT PRIMARY KEY, note VARCHAR(64))", database))
	require.NoError(t, err)

	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	for _, tc := range []struct {
		name       string
		statement  string
		wantAdvice bool
	}{
		{
			name:      "a dry run the server accepts gives no advice",
			statement: fmt.Sprintf("DELETE FROM %s.t WHERE id = 1;", database),
		},
		{
			name:       "a dry run the server refuses gives advice",
			statement:  fmt.Sprintf("DELETE FROM %s.t WHERE missing_column = 1;", database),
			wantAdvice: true,
		},
		{
			// BATCH ... DRY RUN cannot run inside a transaction, so this case
			// only passes when the rule's Autocommit probe is honoured.
			name:      "a batch dry run outside a transaction gives no advice",
			statement: fmt.Sprintf("BATCH ON id LIMIT 2 DELETE FROM %s.t WHERE id > 0;", database),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			adviceList, err := advisor.SQLReviewCheck(ctx, sheet.NewManager(), tc.statement, []*storepb.SQLReviewRule{{
				Type:   storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
				Level:  storepb.SQLReviewRule_WARNING,
				Engine: storepb.Engine_TIDB,
			}}, advisor.Context{
				DBType:          storepb.Engine_TIDB,
				Prober:          driver,
				NoAppendBuiltin: true,
			})
			require.NoError(t, err)
			if !tc.wantAdvice {
				require.Empty(t, adviceList)
				return
			}
			require.Len(t, adviceList, 1)
			require.Contains(t, adviceList[0].GetContent(), "dry runs failed")
		})
	}
}

// TestProbeHonoursAutocommit pins both shapes of the seam. A transactional probe
// opens a rolled-back transaction around its pre-executions and the statement; a
// BATCH ... DRY RUN statement answers only outside one, which is why the rule sets
// Autocommit for it.
func TestProbeHonoursAutocommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	container, database := testcontainer.NewTiDBDatabase(t)
	_, err := container.GetDB().ExecContext(ctx, fmt.Sprintf("CREATE TABLE %s.t (id INT PRIMARY KEY)", database))
	require.NoError(t, err)

	driver := openTestDriverOn(ctx, t, container, database)
	t.Cleanup(func() { require.NoError(t, driver.Close(ctx)) })

	batch := fmt.Sprintf("BATCH ON id LIMIT 2 DRY RUN DELETE FROM %s.t WHERE id > 0", database)
	_, err = driver.Probe(ctx, db.ProbeContext{}, batch)
	require.Error(t, err, "TiDB refuses BATCH ... DRY RUN inside a transaction")
	_, err = driver.Probe(ctx, db.ProbeContext{Autocommit: true}, batch)
	require.NoError(t, err)

	// A transactional probe reads its statement's rows and rolls back.
	probe, err := driver.Probe(ctx, db.ProbeContext{}, fmt.Sprintf("EXPLAIN DELETE FROM %s.t WHERE id > 0", database))
	require.NoError(t, err)
	require.NotEmpty(t, probe.Columns)
	require.NotEmpty(t, probe.Rows)

	// Its pre-executions ran in the same rolled-back transaction, so the write
	// they made is gone.
	probe, err = driver.Probe(ctx, db.ProbeContext{
		PreExecutions: []string{fmt.Sprintf("INSERT INTO %s.t VALUES (42)", database)},
	}, "SELECT COUNT(*) FROM t")
	require.NoError(t, err)
	require.Len(t, probe.Rows, 1)
	require.Equal(t, []any{int64(1)}, probe.Rows[0])

	var count int
	require.NoError(t, container.GetDB().QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s.t", database)).Scan(&count))
	require.Equal(t, 0, count, "the probe's pre-execution was rolled back")

	// Pre-executions cannot run outside a transaction, as the seam states.
	_, err = driver.Probe(ctx, db.ProbeContext{Autocommit: true, PreExecutions: []string{"SET @x = 1"}}, "SELECT 1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "pre-executions cannot run outside a transaction")
}
