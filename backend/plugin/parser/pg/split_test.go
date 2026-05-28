package pg

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestPGSplitSQL(t *testing.T) {
	base.RunSplitTests(t, "test-data/test_split.yaml", base.SplitTestOptions{
		SplitFunc: SplitSQL,
	})
}

func TestPGSplitSQLLargeInsertScriptScalesLinearly(t *testing.T) {
	const rowCount = 2000
	padding := strings.Repeat("x", 1024)
	var builder strings.Builder
	for i := 0; i < rowCount; i++ {
		fmt.Fprintf(&builder, "INSERT INTO perf_omni_pg (id, payload) VALUES (%d, '%s');\n", i, padding)
	}

	started := time.Now()
	statements, err := SplitSQL(builder.String())
	elapsed := time.Since(started)

	require.NoError(t, err)
	require.Len(t, base.FilterEmptyStatements(statements), rowCount)
	require.Less(t, elapsed, time.Second)
}
