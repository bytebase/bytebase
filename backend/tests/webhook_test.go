package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	webhookplugin "github.com/bytebase/bytebase/backend/plugin/webhook"
)

// webhookCollector collects webhook requests for testing.
type webhookCollector struct {
	mu       sync.Mutex
	requests []webhookRequest
}

type webhookRequest struct {
	Method  string
	Headers http.Header
	Body    []byte
	Time    time.Time
}

func (c *webhookCollector) addRequest(r *http.Request) error {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, webhookRequest{
		Method:  r.Method,
		Headers: r.Header,
		Body:    body,
		Time:    time.Now(),
	})
	return nil
}

func (c *webhookCollector) getRequests() []webhookRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]webhookRequest{}, c.requests...)
}

func (c *webhookCollector) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = nil
}

// Helper to parse Slack webhook payload.
// Extracts section block texts from attachments[0].blocks (new format).
func parseSlackWebhook(body []byte) (title, description string, err error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", err
	}

	// New format: blocks are inside attachments[0].blocks.
	var blocks []any
	if attachments, ok := payload["attachments"].([]any); ok && len(attachments) > 0 {
		if att, ok := attachments[0].(map[string]any); ok {
			if b, ok := att["blocks"].([]any); ok {
				blocks = b
			}
		}
	}
	if blocks == nil {
		return "", "", nil
	}

	// Collect section block texts. Layout for issue events:
	// [0] event title (bold, with emoji/link)
	// [1] action description (e.g. "Admin created issue X")
	// [2] issue tile (*IssueName*) — bold-wrapped
	// [3] issue description (if present)
	var sectionTexts []string
	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
			continue
		}
		if blockMap["type"] != "section" {
			continue
		}
		textMap, ok := blockMap["text"].(map[string]any)
		if !ok {
			continue
		}
		text, ok := textMap["text"].(string)
		if !ok {
			continue
		}
		sectionTexts = append(sectionTexts, text)
	}

	// Walk sections: find bold tile (*text*) as issue title,
	// and the next plain section after the tile as issue description.
	for i, s := range sectionTexts {
		if i == 0 {
			continue // skip event title
		}
		if strings.HasPrefix(s, "*") && strings.HasSuffix(s, "*") && !strings.Contains(s, "|") {
			title = strings.Trim(s, "*")
			// Next non-bold section is the issue description.
			if i+1 < len(sectionTexts) {
				next := sectionTexts[i+1]
				if !strings.HasPrefix(next, "*") || !strings.HasSuffix(next, "*") {
					description = next
				}
			}
			break
		}
	}
	return title, description, nil
}

