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
)

const (
	hubEventURL        = "https://hub.bytebase.com/v1/events"
	maxStatementLength = 1000
	cacheCapacity      = 100
)

// Reporter sends telemetry events to hub.bytebase.com with deduplication.
type Reporter struct {
	mu          sync.RWMutex
	workspaceID string
	version     string
	gitCommit   string
	enabled     bool

	// LRU cache for deduplication: tracks reported statement hashes
	cache *lru.Cache[string, struct{}]

	httpClient *http.Client
}

var (
	globalReporter     *Reporter
	globalReporterOnce sync.Once
)

// InitGlobalReporter initializes the global telemetry reporter.
// Must be called once at server startup.
func InitGlobalReporter(workspaceID, version, gitCommit string, enabled bool) {
	globalReporterOnce.Do(func() {
		cache, _ := lru.New[string, struct{}](cacheCapacity)
		globalReporter = &Reporter{
			workspaceID: workspaceID,
			version:     version,
			gitCommit:   gitCommit,
			enabled:     enabled,
			cache:       cache,
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
		}
	})
}

// gomongoFallbackPayload is the JSON payload for gomongo fallback events.
type gomongoFallbackPayload struct {
	WorkspaceID     string `json:"workspaceId"`
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
func ReportGomongoFallback(ctx context.Context, statement string, errorMessage string) {
	if globalReporter == nil {
		return
	}

	globalReporter.mu.RLock()
	if !globalReporter.enabled || globalReporter.workspaceID == "" {
		globalReporter.mu.RUnlock()
		return
	}
	workspaceID := globalReporter.workspaceID
	version := globalReporter.version
	gitCommit := globalReporter.gitCommit
	globalReporter.mu.RUnlock()

	// Skip telemetry in development builds
	if version == "development" {
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

	// Build payload
	payload := gomongoFallbackPayload{
		WorkspaceID: workspaceID,
		Version:     version,
		Commit:      gitCommit,
	}
	payload.GomongoFallback.Statement = truncatedStatement
	payload.GomongoFallback.ErrorMessage = errorMessage

	// Send async to not block query execution
	go globalReporter.send(ctx, payload)
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
