# Sheet API Redesign

## Problem

The current `GetSheet` and `GetSheetStatementByID` APIs are confusing for callers:

1. **Unclear method choice**: Callers don't know when to use `GetSheet` vs `GetSheetStatementByID`
2. **Confusing LoadFull flag**: The `LoadFull` boolean in `FindSheetMessage` is not self-documenting
3. **Dual caching complexity**: Two separate caches (`sheetCache` and `sheetStatementCache`) with different behaviors make reasoning difficult
4. **Dual-call pattern**: Many callers make two database calls when they need both metadata and full statement

### Current Issues

**Pattern 1: Statement-only callers** (5 locations)
```go
statement, err := store.GetSheetStatementByID(ctx, sheetID)
```
Uses `sheetStatementCache` (10 entries).

**Pattern 2: Metadata + Statement callers** (4 locations)
```go
sheet, err := store.GetSheet(ctx, &FindSheetMessage{UID: &id})
// Check sheet.Size > MaxSheetCheckSize
statement, err := store.GetSheetStatementByID(ctx, id)
```
Makes two DB queries because the first call uses `sheetCache` (truncated) and second uses `sheetStatementCache`.

**Pattern 3: API/Display callers**
```go
sheet, err := store.GetSheet(ctx, &FindSheetMessage{UID: &id, LoadFull: raw})
```
LoadFull flag toggles between truncated (2MB) and full statement.

## Solution

Replace the confusing APIs with two purpose-specific methods that clearly communicate intent.

### API Design

**New Methods:**

```go
// GetSheetMetadata gets a sheet with truncated statement (max 2MB).
// Use this when you need to check sheet.Size or other metadata before processing.
// Statement field will be truncated to MaxSheetSize (2MB).
func (s *Store) GetSheetMetadata(ctx context.Context, id int) (*SheetMessage, error)

// GetSheetFull gets a sheet with the complete statement.
// Use this when you need the full statement for execution or processing.
// Statement field contains the complete content regardless of size.
func (s *Store) GetSheetFull(ctx context.Context, id int) (*SheetMessage, error)
```

**Remove:**
```go
// Delete these
GetSheetStatementByID(ctx context.Context, id int) (string, error)
GetSheet(ctx context.Context, find *FindSheetMessage) (*SheetMessage, error)
FindSheetMessage
```

**Internal helper (private):**
```go
// getSheet is the internal implementation shared by both methods
func (s *Store) getSheet(ctx context.Context, id int, loadFull bool) (*SheetMessage, error)
```

### Caching Strategy

**Two separate caches with renamed, clearer purpose:**

```go
type Store struct {
    // sheetMetadataCache stores sheets with truncated statements (max 2MB)
    // Size: 64 entries - larger since metadata checks are frequent
    sheetMetadataCache *lru.Cache[int, *SheetMessage]

    // sheetFullCache stores complete sheets with full statements
    // Size: 10 entries - smaller since full sheets can be huge
    sheetFullCache *lru.Cache[int, *SheetMessage]
}
```

**Cache behavior:**

- `GetSheetMetadata()`:
  - Checks `sheetMetadataCache` first
  - On miss: queries DB with `LEFT(sheet_blob.content, MaxSheetSize)`
  - Stores result in `sheetMetadataCache`

- `GetSheetFull()`:
  - Checks `sheetFullCache` first
  - On miss: queries DB with full `sheet_blob.content`
  - Stores result in `sheetFullCache`

If a caller does `GetSheetMetadata()` then `GetSheetFull()`, the second call will query the database. This is expected and acceptable - they explicitly want the full sheet.

### Migration Strategy

**Pattern 1: Statement-only callers** (5 callers)
- Before: `GetSheetStatementByID(ctx, id)`
- After: `GetSheetFull(ctx, id)` and use `.Statement` field
- Files: data_export_executor.go, database_migrate_executor.go, approval/runner.go

**Pattern 2: Metadata + Statement callers** (4 callers)
- Before:
  ```go
  sheet, err := GetSheet(ctx, &FindSheetMessage{UID: &id})
  if sheet.Size > common.MaxSheetCheckSize { return warning }
  statement, err := GetSheetStatementByID(ctx, id)
  ```
