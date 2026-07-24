# Use row leases for leader coordination

Bytebase will coordinate singleton background responsibilities with PostgreSQL rows instead of session advisory locks because PgBouncer transaction pooling does not preserve session affinity. The `leader` component will use short acquire, renew, and release statements against `leader_lease`, while callbacks run outside transactions; schema sync is the first Leader Type and retains its existing per-pass coordination boundary.

## Consequences

- V1 stores one row per Leader Type with the last replica ID, a monotonic generation, and an expiration evaluated with PostgreSQL `clock_timestamp()`, never replica wall clocks. A future Leader Resource column requires the staged migration in ADR-0002.
- Expired and released rows persist, retaining the last holder and generation; a later acquisition advances the generation.
- A successful lease is valid in PostgreSQL for 30 seconds, renews every 10 seconds, and is treated as locally invalid 10 seconds before database expiry.
- Generation protects renewal and release inside the component. V1 does not expose terms for side-effect fencing, add Prometheus metrics, or terminate replicas whose callbacks fail to stop.
- Schema sync uses the one-shot `TryRun` callback API, preserving current contention, cancellation, and best-effort side-effect semantics.
- Lease coordination has no heartbeat foreign key or lifecycle coupling.
- The obsolete session advisory-lock API is removed; transaction-scoped advisory locks remain available for short transactions and schema migration.
