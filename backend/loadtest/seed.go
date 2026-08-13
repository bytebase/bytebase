package loadtest

import (
	"context"
	"database/sql"
	"time"
)

// seedAll applies cfg.SeedSQL verbatim to each tenant database. The seed SQL
// contains many statements including dollar-quoted plpgsql bodies, so it is
// sent as a single ExecContext over the simple protocol instead of being split.
// nolint:unparam // failures are reported via SeedResult.Stats.Errors; the error result is always nil by contract.
func seedAll(ctx context.Context, _ *sql.DB, cfg *Config, tenants []Tenant) (SeedResult, error) {
	start := time.Now()
	latencies := make([]time.Duration, 0, len(tenants))
	var errors int
	for _, t := range tenants {
		dsn := cfg.tenantDSN(t.Database, t.Role, t.Password) + " default_query_exec_mode=simple_protocol"
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			errors++
			continue
		}
		begin := time.Now()
		if _, err := db.ExecContext(ctx, cfg.SeedSQL); err != nil {
			db.Close()
			errors++
			continue
		}
		latencies = append(latencies, time.Since(begin))
		db.Close()
	}
	return SeedResult{Stats: newDurationStats(latencies, errors), TotalDuration: time.Since(start)}, nil
}
