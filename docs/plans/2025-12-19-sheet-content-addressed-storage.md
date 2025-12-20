# Sheet Content-Addressed Storage Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate sheet storage from ID-based to SHA256-based content addressing, pointing store.SheetMessage directly to sheet_blob table.

**Architecture:** Replace sheet table with direct sheet_blob references. Use SHA256 hash (hex string) as identifier throughout the codebase. Update all foreign keys and proto fields from int32 to string (hex sha256). Create migration to backfill existing data.

**Tech Stack:** Go, Protocol Buffers, PostgreSQL, Bytebase store layer

---

## Task 1: Update Proto Definitions

**Files:**
- Modify: `proto/store/store/task.proto:31`

**Step 1: Update Task proto to use sheet_sha256**

Edit `proto/store/store/task.proto`:

```protobuf
// Change line 31 from:
// int32 sheet_id = 4;

// To:
// The SHA256 hash of the sheet content (hex-encoded).
string sheet_sha256 = 4;
```

**Step 2: Format proto file**

Run: `buf format -w proto`
Expected: File formatted successfully

**Step 3: Lint proto file**

Run: `buf lint proto`
Expected: No lint errors

**Step 4: Generate Go code from proto**

Run: `cd proto && buf generate`
Expected: Generated files updated in backend/generated-go/

**Step 5: Commit proto changes**

```bash
git add proto/store/store/task.proto backend/generated-go/
git commit -m "feat: change Task.sheet_id to sheet_sha256 string

Update proto definition to use SHA256 hash instead of integer ID
for sheet references.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Update SheetMessage Struct

**Files:**
- Modify: `backend/store/sheet.go:17-37`

**Step 1: Update SheetMessage struct**

Edit `backend/store/sheet.go`, replace the struct definition:

```go
// SheetMessage is the message for a sheet.
type SheetMessage struct {
	// SHA256 hash of the statement (hex-encoded)
	Sha256 string
	// SQL statement content
	Statement string
	// Size of the statement in bytes
	Size int64
}
```

Remove the `GetSha256Hex()` method (lines 35-37) since Sha256 is already a hex string.

**Step 2: Run Go format**

Run: `gofmt -w backend/store/sheet.go`
Expected: File formatted

**Step 3: Verify it compiles (will fail, expected)**

Run: `go build ./backend/store/...`
Expected: Compilation errors (methods still use old signature)

**Step 4: Commit struct changes**

```bash
git add backend/store/sheet.go
git commit -m "refactor: simplify SheetMessage to use hex SHA256

Remove ProjectID, Creator, Title, UID, CreatedAt fields.
Change Sha256 from []byte to string (hex-encoded).

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Update GetSheetMetadata Method

**Files:**
- Modify: `backend/store/sheet.go:39-54`

**Step 1: Update GetSheetMetadata signature and implementation**

Edit `backend/store/sheet.go`:

```go
// GetSheetMetadata gets a sheet with truncated statement (max 2MB).
// Use this when you need to check sheet.Size or other metadata before processing.
// Statement field will be truncated to MaxSheetSize (2MB).
func (s *Store) GetSheetMetadata(ctx context.Context, sha256Hex string) (*SheetMessage, error) {
	if v, ok := s.sheetMetadataCache.Get(sha256Hex); ok && s.enableCache {
		return v, nil
	}

	sheet, err := s.getSheet(ctx, sha256Hex, false)
	if err != nil {
		return nil, err
	}

	s.sheetMetadataCache.Add(sha256Hex, sheet)
	return sheet, nil
}
```

**Step 2: Update GetSheetFull method**

Edit `backend/store/sheet.go`:

```go
// GetSheetFull gets a sheet with the complete statement.
// Use this when you need the full statement for execution or processing.
// Statement field contains the complete content regardless of size.
func (s *Store) GetSheetFull(ctx context.Context, sha256Hex string) (*SheetMessage, error) {
	if v, ok := s.sheetFullCache.Get(sha256Hex); ok && s.enableCache {
		return v, nil
	}

	sheet, err := s.getSheet(ctx, sha256Hex, true)
	if err != nil {
		return nil, err
	}

	s.sheetFullCache.Add(sha256Hex, sheet)
	return sheet, nil
}
```

