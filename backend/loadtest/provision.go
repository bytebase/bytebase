package loadtest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
)

// nolint:unparam // failures are reported via Stats.Errors, the error result is always nil by contract.
func provision(ctx context.Context, db *sql.DB, cfg *Config, count int) ([]Tenant, ProvisionResult, error) {
	started := time.Now()
	tenants := make([]Tenant, 0, count)
	latencies := make([]time.Duration, 0, count)
	errors := 0

	for i := 0; i < count; i++ {
		role := cfg.roleName(i)
		dbase := cfg.databaseName(i)

		pwBytes := make([]byte, 16)
		if _, err := rand.Read(pwBytes); err != nil {
			errors++
			if cfg.Verbose {
				log.Printf("provision tenant %d: generate password: %v", i, err)
			}
			continue
		}
		pw := hex.EncodeToString(pwBytes)

		tenantStart := time.Now()
		statements := []string{
			fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD '%s'", role, pw),
			fmt.Sprintf("CREATE DATABASE %s", dbase),
			fmt.Sprintf("REVOKE CONNECT ON DATABASE %s FROM PUBLIC", dbase),
			fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", dbase, role),
			// CREATE ON DATABASE lets the role create schemas (e.g. the seed's
			// bbdataarchive schema) in its own database.
			fmt.Sprintf("GRANT CREATE ON DATABASE %s TO %s", dbase, role),
		}
		failed := false
		for _, stmt := range statements {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				failed = true
				errors++
				if cfg.Verbose {
					log.Printf("provision tenant %d: %v", i, err)
				}
				break
			}
		}
		if failed {
			continue
		}

		latencies = append(latencies, time.Since(tenantStart))
		tenants = append(tenants, Tenant{Index: i, Database: dbase, Role: role, Password: pw})
	}

	return tenants, ProvisionResult{
		Tenants:       tenants,
		Stats:         newDurationStats(latencies, errors),
		TotalDuration: time.Since(started),
	}, nil
}

func cleanup(ctx context.Context, db *sql.DB, cfg *Config, tenants []Tenant) (CleanupResult, error) {
	result := CleanupResult{}
	for _, tenant := range tenants {
		// Terminate lingering backends of the tenant role before dropping its
		// database. This models the 7-day expiry, which must terminate the
		// workspace role's connections first: Cloud SQL's postgres user is not
		// a superuser, so DROP DATABASE ... WITH (FORCE) alone fails with
		// "permission denied to terminate process" when backends of another
		// role linger after the workload phases.
		if err := terminateTenantBackends(ctx, cfg, tenant); err != nil && cfg.Verbose {
			log.Printf("cleanup tenant %d: terminate backends: %v", tenant.Index, err)
		}
		// Backend termination is asynchronous: a backend can linger for a
		// moment after its client socket closes. Wait until none remain on the
		// database before dropping it, so the drop does not race with session
		// teardown.
		if err := waitTenantBackendsDrained(ctx, db, tenant.Database); err != nil && cfg.Verbose {
			log.Printf("cleanup tenant %d: wait backends drained: %v", tenant.Index, err)
		}
		dropped := false
		for attempt := 0; attempt < 3 && !dropped; attempt++ {
			if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", tenant.Database)); err != nil {
				if cfg.Verbose && attempt == 2 {
					log.Printf("cleanup tenant %d: drop database %s: %v", tenant.Index, tenant.Database, err)
				}
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			dropped = true
			result.Dropped++
		}
		if !dropped {
			result.Failed++
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP ROLE %s", tenant.Role)); err != nil {
			result.Failed++
			if cfg.Verbose {
				log.Printf("cleanup tenant %d: drop role %s: %v", tenant.Index, tenant.Role, err)
			}
		}
	}

	if rows, err := db.QueryContext(ctx, "SELECT datname FROM pg_database WHERE datname LIKE $1", cfg.DatabaseNamePrefix+"%"); err != nil {
		if cfg.Verbose {
			log.Printf("cleanup: query orphan databases: %v", err)
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				if cfg.Verbose {
					log.Printf("cleanup: scan orphan database: %v", err)
				}
				continue
			}
			result.OrphanDatabases = append(result.OrphanDatabases, name)
		}
		if err := rows.Err(); err != nil && cfg.Verbose {
			log.Printf("cleanup: iterate orphan databases: %v", err)
		}
	}

	if rows, err := db.QueryContext(ctx, "SELECT rolname FROM pg_roles WHERE rolname LIKE $1", cfg.RoleNamePrefix+"%"); err != nil {
		if cfg.Verbose {
			log.Printf("cleanup: query orphan roles: %v", err)
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				if cfg.Verbose {
					log.Printf("cleanup: scan orphan role: %v", err)
				}
				continue
			}
			result.OrphanRoles = append(result.OrphanRoles, name)
		}
		if err := rows.Err(); err != nil && cfg.Verbose {
			log.Printf("cleanup: iterate orphan roles: %v", err)
		}
	}

	return result, nil
}

// terminateTenantBackends connects as the tenant role and terminates any of its
// lingering backends on its own database. A role can terminate its own
// backends, which the control-plane admin (a non-superuser on Cloud SQL)
// cannot do for another role.
func terminateTenantBackends(ctx context.Context, cfg *Config, t Tenant) error {
	db, err := sql.Open("pgx", cfg.tenantDSN(t.Database, t.Role, t.Password))
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = current_database() AND usename = current_user AND pid <> pg_backend_pid()`); err != nil {
		return err
	}
	return nil
}

// waitTenantBackendsDrained polls pg_stat_activity as the admin user until no
// backends remain on the tenant database. Terminated backends disappear
// asynchronously after their client socket closes, and dropping the database in
// that window would fail because the admin cannot terminate another role's
// processes.
func waitTenantBackendsDrained(ctx context.Context, db *sql.DB, database string, timeout ...time.Duration) error {
	deadline := time.Now().Add(15 * time.Second)
	if len(timeout) > 0 {
		deadline = time.Now().Add(timeout[0])
	}
	for time.Now().Before(deadline) {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname = $1`, database).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return errors.Errorf("backends still present on %s", database)
}

// cleanupPrefix removes any databases and roles left over from a previous run so
// re-runs are idempotent. It is best-effort: errors are ignored here and the main
// cleanup phase reports any remaining orphans.
func cleanupPrefix(ctx context.Context, db *sql.DB, cfg *Config) {
	for _, name := range queryNames(ctx, db, "SELECT datname FROM pg_database WHERE datname LIKE $1", cfg.DatabaseNamePrefix) {
		_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", name))
	}
	for _, name := range queryNames(ctx, db, "SELECT rolname FROM pg_roles WHERE rolname LIKE $1", cfg.RoleNamePrefix) {
		_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP ROLE %s", name))
	}
}

// queryNames returns the single text column of a parameterized LIKE query.
func queryNames(ctx context.Context, db *sql.DB, query, prefix string) []string {
	rows, err := db.QueryContext(ctx, query, prefix+"%")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			names = append(names, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	return names
}
