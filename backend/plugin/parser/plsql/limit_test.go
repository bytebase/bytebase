package plsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestStatementWithResultLimitSeam(t *testing.T) {
	// Oracle 12c+ uses FETCH NEXT.
	got := base.StatementWithResultLimit(storepb.Engine_ORACLE, "SELECT * FROM t", 10, "19.3.0.0.0")
	require.Contains(t, got, "FETCH NEXT 10 ROWS ONLY")

	// Pre-12c uses ROWNUM wrapping.
	got = base.StatementWithResultLimit(storepb.Engine_ORACLE, "SELECT * FROM t", 10, "11.2.0.4.0")
	require.Contains(t, got, "ROWNUM <= 10")

	// A FETCH NEXT rewrite error falls back to ROWNUM.
	got = base.StatementWithResultLimit(storepb.Engine_ORACLE, "SELECT * FROM t FETCH FIRST 50 PERCENT ROWS ONLY", 10, "19.3.0.0.0")
	require.Contains(t, got, "ROWNUM <= 10")
}
