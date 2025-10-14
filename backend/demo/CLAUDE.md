# Demo Data Automation Guide

This directory contains demo data for https://demo.bytebase.com.

`dump.sql` is dumped from the meta database and holds schema and data for the demo.

## Updating Demo Data

### Automated Steps

**Step 1: Start Postgres 16 in Docker**

```bash
docker run -d --name bytebase-demo-pg -e POSTGRES_USER=bbdev -e POSTGRES_DB=bbdev -e POSTGRES_HOST_AUTH_METHOD=trust -p 5433:5432 postgres:16-alpine
```

**Step 2: Build and start Bytebase backend in demo mode**

```bash
go build -p=16 -ldflags "-w -s" -o ./bytebase-build/bytebase ./backend/bin/server/main.go && PG_URL=postgresql://bbdev@localhost:5433/bbdev ./bytebase-build/bytebase --port 8080 --data . --debug --demo
```

In a separate terminal:
```bash
pnpm --dir frontend dev
```

---

### Manual Step Required

**Step 3: Make changes through the UI**

1. Navigate to http://localhost:3000
2. Make the desired changes to the demo data through the UI

---

### Automated Steps (continued)

**Step 4: Dump the database using Docker**

```bash
docker exec bytebase-demo-pg pg_dump -U bbdev bbdev --disable-triggers --no-owner --column-inserts --on-conflict-do-nothing > backend/demo/data/dump.sql
```

Remove the restrict/unrestrict lines from the dump:

```bash
sed -i '' '/^\\restrict /d; /^\\unrestrict /d' backend/demo/data/dump.sql
```

Verify the dump versions match at the top of `backend/demo/data/dump.sql`:
```sql
-- Dumped from database version 16.0
-- Dumped by pg_dump version 16.0
```

**Step 5: Test the updated dump**

Stop the running Bytebase instance from Step 2, clean the Docker database, and restart:

```bash
# Clean the Docker Postgres database
docker exec bytebase-demo-pg psql -U bbdev -d postgres -c "DROP DATABASE IF EXISTS bbdev;" -c "CREATE DATABASE bbdev;"

# Restart Bytebase with the updated dump
PG_URL=postgresql://bbdev@localhost:5433/bbdev ./bytebase-build/bytebase --port 8080 --data . --debug --demo
```

Verify the demo data loads correctly by checking the frontend at http://localhost:3000.

**Step 6: Cleanup**

Stop the Bytebase backend and frontend servers, then remove the Docker container:

```bash
# Stop backend server on port 8080
lsof -ti:8080 | xargs kill -9 2>/dev/null || true

# Stop frontend server on port 3000
lsof -ti:3000 | xargs kill -9 2>/dev/null || true

# Remove Docker container
docker stop bytebase-demo-pg && docker rm bytebase-demo-pg
```

## Service Account

Demo data service account credentials:
- Email: `api@service.bytebase.com`
- Password: `bbs_EDyd8zleJVBEZyw81kLL`

## Claude Code Usage

To automate the demo data update using Claude Code:

**Initial prompt (Steps 1-2):**
```
Update the demo data following backend/demo/CLAUDE.md
```

**After completing manual UI changes in Step 3:**
```
I've finished making changes through the UI. Continue with Steps 4-6.
```
