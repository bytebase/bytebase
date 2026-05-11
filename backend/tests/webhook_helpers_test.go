package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// firstSlackSectionText returns the text of the first Slack section block in the
// captured payload, or "" if none. attachments[0].blocks[0].text.text holds the
// event title decorated with emoji and Slack <link|title> markup — see
// backend/plugin/webhook/slack/slack_test.go:42-47, 85.
func firstSlackSectionText(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	attachments, ok := payload["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		return ""
	}
	att, ok := attachments[0].(map[string]any)
	if !ok {
		return ""
	}
	blocks, ok := att["blocks"].([]any)
	if !ok || len(blocks) == 0 {
		return ""
	}
	block0, ok := blocks[0].(map[string]any)
	if !ok || block0["type"] != "section" {
		return ""
	}
	textMap, ok := block0["text"].(map[string]any)
	if !ok {
		return ""
	}
	text, ok := textMap["text"].(string)
	if !ok {
		return ""
	}
	return text
}

// matchesEvent reports whether a captured Slack payload matches the given
// project and event title. Project filtering prevents cross-subtest
// contamination from the async webhook dispatcher.
func matchesEvent(req webhookRequest, projectName, eventTitle string) bool {
	body := string(req.Body)
	if !strings.Contains(body, projectName) {
		return false
	}
	return strings.Contains(firstSlackSectionText(req.Body), eventTitle)
}

// webhookWaitTimeout is the deadline for waitForWebhookCount and the issue-state
// wait helpers. Webhook delivery + completion-check fanout takes <5s in practice;
// 30s gives ample headroom under CI load.
const webhookWaitTimeout = 30 * time.Second

// waitForWebhookCount blocks until at least n webhooks for (project, eventTitle)
// have arrived, or fails the test after webhookWaitTimeout.
func waitForWebhookCount(t *testing.T, c *webhookCollector, projectName, eventTitle string, n int) {
	t.Helper()
	deadline := time.Now().Add(webhookWaitTimeout)
	for {
		count := countWebhooksFor(c, projectName, eventTitle)
		if count >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d %q webhooks on %s; got %d after %s",
				n, eventTitle, projectName, count, webhookWaitTimeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// requireWebhookCount asserts the exact count of webhooks for (project, eventTitle).
func requireWebhookCount(t *testing.T, c *webhookCollector, projectName, eventTitle string, n int) {
	t.Helper()
	got := countWebhooksFor(c, projectName, eventTitle)
	require.Equalf(t, n, got, "expected %d %q webhooks on %s, got %d", n, eventTitle, projectName, got)
}

func countWebhooksFor(c *webhookCollector, projectName, eventTitle string) int {
	n := 0
	for _, req := range c.getRequests() {
		if matchesEvent(req, projectName, eventTitle) {
			n++
		}
	}
	return n
}

// createTestProject creates a fresh project (default AllowSelfApproval=true;
// approval-flow subtests call disableSelfApproval).
func (ctl *controller) createTestProject(ctx context.Context, t *testing.T, prefix string) *v1pb.Project {
	t.Helper()
	pid := generateRandomString(prefix)
	resp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: pid,
		Project: &v1pb.Project{
			Name:              fmt.Sprintf("projects/%s", pid),
			Title:             prefix,
			AllowSelfApproval: true,
		},
	}))
	require.NoError(t, err)
	return resp.Msg
}

// addWebhookForEvents adds a Slack webhook to the project subscribed to the
// given event types only.
func addWebhookForEvents(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, url string, events []v1pb.Activity_Type) {
	t.Helper()
	_, err := ctl.projectServiceClient.AddWebhook(ctx, connect.NewRequest(&v1pb.AddWebhookRequest{
		Project: project.Name,
		Webhook: &v1pb.Webhook{
			Type:              v1pb.WebhookType_SLACK,
			Title:             "test-webhook-" + generateRandomString("hook"),
			Url:               url,
			NotificationTypes: events,
		},
	}))
	require.NoError(t, err)
}

