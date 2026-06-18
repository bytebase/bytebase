package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common"
)

func TestReportSQLReviewConfigSnapshot(t *testing.T) {
	const snapshot = `{"reviewConfigs":[{"name":"reviewConfigs/basic","enabled":true}]}`

	var (
		mu      sync.Mutex
		payload map[string]any
		done    = make(chan struct{})
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)

		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("failed to decode payload: %v", err)
			return
		}
		mu.Lock()
		payload = got
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oldHubEventURL := hubEventURL
	hubEventURL = server.URL
	defer func() {
		hubEventURL = oldHubEventURL
	}()

	globalReporter = &Reporter{
		version:     "3.19.1",
		gitCommit:   "abcdef",
		releaseMode: common.ReleaseModeProd,
		enabled:     true,
		httpClient:  server.Client(),
	}
	defer func() {
		globalReporter = nil
	}()

	ReportSQLReviewConfigSnapshot(context.Background(), "workspace-id", "owner@example.com", snapshot, []string{"example.com", "test.com"})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for telemetry request")
	}

	mu.Lock()
	defer mu.Unlock()
	if payload["workspaceId"] != "workspace-id" {
		t.Fatalf("workspaceId = %v, want %q", payload["workspaceId"], "workspace-id")
	}
	if payload["email"] != "owner@example.com" {
		t.Fatalf("email = %v, want %q", payload["email"], "owner@example.com")
	}
	if payload["version"] != "3.19.1" {
		t.Fatalf("version = %v, want %q", payload["version"], "3.19.1")
	}
	if payload["commit"] != "abcdef" {
		t.Fatalf("commit = %v, want %q", payload["commit"], "abcdef")
	}
	event, ok := payload["sqlReviewConfigSnapshot"].(map[string]any)
	if !ok {
		t.Fatalf("sqlReviewConfigSnapshot = %T, want object", payload["sqlReviewConfigSnapshot"])
	}
	if event["snapshot"] != snapshot {
		t.Fatalf("snapshot = %v, want %q", event["snapshot"], snapshot)
	}
	domains, ok := event["emailDomains"].([]any)
	if !ok {
		t.Fatalf("emailDomains = %T, want array", event["emailDomains"])
	}
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "test.com" {
		t.Fatalf("emailDomains = %v, want %v", domains, []string{"example.com", "test.com"})
	}
}

func TestReportSQLReviewConfigSnapshotSkipsNonProd(t *testing.T) {
	requestReceived := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		close(requestReceived)
	}))
	defer server.Close()

	oldHubEventURL := hubEventURL
	hubEventURL = server.URL
	defer func() {
		hubEventURL = oldHubEventURL
	}()

	globalReporter = &Reporter{
		version:     "3.19.1",
		gitCommit:   "abcdef",
		releaseMode: common.ReleaseModeDev,
		enabled:     true,
		httpClient:  server.Client(),
	}
	defer func() {
		globalReporter = nil
	}()

	ReportSQLReviewConfigSnapshot(context.Background(), "workspace-id", "owner@example.com", "{}", []string{"example.com"})

	select {
	case <-requestReceived:
		t.Fatal("telemetry request was sent in non-prod release mode")
	case <-time.After(100 * time.Millisecond):
	}
}
