package bigquery

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/require"
)

// sampleStages is a two-stage plan of the shape BigQuery reports for a filtered
// aggregate: one stage reads the table and writes partial groups to shuffle, the
// next reads that shuffle and writes the query's rows.
func sampleStages() []*bigquery.ExplainQueryStage {
	return []*bigquery.ExplainQueryStage{
		{
			ID:                      0,
			Name:                    "S00: Input",
			Status:                  "COMPLETE",
			RecordsRead:             4,
			RecordsWritten:          2,
			ParallelInputs:          1,
			CompletedParallelInputs: 1,
			ShuffleOutputBytes:      72,
			WaitAvg:                 2 * time.Millisecond,
			WaitMax:                 3 * time.Millisecond,
			ReadAvg:                 11 * time.Millisecond,
			ReadMax:                 14 * time.Millisecond,
			ComputeAvg:              7 * time.Millisecond,
			ComputeMax:              9 * time.Millisecond,
			WriteAvg:                time.Millisecond,
			WriteMax:                time.Millisecond,
			Steps: []*bigquery.ExplainQueryStep{
				{Kind: "READ", Substeps: []string{"$1:city", "FROM analytics.customers", "WHERE equal($1, 'berlin')"}},
				{Kind: "AGGREGATE", Substeps: []string{"GROUP BY $10 := $1", "$20 := COUNT(*)"}},
				{Kind: "WRITE", Substeps: []string{"$10, $20", "TO __stage00_output"}},
			},
		},
		{
			ID:             1,
			Name:           "S01: Output",
			Status:         "COMPLETE",
			InputStages:    []int64{0},
			RecordsRead:    2,
			RecordsWritten: 1,
			ParallelInputs: 1,
			WaitAvg:        time.Millisecond,
			ComputeAvg:     3 * time.Millisecond,
			ComputeMax:     3 * time.Millisecond,
			Steps: []*bigquery.ExplainQueryStep{
				{Kind: "READ", Substeps: []string{"$10, $20", "FROM __stage00_output"}},
				{Kind: "AGGREGATE", Substeps: []string{"GROUP BY $30 := $10", "$40 := SUM_OF_COUNTS($20)"}},
				{Kind: "WRITE", Substeps: []string{"$30, $40", "TO __stage01_output"}},
			},
		},
	}
}

// TestToQueryStages pins what the result hands the viewer: BigQuery's own REST
// names, its durations as milliseconds, and the stage ids a stage's inputs
// refer to.
func TestToQueryStages(t *testing.T) {
	encoded, err := json.Marshal(toQueryStages(sampleStages()))
	require.NoError(t, err)

	var stages []map[string]any
	require.NoError(t, json.Unmarshal(encoded, &stages))
	require.Len(t, stages, 2)

	require.Equal(t, float64(0), stages[0]["id"])
	require.Equal(t, "S00: Input", stages[0]["name"])
	require.Equal(t, float64(4), stages[0]["recordsRead"])
	require.Equal(t, float64(2), stages[0]["recordsWritten"])
	require.Equal(t, float64(11), stages[0]["readMsAvg"])
	require.Equal(t, float64(9), stages[0]["computeMsMax"])
	require.Equal(t, float64(72), stages[0]["shuffleOutputBytes"])
	// The stage that reads another names it, which is the plan's one edge.
	require.Equal(t, []any{float64(0)}, stages[1]["inputStages"])
	// A step keeps the engine's own words, which is where the table appears.
	steps, ok := stages[0]["steps"].([]any)
	require.True(t, ok)
	require.Len(t, steps, 3)
	first, ok := steps[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "READ", first["kind"])
	require.Contains(t, first["substeps"], "FROM analytics.customers")

	// The ratios BigQuery reports beside the durations are left out, since each
	// is relative to the whole query's longest segment.
	for _, key := range []string{"waitRatioAvg", "readRatioAvg", "computeRatioAvg", "writeRatioAvg"} {
		require.NotContains(t, stages[0], key)
	}
	// A stage that spent no time writing says nothing about writing.
	require.NotContains(t, stages[1], "writeMsAvg")
}

// TestToQueryStagesKeepsZeroRecords pins the one measurement that carries a
// zero. A stage that wrote no records measured that, and the viewer reads these
// counts as the rows a node really moved, so dropping the zero would leave it
// showing an unknown count instead of an empty stage.
func TestToQueryStagesKeepsZeroRecords(t *testing.T) {
	encoded, err := json.Marshal(toQueryStages([]*bigquery.ExplainQueryStage{
		{ID: 0, Name: "S00: Input", RecordsRead: 0, RecordsWritten: 0},
	}))
	require.NoError(t, err)

	var stages []map[string]any
	require.NoError(t, json.Unmarshal(encoded, &stages))
	require.Len(t, stages, 1)
	require.Equal(t, float64(0), stages[0]["recordsRead"])
	require.Equal(t, float64(0), stages[0]["recordsWritten"])
}

func TestToQueryStagesSkipsNothing(t *testing.T) {
	// A nil stage or step in the list is skipped rather than encoded as null,
	// which the viewer would have to read as an operator.
	stages := toQueryStages([]*bigquery.ExplainQueryStage{
		nil,
		{ID: 1, Name: "S00: Input", Steps: []*bigquery.ExplainQueryStep{nil}},
	})
	require.Len(t, stages, 1)
	require.Empty(t, stages[0].Steps)

	require.Empty(t, toQueryStages(nil))
}

// TestQueryPlanWithoutJob pins that a read with no job behind it reports no
// plan, rather than failing the rows the caller asked for.
func TestQueryPlanWithoutJob(t *testing.T) {
	require.Nil(t, queryPlan(context.Background(), nil))
}