**Step 3: Update getSheet helper method**

Edit `backend/store/sheet.go`:

```go
// getSheet is the internal helper for fetching a single sheet by SHA256.
func (s *Store) getSheet(ctx context.Context, sha256Hex string, loadFull bool) (*SheetMessage, error) {
	statementField := fmt.Sprintf("LEFT(content, %d)", common.MaxSheetSize)
	if loadFull {
		statementField = "content"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			%s,
			OCTET_LENGTH(content)
		FROM sheet_blob
		WHERE sha256 = decode(?, 'hex')`, statementField), sha256Hex)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sheet *SheetMessage
	if rows.Next() {
		sheet = &SheetMessage{
			Sha256: sha256Hex,
		}
		if err := rows.Scan(
			&sheet.Statement,
			&sheet.Size,
		); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if sheet == nil {
		return nil, errors.Errorf("sheet not found with sha256 %s", sha256Hex)
	}

	return sheet, nil
}
```

**Step 4: Run Go format**

Run: `gofmt -w backend/store/sheet.go`
Expected: File formatted

**Step 5: Commit changes**

```bash
git add backend/store/sheet.go
git commit -m "refactor: update sheet get methods to use SHA256

Query sheet_blob directly by SHA256 instead of joining with sheet table.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Update CreateSheets Method

**Files:**
- Modify: `backend/store/sheet.go:140-213`

**Step 1: Update CreateSheets signature and implementation**

Edit `backend/store/sheet.go`:

```go
// CreateSheets creates new sheets.
// You should not use this function directly to create sheets.
// Use CreateSheets in component/sheet instead.
func (s *Store) CreateSheets(ctx context.Context, creates ...*SheetMessage) ([]*SheetMessage, error) {
	var statements []string
	var sha256s [][]byte

	for _, c := range creates {
		statements = append(statements, c.Statement)
		h := sha256.Sum256([]byte(c.Statement))
		c.Sha256 = hex.EncodeToString(h[:])
		sha256s = append(sha256s, h[:])
		c.Size = int64(len(c.Statement))
	}

	if err := s.batchCreateSheetBlob(ctx, sha256s, statements); err != nil {
		return nil, errors.Wrapf(err, "failed to create sheet blobs")
	}

	return creates, nil
}
```

**Step 2: Run Go format**

Run: `gofmt -w backend/store/sheet.go`
Expected: File formatted

**Step 3: Commit changes**

```bash
git add backend/store/sheet.go
git commit -m "refactor: simplify CreateSheets to only create blobs

Remove sheet table insertion, only create sheet_blob entries.
Calculate and return SHA256 hex strings.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Update Store Cache Types

**Files:**
- Modify: `backend/store/store.go` (find sheetMetadataCache and sheetFullCache)

**Step 1: Find cache definitions**

Run: `grep -n "sheetMetadataCache\|sheetFullCache" backend/store/store.go`
Expected: Line numbers for cache field definitions

**Step 2: Update cache key type from int to string**

Find the Store struct and update the cache fields:

```go
// Before:
// sheetMetadataCache *cache.Cache[int, *SheetMessage]
// sheetFullCache     *cache.Cache[int, *SheetMessage]

// After:
sheetMetadataCache *cache.Cache[string, *SheetMessage]
sheetFullCache     *cache.Cache[string, *SheetMessage]
```

**Step 3: Find cache initialization**

Run: `grep -n "NewCache.*SheetMessage" backend/store/store.go`
Expected: Cache initialization code

Update cache initialization from:
```go
// Before:
// cache.NewCache[int, *SheetMessage](...)

// After:
cache.NewCache[string, *SheetMessage](...)
```

**Step 4: Run Go format**

Run: `gofmt -w backend/store/store.go`
Expected: File formatted

**Step 5: Verify it compiles**

Run: `go build ./backend/store/...`
Expected: SUCCESS (store package should compile now)

**Step 6: Commit changes**

```bash
git add backend/store/store.go
git commit -m "refactor: update sheet cache to use string keys

