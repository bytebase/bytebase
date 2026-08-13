package loadtest

import (
	"context"
	"sync"
)

// runOverlapWorkload runs sync, interactive, and DDL simultaneously, each with
// one worker/session per database, modeling the worst case where every workspace
// is syncing, querying, and applying DDL at the same time.
// nolint:unparam // failures are reported via the result structs; the error result is always nil by contract.
func runOverlapWorkload(ctx context.Context, cfg *Config, tenants []Tenant) (SyncResult, InteractiveResult, DDLResult, error) {
	var (
		mu  sync.Mutex
		sy  SyncResult
		ir  InteractiveResult
		ddl DDLResult
	)
	n := len(tenants)
	var wg sync.WaitGroup
	wg.Go(func() {
		r, _ := runSyncWorkload(ctx, nil, cfg, tenants)
		mu.Lock()
		sy = r
		mu.Unlock()
	})
	wg.Go(func() {
		r, _ := runInteractiveWorkload(ctx, nil, cfg, tenants, n)
		mu.Lock()
		ir = r
		mu.Unlock()
	})
	wg.Go(func() {
		r, _ := runDDLWorkload(ctx, nil, cfg, tenants)
		mu.Lock()
		ddl = r
		mu.Unlock()
	})
	wg.Wait()
	return sy, ir, ddl, nil
}
