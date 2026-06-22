package pg

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// TestQueryConnMergeCTEDoesNotLeakRows is a security regression test for the
// PostgreSQL data-masking bypass where a MERGE smuggled into a data-modifying CTE
//
//	WITH x AS (MERGE ... RETURNING masked_col) SELECT masked_col FROM x
//
// was misclassified as a read-only query (hasDMLInTree did not recognise MergeStmt
// as a write). That routed it through QueryConn's row-returning QueryContext path,
// so the RETURNING columns came back to the client and — because the query-span
// extractor cannot trace lineage through a MERGE — were returned UNMASKED.
//
// The fix classifies MERGE as a write, so the editor routes the statement to
// ExecContext, which discards the RETURNING rows. This test asserts the
// security-relevant outcome at the driver boundary: the MERGE-CTE must NOT return
// the secret value as data rows. MERGE ... RETURNING requires PostgreSQL 17.
func TestQueryConnMergeCTEDoesNotLeakRows(t *testing.T) {
	ctx := context.Background()

	pgContainer := testcontainer.GetTestPg17Container(ctx, t)
	defer pgContainer.Close(ctx)

	const secret = "TOP-SECRET-SSN-123-45-6789"
	rawDB := pgContainer.GetDB()
	require.NoError(t, rawDB.Ping())
	_, err := rawDB.ExecContext(ctx, `CREATE TABLE secret_doc (id int PRIMARY KEY, secret text);`)
	require.NoError(t, err)
	_, err = rawDB.ExecContext(ctx, `INSERT INTO secret_doc VALUES (1, '`+secret+`');`)
	require.NoError(t, err)

	driver, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     pgContainer.GetHost(),
			Port:     pgContainer.GetPort(),
			Username: "postgres",
		},
		Password:          "root-password",
		ConnectionContext: db.ConnectionContext{DatabaseName: "postgres"},
	})
	require.NoError(t, err)
	defer driver.Close(ctx)

	conn, err := driver.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// Baseline: a plain SELECT returns the secret as a data row. This proves the
	// harness actually surfaces row data, so the negative assertion below is meaningful.
	selectResults, err := driver.QueryConn(ctx, conn, `SELECT secret FROM secret_doc`, db.QueryContext{Limit: 5000, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.True(t, resultsContainValue(selectResults, secret),
		"baseline plain SELECT should return the secret as a data row")

	// Exploit attempt: a MERGE smuggled into a data-modifying CTE. After the fix it
	// is classified as a write and routed to Exec, so the RETURNING rows are dropped
	// and the secret is never returned as data.
	const mergeCTE = `WITH x AS (
		MERGE INTO secret_doc t USING secret_doc s ON t.id = s.id
		WHEN MATCHED THEN UPDATE SET id = t.id
		RETURNING t.secret AS secret
	) SELECT secret FROM x`
	mergeResults, err := driver.QueryConn(ctx, conn, mergeCTE, db.QueryContext{Limit: 5000, MaximumSQLResultSize: 1 << 30})
	require.NoError(t, err)
	require.False(t, resultsContainValue(mergeResults, secret),
		"SECURITY: MERGE-in-CTE must not return the masked column as data rows; got %v", mergeResults)
}

// resultsContainValue reports whether any string cell in the results contains want.
func resultsContainValue(results []*v1pb.QueryResult, want string) bool {
	for _, r := range results {
		for _, row := range r.GetRows() {
			for _, v := range row.GetValues() {
				if sv, ok := v.GetKind().(*v1pb.RowValue_StringValue); ok && strings.Contains(sv.StringValue, want) {
					return true
				}
			}
		}
	}
	return false
}
