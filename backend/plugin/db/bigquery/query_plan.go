package bigquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// BigQuery has no EXPLAIN. What it has is the plan of a job that ran: the
// stages the query was executed as, each naming the stages that fed it, the
// records it read and wrote, and how long its shards spent waiting, reading,
// computing and writing. That makes it a record of the run rather than an
// estimate of the next one, and it arrives with the job rather than in the
// result's rows — which is why the result carries it in QueryPlan.content.

// queryStage is one stage of a query plan, named as BigQuery's own REST API
// names it (ExplainQueryStage) rather than as the Go client's fields, so the
// viewer reads the shape Google documents. Durations are milliseconds, as that
// API states them; the ids are numbers here, where it quotes them.
//
// Its four ratio fields are left out. Each is relative to the longest-running
// segment of the whole query, which a single node's card cannot explain, and
// the milliseconds beside them say the same thing absolutely.
//
// The record counts carry a zero, where everything else here drops one. A stage
// that wrote no records measured that — a filter left nothing for the aggregate
// above it — and the viewer reads these two as the rows a node really moved, so
// dropping the zero would leave it showing an unknown count instead. A stage
// that spent no time writing has nothing to say about writing, so those fields
// stay out.
type queryStage struct {
	ID                        int64       `json:"id"`
	Name                      string      `json:"name"`
	Status                    string      `json:"status,omitempty"`
	InputStages               []int64     `json:"inputStages,omitempty"`
	Steps                     []queryStep `json:"steps,omitempty"`
	RecordsRead               int64       `json:"recordsRead"`
	RecordsWritten            int64       `json:"recordsWritten"`
	ParallelInputs            int64       `json:"parallelInputs,omitempty"`
	CompletedParallelInputs   int64       `json:"completedParallelInputs,omitempty"`
	ShuffleOutputBytes        int64       `json:"shuffleOutputBytes,omitempty"`
	ShuffleOutputBytesSpilled int64       `json:"shuffleOutputBytesSpilled,omitempty"`
	WaitMsAvg                 int64       `json:"waitMsAvg,omitempty"`
	WaitMsMax                 int64       `json:"waitMsMax,omitempty"`
	ReadMsAvg                 int64       `json:"readMsAvg,omitempty"`
	ReadMsMax                 int64       `json:"readMsMax,omitempty"`
	ComputeMsAvg              int64       `json:"computeMsAvg,omitempty"`
	ComputeMsMax              int64       `json:"computeMsMax,omitempty"`
	WriteMsAvg                int64       `json:"writeMsAvg,omitempty"`
	WriteMsMax                int64       `json:"writeMsMax,omitempty"`
}

// queryStep is one operation within a stage, in dependency order.
type queryStep struct {
	Kind     string   `json:"kind"`
	Substeps []string `json:"substeps,omitempty"`
}

func milliseconds(d time.Duration) int64 {
	return int64(d / time.Millisecond)
}

func toQueryStages(stages []*bigquery.ExplainQueryStage) []queryStage {
	encoded := make([]queryStage, 0, len(stages))
	for _, stage := range stages {
		if stage == nil {
			continue
		}
		steps := make([]queryStep, 0, len(stage.Steps))
		for _, step := range stage.Steps {
			if step == nil {
				continue
			}
			steps = append(steps, queryStep{Kind: step.Kind, Substeps: step.Substeps})
		}
		encoded = append(encoded, queryStage{
			ID:                        stage.ID,
			Name:                      stage.Name,
			Status:                    stage.Status,
			InputStages:               stage.InputStages,
			Steps:                     steps,
			RecordsRead:               stage.RecordsRead,
			RecordsWritten:            stage.RecordsWritten,
			ParallelInputs:            stage.ParallelInputs,
			CompletedParallelInputs:   stage.CompletedParallelInputs,
			ShuffleOutputBytes:        stage.ShuffleOutputBytes,
			ShuffleOutputBytesSpilled: stage.ShuffleOutputBytesSpilled,
			WaitMsAvg:                 milliseconds(stage.WaitAvg),
			WaitMsMax:                 milliseconds(stage.WaitMax),
			ReadMsAvg:                 milliseconds(stage.ReadAvg),
			ReadMsMax:                 milliseconds(stage.ReadMax),
			ComputeMsAvg:              milliseconds(stage.ComputeAvg),
			ComputeMsMax:              milliseconds(stage.ComputeMax),
			WriteMsAvg:                milliseconds(stage.WriteAvg),
			WriteMsMax:                milliseconds(stage.WriteMax),
		})
	}
	return encoded
}

// queryPlan returns the plan of the job a finished read came from, or nil where
// there is none to report.
//
// Reading it costs one jobs.get after the rows, since the responses the row
// iterator fetches on its way through them do not carry the plan. A read that
// came from no job of its own has none; neither does a job still without
// statistics, nor a failure to fetch them, which leaves the rows the caller
// asked for exactly as they were.
func queryPlan(ctx context.Context, job *bigquery.Job) *v1pb.QueryResult_QueryPlan {
	if job == nil {
		return nil
	}
	status, err := job.Status(ctx)
	if err != nil {
		slog.Debug("failed to read the job status for its query plan", log.BBError(err))
		return nil
	}
	stats, ok := statisticsOf(status)
	if !ok || len(stats.QueryPlan) == 0 {
		return nil
	}
	content, err := json.Marshal(toQueryStages(stats.QueryPlan))
	if err != nil {
		slog.Debug("failed to encode the query plan", log.BBError(err))
		return nil
	}
	return &v1pb.QueryResult_QueryPlan{
		Format: v1pb.QueryOption_JSON,
		// The plan records the run that returned these rows rather than
		// planning another, which is what the viewer's measured badge says.
		Executed: true,
		Content:  string(content),
	}
}
