# Overview

This is the demo data for https://demo.bytebase.com.

`dump.sql` itself is dumped from the meta database and holds schema and data for the demo.

# How to update demo data

1. We need to use initdb and pg_dump. Please visit https://github.com/bytebase/bytebase/tree/release/3.5.3/backend/resources/postgres and download the relevant pacakge (e.g for Mac, you will download `postgres-darwin-arm64.txz`). Then extract and move it to `$PWD/bytebase-build/postgres`.

1. Build and start Bytebase backend with `--demo` on a release branch (current demo data is based on release/3.10.0).

   ```bash
   # Make sure to use the local pg binary
   go build -p=16 -ldflags "-w -s" -o ./bytebase-build/bytebase ./backend/bin/server/main.go && PATH="$PWD/bytebase-build/postgres/bin:$PATH" ./bytebase-build/bytebase --port 8080 --data . --debug --demo
   ```

1. Start frontend and make changes.

1. Dump with the following command.

   ```bash
   # Make sure to use the local pg binary
   PATH="$PWD/bytebase-build/postgres/bin:$PATH" pg_dump -h /tmp -p 8082 -U bb --disable-triggers --no-owner --column-inserts --on-conflict-do-nothing > ~/dump.sql
   ```

   On the top of the dump.sql, the version should be consistent

   ```sql
   -- Dumped from database version 16.0
   -- Dumped by pg_dump version 16.0
   ```

1. Copy and replace `dump.sql`.

# Users

Demo data service account: api@service.bytebase.com password: bbs_EDyd8zleJVBEZyw81kLL