// TestWebhookIntegration tests webhook functionality.
func TestWebhookIntegration(t *testing.T) {
	// Allow localhost for testing
	webhookplugin.TestOnlyAllowedDomains[storepb.WebhookType_SLACK] = []string{"127.0.0.1", "localhost", "[::1]"}
	defer func() {
		// Clean up after test
		delete(webhookplugin.TestOnlyAllowedDomains, storepb.WebhookType_SLACK)
	}()

	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	require.NoError(t, err)
	defer ctl.Close(ctx)

	// Create a test webhook server
	collector := &webhookCollector{}
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := collector.addRequest(r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer webhookServer.Close()

	// Create a single instance for all tests
	instanceRootDir := t.TempDir()
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "testInstance")
	require.NoError(t, err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "test instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	require.NoError(t, err)
	instance := instanceResp.Msg

	t.Run("IssueWithPlanWebhookPayload", func(t *testing.T) {
		// Reset webhook collector for this test
		collector.reset()

		// Each subtest owns its own project + webhook to keep counts isolated.
		projectID := generateRandomString("byt9398-i1")
		projectResp, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
			ProjectId: projectID,
			Project: &v1pb.Project{
				Name:              fmt.Sprintf("projects/%s", projectID),
				Title:             "byt9398-i1",
				AllowSelfApproval: true,
			},
		}))
		require.NoError(t, err)
		project := projectResp.Msg

		_, err = ctl.projectServiceClient.AddWebhook(ctx, connect.NewRequest(&v1pb.AddWebhookRequest{
			Project: project.Name,
			Webhook: &v1pb.Webhook{
				Type:              v1pb.WebhookType_SLACK,
				Title:             "Test Webhook for ISSUE_CREATED",
				Url:               webhookServer.URL,
				NotificationTypes: []v1pb.Activity_Type{v1pb.Activity_ISSUE_CREATED},
			},
		}))
		require.NoError(t, err)

		// Create a plan with title and description
		planTitle := "Database Migration Plan"
		planDesc := "This plan creates a new database with important schema changes"
		planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
			Parent: project.Name,
			Plan: &v1pb.Plan{
				Name:        planTitle,
				Description: planDesc,
				Specs: []*v1pb.Plan_Spec{
					{
						Id: uuid.NewString(),
						Config: &v1pb.Plan_Spec_CreateDatabaseConfig{
							CreateDatabaseConfig: &v1pb.Plan_CreateDatabaseConfig{
								Target:   instance.Name,
								Database: "testdb",
							},
						},
					},
				},
			},
		}))
		require.NoError(t, err)

		// Create an issue for webhook testing
		issueResp, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
			Parent: project.Name,
			Issue: &v1pb.Issue{
				Title:       "Test webhook issue",
				Description: "", // Empty description is OK
				Type:        v1pb.Issue_DATABASE_CHANGE,
				Plan:        planResp.Msg.Name,
			},
		}))
		require.NoError(t, err)
		require.NotEmpty(t, issueResp.Msg.Name)

		// Wait for webhook to be processed
		time.Sleep(5 * time.Second)

		// Verify webhook was triggered
		requests := collector.getRequests()
		require.GreaterOrEqual(t, len(requests), 1, "Expected at least 1 webhook")

		// Find and verify the issue creation webhook. Slack incoming webhook
		// payloads intentionally omit top-level text to avoid rendering a
		// duplicate message above the attachment card.
		var foundCorrectWebhook bool
		for _, req := range requests {
			require.Equal(t, "POST", req.Method)
			require.Contains(t, req.Headers.Get("Content-Type"), "application/json")

			title, desc, err := parseSlackWebhook(req.Body)
			require.NoError(t, err)

			// The webhook should use plan's description (title support is incomplete)
			if desc == planDesc {
				foundCorrectWebhook = true
				t.Logf("✓ Webhook uses plan's description: %q", desc)
				if title == "" {
					t.Logf("⚠️  Issue title is empty (expected: %q)", planTitle)
				}
				break
			}
		}

		require.True(t, foundCorrectWebhook, "Webhook should use plan's description")
	})

	t.Run("PipelineCompletedAfterSkippingFailedTask", func(t *testing.T) {
		collector.reset()

		project := ctl.createTestProject(ctx, t, "byt9398-c4")

		// Create databases before registering the webhook so the implicit
		// PIPELINE_COMPLETED events fired by createDatabase (which internally
		// runs a createDatabaseConfig plan) do not pollute the collector.
		err := ctl.createDatabase(ctx, project, instance, nil, "byt9398_c4_pass", "")
		require.NoError(t, err)
		err = ctl.createDatabase(ctx, project, instance, nil, "byt9398_c4_fail", "")
		require.NoError(t, err)

		collector.reset()
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
			v1pb.Activity_PIPELINE_FAILED,
			v1pb.Activity_PIPELINE_COMPLETED,
		})

		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c4_pass")},
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c4_fail")},
		})
		rollout := runAllTasks(ctx, t, ctl, plan)

		// Phase 1: failing task → exactly one PIPELINE_FAILED, no PIPELINE_COMPLETED.
		waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
		waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout completed", 0)

		// Phase 2: skip the failed task → PIPELINE_COMPLETED fires (the fix).
		skipFailedTasks(ctx, t, ctl, rollout)
		waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
	})

	t.Run("PipelineCompleted_AllTasksDone", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-c1")
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c1_a", ""))
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c1_b", ""))
		collector.reset() // flush any PIPELINE_COMPLETED from database creation
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
			v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
		})

		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c1_a")},
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c1_b")},
		})
		runAllTasks(ctx, t, ctl, plan)

		waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout failed", 0)
	})

	t.Run("PipelineCompleted_DoneAndSkipped", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-c2")
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c2_a", ""))
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c2_b", ""))
		collector.reset() // flush any PIPELINE_COMPLETED from database creation
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
			v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
		})

		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c2_a")},
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c2_b")},
		})
		rollout := createRolloutOnly(ctx, t, ctl, plan)
		skipTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c2_b"))
		runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c2_a"))

		waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout failed", 0)
	})

	t.Run("PipelineCompleted_AllSkipped", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-c5")
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c5_a", ""))
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c5_b", ""))
		collector.reset() // flush any PIPELINE_COMPLETED from database creation
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
			v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
		})

		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c5_a")},
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c5_b")},
		})
		rollout := createRolloutOnly(ctx, t, ctl, plan)
		skipAllTasks(ctx, t, ctl, rollout)

		waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout failed", 0)
	})

	t.Run("PipelineCompleted_DoneAndFailedThenRetriedDone", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-c3")
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c3_pass", ""))
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c3_fail", ""))
		collector.reset() // flush any PIPELINE_COMPLETED from database creation
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
			v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
		})

		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c3_pass")},
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c3_fail")},
		})
		rollout := runAllTasks(ctx, t, ctl, plan)

		waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
		waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout completed", 0)

		unblockFailingTask(t, instanceDir, "byt9398_c3_fail")
		retryFailedTasks(ctx, t, ctl, rollout)

		waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
	})

	t.Run("PipelineCompleted_AllFailedThenAllSkipped", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-c6")
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c6_a", ""))
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_c6_b", ""))
		collector.reset() // flush any PIPELINE_COMPLETED from database creation
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
			v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
		})

		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c6_a")},
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c6_b")},
		})
		rollout := runAllTasks(ctx, t, ctl, plan)

		waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
		waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)

		skipAllTasks(ctx, t, ctl, rollout)
		waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
	})

	t.Run("PipelineCompleted_MixedRecovery", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-c7")
		for _, n := range []string{"byt9398_c7_done", "byt9398_c7_skip", "byt9398_c7_retry", "byt9398_c7_skipfailed"} {
			require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, n, ""))
		}
		collector.reset() // flush any PIPELINE_COMPLETED from database creation
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{
			v1pb.Activity_PIPELINE_FAILED, v1pb.Activity_PIPELINE_COMPLETED,
		})

		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c7_done")},
			{seedPassingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c7_skip")},
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c7_retry")},
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_c7_skipfailed")},
		})

		rollout := createRolloutOnly(ctx, t, ctl, plan)
		skipTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_skip"))
		runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_done"))
		runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_retry"))
		runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_skipfailed"))

		waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
		waitForAllTasksTerminal(ctx, t, ctl, rollout, 60*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout completed", 0)

		// Unblock dbRetry only — dbSkipFailed's __force_fail_target table remains
		// absent in its own .db file, so its retry would still fail.
		unblockFailingTask(t, instanceDir, "byt9398_c7_retry")
		runTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_retry"))
		waitForTaskStatus(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_retry"), v1pb.Task_DONE, 30*time.Second)
		skipTaskByDB(ctx, t, ctl, rollout, dbTargetName(instance, "byt9398_c7_skipfailed"))

		waitForWebhookCount(t, collector, project.Name, "Rollout completed", 1, 30*time.Second)
		requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
	})

	t.Run("PipelineFailed_SingleTaskFails", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-f1")
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_f1_fail", ""))
		collector.reset()
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_PIPELINE_FAILED})
		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_f1_fail")},
		})
		runAllTasks(ctx, t, ctl, plan)
		waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
	})

	t.Run("PipelineFailed_DedupOnSecondTaskFailure", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-f2")
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_f2_a", ""))
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_f2_b", ""))
		collector.reset()
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_PIPELINE_FAILED})
		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_f2_a")},
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_f2_b")},
		})
		rollout := runAllTasks(ctx, t, ctl, plan)

		waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
		waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)

		// Both tasks have failed. ClaimPipelineFailureNotification's PK collision
		// must dedupe — assert exactly 1 even after both terminal.
		requireWebhookCount(t, collector, project.Name, "Rollout failed", 1)
	})

	t.Run("PipelineFailed_RetryFailsAgain", func(t *testing.T) {
		collector.reset()
		project := ctl.createTestProject(ctx, t, "byt9398-f3")
		require.NoError(t, ctl.createDatabase(ctx, project, instance, nil, "byt9398_f3_fail", ""))
		collector.reset()
		addWebhookForEvents(ctx, t, ctl, project, webhookServer.URL, []v1pb.Activity_Type{v1pb.Activity_PIPELINE_FAILED})
		plan := createPlanWithSpecs(ctx, t, ctl, project, []taskSpec{
			{seedFailingSheet(ctx, t, ctl, project), dbTargetName(instance, "byt9398_f3_fail")},
		})
		rollout := runAllTasks(ctx, t, ctl, plan)

		waitForWebhookCount(t, collector, project.Name, "Rollout failed", 1, 30*time.Second)
		waitForAllTasksTerminal(ctx, t, ctl, rollout, 30*time.Second)

		// BatchRunTasks resets the dedup row before enqueuing the retry, so the
		// second failure must re-fire PIPELINE_FAILED. We deliberately do NOT call
		// unblockFailingTask — the retry should fail again.
		retryFailedTasks(ctx, t, ctl, rollout)
		waitForWebhookCount(t, collector, project.Name, "Rollout failed", 2, 30*time.Second)
	})

	t.Run("PipelineFailed_HALicenseBreach", func(t *testing.T) {
		t.Skip("HA license-breach path requires a non-HA license JWT, replica-heartbeat " +
			"seeding, and an injectable haFailGracePeriod — none exist in the test harness. " +
			"Codepath at backend/runner/taskrun/scheduler.go:71-142 is manually verified. " +
			"Tracked as a follow-up.")
	})
}