Change cache key type from int to string (SHA256 hex).

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Create Database Migration

**Files:**
- Create: `backend/migrator/migration/4.0/0000##sheet_content_addressed.sql`
- Modify: `backend/migrator/migration/LATEST.sql`

**Step 1: Create migration directory**

Run: `mkdir -p backend/migrator/migration/4.0`
Expected: Directory created

**Step 2: Create migration SQL file**

Create `backend/migrator/migration/4.0/0000##sheet_content_addressed.sql`:

```sql
-- Add new column with sha256 reference to task_run
ALTER TABLE task_run ADD COLUMN sheet_sha256 bytea REFERENCES sheet_blob(sha256);

-- Backfill task_run.sheet_sha256 from sheet table
UPDATE task_run tr
SET sheet_sha256 = s.sha256
FROM sheet s
WHERE tr.sheet_id = s.id;

-- Backfill task.payload JSONB (Task proto)
-- Converts sheetId (int) to sheetSha256 (hex string)
UPDATE task t
SET payload = jsonb_set(
    payload - 'sheetId',
    '{sheetSha256}',
    to_jsonb(encode(s.sha256, 'hex'))
)
FROM sheet s
WHERE (t.payload->>'sheetId')::int = s.id
AND t.payload ? 'sheetId';

-- Drop old column
ALTER TABLE task_run DROP COLUMN sheet_id;

-- Drop sheet table entirely
DROP TABLE sheet;
```

**Step 3: Update LATEST.sql**

Edit `backend/migrator/migration/LATEST.sql`:

Find the `sheet` table definition (around line 177) and remove it entirely:
```sql
-- DELETE THESE LINES:
-- sheet table stores general statements.
CREATE TABLE sheet (
    id serial PRIMARY KEY,
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    project text NOT NULL REFERENCES project(resource_id),
    name text NOT NULL,
    sha256 bytea NOT NULL,
    -- Stored as SheetPayload (proto/store/store/sheet.proto)
    payload jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_sheet_project ON sheet(project);

ALTER SEQUENCE sheet_id_seq RESTART WITH 101;
```

Find `task_run` table definition (around line 227) and update:
```sql
-- Before:
sheet_id integer REFERENCES sheet(id),

-- After:
sheet_sha256 bytea REFERENCES sheet_blob(sha256),
```

**Step 4: Format SQL**

Run: `buf format -w proto` (SQL formatting if available)
Expected: Files formatted or no-op

**Step 5: Commit migration**

```bash
git add backend/migrator/migration/4.0/ backend/migrator/migration/LATEST.sql
git commit -m "feat: add migration for content-addressed sheets

- Add task_run.sheet_sha256 column
- Backfill from existing sheet table
- Backfill task.payload JSONB
- Drop sheet table

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Update Migrator Test

**Files:**
- Modify: `backend/migrator/migrator_test.go`

**Step 1: Find TestLatestVersion**

Run: `grep -n "TestLatestVersion" backend/migrator/migrator_test.go`
Expected: Line number of test

**Step 2: Update test to expect version 4.0**

Edit `backend/migrator/migrator_test.go`, find the test and update the expected latest version:

```go
// Find the assertion that checks latest version
// Update from "3.9" to "4.0"
```

**Step 3: Run the migrator test**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run ^TestLatestVersion$`
Expected: PASS

**Step 4: Commit test update**

