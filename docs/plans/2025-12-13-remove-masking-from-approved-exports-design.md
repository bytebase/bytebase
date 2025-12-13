# Remove Masking from Approved DATABASE_EXPORT Issues

**Date:** 2025-12-13
**Status:** Design Complete

## Problem Statement

Currently, Bytebase has two separate access controls for data export:

1. **Approval workflow** - Controls whether a DATABASE_EXPORT issue can execute
2. **Masking exception policy** - Controls which columns are masked in the export

This creates a confusing user experience: an approver reviews and approves a specific SQL export request, but the exported data may still have sensitive columns masked unless the user also has a masking exception policy with the EXPORT action.

**The Core Issue:** If an approver explicitly approves exporting specific data by reviewing the exact SQL statement, requiring additional masking exceptions is redundant and confusing. The approval itself should be sufficient to grant access to that data.

## Solution

DATABASE_EXPORT issues that complete the approval workflow will export data **without any masking**. The approval process serves as the authorization gate for accessing sensitive data.

**Important:** This only affects DATABASE_EXPORT issues (one-time approved exports). SQL Editor exports where users have the EXPORTER role will continue to respect masking policies for ongoing access protection.

## Design

### Two Independent Export Paths

We will maintain two separate code paths for exports:

#### Path 1: SQL Editor Exports (with masking)
- **Location:** `backend/api/v1/sql_service.go`
- **Function:** `DoExport()` - unchanged behavior
- **Flow:** Execute SQL → Apply masking → Format → Zip → Return bytes
- **Used by:** ExportData API endpoint (SQL Editor)
- **Access model:** User has EXPORTER role (ongoing access)

#### Path 2: Approved DATABASE_EXPORT Tasks (no masking)
- **Location:** `backend/runner/taskrun/data_export_executor.go`
- **Function:** New method `executeExport()`
- **Flow:** Execute SQL → Format → Zip → Return bytes (skip masking entirely)
- **Used by:** `DataExportExecutor.RunOnce()`
- **Access model:** One-time approval for specific SQL statement

### Implementation Details

#### Changes to data_export_executor.go

**Current code (line 84):**
```go
bytes, _, exportErr := apiv1.DoExport(ctx, exec.store, exec.dbFactory, exec.license,
    exportRequest, creatorUser, instance, database, nil, exec.schemaSyncer, dataSource)
```

**New approach:**
```go
func (exec *DataExportExecutor) RunOnce(ctx context.Context, ...) {
    // ... existing code to get issue, database, instance, statement ...

    // Execute the export without masking
    bytes, exportErr := exec.executeExport(ctx, instance, database, statement,
        task.Payload.GetFormat(), creatorUser, dataSource)
    if exportErr != nil {
        return true, nil, errors.Wrap(exportErr, "failed to export data")
    }

    // ... existing code to create export archive ...
}

// executeExport performs the actual export without applying any masking.
// This is used for approved DATABASE_EXPORT tasks where the approval itself
// authorizes access to the data.
func (exec *DataExportExecutor) executeExport(
    ctx context.Context,
    instance *store.InstanceMessage,
    database *store.DatabaseMessage,
    statement string,
    format storepb.ExportFormat,
    user *store.UserMessage,
    dataSource storepb.DataSourceType,
) ([]byte, error) {
    // 1. Get driver and connection
    driver, err := exec.dbFactory.GetDataSourceDriver(ctx, instance, database, dataSource)
    if err != nil {
        return nil, errors.Wrap(err, "failed to get database driver")
    }
    defer driver.Close(ctx)

    conn, err := driver.GetDB()
    if err != nil {
        return nil, errors.Wrap(err, "failed to get database connection")
    }
    defer conn.Close()

    // 2. Get query restrictions from workspace policy
    queryRestriction := exec.getEffectiveQueryDataPolicy(ctx, database.ProjectID)

    // 3. Build query context with limits
    queryContext := db.QueryContext{
        Limit:                int(queryRestriction.MaximumResultRows),
        OperatorEmail:        user.Email,
        MaximumSQLResultSize: queryRestriction.MaximumResultSize,
    }
    if queryRestriction.MaxQueryTimeoutInSeconds > 0 {
        queryContext.Timeout = &durationpb.Duration{
            Seconds: queryRestriction.MaxQueryTimeoutInSeconds,
        }
    }

    // 4. Execute query
    results, err := driver.QueryContext(ctx, conn, statement, queryContext)
    if err != nil {
        return nil, errors.Wrap(err, "failed to execute query")
    }

    // 5. Format and zip results (NO MASKING)
    return exec.formatAndZipResults(ctx, results, instance, database, format, statement)
}
```

