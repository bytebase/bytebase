package store

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/qb"
)

// DBConnectionManager manages database connections with support for dynamic updates.
type DBConnectionManager struct {
	mu          sync.Mutex
	db          *sql.DB
	pgURLOrFile string // Either a PostgreSQL URL or a file path
	watcher     *fsnotify.Watcher
	stopWatcher chan struct{}
}

// NewDBConnectionManager creates a new database connection manager.
func NewDBConnectionManager(pgURLOrFile string) *DBConnectionManager {
	return &DBConnectionManager{
		pgURLOrFile: pgURLOrFile,
		stopWatcher: make(chan struct{}),
	}
}

// Initialize sets up the database connection.
// If pgURLOrFile is a file path, it reads the database URL from that file and watches for changes.
func (m *DBConnectionManager) Initialize(ctx context.Context) error {
	if m.pgURLOrFile == "" {
		return errors.New("database URL is not provided")
	}

	var pgURL string
	var err error

	// Check if it's a file path or direct URL
	if isFilePath(m.pgURLOrFile) {
		pgURL, err = readURLFromFile(m.pgURLOrFile)
		if err != nil {
			return err
		}

		// Start watching the file for changes
		if err := m.startFileWatcher(ctx, m.pgURLOrFile); err != nil {
			return errors.Wrap(err, "failed to start file watcher")
		}
	} else {
		pgURL = m.pgURLOrFile
	}

	// Create initial connection
	db, err := createConnectionWithTracer(ctx, pgURL)
	if err != nil {
		return err
	}

	m.db = db
	return nil
}

// GetDB returns the current database connection.
func (m *DBConnectionManager) GetDB() *sql.DB {
	return m.db
}

// Close stops the file watcher and closes the database connection.
func (m *DBConnectionManager) Close() error {
	if m.watcher != nil {
		close(m.stopWatcher)
		m.watcher.Close()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db == nil {
		return nil
	}

	err := m.db.Close()
	m.db = nil
	return err
}

// startFileWatcher starts watching the PG_URL file for changes.
func (m *DBConnectionManager) startFileWatcher(ctx context.Context, filePath string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return errors.Wrap(err, "failed to create file watcher")
	}
	m.watcher = watcher

	if err := watcher.Add(filePath); err != nil {
		watcher.Close()
		return errors.Wrapf(err, "failed to watch file: %s", filePath)
	}

	go m.watchFile(ctx, filePath)
	return nil
}

// watchFile monitors the file for changes and updates the connection when needed.
func (m *DBConnectionManager) watchFile(ctx context.Context, filePath string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopWatcher:
			return
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				m.reloadConnection(ctx, filePath)
			}
		case err, ok := <-m.watcher.Errors:
			if ok && err != nil {
				slog.Error("File watcher error", "error", err)
			}
		}
	}
}

// reloadConnection reads the updated file and swaps the database connection.
func (m *DBConnectionManager) reloadConnection(ctx context.Context, filePath string) {
	// Small delay to ensure file write is complete
	time.Sleep(100 * time.Millisecond)

	newURL, err := readURLFromFile(filePath)
	if err != nil {
		slog.Error("Failed to read updated PG URL file", "error", err, "file", filePath)
		return
	}

	slog.Info("PG URL file content updated, reconnecting database")

	// Create new connection first (zero downtime)
	newDB, err := createConnectionWithTracer(ctx, newURL)
	if err != nil {
		slog.Error("Failed to create new database connection", "error", err)
		return
	}

	// Swap connections atomically
	m.mu.Lock()
	oldDB := m.db
	m.db = newDB
	m.mu.Unlock()

	// Gracefully drain old connections and force close after 1 hour
	if oldDB != nil {
		// Set max idle connections to 0 to gradually close connections
		// This allows active connections to complete naturally
		oldDB.SetMaxIdleConns(0)
		oldDB.SetConnMaxLifetime(1 * time.Minute)

		// Force close after 1 hour as a safety measure
		go func() {
			time.Sleep(1 * time.Hour)
			if err := oldDB.Close(); err != nil {
				slog.Warn("Failed to force close old database connection", "error", err)
			}
		}()
	}

	slog.Info("Database connection updated successfully", "file", filePath)
}

// Helper functions

func isFilePath(s string) bool {
	if strings.HasPrefix(s, "postgresql://") || strings.HasPrefix(s, "postgres://") {
		return false
	}

	if strings.Contains(s, "host=/tmp") {
		return false
	}

	return true
}

func readURLFromFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read database URL from file: %s", path)
	}
	return strings.TrimSpace(string(content)), nil
}

func createConnectionWithTracer(ctx context.Context, pgURL string) (*sql.DB, error) {
	pgxConfig, err := pgx.ParseConfig(pgURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse database URL")
	}

	pgxConfig.Tracer = &metadataDBTracer{}
	db := stdlib.OpenDB(*pgxConfig)

	// Validate connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.Wrap(err, "failed to ping database")
	}

	// Configure connection pool
	var maxConns, reservedConns int
	q := qb.Q().Space("SHOW max_connections")
	sql, args, err := q.ToSQL()
	if err != nil {
		db.Close()
		return nil, errors.Wrap(err, "failed to build sql")
	}
	if err := db.QueryRowContext(ctx, sql, args...).Scan(&maxConns); err != nil {
		db.Close()
		return nil, errors.Wrap(err, "failed to get max_connections")
	}

	q = qb.Q().Space("SHOW superuser_reserved_connections")
	sql, args, err = q.ToSQL()
	if err != nil {
		db.Close()
		return nil, errors.Wrap(err, "failed to build sql")
	}
	if err := db.QueryRowContext(ctx, sql, args...).Scan(&reservedConns); err != nil {
		db.Close()
		return nil, errors.Wrap(err, "failed to get superuser_reserved_connections")
	}

	maxOpenConns := maxConns - reservedConns
	if maxOpenConns > 50 {
		maxOpenConns = 50
	}
	db.SetMaxOpenConns(maxOpenConns)

	return db, nil
}
