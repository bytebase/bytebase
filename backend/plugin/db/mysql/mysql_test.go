package mysql

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestUpdateTiDBExplainResult(t *testing.T) {
	a := require.New(t)
	got := &v1pb.QueryResult{
		Rows: []*v1pb.QueryRow{
			convertExplainRow("Projection_9", "a"),
			convertExplainRow("└─IndexJoin_13", "b"),
			convertExplainRow("├─TableReader_26(Build)", "c"),
			convertExplainRow("│ └─Selection_25", "d"),
			convertExplainRow("│   └─TableFullScan_24", "e"),
			convertExplainRow("└─IndexLookUp_12(Probe)", "f"),
			convertExplainRow("├─IndexRangeScan_10(Build)", "g"),
			convertExplainRow("└─TableRowIDScan_11(Probe)", "h"),
		},
	}
	want := &v1pb.QueryResult{
		Rows: []*v1pb.QueryRow{
			convertExplainRow("Projection_9", "a"),
			convertExplainRow("--IndexJoin_13", "b"),
			convertExplainRow("--TableReader_26(Build)", "c"),
			convertExplainRow("----Selection_25", "d"),
			convertExplainRow("------TableFullScan_24", "e"),
			convertExplainRow("--IndexLookUp_12(Probe)", "f"),
			convertExplainRow("--IndexRangeScan_10(Build)", "g"),
			convertExplainRow("--TableRowIDScan_11(Probe)", "h"),
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
