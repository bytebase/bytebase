# Helm Chart for Bytebase

[Bytebase](https://bytebase.com) is a Database CI/CD tool for DevOps teams, built for Developers and DBAs.

## TL;DR

```bash
$ helm repo add bytebase-repo https://bytebase.github.io/bytebase
$ helm repo update
$ helm -n <YOUR_NAMESPACE> \
--set "bytebase.option.port"={PORT} \
--set "bytebase.option.external-url"={EXTERNAL_URL} \
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
--set "bytebase.option.external-url"={EXTERNAL_URL} \
--set "bytebase.option.pg"={PGDSN} \
--set "bytebase.version"={VERSION} \
install <RELEASE_NAME> bytebase-repo/bytebase
```

For example:

```bash
$ helm -n bytebase \
--set "bytebase.option.port"=443 \
--set "bytebase.option.external-url"="https://bytebase.com" \
--set "bytebase.option.pg"="postgresql://bytebase:bytebase@database.bytebase.ap-east-1.rds.amazonaws.com/bytebase" \
--set "bytebase.version"=1.7.0 \
install bytebase-release bytebase-repo/bytebase
```

## Uninstalling the Chart

```bash
$ helm delete --namespace <YOUR_NAMESPACE> <RELEASE_NAME>
```

## Upgrade Bytebase Version/Configuration

Use `helm upgrade` command to upgrade the bytebase version or configuration.

```bash
helm -n <YOUR_NAMESPACE> \
--set "bytebase.option.port"={NEW_PORT} \
--set "bytebase.option.external-url"={NEW_EXTERNAL_URL} \
--set "bytebase.option.pg"={NEW_PGDSN} \
--set "bytebase.version"={NEW_VERSION} \
upgrade bytebase-release bytebase-repo/bytebase
```

## Need Help?

- Contact support@bytebase.com
- [Bytebase Docs](https://bytebase.com/docs)
- [Bytebase GitHub Issue Page](https://github.com/bytebase/bytebase/issues/new/choose)
