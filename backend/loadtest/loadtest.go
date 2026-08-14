// Package loadtest implements a reproducible load test for the Bytebase Cloud
// sample PostgreSQL instance. It provisions isolated databases and roles,
// seeds the Bytebase sample schema, replays Bytebase-like sync/DDL/interactive
// workloads, records measurements, and removes everything it created.
//
// The core is DSN-driven so the same harness runs against local Postgres
// (Phase 1) and GCP Cloud SQL (Phase 2). See README.md for the agreed workload
// assumptions and pass/fail thresholds.
package loadtest

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"time"

	"github.com/pkg/errors"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// Config is the full test matrix and connection settings.
type Config struct {
	// Connection settings. The admin user must have CREATEDB and CREATEROLE.
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string `json:"-"`
	SSLMode       string // "disable" locally, "require" or "verify-full" for Cloud SQL.

	DatabaseNamePrefix string // e.g. "lt_ws_"
	RoleNamePrefix     string // e.g. "lt_role_"

	// SeedSQL is applied verbatim to each tenant database after creation.
	SeedSQL string `json:"-"`

	// Matrix.
	DatabaseCounts      []int
	SyncInterval        time.Duration // cadence hint for the sync workload.
	SteadyStateDuration time.Duration // duration for the interactive steady-state phase.

	// Workloads.
	InteractiveQueries []string
	DDLStatements      []string

	// Concurrency for the per-workspace workload model. Zero means "use the
	// realistic default" (see defaults.go). DDL models independent per-workspace
	// schedules with a modest overlap; interactive concurrency is held fixed
	// while the database count varies. Sync runs one worker per database (see
	// sync_workload.go).
	DDLConcurrency         int // max concurrent per-workspace change-ticket DDL.
	InteractiveConcurrency int // steady-state interactive sessions.
	InteractiveBurst       int // burst interactive sessions.

	ReportPath string
	Verbose    bool
}

func (c *Config) adminDSN() string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=postgres sslmode=%s connect_timeout=10",
		c.AdminUser, c.AdminPassword, c.Host, c.Port, c.SSLMode)
}

func (c *Config) tenantDSN(database, role, password string) string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=%s connect_timeout=10",
		role, password, c.Host, c.Port, database, c.SSLMode)
}

func (c *Config) adminDSNForDB(database string) string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=%s connect_timeout=10",
		c.AdminUser, c.AdminPassword, c.Host, c.Port, database, c.SSLMode)
}

func (c *Config) databaseName(i int) string {
	return fmt.Sprintf("%s%d", c.DatabaseNamePrefix, i)
}

func (c *Config) roleName(i int) string {
	return fmt.Sprintf("%s%d", c.RoleNamePrefix, i)
}

// openDB opens a single-connection pool with the given DSN and verifies it,
// retrying a few times because a single transient dial failure should not abort
// an otherwise healthy run.
func openDB(ctx context.Context, dsn string) (*sql.DB, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		pingErr := db.PingContext(ctx)
		if pingErr == nil {
			return db, nil
		}
		lastErr = pingErr
		db.Close()
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return nil, lastErr
}

// Tenant is one provisioned workspace database and its dedicated role.
type Tenant struct {
	Index    int
	Database string
	Role     string
	Password string `json:"-"`
}

// DurationStats summarizes a latency distribution in milliseconds.
type DurationStats struct {
	Count  int
	P50Ms  float64
	P95Ms  float64
	P99Ms  float64
	MaxMs  float64
	Errors int
}

func newDurationStats(latencies []time.Duration, errors int) DurationStats {
	if len(latencies) == 0 {
		return DurationStats{Errors: errors}
	}
	ms := make([]float64, len(latencies))
	for i, d := range latencies {
		ms[i] = float64(d.Microseconds()) / 1000.0
	}
	slices.Sort(ms)
	p := func(q float64) float64 {
		return ms[int(q*float64(len(ms)-1))]
	}
	return DurationStats{
		Count:  len(ms),
		P50Ms:  p(0.50),
		P95Ms:  p(0.95),
		P99Ms:  p(0.99),
		MaxMs:  ms[len(ms)-1],
		Errors: errors,
	}
}

