package loadtest

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// seedAll applies cfg.SeedSQL to each tenant database as the admin user (so the
// tables are owned by the admin, matching Cloud SQL's restriction that the
// postgres user cannot transfer database ownership), then grants the tenant role
// read access. The seed SQL contains many statements including dollar-quoted
// plpgsql bodies, so it is sent as a single ExecContext over the simple protocol.
// nolint:unparam // failures are reported via SeedResult.Stats.Errors; the error result is always nil by contract.
func seedAll(ctx context.Context, _ *sql.DB, cfg *Config, tenants []Tenant) (SeedResult, error) {
	start := time.Now()
	latencies := make([]time.Duration, 0, len(tenants))
	var errors int
	for _, t := range tenants {
		dsn := cfg.adminDSNForDB(t.Database) + " default_query_exec_mode=simple_protocol"
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			errors++
			continue
		}
		begin := time.Now()
		failed := false
		if _, err := db.ExecContext(ctx, cfg.SeedSQL); err != nil {
			failed = true
			if cfg.Verbose {
				log.Printf("seed tenant %s: %v", t.Database, err)
			}
		} else {
			grants := []string{
				fmt.Sprintf("GRANT USAGE ON SCHEMA public TO %s", t.Role),
				fmt.Sprintf("GRANT SELECT ON ALL TABLES IN SCHEMA public TO %s", t.Role),
				fmt.Sprintf("GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s", t.Role),
			}
			for _, g := range grants {
				if _, err := db.ExecContext(ctx, g); err != nil {
					failed = true
					if cfg.Verbose {
						log.Printf("seed grant tenant %s: %v", t.Database, err)
					}
					break
				}
			}
		}
		db.Close()
		if failed {
			errors++
			continue
		}
		latencies = append(latencies, time.Since(begin))
	}
	return SeedResult{Stats: newDurationStats(latencies, errors), TotalDuration: time.Since(start)}, nil
}
