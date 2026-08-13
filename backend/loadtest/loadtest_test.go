package loadtest

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/resources/postgres"
)

// TestSampleInstanceLoadLocal is the Phase 1 smoke run against local Postgres.
// The full authoritative matrix (70/500/1000) runs in Phase 2 against Cloud SQL.
func TestSampleInstanceLoadLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in -short mode")
	}

	ctx := context.Background()
	tc, err := testcontainer.GetPgContainer(ctx)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer tc.Close(ctx)

	port, err := strconv.Atoi(tc.GetPort())
	if err != nil {
		t.Fatalf("parse container port %q: %v", tc.GetPort(), err)
	}

	seed, err := postgres.LoadSampleData()
	if err != nil {
		t.Fatalf("load sample data: %v", err)
	}

	cfg := Config{
		Host:                tc.GetHost(),
		Port:                port,
		AdminUser:           "postgres",
		AdminPassword:       "root-password",
		SSLMode:             "disable",
		DatabaseNamePrefix:  "lt_ws_",
		RoleNamePrefix:      "lt_role_",
		SeedSQL:             seed,
		DatabaseCounts:      []int{3, 10},
		SteadyStateDuration: 10 * time.Second,
		InteractiveQueries:  DefaultInteractiveQueries(),
		DDLStatements:       DefaultDDLStatements(),
		Verbose:             true,
	}

	results, err := Run(ctx, cfg)
	if err != nil {
		t.Fatalf("run load test: %v", err)
	}

	for _, r := range results {
		assertResultClean(t, r)
	}
}

func assertResultClean(t *testing.T, r Result) {
	t.Helper()
	fail := func(format string, args ...any) {
		t.Errorf("database count %d: %s", r.DatabaseCount, fmt.Sprintf(format, args...))
	}
	if r.Provision.Stats.Errors > 0 {
		fail("provision errors = %d", r.Provision.Stats.Errors)
	}
	if r.Seed.Stats.Errors > 0 {
		fail("seed errors = %d", r.Seed.Stats.Errors)
	}
	if r.Sync.Stats.Errors > 0 {
		fail("sync errors = %d", r.Sync.Stats.Errors)
	}
	for _, ir := range r.Interactive {
		if ir.Stats.Errors > 0 || ir.Timeouts > 0 {
			fail("interactive concurrency %d: errors=%d timeouts=%d", ir.Concurrency, ir.Stats.Errors, ir.Timeouts)
		}
	}
	if r.DDL.Errors > 0 {
		fail("ddl errors = %d", r.DDL.Errors)
	}
	if r.Churn.Errors > 0 || r.Churn.Orphans > 0 {
		fail("churn errors=%d orphans=%d", r.Churn.Errors, r.Churn.Orphans)
	}
	if len(r.Cleanup.OrphanDatabases) > 0 || len(r.Cleanup.OrphanRoles) > 0 {
		fail("orphan databases=%v roles=%v", r.Cleanup.OrphanDatabases, r.Cleanup.OrphanRoles)
	}
	if r.Cleanup.Failed > 0 {
		fail("cleanup failures = %d", r.Cleanup.Failed)
	}
}
