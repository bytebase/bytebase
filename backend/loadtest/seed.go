package loadtest

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// seedAll applies cfg.SeedSQL to each tenant database as the tenant role so the
// role owns the schema it seeds and can later run DDL (change tickets) as the
// workspace, matching the per-workspace instance model. The control plane only
// prepares the database: it grants the role CREATE ON the public schema before
// seeding. Seeding as the role is required on Cloud SQL because the postgres
// user is not a superuser and cannot SET ROLE or transfer ownership of objects
// it creates; a role can only own what it creates itself. The seed SQL contains
// many statements including dollar-quoted plpgsql bodies, so it is sent as a
// single ExecContext over the simple protocol.
// nolint:unparam // failures are reported via SeedResult.Stats.Errors; the error result is always nil by contract.
func seedAll(ctx context.Context, _ *sql.DB, cfg *Config, tenants []Tenant) (SeedResult, error) {
	start := time.Now()
	latencies := make([]time.Duration, 0, len(tenants))
	var errors int
	for _, t := range tenants {
		grantDB, err := sql.Open("pgx", cfg.adminDSNForDB(t.Database))
		if err != nil {
			errors++
			continue
		}
		if _, err := grantDB.ExecContext(ctx, fmt.Sprintf("GRANT CREATE ON SCHEMA public TO %s", t.Role)); err != nil {
			grantDB.Close()
			errors++
			if cfg.Verbose {
				log.Printf("seed grant tenant %s: %v", t.Database, err)
			}
			continue
		}
		grantDB.Close()

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
			if cfg.Verbose {
				log.Printf("seed tenant %s: %v", t.Database, err)
			}
			continue
		}
		db.Close()
		latencies = append(latencies, time.Since(begin))
	}
	return SeedResult{Stats: newDurationStats(latencies, errors), TotalDuration: time.Since(start)}, nil
}
