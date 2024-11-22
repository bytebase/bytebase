# Overview

This is the demo data for https://demo.bytebase.com.

`dump.sql` itself is dumped from the meta database and holds schema and data for the demo.

# Dependencies

1. Sample test and prod PG instances running on port `8083` and `8084`.
1. [GitHub hr-sample](https://github.com/s-bytebase/hr-sample) to demonstrate GitOps Workflow.
1. Enterprise license. https://demo.bytebase.com runs in dev mode, thus it bundles a dev license. If
   you want to run the demo in release mode. You need to supply your own release license.

# How to use

Append `--demo default` to the start command.

Demo only works when using embedded PostgreSQL (without supplying [--pg](https://www.bytebase.com/docs/reference/command-line/#pg-string)). This is to prevent accidentally deleting the existing data.

Demo requires the sample test and prod instances run on port `8083` and `8084` respectively. So we need to
supply the --port with `8080` to make sample instances run on those 2 desired ports.

```bash
docker run --init \
  --name bytebase \
  --pull always \
  --restart always \
  --publish 8080:8080 \
  --health-cmd "curl --fail http://localhost:9015/healthz || exit 1" \
  --health-interval 5m \
  --health-timeout 60s \
  --volume ~/.bytebase/data:/var/opt/bytebase \
  bytebase/bytebase:3.1.0 \
  --data /var/opt/bytebase \
  --port 8080 \
  --demo default
```

## Run on render

1. Set the [PORT env](https://render.com/docs/environment-variables#all-services-1) to 8080.
1. Use [/scripts/Dockerfile.render-demo](https://github.com/bytebase/bytebase/blob/main/scripts/Dockerfile.render-demo) as the Dockerfile.
1. Supply `bytebase --port 8080 --data /var/opt/bytebase --demo default` to the Docker Command.

# How to update demo data

1. Demo data is using the dev build because our demo runs in dev mode.

1. Start Bytebase with `--demo default`, and do whatever you want.

1. Dump with the following command.

```bash
pg_dump --username bbdev --host /tmp --port 8082 --disable-triggers --column-inserts --on-conflict-do-nothing bbdev > /tmp/dump.sql
```

3. Copy and replace `dump.sql`.
