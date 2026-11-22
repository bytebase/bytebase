# Helm Chart for Bytebase

[Bytebase](https://www.bytebase.com) is a Database CI/CD tool for DevOps teams, built for Developers and DBAs.

## TL;DR

```bash
$ helm repo add bytebase-repo https://bytebase.github.io/bytebase
$ helm repo update
$ helm -n <YOUR_NAMESPACE> \
--set "bytebase.option.port"={PORT} \
--set "bytebase.option.externalPg.url"={PGDSN} \
--set "bytebase.version"={VERSION} \
install <RELEASE_NAME> bytebase-repo/bytebase
```

## Prerequisites

- Kubernetes 1.24+
- Helm 3.9.0+

## Pinning container images by digest

Set `bytebase.version` to a combined `tag@digest` reference. Kubernetes pulls by digest and treats the tag as documentation.

```bash
helm -n <YOUR_NAMESPACE> \
--set-string "bytebase.version"="{VERSION}@sha256:{DIGEST}" \
install <RELEASE_NAME> bytebase-repo/bytebase
```

The BusyBox init container used when `bytebase.option.externalPg.escapePassword` is enabled defaults to a fixed tag via `bytebase.busyboxVersion`, which also accepts a combined `tag@digest` reference. Azure Marketplace deployments can use the same combined reference in `global.azure.images.bytebase.tag`.

When upgrading from chart 1.1.4, move any digest configured through `bytebase.digest`, `bytebase.busyboxDigest`, or `global.azure.images.bytebase.digest` into the corresponding version or tag value. The chart rejects these removed settings to prevent an upgrade from silently losing digest pinning.

## High availability note

The bundled Helm chart currently deploys Bytebase as a single-replica StatefulSet. It does not expose a Helm value for running multiple Bytebase application replicas.

If you are preparing an HA deployment, treat the multi-replica topology as operator-managed outside this chart. Each HA replica must still start with `--ha` and the same shared external PostgreSQL `PG_URL`. Set `bytebase.option.ha=true` so the replica this chart manages starts with the flag too; review the operator runbook before proceeding:

- [High availability runbook](../../docs/operations/high-availability.md)
- [Upgrade guidance](../../docs/operations/upgrade.md)

## Installing the Chart

```bash
$ helm -n <YOUR_NAMESPACE> \
--set "bytebase.option.port"={PORT} \
--set "bytebase.option.externalPg.url"={PGDSN} \
--set "bytebase.version"={VERSION} \
--set "bytebase.option.external-url"={EXTERNAL_URL} \
--set "bytebase.persistence.enabled"={TRUE/FALSE} \
--set "bytebase.persistence.storage"={STORAGE_SIZE} \
--set "bytebase.persistence.storageClass"={STORAGE_CLASS} \
install <RELEASE_NAME> bytebase-repo/bytebase
```

For example:

```bash
$ helm -n bytebase \
--set "bytebase.option.port"=443 \
--set "bytebase.option.externalPg.url"="postgresql://bytebase:bytebase@database.bytebase.ap-east-1.rds.amazonaws.com/bytebase" \
--set "bytebase.option.external-url"="https://bytebase.ngrok-free.app" \
--set "bytebase.version"=3.20.1 \
--set "bytebase.persistence.enabled"="true" \
--set "bytebase.persistence.storage"="10Gi" \
--set "bytebase.persistence.storageClass"="csi-disk" \
install bytebase-release bytebase-repo/bytebase
```

## Uninstalling the Chart

```bash
helm delete --namespace <YOUR_NAMESPACE> <RELEASE_NAME>
```

## Upgrade Bytebase Version/Configuration

Use `helm upgrade` command to upgrade the bytebase version or configuration.

For HA-aware operations, note that this chart still renders `replicas: 1`. A `helm upgrade` does not enable a multi-replica topology by itself. If you operate multiple Bytebase servers outside this chart, keep `--ha` (`bytebase.option.ha` for the replica this chart manages) and the shared external PostgreSQL `PG_URL` consistent on every replica and follow the operator runbook and upgrade guidance here:

- [High availability runbook](../../docs/operations/high-availability.md)
- [Upgrade guidance](../../docs/operations/upgrade.md)

```bash
helm -n <YOUR_NAMESPACE> \
--set "bytebase.option.port"={NEW_PORT} \
--set "bytebase.option.externalPg.url"={NEW_PGDSN} \
--set "bytebase.version"={NEW_VERSION} \
--set "bytebase.option.external-url"={EXTERNAL_URL} \
--set "bytebase.persistence.enabled"={TRUE/FALSE} \
--set "bytebase.persistence.storage"={STORAGE_SIZE} \
--set "bytebase.persistence.storageClass"={STORAGE_CLASS} \
upgrade bytebase-release bytebase-repo/bytebase
```

Bytebase migrates its metadata database at startup and opens its port only after the migration finishes, which can take a long time on a large metadata database. The chart sets only a readiness probe on `/healthz`, so the pod stays unready during the migration instead of being restarted. If you pass `--wait` or `--atomic` to `helm upgrade`, set `--timeout` above the expected migration time. Helm's default is `5m`, and `--atomic` rolls the release back to the previous Bytebase version when it expires, which does not support the migrated metadata database.

## Scraping /metrics

Since Bytebase 3.22.0, the Prometheus endpoint at `/metrics` answers loopback clients only. An in-cluster scraper gets a 404 until the server starts with `--metrics-remote-access`:

```bash
helm -n <YOUR_NAMESPACE> \
--set "bytebase.option.metrics-remote-access"=true \
upgrade <RELEASE_NAME> bytebase-repo/bytebase
```

`bytebase.podAnnotations` makes the pod discoverable, and needs this value set as well. Keep `prometheus.io/port` equal to `bytebase.option.port`:

```yaml
bytebase:
  option:
    metrics-remote-access: true
  podAnnotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
```

`/metrics` carries no authentication of its own. Enable it only where the port is reachable from the scraper alone, through a gateway path allowlist, a firewall, or a NetworkPolicy. An Ingress or HTTPRoute that matches `/` publishes `/metrics` to everything that reaches the route, along with the rest of the app.

## Parameters

|                        Parameter                         |                                                                                                                Description                                                                                                                 |                         Default Value                          |
| :------------------------------------------------------: | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: | :------------------------------------------------------------: |
|                    `bytebase.version`                    |                                                  The version of Bytebase to be installed. May embed a digest as a combined reference, for example `3.21.0@sha256:abc123`.                                                  |                            "latest"                            |
|              `bytebase.registryMirrorHost`               |                                                                              The host for the Docker registry mirror. Leave empty for default registry usage.                                                                              |                               ""                               |
|                `bytebase.busyboxVersion`                 |                                                  Tag of the BusyBox init container used when PostgreSQL password escaping is enabled. May embed a digest, for example `1.37.0@sha256:abc123`.                                                  |                            "1.37.0"                            |
|                  `bytebase.option.port`                  |                                                                                                      Port where Bytebase server runs.                                                                                                      |                              8080                              |
|                  `bytebase.option.data`                  |                                                                                                  Data directory of Bytebase data stored.                                                                                                   |                       /var/opt/bytebase                        |
|              `bytebase.option.external-url`              |                                                The address for users to visit Bytebase, visit [our docs](https://docs.bytebase.com/get-started/self-host/external-url/) to get more details.                                                 | "<https://docs.bytebase.com/get-started/self-host/external-url>" |
|          `bytebase.option.enable-json-logging`           |                                                                      Output Bytebase logs in JSON format, for log aggregators such as Splunk, ELK, Datadog, and Loki.                                                                      |                             false                              |
|                   `bytebase.option.ha`                   |                                           Start the Bytebase server with `--ha`. Required on every replica once more than one server shares an external PostgreSQL; this chart still renders `replicas: 1`.                                           |                             false                              |
|       `bytebase.option.metrics-remote-access`            |                                                       Allow non-loopback clients to scrape `/metrics`, which answers loopback only by default. See [Scraping /metrics](#scraping-metrics).                                                       |                             false                              |
|             `bytebase.option.externalPg.url`             |                                                                                        The PostgreSQL url(DSN) for Bytebase to store the metadata.                                                                                         |                               ""                               |
|     `bytebase.option.externalPg.existingPgURLSecret`     |                                                                           The name of Secret stores the PostgreSQL url(DSN) for Bytebase to store the metadata.                                                                            |                               ""                               |
|   `bytebase.option.externalPg.existingPgURLSecretKey`    |                                    The key of Secret stores the PostgreSQL url(DSN) for Bytebase to store the metadata. Should be used with `bytebase.option.externalPg.existingPgURLSecret` together.                                     |                               ""                               |
|           `bytebase.option.externalPg.pgHost`            |                                                                                             The PostgreSQL host for Bytebase metadata storage.                                                                                             |                             "host"                             |
|           `bytebase.option.externalPg.pgPort`            |                                                                                             The PostgreSQL port for Bytebase metadata storage.                                                                                             |                             "port"                             |
|         `bytebase.option.externalPg.pgUsername`          |                                                                                           The PostgreSQL username for Bytebase metadata storage.                                                                                           |                           "username"                           |
|         `bytebase.option.externalPg.pgPassword`          |                                                                                           The PostgreSQL password for Bytebase metadata storage.                                                                                           |                           "password"                           |
|         `bytebase.option.externalPg.pgDatabase`          |                                                                                             The name of the PostgreSQL database for Bytebase.                                                                                              |                           "database"                           |
|  `bytebase.option.externalPg.existingPgPasswordSecret`   |                                                                       The name of Secret that stores the existing PostgreSQL password for Bytebase metadata storage.                                                                       |                               ""                               |
| `bytebase.option.externalPg.existingPgPasswordSecretKey` |                                                   The key of Secret storing the existing PostgreSQL password. Should be used with `bytebase.option.externalPg.existingPgPasswordSecret`.                                                   |                               ""                               |
|       `bytebase.option.externalPg.escapePassword`        | Controls whether to escape the password in the connection string. `bytebase.option.externalPg.existingPgPasswordSecret` or `bytebase.option.externalPg.pgPassword` should be specified with this value together. **Experimental feature.** |                             false                              |
|    `bytebase.option.externalPg.awsRdsIam.enabled`        |                                                        Enables AWS RDS IAM authentication for the Bytebase metadata PostgreSQL database when the chart constructs PG_URL from externalPg fields, including `sslmode=verify-full`.                                                        |                             false                              |
|     `bytebase.option.externalPg.awsRdsIam.region`        |                                                        AWS region used to sign RDS IAM authentication tokens for the Bytebase metadata PostgreSQL database. Required when awsRdsIam.enabled is true.                                                        |                               ""                               |
|   `bytebase.option.externalPg.gcpCloudSqlIam.enabled`    |                                                        Enables GCP Cloud SQL IAM authentication for the Bytebase metadata PostgreSQL database when the chart constructs PG_URL from externalPg fields.                                                        |                             false                              |
| `bytebase.option.externalPg.gcpCloudSqlIam.instanceConnectionName` |                                                        GCP Cloud SQL instance connection name (`project:region:instance`) for the Bytebase metadata PostgreSQL database. Required when gcpCloudSqlIam.enabled is true.                                                        |                               ""                               |
|              `bytebase.persistence.enabled`              |                                                                                                  Enable/disable persistence for Bytebase.                                                                                                  |                             false                              |
|           `bytebase.persistence.existingClaim`           |                                                                                    Name of the existing PersistentVolumeClaim for Bytebase persistence.                                                                                    |                               ""                               |
|              `bytebase.persistence.storage`              |                                                                                              Size of the persistent volume for Bytebase data.                                                                                              |                             "2Gi"                              |
|           `bytebase.persistence.storageClass`            |                                                                                         Storage class for the persistent volume used by Bytebase.                                                                                          |                               ""                               |
|               `bytebase.extraSecretMounts`               |                                                                               Additional Bytebase secret mounts. Defined as an array of volumeMount objects.                                                                               |                               []                               |
|                 `bytebase.extraVolumes`                  |                                                                                    Additional Bytebase volumes. Defined as an array of volume objects.                                                                                     |                               []                               |
|                 `bytebase.podAnnotations`                 |                                                                                       Annotations added to the Bytebase pod. Useful for metrics scrapers.                                                                                       |                               {}                               |
|        `bytebase.readinessProbe.periodSeconds`           |                                                                 How often the readiness probe calls `/healthz`. Other probe timing fields such as `initialDelaySeconds` and `successThreshold` are also accepted.                                                                 |                               10                               |
|        `bytebase.readinessProbe.timeoutSeconds`          |                                                                                      Seconds before a single `/healthz` call times out.                                                                                       |                               10                               |
|       `bytebase.readinessProbe.failureThreshold`         |                                                                          Consecutive failed calls before the pod is marked unready and removed from the Service.                                                                          |                               3                                |

**If you enable bytebase persistence, you should provide storageClass and storage to bytebase to request a PVC, or provide the already existed PVC by existingClaim.**

## Need Help?

- Contact <support@bytebase.com>
- [Bytebase Docs](https://docs.bytebase.com)
- [High availability runbook](../../docs/operations/high-availability.md)
- [Upgrade guidance](../../docs/operations/upgrade.md)
- [Bytebase GitHub Issue Page](https://github.com/bytebase/bytebase/issues/new/choose)
