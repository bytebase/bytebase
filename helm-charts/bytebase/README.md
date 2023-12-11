# Helm Chart for Bytebase

[Bytebase](https://bytebase.com) is a Database CI/CD tool for DevOps teams, built for Developers and DBAs.

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
--set "bytebase.version"=2.11.1 \
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

## Parameters

|                        Parameter                         |                                                                                                                Description                                                                                                                 |                           Default Value                            |
| :------------------------------------------------------: | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: | :----------------------------------------------------------------: |
|                    `bytebase.version`                    |                                                                                                  The version of Bytebase to be installed.                                                                                                  |                              "2.11.1"                              |
|              `bytebase.registryMirrorHost`               |                                                                              The host for the Docker registry mirror. Leave empty for default registry usage.                                                                              |                                 ""                                 |
|                  `bytebase.option.port`                  |                                                                                                      Port where Bytebase server runs.                                                                                                      |                                8080                                |
|                  `bytebase.option.data`                  |                                                                                                  Data directory of Bytebase data stored.                                                                                                   |                         /var/opt/bytebase                          |
|              `bytebase.option.external-url`              |                                              The address for users to visit Bytebase, visit [our docs](https://www.bytebase.com/docs/get-started/install/external-url/) to get more details.                                               | "<https://www.bytebase.com/docs/get-started/install/external-url>" |
|             `bytebase.option.disable-sample`             |                                                                                                        Disable the sample instance.                                                                                                        |                               false                                |
|             `bytebase.option.externalPg.url`             |                                                                                        The PostgreSQL url(DSN) for Bytebase to store the metadata.                                                                                         |                                 ""                                 |
|     `bytebase.option.externalPg.existingPgURLSecret`     |                                                                           The name of Secret stores the PostgreSQL url(DSN) for Bytebase to store the metadata.                                                                            |                                 ""                                 |
|   `bytebase.option.externalPg.existingPgURLSecretKey`    |                                     The key of Secret stores the PostgreSQL url(DSN) for Bytebase to store the metadata. Should be used with `bytebase.option.externalPg.existingPgURLSecret` together.                                      |                                 ""                                 |
|           `bytebase.option.externalPg.pgHost`            |                                                                                             The PostgreSQL host for Bytebase metadata storage.                                                                                             |                               "host"                               |
|           `bytebase.option.externalPg.pgPort`            |                                                                                             The PostgreSQL port for Bytebase metadata storage.                                                                                             |                               "port"                               |
|         `bytebase.option.externalPg.pgUsername`          |                                                                                           The PostgreSQL username for Bytebase metadata storage.                                                                                           |                             "username"                             |
|         `bytebase.option.externalPg.pgPassword`          |                                                                                           The PostgreSQL password for Bytebase metadata storage.                                                                                           |                             "password"                             |
|         `bytebase.option.externalPg.pgDatabase`          |                                                                                             The name of the PostgreSQL database for Bytebase.                                                                                              |                             "database"                             |
|  `bytebase.option.externalPg.existingPgPasswordSecret`   |                                                                       The name of Secret that stores the existing PostgreSQL password for Bytebase metadata storage.                                                                       |                                 ""                                 |
| `bytebase.option.externalPg.existingPgPasswordSecretKey` |                                                    The key of Secret storing the existing PostgreSQL password. Should be used with `bytebase.option.externalPg.existingPgPasswordSecret`.                                                    |                                 ""                                 |
|       `bytebase.option.externalPg.escapePassword`        | Controls whether to escape the password in the connection string. `bytebase.option.externalPg.existingPgPasswordSecret` or `bytebase.option.externalPg.pgPassword` should be specified with this value together. **Experimental feature.** |                               false                                |
|              `bytebase.persistence.enabled`              |                                                                                                  Enable/disable persistence for Bytebase.                                                                                                  |                               false                                |
|           `bytebase.persistence.existingClaim`           |                                                                                    Name of the existing PersistentVolumeClaim for Bytebase persistence.                                                                                    |                                 ""                                 |
|              `bytebase.persistence.storage`              |                                                                                              Size of the persistent volume for Bytebase data.                                                                                              |                               "2Gi"                                |
|           `bytebase.persistence.storageClass`            |                                                                                         Storage class for the persistent volume used by Bytebase.                                                                                          |                                 ""                                 |
|               `bytebase.extraSecretMounts`               |                                                                               Additional Bytebase secret mounts. Defined as an array of volumeMount objects.                                                                               |                                 []                                 |
|                 `bytebase.extraVolumes`                  |                                                                                    Additional Bytebase volumes. Defined as an array of volume objects.                                                                                     |                                 []                                 |

**If you enable bytebase persistence, you should provide storageClass and storage to bytebase to request a PVC, or provide the already existed PVC by existingClaim.**

## Need Help?

- Contact <support@bytebase.com>
- [Bytebase Docs](https://bytebase.com/docs)
- [Bytebase GitHub Issue Page](https://github.com/bytebase/bytebase/issues/new/choose)