- After:
  ```go
  sheet, err := GetSheetMetadata(ctx, id)
  if sheet.Size > common.MaxSheetCheckSize { return warning }
  fullSheet, err := GetSheetFull(ctx, id)
  statement := fullSheet.Statement
  ```
- Files: statement_advise_executor.go, ghost_sync_executor.go, statement_report_executor.go

**Pattern 3: API/Display callers**
- Before: `GetSheet(ctx, &FindSheetMessage{UID: &id, LoadFull: raw})`
- After:
  ```go
  if raw {
      sheet, err := GetSheetFull(ctx, id)
  } else {
      sheet, err := GetSheetMetadata(ctx, id)
  }
  ```
- Files: sheet_service.go, release_service.go

**Pattern 4: Mixed callers** (database_create_executor.go)
- Currently calls both `GetSheetStatementByID` and `GetSheet` separately
- After: `GetSheetFull(ctx, id)` - single call gets everything

### Implementation

```go
func (s *Store) GetSheetMetadata(ctx context.Context, id int) (*SheetMessage, error) {
    if v, ok := s.sheetMetadataCache.Get(id); ok && s.enableCache {
        return v, nil
    }

    sheet, err := s.getSheet(ctx, id, false)
    if err != nil {
        return nil, err
    }

    s.sheetMetadataCache.Add(id, sheet)
    return sheet, nil
}

func (s *Store) GetSheetFull(ctx context.Context, id int) (*SheetMessage, error) {
    if v, ok := s.sheetFullCache.Get(id); ok && s.enableCache {
        return v, nil
    }

    sheet, err := s.getSheet(ctx, id, true)
    if err != nil {
        return nil, err
    }

    s.sheetFullCache.Add(id, sheet)
    return sheet, nil
}

func (s *Store) getSheet(ctx context.Context, id int, loadFull bool) (*SheetMessage, error) {
    statementField := fmt.Sprintf("LEFT(sheet_blob.content, %d)", common.MaxSheetSize)
    if loadFull {
        statementField = "sheet_blob.content"
    }

    q := qb.Q().Space(fmt.Sprintf(`
        SELECT
            sheet.id,
            sheet.creator,
            sheet.created_at,
            sheet.project,
            sheet.name,
            %s,
            sheet.sha256,
            sheet.payload,
            OCTET_LENGTH(sheet_blob.content)
        FROM sheet
        LEFT JOIN sheet_blob ON sheet.sha256 = sheet_blob.sha256
        WHERE sheet.id = ?`, statementField), id)

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
        sheet = &SheetMessage{}
        var payload []byte
        if err := rows.Scan(
            &sheet.UID,
            &sheet.Creator,
            &sheet.CreatedAt,
            &sheet.ProjectID,
            &sheet.Title,
            &sheet.Statement,
            &sheet.Sha256,
            &payload,
            &sheet.Size,
        ); err != nil {
            return nil, err
        }
        sheetPayload := &storepb.SheetPayload{}
        if err := common.ProtojsonUnmarshaler.Unmarshal(payload, sheetPayload); err != nil {
            return nil, err
        }
        sheet.Payload = sheetPayload
    }

    if err := rows.Err(); err != nil {
        return nil, err
    }
    if err := tx.Commit(); err != nil {
        return nil, err
    }

    if sheet == nil {
        return nil, errors.Errorf("sheet not found with id %d", id)
    }

    return sheet, nil
}
```

**Edge cases handled:**
- Sheet not found: return error with clear message
- Cache disabled: still works, queries DB every time
- Multiple sheets returned: impossible with ID-based WHERE clause

## Benefits

1. **Clear intent**: Method names communicate exactly what you get
2. **No confusing flags**: No LoadFull boolean to explain
3. **Simpler mental model**: Two caches with obvious purposes
4. **Consistent behavior**: Each method has one caching strategy
5. **Better errors**: Single method to fix when sheet not found
6. **Type safety**: Return full SheetMessage instead of string-only

## Trade-offs

- Pattern 2 callers still make two calls (metadata check + full fetch), but now it's explicit and intentional
- Migration effort across ~15 call sites
- Two methods instead of one flexible method (but flexibility was the source of confusion)
