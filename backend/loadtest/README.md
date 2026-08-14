# Sample-instance load test

Reproducible load test that answers: how many isolated sample databases can one
PostgreSQL/Cloud SQL instance support without unacceptable degradation.

The core (`loadtest.go`) is DSN-driven, so it runs identically against local
Postgres (Phase 1) and GCP Cloud SQL (Phase 2). Components are split across
`provision.go`, `seed.go`, `sync_workload.go`, `workload.go`, `metrics.go`, and
`report.go`.

## Agreed decisions

- Vendor: GCP Cloud SQL for PostgreSQL. AWS out of scope. Docker Postgres only
  for free harness development.
- Tier under test: smallest shippable tier (1 vCPU / ~3.75 GB).
- Connection: direct, no pooler, public IP + TLS; password superuser admin.
- Isolation: one database + one role per workspace. The database is owned by the
  admin user (Cloud SQL's `postgres` superuser cannot `SET ROLE` / transfer
  database ownership); the role is granted `CONNECT` on only its own database
  and is transferred ownership of its public schema and objects so it can run
  change-ticket DDL as the workspace. `pg_database` name disclosure is accepted.
- Seed: the Bytebase sample HR schema (7 tables, 2 views, 1 function, ~13k rows),
  applied by the admin (control plane); ownership is then transferred to the
  workspace role.

## Workload assumptions (labeled, not measured traffic)

- Sync load: replay Bytebase's per-workspace catalog-introspection + schema-dump
  pattern. Each workspace's instance syncs only its own database as its
  per-database role, on an independent 15-minute cadence; the realistic overlap
  is a small number of concurrent syncs (default 10, conservative vs the
  expected ~N x sync-duration/15min at 1000 databases).
- DDL load: replay change-ticket DDL (idempotent, non-rewriting) as the
  per-database role, capped at a modest overlap (default 5) because change
  tickets are rare and independently scheduled.
- Interactive: 10 concurrent sessions steady state, 50 burst, held fixed while
  database count varies. Database count and connection count are independent.

All three concurrency levels are configurable (`-sync-concurrency`,
`-ddl-concurrency`, `-interactive-concurrency`, `-interactive-burst`).

## Pass/fail thresholds (at 500 databases, steady + 50-session burst + sync)

| Dimension | Criterion |
|---|---|
| Interactive query p50/p95/p99 | p95 <= 100 ms, p99 <= 500 ms |
| Query error / timeout rate | 0 |
| Full sync (all DBs) | <= 5 min |
| Sync errors | 0 |
| Sustained / peak CPU | <= 60% / <= 80% |
| Connection utilization | <= 70% of max_connections |
| Rejected connections | 0 |
| Provisioning / deletion failures | 0, no orphaned DBs/roles |
| Memory / disk | stable, no unbounded growth |

CPU/memory/IOPS are captured from Cloud Monitoring in Phase 2; Phase 1 records
PostgreSQL-side statistics only.

## Running

Phase 1 (local smoke, needs Docker):

    go test ./backend/loadtest/ -run TestSampleInstanceLoadLocal -v -count=1

Phase 2 (Cloud SQL, needs GCP credentials):

    go run ./backend/cmd/sample-loadtest \
      -host <public-ip> -port 5432 -user postgres -password <secret> \
      -sslmode require -counts 70,500,1000 \
      -sync-concurrency 10 -ddl-concurrency 5 \
      -interactive-concurrency 10 -interactive-burst 50 \
      -report /tmp/loadtest-report.json