// Metrics is an instance-level snapshot from PostgreSQL statistics views.
// Instance-level CPU/memory/IOPS are out of scope for Phase 1 (local Postgres)
// and will be captured from Cloud Monitoring in Phase 2.
type Metrics struct {
	Timestamp         time.Time
	TotalConnections  int
	ActiveConnections int
	IdleConnections   int
	MaxConnections    int
	DatabaseCount     int
	XactCommit        int64
	XactRollback      int64
	BlocksRead        int64
	BlocksHit         int64
	TempFiles         int64
	TempBytes         int64
	Deadlocks         int64
}

type ProvisionResult struct {
	Tenants       []Tenant
	Stats         DurationStats
	TotalDuration time.Duration
}

type SeedResult struct {
	Stats         DurationStats
	TotalDuration time.Duration
}

type SyncResult struct {
	Stats         DurationStats // per-database sync latency
	TotalDuration time.Duration
	Databases     int
}

type InteractiveResult struct {
	Concurrency int
	Stats       DurationStats
	Timeouts    int
	Duration    time.Duration
}

type DDLResult struct {
	Stats  DurationStats
	Errors int
}

type ChurnResult struct {
	Created int
	Dropped int
	Errors  int
	Orphans int
}

type CleanupResult struct {
	Dropped         int
	Failed          int
	OrphanDatabases []string
	OrphanRoles     []string
}

// Result is the outcome of one database-count run through all phases.
type Result struct {
	DatabaseCount int
	StartedAt     time.Time
	FinishedAt    time.Time
	Provision     ProvisionResult
	Seed          SeedResult
	Idle          Metrics
	Sync          SyncResult
	Interactive   []InteractiveResult
	DDL           DDLResult
	Churn         ChurnResult
	Cleanup       CleanupResult
}

// Run executes the full matrix and returns one Result per database count.
func Run(ctx context.Context, cfg Config) ([]Result, error) {
	db, err := openDB(ctx, cfg.adminDSN())
	if err != nil {
		return nil, errors.Wrapf(err, "open admin connection")
	}
	defer db.Close()

	cleanupPrefix(ctx, db, &cfg)

	results := make([]Result, 0, len(cfg.DatabaseCounts))
	for _, count := range cfg.DatabaseCounts {
		r := Result{DatabaseCount: count, StartedAt: time.Now()}

		tenants, prov, err := provision(ctx, db, &cfg, count)
		if err != nil {
			return results, errors.Wrapf(err, "provision %d databases", count)
		}
		r.Provision = prov
		if len(tenants) == 0 {
			r.FinishedAt = time.Now()
			results = append(results, r)
			continue
		}

		seed, err := seedAll(ctx, db, &cfg, tenants)
		if err != nil {
			_, _ = cleanup(ctx, db, &cfg, tenants)
			return results, errors.Wrapf(err, "seed %d databases", count)
		}
		r.Seed = seed

		idle, err := snapshotMetrics(ctx, db)
		if err != nil {
			_, _ = cleanup(ctx, db, &cfg, tenants)
			return results, errors.Wrapf(err, "idle snapshot")
		}
		r.Idle = idle

		syncRes, irs, ddl, err := runOverlapWorkload(ctx, &cfg, tenants)
		if err != nil {
			_, _ = cleanup(ctx, db, &cfg, tenants)
			return results, errors.Wrapf(err, "overlap workload")
		}
		r.Sync = syncRes
		r.Interactive = append(r.Interactive, irs...)
		r.DDL = ddl

		churn, err := runChurn(ctx, db, &cfg, tenants)
		if err != nil {
			_, _ = cleanup(ctx, db, &cfg, tenants)
			return results, errors.Wrapf(err, "churn workload")
		}
		r.Churn = churn

		cleanupRes, err := cleanup(ctx, db, &cfg, tenants)
		if err != nil {
			return results, errors.Wrapf(err, "cleanup %d databases", count)
		}
		r.Cleanup = cleanupRes
		r.FinishedAt = time.Now()

		results = append(results, r)
	}

	if cfg.ReportPath != "" {
		if err := writeReport(cfg.ReportPath, results, cfg); err != nil {
			return results, errors.Wrapf(err, "write report")
		}
	}
	return results, nil
}
