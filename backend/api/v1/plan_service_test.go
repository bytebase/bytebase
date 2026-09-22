package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestValidateSpecsRejectsShapesBeforeAnyLookup pins the two plan shapes
// creation refuses on the request alone: a plan mixing create-database and
// change-database specs, and a spec with no config at all — which is what a
// legacy client sending the retired export_data_config decodes to. Neither
// reaches the store, so nil is one.
func TestValidateSpecsRejectsShapesBeforeAnyLookup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	_, err := validateSpecs(ctx, nil, "project-a", []*v1pb.Plan_Spec{
		{Id: "create", Config: &v1pb.Plan_Spec_CreateDatabaseConfig{CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{}}},
		{Id: "change", Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{}}},
	})
	require.ErrorContains(t, err, "each plan must contain only one type")

	_, err = validateSpecs(ctx, nil, "project-a", []*v1pb.Plan_Spec{{Id: "spec-1"}})
	require.ErrorContains(t, err, "invalid spec type")
}

// TestBuildV1PlansScopesRelationsByProject pins the join the plan list depends
// on: issue status, approval status, plan check counts and rollout stage
// summaries are attached by (project, plan UID), never by UID alone — plan
// UIDs are allocated per project, so two projects' plans share every number.
func TestBuildV1PlansScopesRelationsByProject(t *testing.T) {
	t.Parallel()
	changeConfig := &storepb.PlanConfig{Specs: []*storepb.PlanConfig_Spec{{
		Id:     "change",
		Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{}},
	}}}
	plans := []*store.PlanMessage{
		{ProjectID: "project-a", UID: 101, Name: "canceled in a", Config: changeConfig},
		{ProjectID: "project-b", UID: 101, Name: "done in b", Config: changeConfig},
		{ProjectID: "project-a", UID: 102, Name: "draft", Config: changeConfig},
		{ProjectID: "project-a", UID: 103, Name: "open and skipped", Config: changeConfig},
		{ProjectID: "project-a", UID: 104, Name: "no issue", Config: changeConfig},
	}
	planUID := func(uid int64) *int64 { return &uid }
	issues := []*store.IssueMessage{
		{ProjectID: "project-a", UID: 1, PlanUID: planUID(101), Status: storepb.Issue_CANCELED, Payload: &storepb.Issue{}},
		{ProjectID: "project-b", UID: 1, PlanUID: planUID(101), Status: storepb.Issue_DONE, Payload: &storepb.Issue{}},
		{ProjectID: "project-a", UID: 2, PlanUID: planUID(102), Status: storepb.Issue_OPEN, Payload: &storepb.Issue{
			Draft:    true,
			Approval: &storepb.IssuePayloadApproval{ApprovalFindingDone: true},
		}},
		{ProjectID: "project-a", UID: 3, PlanUID: planUID(103), Status: storepb.Issue_OPEN, Payload: &storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{ApprovalFindingDone: true},
		}},
	}

	planCheckRuns := []*store.PlanCheckRunMessage{
		{ProjectID: "project-a", PlanUID: 101, Status: store.PlanCheckRunStatusDone, Result: &storepb.PlanCheckRunResult{
			Results: []*storepb.PlanCheckRunResult_Result{{Status: storepb.Advice_WARNING}},
		}},
		{ProjectID: "project-b", PlanUID: 101, Status: store.PlanCheckRunStatusFailed, Result: &storepb.PlanCheckRunResult{}},
	}
	taskStatusCounts := []*store.TaskStatusCount{
		{ProjectID: "project-a", PlanID: 101, Environment: "prod", Status: storepb.TaskRun_DONE.String(), Count: 2},
		{ProjectID: "project-b", PlanID: 101, Environment: "prod", Status: storepb.TaskRun_FAILED.String(), Count: 1},
	}

	got := buildV1Plans(plans, issues, planCheckRuns, taskStatusCounts, map[string]int{"prod": 0})
	require.Len(t, got, len(plans))
	byTitle := map[string]*v1pb.Plan{}
	for _, plan := range got {
		byTitle[plan.Title] = plan
	}

	require.Equal(t, "projects/project-a/issues/1", byTitle["canceled in a"].Issue)
	require.Equal(t, v1pb.IssueStatus_CANCELED, byTitle["canceled in a"].IssueStatus)
	require.Equal(t, "projects/project-b/issues/1", byTitle["done in b"].Issue)
	require.Equal(t, v1pb.IssueStatus_DONE, byTitle["done in b"].IssueStatus)

	require.Equal(t, map[string]int32{"DONE": 1, "WARNING": 1}, byTitle["canceled in a"].PlanCheckRunStatusCount)
	require.Equal(t, map[string]int32{"FAILED": 1}, byTitle["done in b"].PlanCheckRunStatusCount)
	require.Empty(t, byTitle["draft"].PlanCheckRunStatusCount)
	stageCounts := func(plan *v1pb.Plan) []*v1pb.Plan_TaskStatusCount {
		require.Len(t, plan.RolloutStageSummaries, 1)
		return plan.RolloutStageSummaries[0].TaskStatusCounts
	}
	require.Len(t, stageCounts(byTitle["canceled in a"]), 1)
	require.Equal(t, v1pb.Task_DONE, stageCounts(byTitle["canceled in a"])[0].Status)
	require.EqualValues(t, 2, stageCounts(byTitle["canceled in a"])[0].Count)
	require.Equal(t, v1pb.Task_FAILED, stageCounts(byTitle["done in b"])[0].Status)
	require.Empty(t, byTitle["draft"].RolloutStageSummaries)

	require.Equal(t, v1pb.IssueStatus_OPEN, byTitle["draft"].IssueStatus)
	require.Equal(t, v1pb.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED, byTitle["draft"].ApprovalStatus,
		"a draft has no approval yet, whatever its payload says")
	require.Equal(t, v1pb.IssueStatus_OPEN, byTitle["open and skipped"].IssueStatus)
	require.Equal(t, v1pb.ApprovalStatus_SKIPPED, byTitle["open and skipped"].ApprovalStatus)

	require.Empty(t, byTitle["no issue"].Issue)
	require.Equal(t, v1pb.IssueStatus_ISSUE_STATUS_UNSPECIFIED, byTitle["no issue"].IssueStatus)
	require.Equal(t, v1pb.ApprovalStatus_APPROVAL_STATUS_UNSPECIFIED, byTitle["no issue"].ApprovalStatus)
}
