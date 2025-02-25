# Overview

This is the demo data for https://demo.bytebase.com.

`dump.sql` itself is dumped from the meta database and holds schema and data for the demo.

# How to update demo data

1. Build and start Bytebase backend with `--demo default` on main branch.

1. Dump with the following command.

```bash
pg_dump -h /tmp -p 8082 -U bb --disable-triggers --no-owner --column-inserts --on-conflict-do-nothing > ~/dump.sql
```

1. Copy and replace `dump.sql`.

# Users

Demo data service account: ci@service.bytebase.com password: bbs_iqysPHMqhNpG4rQ5SFEJ
