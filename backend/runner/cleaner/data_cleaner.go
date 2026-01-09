package cleaner

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	cleanupInterval              = 1 * time.Hour
	staleDetectionInterval       = 30 * time.Second
	stalenessThreshold           = 1 * time.Minute
	planCheckRunTimeout          = 10 * time.Minute
	heartbeatRetentionPeriod     = 1 * time.Hour
	exportArchiveRetentionPeriod = 24 * time.Hour
	oauth2ClientRetentionPeriod  = 30 * 24 * time.Hour // 30 days of inactivity
)

// DataCleaner periodically cleans up expired data from the database.
type DataCleaner struct {
	store *store.Store
}

// NewDataCleaner creates a new DataCleaner.
func NewDataCleaner(store *store.Store) *DataCleaner {
	return &DataCleaner{
		store: store,
	}
}

// Run starts the DataCleaner.
func (c *DataCleaner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	cleanupTicker := time.NewTicker(cleanupInterval)
	defer cleanupTicker.Stop()

	staleTicker := time.NewTicker(staleDetectionInterval)
	defer staleTicker.Stop()

	slog.Debug("Data cleaner started",
		slog.Duration("cleanupInterval", cleanupInterval),
		slog.Duration("staleDetectionInterval", staleDetectionInterval))

	// Run cleanup immediately on startup
	c.cleanup(ctx)
	c.detectStaleTaskRuns(ctx)
	c.detectStalePlanCheckRuns(ctx)

	for {
		select {
		case <-cleanupTicker.C:
			c.cleanup(ctx)
		case <-staleTicker.C:
			c.detectStaleTaskRuns(ctx)
			c.detectStalePlanCheckRuns(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *DataCleaner) cleanup(ctx context.Context) {
	c.cleanupExportArchives(ctx)
	c.cleanupOAuth2Data(ctx)
	c.cleanupWebRefreshTokens(ctx)
	c.cleanupStaleHeartbeats(ctx)
}

func (c *DataCleaner) detectStaleTaskRuns(ctx context.Context) {
	rowsAffected, err := c.store.FailStaleTaskRuns(ctx, stalenessThreshold)
	if err != nil {
		slog.Error("Failed to detect stale task runs", log.BBError(err))
		return
	}
	if rowsAffected > 0 {
		slog.Info("Marked stale task runs as failed", slog.Int64("count", rowsAffected))
	}
}

func (c *DataCleaner) detectStalePlanCheckRuns(ctx context.Context) {
	rowsAffected, err := c.store.FailStalePlanCheckRuns(ctx, planCheckRunTimeout)
	if err != nil {
		slog.Error("Failed to detect stale plan check runs", log.BBError(err))
		return
	}
	if rowsAffected > 0 {
		slog.Info("Marked stale plan check runs as failed", slog.Int64("count", rowsAffected))
	}
}

func (c *DataCleaner) cleanupStaleHeartbeats(ctx context.Context) {
	rowsAffected, err := c.store.DeleteStaleReplicaHeartbeats(ctx, heartbeatRetentionPeriod)
	if err != nil {
		slog.Error("Failed to clean up stale replica heartbeats", log.BBError(err))
		return
	}
	if rowsAffected > 0 {
		slog.Info("Cleaned up stale replica heartbeats", slog.Int64("count", rowsAffected))
	}
}

func (c *DataCleaner) cleanupExportArchives(ctx context.Context) {
	rowsAffected, err := c.store.DeleteExpiredExportArchives(ctx, exportArchiveRetentionPeriod)
	if err != nil {
		slog.Error("Failed to clean up expired export archives", log.BBError(err))
		return
	}
	if rowsAffected > 0 {
		slog.Info("Cleaned up expired export archives", slog.Int64("count", rowsAffected))
	}
}

func (c *DataCleaner) cleanupOAuth2Data(ctx context.Context) {
	// Clean up expired authorization codes
	if rowsAffected, err := c.store.DeleteExpiredOAuth2AuthorizationCodes(ctx); err != nil {
		slog.Error("Failed to clean up expired OAuth2 authorization codes", log.BBError(err))
	} else if rowsAffected > 0 {
		slog.Info("Cleaned up expired OAuth2 authorization codes", slog.Int64("count", rowsAffected))
	}

	// Clean up expired refresh tokens
	if rowsAffected, err := c.store.DeleteExpiredOAuth2RefreshTokens(ctx); err != nil {
		slog.Error("Failed to clean up expired OAuth2 refresh tokens", log.BBError(err))
	} else if rowsAffected > 0 {
		slog.Info("Cleaned up expired OAuth2 refresh tokens", slog.Int64("count", rowsAffected))
	}

	// Clean up inactive OAuth2 clients (DCR clients that haven't been used)
	expireBefore := time.Now().Add(-oauth2ClientRetentionPeriod)
	if rowsAffected, err := c.store.DeleteExpiredOAuth2Clients(ctx, expireBefore); err != nil {
		slog.Error("Failed to clean up inactive OAuth2 clients", log.BBError(err))
	} else if rowsAffected > 0 {
		slog.Info("Cleaned up inactive OAuth2 clients", slog.Int64("count", rowsAffected))
	}
}

func (c *DataCleaner) cleanupWebRefreshTokens(ctx context.Context) {
	if rowsAffected, err := c.store.DeleteExpiredWebRefreshTokens(ctx); err != nil {
		slog.Error("Failed to clean up expired web refresh tokens", log.BBError(err))
	} else if rowsAffected > 0 {
		slog.Info("Cleaned up expired web refresh tokens", slog.Int64("count", rowsAffected))
	}
}