// dbTargetName returns the canonical instance/databases/<name> resource form.
func dbTargetName(instance *v1pb.Instance, dbName string) string {
	return fmt.Sprintf("%s/databases/%s", instance.Name, dbName)
}

// seedPassingSheet creates a sheet whose SQL trivially succeeds.
func seedPassingSheet(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project) string {
	t.Helper()
	resp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
	}))
	require.NoError(t, err)
	return resp.Msg.Name
}

// seedFailingSheet creates a sheet whose SQL fails until unblockFailingTask runs.
func seedFailingSheet(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project) string {
	t.Helper()
	resp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("INSERT INTO __force_fail_target VALUES(1);")},
	}))
	require.NoError(t, err)
	return resp.Msg.Name
}

// unblockFailingTask creates the missing table inside the SQLite database file
// so subsequent runs of seedFailingSheet's SQL succeed. db.Close() flushes
// WAL/journal so the file is consistent before the retry is enqueued.
func unblockFailingTask(t *testing.T, instanceDir, dbName string) {
	t.Helper()
	dbPath := filepath.Join(instanceDir, dbName+".db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer func() {
		if cerr := db.Close(); cerr != nil {
			t.Errorf("close sqlite handle for %s: %v", dbPath, cerr)
		}
	}()
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS __force_fail_target(id INT);")
	require.NoError(t, err)
}

type taskSpec struct {
	sheetName string
	dbTarget  string // full instance/databases/<name>
}

// createPlanWithSpecs creates a plan with one ChangeDatabaseConfig spec per entry.
func createPlanWithSpecs(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, specs []taskSpec) *v1pb.Plan {
	t.Helper()
	var planSpecs []*v1pb.Plan_Spec
	for _, s := range specs {
		planSpecs = append(planSpecs, &v1pb.Plan_Spec{
			Id: uuid.NewString(),
			Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
				ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
					Targets: []string{s.dbTarget},
					Sheet:   s.sheetName,
				},
			},
		})
	}
	resp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan:   &v1pb.Plan{Specs: planSpecs},
	}))
	require.NoError(t, err)
	return resp.Msg
}

func createRolloutOnly(ctx context.Context, t *testing.T, ctl *controller, plan *v1pb.Plan) *v1pb.Rollout {
	t.Helper()
	resp, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: plan.Name}))
	require.NoError(t, err)
	return resp.Msg
}

// runAllTasks materializes the rollout and starts every task in every stage.
func runAllTasks(ctx context.Context, t *testing.T, ctl *controller, plan *v1pb.Plan) *v1pb.Rollout {
	t.Helper()
	rollout := createRolloutOnly(ctx, t, ctl, plan)
	for _, stage := range rollout.Stages {
		var taskNames []string
		for _, task := range stage.Tasks {
			taskNames = append(taskNames, task.Name)
		}
		if len(taskNames) == 0 {
			continue
		}
		_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
			Parent: stage.Name,
			Tasks:  taskNames,
		}))
		require.NoError(t, err)
	}
	return rollout
}

func refreshRollout(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout) *v1pb.Rollout {
	t.Helper()
	resp, err := ctl.rolloutServiceClient.GetRollout(ctx, connect.NewRequest(&v1pb.GetRolloutRequest{Name: rollout.Name}))
	require.NoError(t, err)
	return resp.Msg
}

// findTaskByDB scans the rollout for a task targeting dbResource.
func findTaskByDB(t *testing.T, rollout *v1pb.Rollout, dbResource string) (stageName, taskName string) {
	t.Helper()
	for _, stage := range rollout.Stages {
		for _, task := range stage.Tasks {
			if task.Target == dbResource {
				return stage.Name, task.Name
			}
		}
	}
	t.Fatalf("no task targets database %s in rollout %s", dbResource, rollout.Name)
	return "", ""
}

func runTaskByDB(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, dbResource string) {
	t.Helper()
	stageName, taskName := findTaskByDB(t, rollout, dbResource)
	_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
		Parent: stageName,
		Tasks:  []string{taskName},
	}))
	require.NoError(t, err)
}

