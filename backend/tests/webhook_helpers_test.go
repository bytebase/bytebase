package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

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

// waitForWebhookCount blocks until at least n webhooks for (project, eventTitle)
// have arrived, or fails the test.
func waitForWebhookCount(t *testing.T, c *webhookCollector, projectName, eventTitle string, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		count := countWebhooksFor(c, projectName, eventTitle)
		if count >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d %q webhooks on %s; got %d after %s",
				n, eventTitle, projectName, count, timeout)
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
//
//nolint:unused
func unblockFailingTask(t *testing.T, instanceDir, dbName string) {
	t.Helper()
	dbPath := filepath.Join(instanceDir, dbName+".db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()
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
//
//nolint:unused
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

//nolint:unused
func runTaskByDB(ctx context.Context, t *testing.T, ctl *controller, rollout *v1pb.Rollout, dbResource string) {
	t.Helper()
	stageName, taskName := findTaskByDB(t, rollout, dbResource)
	_, err := ctl.rolloutServiceClient.BatchRunTasks(ctx, connect.NewRequest(&v1pb.BatchRunTasksRequest{
		Parent: stageName,
		Tasks:  []string{taskName},
	}))
	require.NoError(t, err)
}

//nolint:unused
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

//nolint:unused
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
//
//nolint:unused
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
//
//nolint:unused
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
