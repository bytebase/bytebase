# Product metrics

Bytebase exposes product-health metrics from the server's `/metrics` endpoint.
They complement Go runtime and HTTP metrics with installation state and
process-local background-work health.

## Installation-state metrics

Installation-state metrics are read from shared metadata on every scrape so all
replicas report equivalent values. If the metadata or configured license cannot
be read and verified, the scrape fails instead of returning a partial state.

| Metric | Type | Description |
| --- | --- | --- |
| `bytebase_license_expiry_timestamp_seconds` | Gauge | License expiry as a Unix timestamp. `+Inf` means the Free plan or a perpetual license. A valid expired license retains its expiry timestamp while using Free-plan limits. |
| `bytebase_license_seats_used` | Gauge | Distinct end users currently occupying license seats. |
| `bytebase_license_seats_limit` | Gauge | Effective seat limit. `+Inf` means unlimited. |
| `bytebase_license_instances_used` | Gauge | Non-deleted registered instances. |
| `bytebase_license_instances_limit` | Gauge | Effective registered-instance limit. `+Inf` means unlimited. |

## Process-local metrics

Process metrics record work performed by the serving Bytebase process. In a
multi-replica deployment, query each replica or aggregate across replicas as
appropriate. Canceled shutdown work is not recorded as a failure.

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `bytebase_instance_sync_duration_seconds` | Native histogram | `result`, `bytebase_instance_id`, resource labels | Duration of completed instance synchronization attempts. |
| `bytebase_database_sync_total` | Counter | `result`, `bytebase_instance_id`, `database`, resource labels | Completed database schema synchronization attempts. |
| `bytebase_runner_run_duration_seconds` | Native histogram | `runner`, `result` | Duration of synchronous background runner cycles. |

`result` is `success`, `failure`, or `skipped`. Runner values are
`plan_check`, `task_pending`, `task_dispatch`, `instance_sync`, and
`database_sync`.

Resource-label names are exposed with a `label_` prefix. Characters outside
ASCII letters, digits, and underscores are replaced with underscores, and name
collisions receive deterministic `_conflictN` suffixes. The label schema is the
union of labels observed by that process; a resource without a union label emits
an empty value for it.

The duration metrics use the Prometheus client library's native histogram
implementation. Their bucket factor is at most 1.1, populated buckets are capped
at 100, and bucket state is not reset more often than once per hour.
