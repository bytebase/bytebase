# High availability for Bytebase

This runbook describes the current operator-facing behavior for running Bytebase in a high-availability (HA) topology.

## Current support boundary

- Bytebase can detect and operate with multiple active replicas that share the same metadata database.
- HA is license-gated. The subscription API exposes this as the `ha` field on `Subscription`.
- The bundled Helm chart in `helm-charts/bytebase` currently renders a single-replica StatefulSet (`replicas: 1`). It does not provide a multi-replica deployment switch.

In other words: the runtime has HA awareness, but operators must still supply the multi-replica deployment mechanics outside the current Helm chart.

## Prerequisites

Before you keep more than one Bytebase server active at the same time, make sure all of the following are true:

1. **HA is enabled in the license.**
   - Check `GET /v1/subscription` and confirm `ha: true`.
2. **All replicas use the same metadata database.**
   - Active replicas are counted as Bytebase servers sharing the same database.
   - For Kubernetes operators, this means using an external PostgreSQL metadata database that every replica can reach.
3. **All replicas use the same external URL.**
   - Bytebase exposes a single external URL through actuator info and uses that address for user access and callbacks.
4. **Your platform provides traffic management and rollout control.**
   - For example, a load balancer plus a rolling update strategy managed by your orchestrator.

## How replica detection works

Bytebase tracks live replicas with heartbeats:

- Each replica writes a heartbeat immediately on startup and then every 10 seconds.
- A replica is considered active when it has sent a heartbeat within the last 30 seconds.
- The actuator API exposes the current active replica count as `replica_count` on `GET /v1/actuator/info`.
- Bytebase always reports at least one active replica for the current server, even if heartbeat counting fails.

## What happens when HA is not licensed

If more than one active replica is detected and the license does not enable HA, Bytebase does not permit the HA topology.

The current runtime behavior is to log warnings such as:

```text
multiple replicas detected (<count>) but HA is not enabled in license
```

When that condition is present, background runners that check the replica limit skip work instead of continuing in an unsupported topology. This includes scheduler and cleaner paths used for task execution, plan checks, schema sync, approvals, and stale-run cleanup.

## Operator validation checklist

Use this checklist when enabling or validating HA:

1. Call `GET /v1/subscription` and verify `ha` is `true`.
2. Call `GET /v1/actuator/info` and record:
   - `version`
   - `external_url`
   - `replica_count`
3. Confirm every replica points to the same metadata database and external URL.
4. After adding or restarting replicas, allow at least 30 seconds for the active replica count to settle.
5. Review logs for HA-license warnings before declaring the topology healthy.

## Troubleshooting

### `replica_count` stays at `1`

Check the following:

- The additional Bytebase server is actually running.
- The replica can reach the shared metadata database and write heartbeats.
- You waited long enough for the new replica to start and publish heartbeats.

### `replica_count` is lower than expected during a restart

A replica falls out of the active set after roughly 30 seconds without a heartbeat. A brief drop during restarts or node moves can therefore be expected.

### Logs show `multiple replicas detected ... but HA is not enabled in license`

This means the deployment topology and license do not match. Resolve it by doing one of the following:

- reduce the deployment back to a single active replica, or
- install a license where `GET /v1/subscription` returns `ha: true`.

### Old heartbeat rows exist in the database

Stale heartbeat rows are cleaned up separately and do not define the active replica count. Active counting only considers heartbeats from the last 30 seconds.

## Related docs

- [Upgrade guidance](./upgrade.md)
- [Helm chart README](../../helm-charts/bytebase/README.md)
