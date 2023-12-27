package starrocks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTemporaryView(t *testing.T) {
	a := require.New(t)
	got := getTemporaryView("db1", []string{"col1", "col2"})
	want := "--\n-- Temporary view structure for `db1`\n--\nCREATE VIEW `db1` AS SELECT\n  1 AS `col1`,\n  1 AS `col2`;\n\n"
	a.Equal(want, got)
}