func skipTaskByDB(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, dbResource string) {
	t.Helper()
	stageName, taskName := findTaskByDB(t, rollout, dbResource)
	_, err := ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
		Parent: stageName,
		Tasks:  []string{taskName},
		Reason: "test: skip task by db",
	}))
	require.NoError(t, err)
}

// skipFailedTasks finds every FAILED task in the rollout and skips them.
func skipFailedTasks(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout) {
	t.Helper()
	fresh := refreshRollout(ctx, t, ctl, rollout)
	perStage := map[string][]string{}
	for _, stage := range fresh.Stages {
		for _, task := range stage.Tasks {
			if task.Status == v1pb.Task_FAILED {
				perStage[stage.Name] = append(perStage[stage.Name], task.Name)
			}
		}
	}
	require.NotEmpty(t, perStage, "expected at least one failed task to skip")
	for stageName, names := range perStage {
		_, err := ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
			Parent: stageName,
			Tasks:  names,
			Reason: "test: skip failed tasks",
		}))
		require.NoError(t, err)
	}
}

func skipAllTasks(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout) {
	t.Helper()
	fresh := refreshRollout(ctx, t, ctl, rollout)
	for _, stage := range fresh.Stages {
		var names []string
		for _, task := range stage.Tasks {
			names = append(names, task.Name)
		}
		if len(names) == 0 {
			continue
		}
		_, err := ctl.rolloutServiceClient.BatchSkipTasks(ctx, connect.NewRequest(&v1pb.BatchSkipTasksRequest{
			Parent: stage.Name,
			Tasks:  names,
			Reason: "test: skip all",
		}))
		require.NoError(t, err)
	}
}

// retryFailedTasks reruns BatchRunTasks on every FAILED task. Caller decides
// whether to call unblockFailingTask first (to make the retry pass) or not
// (to test that BatchRunTasks resets the dedup row so a second
// PIPELINE_FAILED can fire).
func retryFailedTasks(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout) {
	t.Helper()
	fresh := refreshRollout(ctx, t, ctl, rollout)
	perStage := map[string][]string{}
	for _, stage := range fresh.Stages {
		for _, task := range stage.Tasks {
			if task.Status == v1pb.Task_FAILED {
				perStage[stage.Name] = append(perStage[stage.Name], task.Name)
			}
		}
	}
	require.NotEmpty(t, perStage)
	for stageName, names := range perStage {
		_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
			Parent: stageName,
			Tasks:  names,
		}))
		require.NoError(t, err)
	}
}

