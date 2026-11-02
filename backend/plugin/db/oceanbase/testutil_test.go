package oceanbase

import (
	"database/sql/driver"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

func columnNames(columns []*metadatapb.ColumnMetadata) []string {
	var names []string
	for _, column := range columns {
		names = append(names, column.Name)
	}
	return names
}

// testExplainResultRows is a fake result set for the EXPLAIN FORMAT=JSON plans the
// sync and stats tests feed through a registered test driver.
type testExplainResultRows struct {
	// columns defaults to the single EXPLAIN column of a JSON plan.
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *testExplainResultRows) Columns() []string {
	if r.columns != nil {
		return r.columns
	}
	return []string{"EXPLAIN"}
}

func (*testExplainResultRows) Close() error {
	return nil
}

func (r *testExplainResultRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}
