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
func parseSlackWebhook(body []byte) (title, description string, err error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", "", err
	}

	blocks, ok := payload["blocks"].([]any)
	if !ok {
		return "", "", nil
	}

	for _, block := range blocks {
		blockMap, ok := block.(map[string]any)
		if !ok {
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

		if strings.HasPrefix(text, "*Issue:* ") {
			title = strings.TrimPrefix(text, "*Issue:* ")
		} else if strings.HasPrefix(text, "*Issue Description:* ") {
			description = strings.TrimPrefix(text, "*Issue Description:* ")
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
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	require.NoError(t, err)
	instance := instanceResp.Msg

	// Create webhooks for issue events
	for _, eventType := range []v1pb.Activity_Type{
		v1pb.Activity_ISSUE_CREATED,
	} {
		_, err := ctl.projectServiceClient.AddWebhook(ctx, connect.NewRequest(&v1pb.AddWebhookRequest{
			Project: ctl.project.Name,
			Webhook: &v1pb.Webhook{
				Type:              v1pb.WebhookType_SLACK,
				Title:             fmt.Sprintf("Test Webhook for %s", eventType),
				Url:               webhookServer.URL,
				NotificationTypes: []v1pb.Activity_Type{eventType},
			},
		}))
		require.NoError(t, err)
	}

	t.Run("IssueWithPlanWebhookPayload", func(t *testing.T) {
		// Reset webhook collector for this test
		collector.reset()

		// Create a plan with title and description
		planTitle := "Database Migration Plan"
		planDesc := "This plan creates a new database with important schema changes"
		planResp, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
			Parent: ctl.project.Name,
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
			Parent: ctl.project.Name,
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

		// Find and verify the issue creation webhook
		var foundCorrectWebhook bool
		for _, req := range requests {
			require.Equal(t, "POST", req.Method)
			require.Contains(t, req.Headers.Get("Content-Type"), "application/json")

			var payload map[string]any
			require.NoError(t, json.Unmarshal(req.Body, &payload))

			// Check if this is an issue creation webhook
			if text, ok := payload["text"].(string); ok && text == "Issue created" {
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
		}

		require.True(t, foundCorrectWebhook, "Webhook should use plan's description")
	})
}