```bash
git add backend/migrator/migrator_test.go
git commit -m "test: update migrator test for version 4.0

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Update Sheet API Service

**Files:**
- Modify: `backend/api/v1/sheet_service.go`

**Step 1: Find GetSheet method**

Run: `grep -n "func.*GetSheet" backend/api/v1/sheet_service.go`
Expected: Line numbers of GetSheet methods

**Step 2: Update GetSheet to parse SHA256 from resource name**

Update the method to:
1. Parse `projects/{project}/sheets/{sha256}` resource name
2. Extract sha256 hex string (not integer ID)
3. Call `store.GetSheetFull(ctx, sha256Hex)` instead of `store.GetSheetFull(ctx, id)`
4. Build response with resource name including sha256

**Step 3: Find CreateSheet method**

Run: `grep -n "func.*CreateSheet" backend/api/v1/sheet_service.go`
Expected: Line number

**Step 4: Update CreateSheet response**

Update to return resource name with SHA256:
```go
sheetName := fmt.Sprintf("projects/%s/sheets/%s", projectID, sheet.Sha256)
```

**Step 5: Update any other sheet service methods**

Search for all methods that reference sheet IDs and update them to use SHA256.

**Step 6: Run Go format**

Run: `gofmt -w backend/api/v1/sheet_service.go`
Expected: File formatted

**Step 7: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/sheet_service.go`
Expected: No errors (or fix any errors found)

**Step 8: Commit API service changes**

```bash
git add backend/api/v1/sheet_service.go
git commit -m "refactor: update sheet API to use SHA256 identifiers

Parse SHA256 from resource names instead of integer IDs.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Update Plan Service

**Files:**
- Modify: `backend/api/v1/plan_service.go`
- Modify: `backend/api/v1/plan_service_converter.go`

**Step 1: Find sheet references in plan_service.go**

Run: `grep -n "GetSheetMetadata\|GetSheetFull\|sheet.UID\|sheet.GetSha256Hex" backend/api/v1/plan_service.go`
Expected: Line numbers where sheets are accessed

**Step 2: Update sheet access**

For each occurrence:
- Change from accessing `sheet.UID` to `sheet.Sha256`
- Change from `GetSheetMetadata(ctx, sheetID)` to `GetSheetMetadata(ctx, sha256Hex)`
- Remove calls to `sheet.GetSha256Hex()`, use `sheet.Sha256` directly

**Step 3: Find sheet references in plan_service_converter.go**

Run: `grep -n "GetSheetMetadata\|GetSheetFull\|SheetId\|sheet_id" backend/api/v1/plan_service_converter.go`
Expected: Line numbers

**Step 4: Update converters**

Update code that:
- Converts proto `sheet_id` (int32) to `sheet_sha256` (string)
- Builds resource names with SHA256 instead of integer IDs

**Step 5: Run Go format**

Run: `gofmt -w backend/api/v1/plan_service.go backend/api/v1/plan_service_converter.go`
Expected: Files formatted

**Step 6: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/plan_service*.go`
Expected: No errors (fix any that appear)

**Step 7: Commit changes**

```bash
git add backend/api/v1/plan_service.go backend/api/v1/plan_service_converter.go
git commit -m "refactor: update plan service to use sheet SHA256

Update converters and sheet access to use SHA256 identifiers.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Update Rollout Service

**Files:**
- Modify: `backend/api/v1/rollout_service.go`
- Modify: `backend/api/v1/rollout_service_task.go`

**Step 1: Find sheet references**

Run: `grep -n "GetSheetMetadata\|GetSheetFull\|sheet.UID\|SheetId" backend/api/v1/rollout_service.go backend/api/v1/rollout_service_task.go`
Expected: Line numbers

**Step 2: Update all sheet access patterns**

For both files:
- Change `sheet.UID` to `sheet.Sha256`
- Change `task.SheetId` (int32) to `task.SheetSha256` (string)
- Update `GetSheetMetadata(ctx, id)` to `GetSheetMetadata(ctx, sha256Hex)`

**Step 3: Run Go format**

Run: `gofmt -w backend/api/v1/rollout_service*.go`
Expected: Files formatted

**Step 4: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/rollout_service*.go`
Expected: No errors

**Step 5: Commit changes**

```bash
git add backend/api/v1/rollout_service*.go
git commit -m "refactor: update rollout service to use sheet SHA256

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Update Release Service

**Files:**
- Modify: `backend/api/v1/release_service.go`
- Modify: `backend/api/v1/release_service_check.go`

**Step 1: Find sheet references**

Run: `grep -n "GetSheetMetadata\|GetSheetFull\|CreateSheets" backend/api/v1/release_service*.go`
Expected: Line numbers

**Step 2: Update CreateSheets calls**

Update calls to `CreateSheets`:
- Remove `projectID` parameter (no longer needed)
- Access `sheet.Sha256` instead of `sheet.UID`

**Step 3: Update sheet get calls**

Change from ID-based to SHA256-based lookups.

**Step 4: Run Go format**

Run: `gofmt -w backend/api/v1/release_service*.go`
Expected: Files formatted

**Step 5: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/release_service*.go`
Expected: No errors