// waitForTaskStatus polls the rollout until the task targeting dbResource has
// the requested status, or fails the test.
func waitForTaskStatus(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, dbResource string, want v1pb.Task_Status, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		fresh := refreshRollout(ctx, t, ctl, rollout)
		for _, stage := range fresh.Stages {
			for _, task := range stage.Tasks {
				if task.Target != dbResource {
					continue
				}
				if task.Status == want {
					return
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for task on %s to reach %s", dbResource, want)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// waitForAllTasksTerminal polls until every task in the rollout is in a
// terminal status (DONE, FAILED, SKIPPED, or CANCELED). Used before
// "skip remaining" actions to avoid races with still-running tasks. On
// timeout, dumps per-task status to t.Logf so failures are debuggable
// without re-running.
func waitForAllTasksTerminal(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	isTerminal := func(s v1pb.Task_Status) bool {
		return s == v1pb.Task_DONE || s == v1pb.Task_FAILED || s == v1pb.Task_SKIPPED || s == v1pb.Task_CANCELED
	}
	var fresh *v1pb.Rollout
	for {
		fresh = refreshRollout(ctx, t, ctl, rollout)
		allTerminal := true
		for _, stage := range fresh.Stages {
			for _, task := range stage.Tasks {
				if !isTerminal(task.Status) {
					allTerminal = false
				}
			}
		}
		if allTerminal {
			return
		}
		if time.Now().After(deadline) {
			for _, stage := range fresh.Stages {
				for _, task := range stage.Tasks {
					t.Logf("non-terminal task: %s status=%s target=%s", task.Name, task.Status, task.Target)
				}
			}
			t.Fatalf("timed out waiting for all tasks terminal on rollout %s after %s", rollout.Name, timeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// installWorkspaceApprovalRule writes a WORKSPACE_APPROVAL setting rule that
// scopes to the given project via CEL and registers a cleanup that restores
// the prior workspace setting. flowRoles is the ordered list of roles for
// sequential approval steps (one role per step).
func installWorkspaceApprovalRule(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, flowRoles []string) {
	t.Helper()
	projectID := path.Base(project.Name)

	prior := snapshotWorkspaceApproval(ctx, t, ctl)

	rules := []*v1pb.WorkspaceApprovalSetting_Rule{{
		Source: v1pb.WorkspaceApprovalSetting_Rule_SOURCE_UNSPECIFIED,
		Condition: &expr.Expr{
			Expression: fmt.Sprintf(`resource.project_id == "%s"`, projectID),
		},
		Template: &v1pb.ApprovalTemplate{
			Title: "Test approval flow for " + projectID,
			Flow:  &v1pb.ApprovalFlow{Roles: flowRoles},
		},
	}}
	if prior != nil {
		// Append to the snapshotted rules. Each subtest's t.Cleanup restores
		// its own snapshot, so LIFO ordering of cleanups still ends with the
		// truly-original workspace state. If a future test installs twice in
		// the same subtest, cleanups run LIFO and each writes the snapshot
		// taken at its install time, so the final state matches the original.
		rules = append(prior.Rules, rules...)
	}

	writeWorkspaceApproval(ctx, t, ctl, &v1pb.WorkspaceApprovalSetting{Rules: rules})
	t.Cleanup(func() {
		restoreWorkspaceApproval(t, ctl, prior)
	})
}

// clearWorkspaceApprovalRules removes all workspace approval rules (including the
// default catch-all) and restores the prior setting on cleanup. Use this in
// subtests that assert ISSUE_APPROVAL_REQUESTED does NOT fire when truly no rule
// applies, or when a custom multi-step flow must not coexist with the default
// projectOwner-only rule.
func clearWorkspaceApprovalRules(ctx context.Context, t *testing.T, ctl *controller) {
	t.Helper()

	prior := snapshotWorkspaceApproval(ctx, t, ctl)
	writeWorkspaceApproval(ctx, t, ctl, &v1pb.WorkspaceApprovalSetting{})
	t.Cleanup(func() {
		restoreWorkspaceApproval(t, ctl, prior)
	})
}

// snapshotWorkspaceApproval reads the current WORKSPACE_APPROVAL setting and
// fails the test if the read errors — silent failures here would let a later
// "restore" wipe the workspace's real config.
func snapshotWorkspaceApproval(ctx context.Context, t *testing.T, ctl *controller) *v1pb.WorkspaceApprovalSetting {
	t.Helper()
	resp, err := ctl.settingServiceClient.GetSetting(ctx, connect.NewRequest(&v1pb.GetSettingRequest{
		Name: "settings/WORKSPACE_APPROVAL",
	}))
	require.NoError(t, err, "snapshot WORKSPACE_APPROVAL setting")
	return resp.Msg.Value.GetWorkspaceApproval() // may be nil when no rules are set
}

func writeWorkspaceApproval(ctx context.Context, t *testing.T, ctl *controller, value *v1pb.WorkspaceApprovalSetting) {
	t.Helper()
	_, err := ctl.settingServiceClient.UpdateSetting(ctx, connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{WorkspaceApproval: value},
			},
		},
	}))
	require.NoError(t, err, "write WORKSPACE_APPROVAL setting")
}

// restoreWorkspaceApproval writes the snapshotted setting back. Runs from
// t.Cleanup; logs failures loudly so a corrupted teardown doesn't bleed into
// later subtests as an order-dependent flake.
func restoreWorkspaceApproval(t *testing.T, ctl *controller, prior *v1pb.WorkspaceApprovalSetting) {
	t.Helper()
	restored := prior
	if restored == nil {
		restored = &v1pb.WorkspaceApprovalSetting{}
	}
	if _, err := ctl.settingServiceClient.UpdateSetting(context.Background(), connect.NewRequest(&v1pb.UpdateSettingRequest{
		AllowMissing: true,
		Setting: &v1pb.Setting{
			Name: "settings/WORKSPACE_APPROVAL",
			Value: &v1pb.SettingValue{
				Value: &v1pb.SettingValue_WorkspaceApproval{WorkspaceApproval: restored},
			},
		},
	})); err != nil {
		t.Errorf("failed to restore WORKSPACE_APPROVAL setting in cleanup: %v", err)
	}
}

func disableSelfApproval(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project) {
	t.Helper()
	_, err := ctl.projectServiceClient.UpdateProject(ctx, connect.NewRequest(&v1pb.UpdateProjectRequest{
		Project: &v1pb.Project{
			Name:              project.Name,
			AllowSelfApproval: false,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"allow_self_approval"}},
	}))
	require.NoError(t, err)
}

type testApprover struct {
	Email    string
	Password string
}

func provisionApprover(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, suffix, projectRole string) testApprover {
	t.Helper()
	email := fmt.Sprintf("approver-%s@example.com", suffix)
	password := "1024bytebase"

	newUser, err := ctl.userServiceClient.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
		User: &v1pb.User{Email: email, Password: password, Title: "Approver " + suffix},
	}))
	require.NoError(t, err)

	_, err = ctl.addMemberToWorkspaceIAM(ctx, newUser.Msg.Workspace, fmt.Sprintf("user:%s", email), "roles/workspaceMember")
	require.NoError(t, err)

	policyResp, err := ctl.projectServiceClient.GetIamPolicy(ctx, connect.NewRequest(&v1pb.GetIamPolicyRequest{
		Resource: project.Name,
	}))
	require.NoError(t, err)
	policy := policyResp.Msg
	policy.Bindings = append(policy.Bindings, &v1pb.Binding{
		Role:    projectRole,
		Members: []string{fmt.Sprintf("user:%s", email)},
	})
	_, err = ctl.projectServiceClient.SetIamPolicy(ctx, connect.NewRequest(&v1pb.SetIamPolicyRequest{
		Resource: project.Name,
		Policy:   policy,
	}))
	require.NoError(t, err)

	return testApprover{Email: email, Password: password}
}

