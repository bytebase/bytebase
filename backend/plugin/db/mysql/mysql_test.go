package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestUpdateTiDBExplainResult(t *testing.T) {
	a := require.New(t)
	firstValues := []string{
		"Projection_9",
		"└─IndexJoin_13",
		"├─TableReader_26(Build)",
		"│ └─Selection_25",
		"│   └─TableFullScan_24",
		"└─IndexLookUp_12(Probe)",
		"├─IndexRangeScan_10(Build)",
		"└─TableRowIDScan_11(Probe)",
	}
	secondValues := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	got := &v1pb.QueryResult{}
	for i := 0; i < len(firstValues); i++ {
		got.Rows = append(got.Rows, convertExplainRow(firstValues[i], secondValues[i]))
	}

	want := &v1pb.QueryResult{
		Rows: []*v1pb.QueryRow{
			convertExplainRow(strings.Join(firstValues, "\n"), strings.Join(secondValues, "\n")),
		},
	}
	err := updateTiDBExplainResult(got)
	a.NoError(err)
	a.Equal(want, got)
}

func convertExplainRow(values ...string) *v1pb.QueryRow {
	row := &v1pb.QueryRow{}
	for _, v := range values {
		row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}})
	}
	return row
}
