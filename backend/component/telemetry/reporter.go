// Package telemetry provides telemetry reporting to hub.bytebase.com.
package telemetry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	hubEventURL        = "https://hub.bytebase.com/v1/events"
	maxStatementLength = 1000
	cacheCapacity      = 100
)

// Reporter sends telemetry events to hub.bytebase.com with deduplication.
type Reporter struct {
	mu        sync.RWMutex
	version   string
	gitCommit string
	enabled   bool

	// LRU cache for deduplication: tracks reported statement hashes
	cache *lru.Cache[string, struct{}]

	httpClient *http.Client
}

var (
	globalReporter     *Reporter
	globalReporterOnce sync.Once
)

// InitGlobalReporter initializes the global telemetry reporter.
// The workspace ID is not needed at init time — it's resolved from request context
// when events are reported, so this works in both single-workspace and SaaS modes.
func InitGlobalReporter(version, gitCommit string, enabled bool) {
	globalReporterOnce.Do(func() {
		cache, _ := lru.New[string, struct{}](cacheCapacity)
		globalReporter = &Reporter{
			version:   version,
			gitCommit: gitCommit,
			enabled:   enabled,
			cache:     cache,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	})
}

// gomongoFallbackPayload is the JSON payload for gomongo fallback events.
type gomongoFallbackPayload struct {
	WorkspaceID     string `json:"workspaceId"`
	Email           string `json:"email"`
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	GomongoFallback struct {
		Statement    string `json:"statement"`
		ErrorMessage string `json:"errorMessage"`
	} `json:"gomongoFallback"`
}

// ReportGomongoFallback reports a gomongo fallback event.
// It deduplicates based on statement hash using an LRU cache.
// Only reports in release versions (non-development builds).
// The workspace ID is resolved from the request context if available,
// otherwise from the explicit workspaceID parameter (for runner contexts).
func ReportGomongoFallback(ctx context.Context, workspaceID, statement, errorMessage string) {
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
	globalReporter.mu.RUnlock()

	// Skip telemetry in development builds
	if version == "development" {
		return
	}

	// Prefer workspace from context (API requests), fall back to parameter (runners).
	if wsFromCtx := common.GetWorkspaceIDFromContext(ctx); wsFromCtx != "" {
		workspaceID = wsFromCtx
	}
	if workspaceID == "" {
		return
	}

	// Truncate statement
	truncatedStatement, truncated := common.TruncateString(statement, maxStatementLength)
	if truncated {
		truncatedStatement += "..."
	}

	// Check deduplication
	hash := hashStatement(truncatedStatement)
	if !globalReporter.shouldReport(hash) {
		return
	}

	// Extract email from context user.
	var email string
	if user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage); ok {
		email = user.Email
	}

	// Build payload
	payload := gomongoFallbackPayload{
		WorkspaceID: workspaceID,
		Email:       email,
		Version:     version,
		Commit:      gitCommit,
	}
	payload.GomongoFallback.Statement = truncatedStatement
	payload.GomongoFallback.ErrorMessage = errorMessage

	// Send async with a detached context so the request context cancellation
	// does not abort the telemetry HTTP call.
	go globalReporter.send(context.WithoutCancel(ctx), payload)
}

func (r *Reporter) shouldReport(hash string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.cache.Get(hash); ok {
		return false
	}

	r.cache.Add(hash, struct{}{})
	return true
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

func hashStatement(statement string) string {
	h := sha256.Sum256([]byte(statement))
	return hex.EncodeToString(h[:])
}
