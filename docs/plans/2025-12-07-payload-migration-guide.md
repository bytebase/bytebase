# SQL Review Rule Payload Migration Guide

## Summary

Migrated `SQLReviewRule.payload` from magic JSON string to type-safe proto oneof with 7 distinct message types.

## What Changed

**Before:**
```protobuf
message SQLReviewRule {
  Type type = 1;
  Level level = 2;
  string payload = 3;  // JSON string
  Engine engine = 4;
}
```

**After:**
```protobuf
message SQLReviewRule {
  Type type = 1;
  Level level = 2;
  oneof payload {
    NamingRulePayload naming_payload = 3;
    NumberRulePayload number_payload = 4;
    StringArrayRulePayload string_array_payload = 5;
    CommentConventionRulePayload comment_convention_payload = 6;
    RequiredColumnRulePayload required_column_payload = 7;
    StringRulePayload string_payload = 8;
    NamingCaseRulePayload naming_case_payload = 9;
  }
  Engine engine = 10;
}
```

## Benefits

1. **Type Safety** - Compile-time validation prevents accessing wrong fields
2. **Clear API** - IDE autocomplete shows only valid fields for each rule type
3. **Better Validation** - Each payload type has specific validation
4. **Maintainability** - Clear structure for 109+ rule types

## Breaking Changes

- Proto API: `SQLReviewRule.payload` field changed from `string` to `oneof`
- Go API: Advisors now use `rule.GetXXXPayload()` instead of `UnmarshalXXXPayload()`

## Backward Compatibility

- v1 REST API unchanged - still uses JSON string format
- Database JSONB column unchanged - stores both formats
- Old review configs continue to work during migration period

## Migration Strategy

Application code handles conversion:
- **Reading**: Works with both old JSON strings and new typed protos
- **Writing**: Always writes typed proto format
- **API Layer**: Converts between JSON (v1 API) and proto (internal)

Database migration automatically converts existing data from flat JSON format:
```
Old: {"maxLength": 64, "format": "^[a-z]+$"}
New: {"namingPayload": {"maxLength": 64, "format": "^[a-z]+$"}}
```

No user action required - migration is transparent.

## Payload Types

### 1. NamingRulePayload
Used by naming convention rules (table, column, index names).

**Fields:**
- `max_length` (int32): Maximum allowed length
- `format` (string): Regex pattern or template string

**Rules using this type:**
- NAMING_TABLE
- NAMING_COLUMN
- NAMING_INDEX_PK
- NAMING_INDEX_UK
- NAMING_INDEX_FK
- NAMING_INDEX_IDX
- NAMING_COLUMN_AUTO_INCREMENT
- TABLE_DROP_NAMING_CONVENTION

### 2. NumberRulePayload
Used by rules that enforce numeric limits.

**Fields:**
- `number` (int32): The numeric limit value

**Rules using this type:**
- STATEMENT_INSERT_ROW_LIMIT
- STATEMENT_AFFECTED_ROW_LIMIT
- STATEMENT_MAXIMUM_LIMIT_VALUE
- COLUMN_MAXIMUM_CHARACTER_LENGTH
- INDEX_KEY_NUMBER_LIMIT
- And 12 more...

### 3. StringArrayRulePayload
Used by rules that specify lists of allowed/disallowed items.

**Fields:**
- `list` (repeated string): Array of string values

**Rules using this type:**
- COLUMN_REQUIRED
- COLUMN_TYPE_DISALLOW_LIST
- SYSTEM_CHARSET_ALLOWLIST
- TABLE_DISALLOW_DDL
- And 5 more...

### 4. CommentConventionRulePayload
Used by comment requirement rules.

**Fields:**
- `required` (bool): Whether comment is required
- `max_length` (int32): Maximum comment length

**Rules using this type:**
- TABLE_COMMENT
- COLUMN_COMMENT

### 5. NamingCaseRulePayload
Used by identifier case rules.

**Fields:**
- `upper` (bool): Whether uppercase is required

**Rules using this type:**
- NAMING_IDENTIFIER_CASE

### 6. StringRulePayload
Used by single-string configuration rules.

**Fields:**
- `value` (string): The string value

**Rules using this type:**
- NAMING_FULLY_QUALIFIED

### 7. RequiredColumnRulePayload
Legacy payload type for backward compatibility.

**Fields:**
- `column_list` (repeated string): List of required columns

**Note:** This is converted to StringArrayRulePayload internally.

## File Changes

- Proto: `proto/store/store/review_config.proto`
- Helpers: `backend/plugin/advisor/sql_review.go`
- Advisors: ~105 files in `backend/plugin/advisor/`
- API: `backend/api/v1/review_config_service.go`
- Tests: `backend/plugin/advisor/utils_for_tests.go`
- Migration: `backend/migrator/migration/3.13/0018##migrate_review_config_payload_to_proto.sql`

## Code Examples

### Before (Old String Payload)
```go
format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
if err != nil {
    return nil, err
}
```

### After (Typed Proto Payload)
```go
namingPayload := checkCtx.Rule.GetNamingPayload()
if namingPayload == nil {
    return nil, errors.New("naming_payload is required for naming table rule")
}

format, err := regexp.Compile(namingPayload.Format)
if err != nil {
    return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
}

maxLength := namingPayload.MaxLength
if maxLength == 0 {
    maxLength = advisor.DefaultNameLengthLimit
}
```

## Testing

All existing tests pass with no changes to test behavior. The migration includes:

1. Unit tests for all 105+ advisor files
2. Integration tests for API layer conversion
3. Database migration test
4. Test utilities updated to use typed payloads

## Database Migration

Migration file: `backend/migrator/migration/3.13/0018##migrate_review_config_payload_to_proto.sql`

The migration:
1. Iterates through all review_config records
2. Wraps flat payload JSON in typed field based on rule type
3. Preserves rules with no payload
4. Skips already-migrated rules
5. Runs automatically on server startup

No downtime required - the JSONB column supports both formats.

## Rollback Plan

If issues arise:
1. Revert to previous version - old code can read old format
2. New data will be in typed format but readable by old parsers
3. Future forward migration will re-wrap payloads if needed

## Future Work

1. Remove deprecated unmarshal functions after full migration
2. Add stricter validation for each payload type
3. Consider moving validation rules into proto definitions
4. Explore using proto validation plugins

## References

- Implementation plan: `docs/plans/2025-12-07-migrate-review-config-payload-to-proto.md`
- Proto file: `proto/store/store/review_config.proto`
- Migration file: `backend/migrator/migration/3.13/0018##migrate_review_config_payload_to_proto.sql`