**Step 6: Commit changes**

```bash
git add backend/api/v1/release_service*.go
git commit -m "refactor: update release service to use sheet SHA256

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 12: Update TaskRun Executors

**Files:**
- Modify: `backend/runner/taskrun/executor.go`
- Modify: `backend/runner/taskrun/database_create_executor.go`
- Modify: `backend/runner/taskrun/database_migrate_executor.go`
- Modify: `backend/runner/taskrun/schema_update_sdl_executor.go`
- Modify: `backend/runner/taskrun/data_export_executor.go`

**Step 1: Find all sheet references in executor files**

Run: `grep -n "GetSheetMetadata\|GetSheetFull\|SheetId" backend/runner/taskrun/*.go`
Expected: Line numbers across multiple files

**Step 2: Update executor.go**

Edit `backend/runner/taskrun/executor.go`:
- Change `task.SheetId` to `task.SheetSha256`
- Update any sheet retrieval calls

**Step 3: Update database_create_executor.go**

Same pattern: `SheetId` → `SheetSha256`, ID lookups → SHA256 lookups

**Step 4: Update database_migrate_executor.go**

Same pattern.

**Step 5: Update schema_update_sdl_executor.go**

Same pattern.

**Step 6: Update data_export_executor.go**

Same pattern.

**Step 7: Run Go format**

Run: `gofmt -w backend/runner/taskrun/*.go`
Expected: Files formatted

**Step 8: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/runner/taskrun/`
Expected: No errors

**Step 9: Commit changes**

```bash
git add backend/runner/taskrun/
git commit -m "refactor: update taskrun executors to use sheet SHA256

All executors now use SHA256 identifiers for sheets.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 13: Update PlanCheck Executors

**Files:**
- Modify: `backend/runner/plancheck/statement_advise_executor.go`
- Modify: `backend/runner/plancheck/statement_report_executor.go`
- Modify: `backend/runner/plancheck/ghost_sync_executor.go`

**Step 1: Find sheet references**

Run: `grep -n "GetSheetMetadata\|GetSheetFull\|SheetId" backend/runner/plancheck/*.go`
Expected: Line numbers

**Step 2: Update all three executor files**

For each file, update:
- `task.SheetId` → `task.SheetSha256`
- Sheet retrieval to use SHA256

**Step 3: Run Go format**

Run: `gofmt -w backend/runner/plancheck/*.go`
Expected: Files formatted

**Step 4: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/runner/plancheck/`
Expected: No errors

**Step 5: Commit changes**

```bash
git add backend/runner/plancheck/
git commit -m "refactor: update plancheck executors to use sheet SHA256

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 14: Update Approval Runner

**Files:**
- Modify: `backend/runner/approval/runner.go`

**Step 1: Find sheet references**

Run: `grep -n "GetSheetMetadata\|GetSheetFull" backend/runner/approval/runner.go`
Expected: Line numbers (if any)

**Step 2: Update sheet access (if present)**

Update any sheet-related code to use SHA256.

**Step 3: Run Go format**

Run: `gofmt -w backend/runner/approval/runner.go`
Expected: File formatted

**Step 4: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/runner/approval/`
Expected: No errors

**Step 5: Commit changes**

```bash
git add backend/runner/approval/runner.go
git commit -m "refactor: update approval runner to use sheet SHA256

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 15: Update Revision Service

**Files:**
- Modify: `backend/api/v1/revision_service.go`

**Step 1: Find sheet references**

Run: `grep -n "GetSheetMetadata\|GetSheetFull\|CreateSheets" backend/api/v1/revision_service.go`
Expected: Line numbers

**Step 2: Update sheet operations**

Update to use SHA256-based access.

**Step 3: Run Go format**

Run: `gofmt -w backend/api/v1/revision_service.go`
Expected: File formatted

**Step 4: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/revision_service.go`
Expected: No errors

**Step 5: Commit changes**

```bash
git add backend/api/v1/revision_service.go
git commit -m "refactor: update revision service to use sheet SHA256

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 16: Update Plan Service Plan Check

**Files:**
- Modify: `backend/api/v1/plan_service_plan_check.go`

**Step 1: Find sheet references**

Run: `grep -n "GetSheetMetadata\|GetSheetFull\|SheetId" backend/api/v1/plan_service_plan_check.go`
Expected: Line numbers

**Step 2: Update sheet access**

Update to use `task.SheetSha256` and SHA256-based lookups.

**Step 3: Run Go format**

Run: `gofmt -w backend/api/v1/plan_service_plan_check.go`
Expected: File formatted

**Step 4: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/plan_service_plan_check.go`
Expected: No errors

**Step 5: Commit changes**

```bash
git add backend/api/v1/plan_service_plan_check.go
git commit -m "refactor: update plan check service to use sheet SHA256

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 17: Build and Run Linter

**Step 1: Build the entire backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 2: Run golangci-lint on entire codebase**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No errors

**Step 3: If errors found, fix them iteratively**

Run: `golangci-lint run --fix --allow-parallel-runners`
Expected: Auto-fixes applied

Repeat step 2 until no errors remain.

**Step 4: Commit any lint fixes**

```bash
git add -A
git commit -m "fix: resolve linting issues for sheet SHA256 migration

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 18: Run Backend Tests

**Step 1: Run store tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store -run Sheet`
Expected: Tests pass (or identify failures to fix)

**Step 2: Run migrator tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator`
Expected: Tests pass

**Step 3: Run API tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run Sheet`
Expected: Tests pass

**Step 4: Fix any test failures**

If tests fail:
1. Analyze failure
2. Update test or code
3. Commit fix
4. Re-run tests

**Step 5: Commit test fixes (if any)**

```bash
git add -A
git commit -m "test: fix sheet tests for SHA256 migration

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 19: Update Frontend (if applicable)

**Note:** This task depends on frontend code structure. If frontend directly references sheet IDs, update to use SHA256.

**Step 1: Search for sheet ID references in frontend**

Run: `grep -r "sheets/" frontend/src --include="*.ts" --include="*.vue" | grep -v node_modules`
Expected: List of files with sheet references

**Step 2: Update TypeScript/Vue files**

For each file:
- Update resource name parsing to expect SHA256 instead of integers
- Update API calls to use SHA256

**Step 3: Run frontend type check**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 4: Run frontend lint**

Run: `pnpm --dir frontend lint`
Expected: No lint errors

**Step 5: Run frontend format**

Run: `pnpm --dir frontend biome:check`
Expected: Files formatted

**Step 6: Commit frontend changes**

```bash
git add frontend/
git commit -m "refactor: update frontend to use sheet SHA256

Parse SHA256 from resource names instead of integer IDs.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 20: Final Verification

**Step 1: Build entire project**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Build succeeds

**Step 2: Run full linter**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No errors

**Step 3: Run all backend tests**

Run: `go test -v -count=1 ./backend/...`
Expected: All tests pass

**Step 4: Run frontend tests**

Run: `pnpm --dir frontend test`
Expected: All tests pass

**Step 5: Manual smoke test (optional)**

Start the server:
```bash
PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug
```

Test:
1. Create a sheet via API
2. Verify resource name uses SHA256
3. Retrieve sheet by SHA256
4. Create task with sheet reference
5. Verify task stores SHA256 correctly

**Step 6: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix: final adjustments for sheet SHA256 migration

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Summary

This plan migrates sheets from ID-based to SHA256-based content addressing in 20 tasks:

1-5: Proto and struct changes
6-7: Database migration
8-16: Update all callers (API services, runners, executors)
17-18: Build and test backend
19: Update frontend
20: Final verification

Each task is atomic and can be committed independently. Follow TDD principles where applicable. Run linters after each task to catch issues early.