// withImpersonation logs in as the given approver, runs fn with that identity,
// and restores the original token via defer.
func withImpersonation(ctx context.Context, t *testing.T, ctl *controller, who testApprover, fn func()) {
	t.Helper()
	loginResp, err := ctl.authServiceClient.Login(ctx, connect.NewRequest(&v1pb.LoginRequest{
		Email:    who.Email,
		Password: who.Password,
	}))
	require.NoError(t, err)

	original := ctl.authInterceptor.token
	ctl.authInterceptor.token = loginResp.Msg.Token
	defer func() { ctl.authInterceptor.token = original }()

	fn()
}

func createIssueForPlan(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, plan *v1pb.Plan, title string) *v1pb.Issue {
	t.Helper()
	resp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title:       title,
			Description: title + " description",
			Type:        v1pb.Issue_DATABASE_CHANGE,
			Plan:        plan.Name,
		},
	}))
	require.NoError(t, err)
	return resp.Msg
}

func approveIssueAs(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue, who testApprover) {
	t.Helper()
	withImpersonation(ctx, t, ctl, who, func() {
		_, err := ctl.issueServiceClient.ApproveIssue(ctx, connect.NewRequest(&v1pb.ApproveIssueRequest{
			Name: issue.Name,
		}))
		require.NoError(t, err)
	})
}

