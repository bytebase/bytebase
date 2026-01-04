package cleaner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	cleanupInterval              = 1 * time.Hour
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
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	slog.Debug(fmt.Sprintf("Data cleaner started and will run every %v", cleanupInterval))

	// Run once immediately on startup
	c.cleanup(ctx)

	for {
		select {
		case <-ticker.C:
			c.cleanup(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *DataCleaner) cleanup(ctx context.Context) {
	c.cleanupExportArchives(ctx)
	c.cleanupOAuth2Data(ctx)
	c.cleanupWebRefreshTokens(ctx)
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
