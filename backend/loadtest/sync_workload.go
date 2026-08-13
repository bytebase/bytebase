package loadtest

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/sourcegraph/conc/pool"
)

// syncQueries are representative catalog-introspection and schema-dump queries
// Bytebase's schema sync runs per database.
var syncQueries = []string{
	`SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind IN ('r','p','v','m','S')`,
	`SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema')`,
	`SELECT table_schema, table_name, column_name, data_type FROM information_schema.columns WHERE table_schema NOT IN ('pg_catalog','information_schema')`,
	`SELECT schemaname, tablename, indexname FROM pg_indexes WHERE schemaname NOT IN ('pg_catalog','information_schema')`,
	`SELECT schemaname, tablename, tableowner FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema')`,
}

// runSyncWorkload replays Bytebase's schema-sync load: up to
// cfg.syncConcurrency() workers each open a connection to one tenant database,
// run catalog introspection plus a light schema dump, then close it.
// nolint:unparam // failures are recorded in DurationStats.Errors; the error result is always nil by contract.
func runSyncWorkload(ctx context.Context, _ *sql.DB, cfg *Config, tenants []Tenant) (SyncResult, error) {
	start := time.Now()

	var mu sync.Mutex
	latencies := make([]time.Duration, 0, len(tenants))
	errors := 0

	p := pool.New().WithMaxGoroutines(cfg.syncConcurrency())
	for _, t := range tenants {
		p.Go(func() {
			start := time.Now()
			err := syncOneDatabase(ctx, cfg, t, syncQueries)
			elapsed := time.Since(start)
			mu.Lock()
			if err != nil {
				errors++
			} else {
				latencies = append(latencies, elapsed)
			}
			mu.Unlock()
		})
	}
	p.Wait()

	return SyncResult{
		Stats:         newDurationStats(latencies, errors),
		TotalDuration: time.Since(start),
		Databases:     len(tenants),
	}, nil
}

// syncOneDatabase opens one connection to a tenant database and runs the sync
// queries sequentially, returning the first error encountered.
func syncOneDatabase(ctx context.Context, cfg *Config, t Tenant, queries []string) error {
	db, err := sql.Open("pgx", cfg.adminDSNForDB(t.Database))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	for _, q := range queries {
		if err := func() error {
			rows, err := db.QueryContext(ctx, q)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
			}
			return rows.Err()
		}(); err != nil {
			return err
		}
	}
	return nil
}
