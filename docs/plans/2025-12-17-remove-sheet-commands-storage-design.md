# Remove Sheet Commands Storage - Design Document

**Date:** 2025-12-17
**Status:** Draft

## Overview

This design removes the `repeated Range commands` field from the `SheetPayload` proto message and moves SQL command range information to the task run log where it's actually needed.

### Problem Statement

Currently, `SheetPayload.commands` stores pre-parsed SQL statement ranges for every sheet in the database. This creates three issues:

1. **Storage overhead** - Commands data is stored alongside every sheet, consuming database space
2. **Architectural concern** - Parsing logic results are persisted in sheet storage rather than where they're used (task run logs)
3. **Unnecessary persistence** - Sheets store commands but they're only needed when displaying task run log entries

### Solution

Instead of storing commands in sheets, store the ranges directly in task run log entries where they're used. When a task executes SQL commands, compute the ranges at that time and store them in the `CommandExecute` log entry.

### Goals

- Remove `commands` field from `SheetPayload` proto
- Change `command_indexes` to `command_ranges` in `TaskRunLog.CommandExecute`
- Store ranges when creating task run log entries (at execution time)
- Maintain exact same UX - users see no difference in functionality

### Non-Goals

- Changing how sheets are stored or created
- Modifying sheet immutability
- Changing task run log display logic beyond the data source for ranges

## Architecture

### Proto Changes

**Location:** `proto/v1/v1/sheet_service.proto`

Remove commands from SheetPayload:
```proto
message SheetPayload {
  // DELETE this field:
  // repeated Range commands = 1;
}
```

**Location:** `proto/store/store/task_run_log.proto`

Change command_indexes to command_ranges:
```proto
message CommandExecute {
  // REPLACE:
  // repeated int32 command_indexes = 1;

  // WITH:
  // The ranges of the executed commands in the sheet.
  repeated Range command_ranges = 3;

  // The statement to be executed.
  string statement = 2;
}
```

Note: Use field number 3 to avoid reusing deprecated field 1.

### Data Flow

**Before (current):**
1. Sheet created → parse commands → store in `SheetPayload.commands`
2. Task executes → log entry stores `command_indexes`
3. UI displays → read sheet, look up command by index, extract using stored range

**After (new):**
1. Sheet created → no command parsing
2. Task executes → parse commands → store `command_ranges` in log entry
3. UI displays → read command ranges from log entry, extract directly

### Benefits

- **No API needed** - No ParseSheet endpoint, no additional API calls
- **No caching needed** - Ranges are in the log entry, already loaded
- **Data locality** - Ranges stored where they're used (task run logs), not in sheets
- **Same parsing logic** - Reuses existing `getSheetCommands()` function, just at a different time

## Component Changes

### Proto Changes

**File:** `proto/v1/v1/sheet_service.proto`

Remove `repeated Range commands = 1;` from `SheetPayload` message.

**File:** `proto/store/store/task_run_log.proto`

1. Add `repeated Range command_ranges = 3;` to `CommandExecute` message
2. Mark `repeated int32 command_indexes = 1;` as deprecated

After proto changes, run:
```bash
buf format -w proto
buf lint proto
cd proto && buf generate
```

### Backend Changes

**File:** `backend/component/sheet/sheet.go`

Remove lines that set `sheet.Payload.Commands`:
```go
// DELETE this line from CreateSheets (line 66):
sheet.Payload.Commands = getSheetCommands(sheet.Payload.Engine, sheet.Statement)
```

Keep `getSheetCommands()` function and its helpers - they'll be used when creating task run log entries.

Optionally, export `getSheetCommands()` as `GetSheetCommands()` if it needs to be called from other packages.

**File:** Backend task run log creation (location TBD - wherever `CommandExecute` log entries are created)

When creating `CommandExecute` log entries:

```go
// Compute command ranges for the sheet
commands := sheet.GetSheetCommands(sheetEngine, sheetContent)

// Create log entry with ranges
logEntry := &storepb.TaskRunLogEntry{
  Type: storepb.TaskRunLogEntry_COMMAND_EXECUTE,
  CommandExecute: &storepb.TaskRunLogEntry_CommandExecute{
    CommandRanges: commands, // Store ranges, not indexes
    Statement: statement,
  },
}
```

**File:** `backend/api/v1/sheet_service.go`

Remove fallback logic in sheet conversion (lines 199-205):
```go
// DELETE this entire block:
v1SheetPayload := &v1pb.SheetPayload{}
if len(sheet.Payload.GetCommands()) > 0 {
  v1SheetPayload.Commands = convertToRanges(sheet.Payload.GetCommands())
} else {
  v1SheetPayload.Commands = []*v1pb.Range{
    {Start: 0, End: int32(sheet.Size)},
  }
}
```

Return empty `SheetPayload{}` or omit the field entirely.

### Frontend Changes

**File:** `frontend/src/components/Plan/components/RolloutView/v2/TaskRunLogViewer/useTaskRunLogSections.ts`

