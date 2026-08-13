package loadtest

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

func runInteractiveWorkload(ctx context.Context, _ *sql.DB, cfg *Config, tenants []Tenant, concurrency int) (InteractiveResult, error) {
	d := cfg.SteadyStateDuration
	if d <= 0 {
		d = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, d)
	defer cancel()

	queries := cfg.InteractiveQueries
	if len(queries) == 0 {
		queries = DefaultInteractiveQueries()
	}

	var (
		mu         sync.Mutex
		latencies  []time.Duration
		errorCount int
		timeouts   int
	)
	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Go(func() {
			// One persistent connection per session, modeling a single active
			// query-editor session pinned to one workspace database. Reusing the
			// connection avoids per-query connect/teardown churn, which exhausts
			// ephemeral ports on a local Docker host and is not how a real query
			// editor behaves.
			t := tenants[rand.Intn(len(tenants))]
			db, err := sql.Open("pgx", cfg.tenantDSN(t.Database, t.Role, t.Password))
			if err != nil {
				mu.Lock()
				errorCount++
				mu.Unlock()
				if cfg.Verbose {
					log.Printf("interactive: open tenant %s: %v", t.Database, err)
				}
				return
			}
			defer db.Close()
			db.SetMaxOpenConns(1)
			db.SetMaxIdleConns(1)
			if err := db.PingContext(runCtx); err != nil {
				mu.Lock()
				errorCount++
				mu.Unlock()
				if cfg.Verbose {
					log.Printf("interactive: ping tenant %s: %v", t.Database, err)
				}
				return
			}
			for runCtx.Err() == nil {
				qctx, qcancel := context.WithTimeout(ctx, 10*time.Second)
				begin := time.Now()
				_, err := db.ExecContext(qctx, queries[rand.Intn(len(queries))])
				latency := time.Since(begin)
				qcancel()
				if err != nil {
					mu.Lock()
					if errors.Is(err, context.DeadlineExceeded) {
						timeouts++
					} else {
						errorCount++
					}
					mu.Unlock()
					if cfg.Verbose && !errors.Is(err, context.DeadlineExceeded) {
						log.Printf("interactive: query %s: %v", t.Database, err)
					}
					continue
				}
				mu.Lock()
				latencies = append(latencies, latency)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return InteractiveResult{
		Concurrency: concurrency,
		Stats:       newDurationStats(latencies, errorCount),
		Timeouts:    timeouts,
		Duration:    time.Since(start),
	}, nil
}

// nolint:unparam // failures are reported via DDLResult.Errors; the error result is always nil by contract.
func runDDLWorkload(ctx context.Context, _ *sql.DB, cfg *Config, tenants []Tenant) (DDLResult, error) {
	var (
		latencies  []time.Duration
		errorCount int
	)
	for _, t := range tenants {
		db, err := sql.Open("pgx", cfg.adminDSNForDB(t.Database))
		if err != nil {
			errorCount++
			if cfg.Verbose {
				log.Printf("ddl: open tenant %s: %v", t.Database, err)
			}
			continue
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		for _, stmt := range cfg.DDLStatements {
			begin := time.Now()
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				errorCount++
				if cfg.Verbose {
					log.Printf("ddl: tenant %s: %v", t.Database, err)
				}
				continue
			}
			latencies = append(latencies, time.Since(begin))
		}
		db.Close()
	}
	return DDLResult{Stats: newDurationStats(latencies, errorCount), Errors: errorCount}, nil
}

// nolint:unparam // failures are reported via ChurnResult.Errors; the error result is always nil by contract.
func runChurn(ctx context.Context, db *sql.DB, cfg *Config, tenants []Tenant) (ChurnResult, error) {
	k := 10
	if len(tenants) < k {
		k = len(tenants)
	}
	const offset = 1_000_000

	var result ChurnResult
	created := make(map[int]bool)
	for j := 0; j < k; j++ {
		index := offset + j
		role := cfg.roleName(index)
		dbase := cfg.databaseName(index)
		pw, err := randomHexPassword(16)
		if err != nil {
			result.Errors++
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD '%s'", role, pw)); err != nil {
			result.Errors++
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbase)); err != nil {
			result.Errors++
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("REVOKE CONNECT ON DATABASE %s FROM PUBLIC", dbase)); err != nil {
			result.Errors++
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", dbase, role)); err != nil {
			result.Errors++
			continue
		}
		created[index] = true
		result.Created++
	}

	for index := range created {
		role := cfg.roleName(index)
		dbase := cfg.databaseName(index)
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", dbase)); err != nil {
			result.Errors++
			continue
		}
		if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP ROLE %s", role)); err != nil {
			result.Errors++
			continue
		}
		result.Dropped++
	}

	for index := range created {
		role := cfg.roleName(index)
		dbase := cfg.databaseName(index)
		var exists bool
		if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", dbase).Scan(&exists); err != nil {
			result.Errors++
			continue
		}
		if exists {
			result.Orphans++
		}
		if err := db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", role).Scan(&exists); err != nil {
			result.Errors++
			continue
		}
		if exists {
			result.Orphans++
		}
	}
	return result, nil
}

// randomHexPassword returns a cryptographically random hex-encoded password.
func randomHexPassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := cryptorand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
