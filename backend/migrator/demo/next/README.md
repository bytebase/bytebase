# Overview

*Work In Progress*

This is the next version of demo data.

`demo.sql` holds the data for demo.

`demo.sql` itself is dumped from the meta database.

# How to use

Append `--demo next` to the start command.

Beware that this will overwrite your data, so it's recommended to use a fresh database.

## Example

```bash
docker run --init \
  --name bytebase \
  --restart always \
  --publish 9015:9015 \
  --health-cmd "curl --fail http://localhost:9015/healthz || exit 1" \
  --health-interval 5m \
  --health-timeout 60s \
  --volume ~/.bytebase/data:/var/opt/bytebase \
  bytebase/bytebase:2.8.0 \
  --data /var/opt/bytebase \
  --port 9015 \
  --demo next \
  --pg postgresql://user:secret@host:port/dbname
```



# How to update demo data

1. Start Bytebase with `--demo next`, and do whatever you want.

2. Dump with the following command.

```bash
pg_dump --username ${user} --host ${host} --port ${port} --password ${secret} --data-only --disable-triggers --column-inserts --on-conflict-do-nothing ${dbname} > /tmp/demo.sql
```

3. Copy and replace `demo.sql`.