Update `getEntryDetail` to use `command_ranges` instead of `extractSheetCommandByIndex`:

```typescript
const getEntryDetail = (entry: TaskRunLogEntry): string => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE: {
      const cmd = entry.commandExecute;
      if (!cmd) return "";
      if (cmd.response?.error) return cmd.response.error;

      let statement: string | undefined;
      if (cmd.statement) {
        statement = cmd.statement;
      } else if (cmd.commandRanges && cmd.commandRanges.length > 0) {
        // NEW: Extract using ranges from log entry
        const sheetValue = toValue(sheet);
        if (sheetValue && cmd.commandRanges[0]) {
          const range = cmd.commandRanges[0];
          const subarray = sheetValue.content.subarray(range.start, range.end);
          statement = new TextDecoder().decode(subarray);
        }
      }

      if (statement) {
        const stmt = statement.trim().replace(/\s+/g, " ");
        return stmt.length > 80 ? stmt.substring(0, 80) + "..." : stmt;
      }
      return "-";
    }
    // ... rest of cases
  }
};
```

**File:** `frontend/src/utils/v1/sheet.ts`

The `extractSheetCommandByIndex` function can be removed entirely, or marked as deprecated if it's used elsewhere.

**File:** `frontend/src/store/modules/v1/sheet.ts`

No changes needed - no caching logic required.

**File:** `frontend/src/components/Plan/components/RolloutView/v2/composables/useTaskRunLogSummary.ts`

No changes needed - no command fetching required.

## Error Handling

### Backend Errors

**Task run log entry creation:**
- **Parsing errors:** Log warning, store empty command_ranges array (graceful degradation)
- **Large sheets (>MaxSheetCheckSize):** Store empty command_ranges array (same as current behavior)
- **No sheet available:** Store statement directly (existing fallback)

**Implementation:**
```go
commands := getSheetCommands(engine, content)
if commands == nil {
  commands = []*storepb.Range{} // Empty array, not nil
}
```

### Frontend Errors

**Missing command_ranges:**
- If `command_ranges` is empty/nil, fall back to `statement` field (existing behavior)
- If neither available, show "-" or nothing (existing behavior)

**Backward compatibility:**
- Old task run logs still have `command_indexes` populated
- Frontend should handle both `command_indexes` (deprecated) and `command_ranges` (new)
- During migration period, check `command_ranges` first, fall back to `command_indexes`

**Graceful degradation:**
```typescript
let statement: string | undefined;
if (cmd.statement) {
  statement = cmd.statement;
} else if (cmd.commandRanges?.length > 0) {
  // NEW: Use ranges from log entry
  statement = extractFromRange(sheet, cmd.commandRanges[0]);
} else if (cmd.commandIndexes?.length > 0) {
  // DEPRECATED: Fall back for old logs
  // This can be removed after all old logs are migrated/expired
  statement = extractSheetCommandByIndex(sheet, cmd.commandIndexes[0]);
}
```

### Backward Compatibility

**Migration period:**
- Old task run logs have `command_indexes` populated
- New task run logs have `command_ranges` populated
- Frontend handles both during transition
- After sufficient time (e.g., all old task runs expired), remove `command_indexes` support from frontend

**Proto compatibility:**
- `command_indexes` marked as deprecated but not removed immediately
- `command_ranges` uses new field number (3) to avoid conflicts
- No breaking changes - old and new systems can coexist

## Migration Strategy

1. **Phase 1:** Add `command_ranges` field to proto, deploy backend + frontend that writes new field but still reads both
2. **Phase 2:** Update sheet creation to stop populating `SheetPayload.commands`
3. **Phase 3:** After transition period (all old task runs expired), remove deprecated fields and frontend fallback logic
4. **Phase 4:** Remove `SheetPayload.commands` field from proto entirely

## Testing

### Backend Tests

**File:** `backend/component/sheet/sheet_test.go` (if exists)

- Test that new sheets don't have `Payload.Commands` populated
- Test that `getSheetCommands()` still works correctly

**File:** Task run log creation tests (location TBD)

- Test that `CommandExecute` entries have `command_ranges` populated
- Test graceful degradation when parsing fails (empty ranges)
- Test large sheets (>MaxSheetCheckSize) return empty ranges

### Frontend Tests

**File:** `frontend/src/components/Plan/components/RolloutView/v2/TaskRunLogViewer/useTaskRunLogSections.test.ts`

- Test `getEntryDetail` extracts statements using `command_ranges`
- Test fallback to `statement` field when ranges unavailable
- Test backward compatibility with `command_indexes`

### Manual Testing

1. Create a new task with SQL sheet
2. Execute the task
3. Verify task run log displays individual commands correctly
4. Verify no regression in existing task run logs (old data)
5. Verify sheet payload no longer contains commands field

## Rollout Plan

1. Deploy proto changes + backend + frontend together (atomic deployment)
2. Monitor for errors in task run log display
3. After stabilization period, stop writing to deprecated `SheetPayload.commands`
4. After all old task runs expire, remove deprecated field support