func rejectIssueAs(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue, who testApprover, comment string) {
	t.Helper()
	withImpersonation(ctx, t, ctl, who, func() {
		_, err := ctl.issueServiceClient.RejectIssue(ctx, connect.NewRequest(&v1pb.RejectIssueRequest{
			Name:    issue.Name,
			Comment: comment,
		}))
		require.NoError(t, err)
	})
}

// requestIssueAsCreator clears the rejected-approver state on a sent-back issue.
// MUST be called as the issue creator (canRequestIssue at issue_service.go:803-805).
// Run with the default ctl token, which is the demo user used as default creator.
func requestIssueAsCreator(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue, comment string) {
	t.Helper()
	_, err := ctl.issueServiceClient.RequestIssue(ctx, connect.NewRequest(&v1pb.RequestIssueRequest{
		Name:    issue.Name,
		Comment: comment,
	}))
	require.NoError(t, err)
}

// waitForApprovalFindingDone blocks until the issue's approval-finding pipeline
// finishes — i.e., the runner has resolved the workspace approval rules and
// the issue's ApprovalStatus is no longer CHECKING. Use when the test does not
// care WHICH terminal status was reached, only that the runner completed (e.g.
// when asserting that no approval webhook fires for an issue that should auto-
// approve because no rule applies).
func waitForApprovalFindingDone(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue) {
	t.Helper()
	deadline := time.Now().Add(webhookWaitTimeout)
	for {
		resp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issue.Name,
		}))
		require.NoError(t, err)
		if resp.Msg.ApprovalStatus != v1pb.ApprovalStatus_CHECKING {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("issue %s still CHECKING after %s", issue.Name, webhookWaitTimeout)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// waitForIssueApproved blocks until the issue reaches APPROVED.
//
// Use only as a state-transition post-condition. This helper is NOT a barrier
// for webhook delivery — webhook.Manager.CreateEvent runs asynchronously after
// the issue row is updated. The current absence-assertion call sites are
// race-free because they subscribe only to event types the gRPC handler does
// NOT emit on the awaited transition (e.g., I2 subscribes to ISSUE_CREATED but
// drives an approval that emits ISSUE_APPROVED — which has no subscriber, so
// CreateEvent's synchronous len(webhookList) == 0 short-circuit never spawns
// a goroutine). A future caller that subscribes to the awaited event must wait
// on the collector, not on issue state.
func waitForIssueApproved(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue) {
	t.Helper()
	deadline := time.Now().Add(webhookWaitTimeout)
	for {
		resp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issue.Name,
		}))
		require.NoError(t, err)
		if resp.Msg.ApprovalStatus == v1pb.ApprovalStatus_APPROVED {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("issue %s did not reach APPROVED within %s; current status %s",
				issue.Name, webhookWaitTimeout, resp.Msg.ApprovalStatus)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// waitForIssuePending blocks until the issue's approval-finding pipeline
// finishes and the issue is in PENDING — i.e., the rule has been resolved and
// the system is ready for an approver action. Fails the test on any other
// terminal status (APPROVED, REJECTED), which would indicate a setup bug.
func waitForIssuePending(ctx context.Context, t *testing.T, ctl *controller, issue *v1pb.Issue) {
	t.Helper()
	deadline := time.Now().Add(webhookWaitTimeout)
	for {
		resp, err := ctl.issueServiceClient.GetIssue(ctx, connect.NewRequest(&v1pb.GetIssueRequest{
			Name: issue.Name,
		}))
		require.NoError(t, err)
		switch resp.Msg.ApprovalStatus {
		case v1pb.ApprovalStatus_PENDING:
			return
		case v1pb.ApprovalStatus_CHECKING:
			// keep waiting
		default:
			t.Fatalf("issue %s reached unexpected approval status %s while waiting for PENDING",
				issue.Name, resp.Msg.ApprovalStatus)
		}
		if time.Now().After(deadline) {
			t.Fatalf("issue %s still CHECKING after %s", issue.Name, webhookWaitTimeout)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
