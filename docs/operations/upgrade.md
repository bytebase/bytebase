# Upgrade guidance for HA-aware Bytebase deployments

This runbook focuses on upgrade-time checks for environments that may run Bytebase with more than one active replica.

## Scope

- Covers operator checks before, during, and after an upgrade.
- Applies to the current runtime behavior in this repository.
- Does **not** introduce a new deployment mechanism.

The bundled Helm chart still deploys a single replica by default, so HA rollouts remain operator-managed outside the chart.

## Before you upgrade

1. **Confirm the current topology.**
   - Call `GET /v1/actuator/info` and capture `version`, `externalUrl`, and `replicaCount`.
2. **Confirm whether HA is licensed.**
   - Call `GET /v1/subscription` and verify the `ha` field.
3. **Protect the metadata database.**
   - Take the backup or snapshot your standard operating procedure requires before upgrading Bytebase.
4. **Confirm shared configuration is consistent.**
   - Every HA replica should start with `--ha`, use the same `PG_URL`, and keep the same external URL during the upgrade.

## Helm chart behavior during upgrades

The chart README uses `helm upgrade` for version and configuration changes, and that is still the correct command.

However, the chart currently renders a single-replica StatefulSet. Upgrading the chart does not enable an HA topology and does not add a chart value for running multiple Bytebase application replicas.

If you operate multiple Bytebase servers today, treat that topology as operator-managed and keep the upgrade procedure for those extra replicas in your own platform tooling.

## Multi-replica upgrade guidance

If you operate multiple Bytebase replicas outside the bundled chart:

1. **Only keep multiple active replicas if HA is licensed.**
   - Without `ha: true`, Bytebase logs HA restriction warnings and background runners skip work when more than one replica is active.
2. **Keep HA startup arguments consistent across replicas.**
   - Start each replica with `--ha` and the same shared external PostgreSQL `PG_URL` before beginning the rollout.
3. **Prefer a rolling replacement instead of a full stop/start.**
   - Replace replicas gradually with your orchestrator so at least one healthy replica remains available.
4. **Wait for replica health to settle between steps.**
   - After each rollout step, check `GET /v1/actuator/info` and confirm `replicaCount` matches your expectation.
5. **Watch logs during the rollout.**
   - Investigate any HA-license warnings before continuing to the next replica.

## Single-replica upgrade guidance

If your environment uses only the bundled Helm chart with its default topology:

1. Run `helm upgrade` with the new Bytebase version or configuration.
2. Wait for the single StatefulSet replica to become healthy.
3. Verify the updated version via `GET /v1/actuator/info`.

## After the upgrade

Validate the following before closing the change:

1. `GET /v1/actuator/info` returns the expected `version`.
2. `GET /v1/actuator/info` returns the expected `replicaCount`.
3. `GET /v1/subscription` still shows the expected `ha` value.
4. Logs do not show `multiple replicas detected ... but HA is not enabled in license`.
5. Background operations resume normally after the rollout window.

## When to stop the rollout

Pause the upgrade and investigate if any of the following occur:

- `replicaCount` does not recover to the expected value.
- Replicas disagree on the external URL or metadata database configuration.
- HA-license warnings appear in a topology that is supposed to stay multi-replica.

## Related docs

- [High availability runbook](./high-availability.md)
- [Helm chart README](../../helm-charts/bytebase/README.md)
