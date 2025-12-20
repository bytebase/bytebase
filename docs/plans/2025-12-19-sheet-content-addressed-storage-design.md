# Content-Addressed Storage for Sheets

## Overview

Migrate sheet storage from ID-based to content-addressed (SHA256-based) storage. The `store.SheetMessage` will reference `sheet_blob` table directly instead of the `sheet` table. The `sheet` table will be dropped entirely.

## Goals

- Use SHA256 hash as the primary identifier for sheets
- Share sheet blobs across projects (immutable, content-addressed storage)
- Simplify data model by removing sheet metadata (creator, project, title)
- Change v1 API resource names from `projects/{project}/sheets/{id}` to `projects/{project}/sheets/{sha256}`

## Non-Goals

- Change authorization model (already project-scoped, no changes needed)
- Gradual migration or backward compatibility (direct cutover)

## Data Model Changes

### SheetMessage Struct

**Before:**
```go
type SheetMessage struct {
    ProjectID string
    Creator   string
    Title     string
    Statement string
    Sha256    []byte
    UID       int
    Size      int64
    CreatedAt time.Time
}
```

**After:**
```go
type SheetMessage struct {
    // SHA256 hash of the statement (hex-encoded)
    Sha256 string
    // SQL statement content
    Statement string
    // Size of the statement in bytes
    Size int64
}
```

### Database Schema Changes

**sheet_blob table** - no changes:
```sql
CREATE TABLE sheet_blob (
    sha256 bytea NOT NULL PRIMARY KEY,
    content text NOT NULL
);
```

**task_run table** - replace integer foreign key:
```sql
-- Before:
CREATE TABLE task_run (
    ...
    sheet_id integer REFERENCES sheet(id),
    ...
);

-- After:
CREATE TABLE task_run (
    ...
    sheet_sha256 bytea REFERENCES sheet_blob(sha256),
    ...
);
```

**sheet table** - will be dropped entirely after migration.

### Proto Changes

**proto/store/store/task.proto:**
```protobuf
message Task {
    // Before:
    // int32 sheet_id = 4;

    // After:
    // SHA256 hash of the sheet content (hex-encoded)
    string sheet_sha256 = 4;
}
```

The proto field stores the hex-encoded SHA256 string. When exposed via v1 API, it gets embedded in the full resource name `projects/{project}/sheets/{sha256}`.

## Migration Strategy

### Direct Cutover

All changes happen in a single migration. No intermediate state.

**Migration SQL:**

```sql
-- 1. Add new column with sha256 reference
ALTER TABLE task_run ADD COLUMN sheet_sha256 bytea REFERENCES sheet_blob(sha256);

-- 2. Backfill task_run.sheet_sha256 from sheet table
UPDATE task_run tr
SET sheet_sha256 = s.sha256
FROM sheet s
WHERE tr.sheet_id = s.id;

-- 3. Backfill task.payload JSONB (Task proto)
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

-- 4. Drop old column
ALTER TABLE task_run DROP COLUMN sheet_id;

-- 5. Drop sheet table entirely
DROP TABLE sheet;
```

### Code Deployment

All code changes deploy atomically with the database migration:

- Update proto definitions (regenerate with `buf generate`)
- Update `store.SheetMessage` struct
- Update all store methods that query/create sheets
- Update all API handlers to use sha256-based resource names
- Update frontend to use new resource name format

No feature flags or compatibility layers needed.

## Code Changes

### backend/store/sheet.go

**SheetMessage struct:**
```go
type SheetMessage struct {
    Sha256    string
    Statement string
    Size      int64
}
// Remove GetSha256Hex() method - Sha256 is already hex string
```

**GetSheetMetadata / GetSheetFull:**
```go
// Before:
func (s *Store) GetSheetMetadata(ctx context.Context, id int) (*SheetMessage, error)
func (s *Store) GetSheetFull(ctx context.Context, id int) (*SheetMessage, error)

// After:
func (s *Store) GetSheetMetadata(ctx context.Context, sha256Hex string) (*SheetMessage, error)
func (s *Store) GetSheetFull(ctx context.Context, sha256Hex string) (*SheetMessage, error)
```

**Query changes:**
```go
// Before:
// SELECT ... FROM sheet
// LEFT JOIN sheet_blob ON sheet.sha256 = sheet_blob.sha256
// WHERE sheet.id = ?

// After:
// SELECT ... FROM sheet_blob
// WHERE sha256 = decode(?, 'hex')
```

**CreateSheets:**
```go
// No longer creates rows in sheet table
// Just calls batchCreateSheetBlob
// Returns SheetMessage with sha256, statement, size
func (s *Store) CreateSheets(ctx context.Context, creates ...*SheetMessage) ([]*SheetMessage, error)
```

### Proto Changes

Run after updating task.proto:
```bash
cd proto && buf generate
```

### Callers

Any code calling `store.GetSheetMetadata(ctx, sheetID)` changes to pass sha256 hex string instead of integer ID. Similarly for creating/updating tasks that reference sheets.

## API Changes

### Resource Name Format

**Before:**
```
projects/{project}/sheets/{id}
```
Example: `projects/my-project/sheets/123`

**After:**
```
projects/{project}/sheets/{sha256}
```
Example: `projects/my-project/sheets/a1b2c3d4e5f67890abcdef...`

### Authorization

No change. Authorization works the same way:
- User must have IAM permission to access the project
- Sheet retrieval is project-scoped via the resource name
- No verification that the project actually "owns" the sheet

### Converting Between Store and API Formats

**Store → API:**
```go
// Store has: sheet_sha256 = "a1b2c3d4..."
// API returns: "projects/my-project/sheets/a1b2c3d4..."
sheetResourceName := fmt.Sprintf("projects/%s/sheets/%s", project, sheetSha256)
```

**API → Store:**
```go
// Parse resource name to extract sha256
// Call store.GetSheetMetadata(ctx, sha256Hex)
```

### Client Impact

Clients see sheet IDs change from integers (`123`) to SHA256 hashes (`a1b2c3d4...`). The API endpoints and request/response structure remain the same.

## Benefits

- **Deduplication**: Same SQL content stored once, referenced by multiple tasks/projects
- **Immutability**: SHA256-based addressing guarantees content integrity
- **Simplicity**: Removes unnecessary metadata, cleaner data model
- **Scalability**: Content-addressed storage scales better than relational references

## Risks

- **Breaking change**: Existing sheet IDs become invalid
- **Client compatibility**: All API clients must handle sha256-based IDs
- **Migration complexity**: JSONB backfill must handle all edge cases correctly

## Testing

- Unit tests for store methods with sha256-based lookups
- Integration tests for migration SQL (verify data integrity)
- API tests for new resource name format
- Frontend tests for displaying sha256-based sheet references
