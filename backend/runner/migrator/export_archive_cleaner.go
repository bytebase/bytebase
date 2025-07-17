package migrator

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
	exportArchiveCleanupInterval = 1 * time.Hour
	exportArchiveRetentionPeriod = 24 * time.Hour
)

// ExportArchiveCleaner is the cleaner for expired export archives.
// It runs periodically to delete export archives older than the retention period.
type ExportArchiveCleaner struct {
	store *store.Store
}

// NewExportArchiveCleaner creates a new ExportArchiveCleaner.
func NewExportArchiveCleaner(store *store.Store) *ExportArchiveCleaner {
	return &ExportArchiveCleaner{
		store: store,
	}
}

// Run starts the ExportArchiveCleaner.
func (c *ExportArchiveCleaner) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(exportArchiveCleanupInterval)
	defer ticker.Stop()
	slog.Debug(fmt.Sprintf("Export archive cleaner started and will run every %v", exportArchiveCleanupInterval))

	// Run once immediately on startup
	if err := c.cleanup(ctx); err != nil {
		slog.Error("Failed to run export archive cleanup on startup", log.BBError(err))
	}

	for {
		select {
		case <-ticker.C:
			if err := c.cleanup(ctx); err != nil {
				slog.Error("Failed to run export archive cleanup", log.BBError(err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *ExportArchiveCleaner) cleanup(ctx context.Context) error {
	rowsAffected, err := c.store.DeleteExpiredExportArchives(ctx, exportArchiveRetentionPeriod)
	if err != nil {
		return err
	}

	if rowsAffected > 0 {
		slog.Info("Cleaned up expired export archives", slog.Int64("count", rowsAffected))
	}

	return nil
}
