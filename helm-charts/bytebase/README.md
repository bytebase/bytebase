# Helm Chart for Bytebase

[Bytebase](https://bytebase.com) is a Database CI/CD tool for DevOps teams, built for Developers and DBAs.

## TL;DR

```bash
$ helm repo add bytebase-repo https://bytebase.github.io/bytebase
$ helm repo update
$ helm -n <YOUR_NAMESPACE> \
--set "bytebase.option.port"={PORT} \
--set "bytebase.option.pg"={PGDSN} \
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
--set "bytebase.option.pg"={PGDSN} \
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
--set "bytebase.option.pg"="postgresql://bytebase:bytebase@database.bytebase.ap-east-1.rds.amazonaws.com/bytebase" \
--set "bytebase.option.external-url"="https://bytebase.ngrok-free.app" \
--set "bytebase.version"=2.5.0 \
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
--set "bytebase.option.pg"={NEW_PGDSN} \
--set "bytebase.version"={NEW_VERSION} \
--set "bytebase.option.external-url"={EXTERNAL_URL} \
--set "bytebase.persistence.enabled"={TRUE/FALSE} \
--set "bytebase.persistence.storage"={STORAGE_SIZE} \
--set "bytebase.persistence.storageClass"={STORAGE_CLASS} \
upgrade bytebase-release bytebase-repo/bytebase
```

## Parameters

|               Parameter                |                                                                  Description                                                                   |                                       Default Value                                       |
| :------------------------------------: | :--------------------------------------------------------------------------------------------------------------------------------------------: | :---------------------------------------------------------------------------------------: |
|          bytebase.option.port          |                                                        Port where Bytebase server runs.                                                        |                                           8080                                            |
|           bytebase.option.pg           |   External PostgreSQL instance connection url(must provide dbname).It will be ignored if you specify `bytebase.option.existingPgURLSecret`.    | "postgresql://bytebase:<bytebase@database.bytebase.ap-east-1.rds.amazonaws.com>/bytebase" |
|      bytebase.option.external-url      | The address for users to visit Bytebase, visit [our docs](https://www.bytebase.com/docs/get-started/install/external-url/) to get more details |            "<https://www.bytebase.com/docs/get-started/install/external-url>"             |
|  bytebase.option.existingPgURLSecret   |                                          Existing secret with external PostgreSQL connection string.                                           |                                            ""                                             |
| bytebase.option.existingPgURLSecretKey |      Existing secret key with external PostgreSQL connection(must specfied if you specify `bytebase.option.existingPgURLSecret`) string.       |                                            ""                                             |
|            bytebase.version            |                                                             The Bytebase version.                                                              |                                          "2.5.0"                                          |
|      bytebase.persistence.enabled      |                                                         Persist bytebase data switch.                                                          |                                           false                                           |
|   bytebase.persistence.storageClass    |                                                    The storage class used by Bytebase PVC.                                                     |                                            ""                                             |
|      bytebase.persistence.storage      |                                                     The storage size of Bytebase PVC used.                                                     |                                           "2Gi"                                           |
|   bytebase.persistence.existingClaim   |                                                  The existing PVC that bytebase need to use.                                                   |                                            ""                                             |
|      bytebase.registryMirrorHost       |                                                                       ""                                                                       |      The host of the registry mirror used by downloading bytebase container images.       |
|     bytebase.option.disable-sample     |                                                          Disable the sample instance.                                                          |                                           false                                           |

**If you enable bytebase persistence, you should provide storageClass and storage to bytebase to request a PVC, or provide the already existed PVC by existingClaim.**

## Need Help?

- Contact <support@bytebase.com>
- [Bytebase Docs](https://bytebase.com/docs)
- [Bytebase GitHub Issue Page](https://github.com/bytebase/bytebase/issues/new/choose)