#### Helper Methods to Add

The following methods will be duplicated from `sql_service.go` into `data_export_executor.go`:

1. **getEffectiveQueryDataPolicy()** - Gets workspace query limits (row count, size, timeout)
2. **formatAndZipResults()** - Formats query results and packages into ZIP archive
3. **logExportError()** - Consistent error logging for export failures

### Architecture Rationale

**Why duplicate code instead of sharing?**

We chose code duplication over creating shared functions because:

1. **No layering violations:** `runner/taskrun` should not depend on `api/v1`
2. **Clear separation of concerns:** Approved exports vs ad-hoc exports are fundamentally different
3. **Independent evolution:** Each path can evolve separately (different validation, limits, etc.)
4. **Easier to understand:** Clear which code path applies in each context

The duplicated logic is straightforward (query execution, formatting, zipping), and having two independent paths makes the intent explicit.

### What Stays Shared

- `backend/component/export` package - CSV, JSON, SQL, XLSX format converters
- Database/instance/project lookups via store
- Query context and policy structures

## Security Considerations

### Approved Exports Bypass Masking
- **Justification:** Approver explicitly reviews the SQL statement before approving
- **Audit trail:** Issue approval is logged, export archive is stored
- **Access model:** One-time access for specific query, not ongoing access

### Workspace Limits Still Apply
Approved exports continue to respect:
- Maximum result rows (from Query Data Policy)
- Maximum result size in bytes
- Query timeout duration

This prevents accidentally huge exports even with approval.

### SQL Editor Exports Unchanged
Users with EXPORTER role continue to have exports masked according to masking exception policies. This maintains column-level protection for ongoing export access.

## Migration & Compatibility

**No breaking changes:**
- Existing DATABASE_EXPORT issues will get unmasked exports after this change
- No proto changes needed
- No API changes needed
- No frontend changes needed
- Users will simply see unmasked data in approved exports (which is the desired behavior)

**No data migration needed.**

## Testing Considerations

### Unit Tests
- Test `executeExport()` with various SQL statements
- Verify workspace limits are enforced
- Test error handling (connection failures, query errors)
- Verify format conversion (CSV, JSON, SQL, XLSX)

### Integration Tests
- Create DATABASE_EXPORT issue with sensitive data
- Approve the issue
- Verify exported data is NOT masked
- Verify SQL Editor exports still apply masking

### Manual Testing
- Test with actual masked columns in production-like setup
- Verify audit logs capture unmasked exports

## Files Modified

**Changes:**
- `backend/runner/taskrun/data_export_executor.go` - Add export logic, remove api/v1 dependency

**No changes:**
- `backend/api/v1/sql_service.go` - Unchanged
- Proto definitions - Unchanged
- Frontend code - Unchanged

## Future Considerations

### Potential Enhancements
1. Allow approvers to specify which columns to unmask (partial approval)
2. Add approval template conditions that automatically approve certain export patterns
3. Time-limited export access (export link expires after N hours)

### Out of Scope
- GRANT_REQUEST issues (EXPORTER role) - still use masking exception policies
- Query exports from SQL Editor - still masked
- Changing the approval workflow itself

## Summary

DATABASE_EXPORT issues that pass approval workflow will export data without masking. The approval itself authorizes access to the sensitive data in that specific query. This eliminates the confusing dual-gate system (approval + masking exception) for one-time exports while maintaining column-level protection for ongoing access via EXPORTER role.
