package lsp

import (
	"context"
	"log/slog"
	"sync"
	"time"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/sourcegraph/jsonrpc2"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// DiagnosticsDebouncer manages debounced diagnostics for performance optimization
type DiagnosticsDebouncer struct {
	mu                 sync.Mutex
	pendingDiagnostics map[lsp.DocumentURI]*pendingDiagnostic
	defaultDelay       time.Duration
}

type pendingDiagnostic struct {
	uri       lsp.DocumentURI
	content   string
	timer     *time.Timer
	cancelled bool
}

// NewDiagnosticsDebouncer creates a new diagnostics debouncer
func NewDiagnosticsDebouncer(defaultDelay time.Duration) *DiagnosticsDebouncer {
	return &DiagnosticsDebouncer{
		pendingDiagnostics: make(map[lsp.DocumentURI]*pendingDiagnostic),
		defaultDelay:       defaultDelay,
	}
}

// ScheduleDiagnostics schedules diagnostics to be run after a delay, cancelling any pending diagnostics for the same URI
func (d *DiagnosticsDebouncer) ScheduleDiagnostics(
	ctx context.Context,
	conn *jsonrpc2.Conn,
	uri lsp.DocumentURI,
	content string,
	handler *Handler,
) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing pending diagnostic for this URI
	if existing, exists := d.pendingDiagnostics[uri]; exists {
		existing.cancelled = true
		if existing.timer != nil {
			existing.timer.Stop()
		}
	}

	// Create new pending diagnostic
	pending := &pendingDiagnostic{
		uri:       uri,
		content:   content,
		cancelled: false,
	}

	// Schedule the diagnostic
	pending.timer = time.AfterFunc(d.defaultDelay, func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		// Check if this diagnostic was cancelled
		if pending.cancelled {
			return
		}

		// Remove from pending map
		delete(d.pendingDiagnostics, uri)

		// Run diagnostics in a goroutine to avoid blocking
		go d.runDiagnostics(ctx, conn, uri, content, handler)
	})

	d.pendingDiagnostics[uri] = pending
}

// runDiagnostics performs the actual diagnostics and sends results
func (*DiagnosticsDebouncer) runDiagnostics(
	ctx context.Context,
	conn *jsonrpc2.Conn,
	uri lsp.DocumentURI,
	content string,
	handler *Handler,
) {
	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return
	default:
	}

	start := time.Now()
	engineType := handler.getEngineType(ctx)

	// Check cache first for statement ranges
	var diagnostics []lsp.Diagnostic
	var statementRanges []lsp.Range
	var cacheHit bool

	if handler.contentCache != nil {
		if cached, exists := handler.contentCache.Get(string(uri)); exists && cached.Content == content {
			// Use cached statement ranges if content hasn't changed
			statementRanges = cached.StatementRanges
			cacheHit = true
			slog.Debug("Using cached statement ranges",
				slog.String("uri", string(uri)))
		}
	}

	// Run expensive diagnostics (always fresh, not cached)
	var err error
	diagnostics, err = base.Diagnose(ctx, base.DiagnoseContext{}, engineType, content)
	if err != nil {
		slog.Warn("diagnose error", log.BBError(err))
	}

	// Check context cancellation after diagnostics
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Log diagnostic performance
	slog.Debug("Debounced diagnostics completed",
		slog.String("uri", string(uri)),
		slog.Duration("duration", time.Since(start)),
		slog.Int("diagnosticCount", len(diagnostics)),
		slog.Bool("cacheHit", cacheHit))

	// Send diagnostics notification
	if err := conn.Notify(ctx, string(LSPMethodPublishDiagnostics), &lsp.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}); err != nil {
		slog.Warn("failed to publish diagnostics", log.BBError(err))
	}

	// Parse statement ranges if not cached
	if !cacheHit {
		rangeStart := time.Now()
		statementRanges, err = base.GetStatementRanges(ctx, base.StatementRangeContext{}, engineType, content)
		if err != nil {
			slog.Warn("get statement ranges error", log.BBError(err))
		} else {
			slog.Debug("Statement range parsing completed",
				slog.String("uri", string(uri)),
				slog.Duration("duration", time.Since(rangeStart)),
				slog.Int("rangeCount", len(statementRanges)))

			// Cache the results
			if handler.contentCache != nil {
				handler.contentCache.Set(string(uri), &CachedContent{
					Content:         content,
					LastModified:    time.Now(),
					StatementRanges: statementRanges,
				})
			}
		}
	}

	// Send statement ranges if available
	if len(statementRanges) != 0 {
		if err := conn.Notify(ctx, string(LSPCustomMethodSQLStatementRanges), &SQLStatementRangesParams{
			URI:    uri,
			Ranges: statementRanges,
		}); err != nil {
			slog.Warn("failed to publish statement ranges", log.BBError(err))
		}
	}
}

// Clear cancels all pending diagnostics
func (d *DiagnosticsDebouncer) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, pending := range d.pendingDiagnostics {
		pending.cancelled = true
		if pending.timer != nil {
			pending.timer.Stop()
		}
	}
	d.pendingDiagnostics = make(map[lsp.DocumentURI]*pendingDiagnostic)
}

// ContentCache implements a simple LRU cache for parsed content
type ContentCache struct {
	mu         sync.RWMutex
	cache      map[string]*CachedContent
	maxEntries int
	order      []string // Track order for LRU eviction
}

type CachedContent struct {
	Content         string
	LastModified    time.Time
	StatementRanges []lsp.Range
}

// NewContentCache creates a new content cache
func NewContentCache(maxEntries int) *ContentCache {
	return &ContentCache{
		cache:      make(map[string]*CachedContent),
		maxEntries: maxEntries,
		order:      make([]string, 0, maxEntries),
	}
}

// Get retrieves cached content
func (c *ContentCache) Get(uri string) (*CachedContent, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	content, exists := c.cache[uri]
	return content, exists
}

// Set stores content in cache with LRU eviction
func (c *ContentCache) Set(uri string, content *CachedContent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	if _, exists := c.cache[uri]; exists {
		// If already exists, just update and move to end
		for i, u := range c.order {
			if u == uri {
				// More efficient removal for common case of updating existing entry
				copy(c.order[i:], c.order[i+1:])
				c.order = c.order[:len(c.order)-1]
				break
			}
		}
	}

	// Add to end (most recently used)
	c.order = append(c.order, uri)

	// Evict oldest if over capacity
	if len(c.order) > c.maxEntries {
		oldest := c.order[0]
		delete(c.cache, oldest)
		c.order = c.order[1:]
	}

	c.cache[uri] = content
}
