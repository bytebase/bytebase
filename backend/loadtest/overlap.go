package loadtest

import (
	"context"
	"sync"
)

// runOverlapWorkload runs sync, interactive, and DDL concurrently at the
// realistic per-workspace concurrency levels: sync and DDL are capped at a
// modest overlap (independent per-workspace schedules) and interactive runs a
// fixed steady-state number of sessions, followed by a fixed burst phase. This
// models the per-workspace model where each workspace's instance syncs, queries,
// and applies DDL only against its own database, so concurrency is not N-wide.
// nolint:unparam // failures are reported via the result structs; the error result is always nil by contract.
func runOverlapWorkload(ctx context.Context, cfg *Config, tenants []Tenant) (SyncResult, []InteractiveResult, DDLResult, error) {
	var (
		mu  sync.Mutex
		sy  SyncResult
		irs []InteractiveResult
		ddl DDLResult
	)
	var wg sync.WaitGroup
	wg.Go(func() {
		r, _ := runSyncWorkload(ctx, nil, cfg, tenants)
		mu.Lock()
		sy = r
		mu.Unlock()
	})
	wg.Go(func() {
		r, _ := runInteractiveWorkload(ctx, nil, cfg, tenants, cfg.interactiveConcurrency())
		mu.Lock()
		irs = append(irs, r)
		mu.Unlock()
	})
	wg.Go(func() {
		r, _ := runDDLWorkload(ctx, nil, cfg, tenants)
		mu.Lock()
		ddl = r
		mu.Unlock()
	})
	wg.Wait()
	// The burst phase runs after the steady-state overlap so its concurrency is
	// measured independently (steady + burst, per the README workload model).
	ir, _ := runInteractiveWorkload(ctx, nil, cfg, tenants, cfg.interactiveBurst())
	irs = append(irs, ir)
	return sy, irs, ddl, nil
}
