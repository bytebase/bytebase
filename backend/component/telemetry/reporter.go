// Package telemetry provides telemetry reporting to hub.bytebase.com.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/bytebase/bytebase/backend/common"
)

var hubEventURL = "https://hub.bytebase.com/v1/events"

// Reporter sends telemetry events to hub.bytebase.com.
type Reporter struct {
	mu          sync.RWMutex
	version     string
	gitCommit   string
	releaseMode common.ReleaseMode
	enabled     bool

	httpClient *http.Client
}

var (
	globalReporter     *Reporter
	globalReporterOnce sync.Once
)

// InitGlobalReporter initializes the global telemetry reporter.
func InitGlobalReporter(version, gitCommit string, releaseMode common.ReleaseMode, enabled bool) {
	globalReporterOnce.Do(func() {
		globalReporter = &Reporter{
			version:     version,
			gitCommit:   gitCommit,
			releaseMode: releaseMode,
			enabled:     enabled,
			httpClient: &http.Client{
				Timeout: 120 * time.Second,
			},
		}
	})
}

type sqlReviewConfigSnapshotPayload struct {
	WorkspaceID             string `json:"workspaceId"`
	Email                   string `json:"email"`
	Version                 string `json:"version"`
	Commit                  string `json:"commit"`
	SQLReviewConfigSnapshot struct {
		Snapshot     string   `json:"snapshot"`
		EmailDomains []string `json:"emailDomains"`
	} `json:"sqlReviewConfigSnapshot"`
}

// ReportSQLReviewConfigSnapshot reports the current SQL review config snapshot.
func ReportSQLReviewConfigSnapshot(ctx context.Context, workspaceID string, email string, snapshot string, emailDomains []string) {
	if globalReporter == nil {
		return
	}

	globalReporter.mu.RLock()
	if !globalReporter.enabled {
		globalReporter.mu.RUnlock()
		return
	}
	version := globalReporter.version
	gitCommit := globalReporter.gitCommit
	releaseMode := globalReporter.releaseMode
	globalReporter.mu.RUnlock()

	if releaseMode != common.ReleaseModeProd || workspaceID == "" || email == "" || snapshot == "" {
		return
	}

	payload := sqlReviewConfigSnapshotPayload{
		WorkspaceID: workspaceID,
		Email:       email,
		Version:     version,
		Commit:      gitCommit,
	}
	payload.SQLReviewConfigSnapshot.Snapshot = snapshot
	payload.SQLReviewConfigSnapshot.EmailDomains = emailDomains

	go globalReporter.send(context.WithoutCancel(ctx), payload)
}

func (r *Reporter) send(ctx context.Context, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hubEventURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
