package plancheck

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/productmetrics"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestRunOnceRecordsEmptySuccess(t *testing.T) {
	ctx := context.Background()
	stores := setupPlancheckStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)
	metrics := productmetrics.New(nil, nil)

	NewScheduler(stores, b, nil, nil, metrics).runOnce(ctx)
	NewScheduler(nil, nil, nil, nil, metrics).runOnce(ctx)

	require.Equal(t, uint64(1), runnerRunCount(t, metrics, productmetrics.RunnerPlanCheck, productmetrics.ResultSuccess))
	require.Equal(t, uint64(1), runnerRunCount(t, metrics, productmetrics.RunnerPlanCheck, productmetrics.ResultFailure))
}

func runnerRunCount(t *testing.T, metrics *productmetrics.ProductMetrics, runner productmetrics.Runner, result productmetrics.RunnerResult) uint64 {
	t.Helper()
	ch := make(chan prometheus.Metric, 16)
	go func() {
		metrics.Collect(ch)
		close(ch)
	}()

	var count uint64
	for metric := range ch {
		var dtoMetric dto.Metric
		if metric.Write(&dtoMetric) != nil {
			continue
		}
		labels := map[string]string{}
		for _, label := range dtoMetric.GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}
		if labels["runner"] == string(runner) && labels["result"] == string(result) {
			count += dtoMetric.GetHistogram().GetSampleCount()
		}
	}
	return count
}

func TestMarkPlanCheckRunDoneSkipsDraftIssue(t *testing.T) {
	ctx := context.Background()
	stores := setupPlancheckStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "draft plan",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)
	_, err = stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "project-a",
		CreatorEmail: "creator@example.com",
		Title:        "draft issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{Draft: true},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)
	claimed, err := stores.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	scheduler := NewScheduler(stores, b, nil, nil, nil)
	scheduler.markPlanCheckRunDone(ctx, "project-a", claimed[0].UID, plan.UID, 1, nil)

	run, err := stores.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.Equal(t, store.PlanCheckRunStatusDone, run.Status)
	require.Empty(t, b.ApprovalCheckChan)
}

func TestRunPlanCheckRunStopsAfterProjectArchive(t *testing.T) {
	ctx := context.Background()
	stores := setupPlancheckStore(ctx, t)
	b, err := bus.New()
	require.NoError(t, err)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "project-a",
		Name:      "archived plan",
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)
	created, err := stores.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)
	claimed, err := stores.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	archived := true
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "project-a",
		Workspace:  "default",
		Delete:     &archived,
	}))
	scheduler := NewScheduler(stores, b, nil, nil, nil)
	scheduler.runPlanCheckRun(ctx, "project-a", claimed[0].UID, plan.UID, 1)

	run, err := stores.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.Equal(t, store.PlanCheckRunStatusCanceled, run.Status)
	require.Equal(t, "project is archived", run.Result.GetError())
}
