package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetTemporalView(t *testing.T) {
	a := require.New(t)
	got := getTemporalView("db1", []string{"col1", "col2"})
	want := "--\n-- Temporal view structure for `db1`\n--\nCREATE VIEW `db1` AS SELECT\n  1 AS `col1`,\n  1 AS `col2`;\n\n"
	a.Equal(want, got)
}
