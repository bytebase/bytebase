package loadtest

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// seedAll applies cfg.SeedSQL to each tenant database as the admin user (the
// control plane creates and seeds the database before the workspace exists),
// then transfers ownership of the public schema and its objects to the tenant
// role so the role can run DDL (change tickets) as the workspace, matching the
// per-workspace instance model. On Cloud SQL the postgres user can transfer
// object ownership inside a database even though it cannot transfer database
// ownership itself. The seed SQL contains many statements including
// dollar-quoted plpgsql bodies, so it is sent as a single ExecContext over the
// simple protocol.
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
				fmt.Sprintf("ALTER SCHEMA public OWNER TO %s", t.Role),
				ownershipTransferSQL(t.Role),
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

// ownershipTransferSQL transfers ownership of every user object in the public
// schema (tables, views, materialized views, and functions) to the tenant role
// so the role can run DDL against its own database. Serial-owned sequences
// follow their table's ownership when the table is transferred, so they are not
// listed explicitly (transferring one before its table errors with
// "cannot change owner of sequence"); indexes and triggers also follow their
// parent table. The sample schema has no standalone sequences.
func ownershipTransferSQL(role string) string {
	return fmt.Sprintf(`DO $$
DECLARE
  r record;
BEGIN
  FOR r IN
    SELECT n.nspname AS schema, c.relname AS name
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relkind IN ('r', 'v', 'm', 'p')
  LOOP
    EXECUTE format('ALTER TABLE %%I.%%I OWNER TO %s', r.schema, r.name);
  END LOOP;
  FOR r IN
    SELECT p.oid AS oid
    FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE n.nspname = 'public'
  LOOP
    EXECUTE format('ALTER FUNCTION %%s OWNER TO %s', r.oid::regprocedure);
  END LOOP;
END $$;`, role, role)
}
