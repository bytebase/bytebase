# Migrate ENTERPRISE_LICENSE to SYSTEM Setting

**Date**: 2025-12-05

## Overview

Migrate the `ENTERPRISE_LICENSE` setting from its own database row into the `SYSTEM` setting JSON object, consolidating all internal system settings in one place. This follows the same pattern as the recent `AUTH_SECRET` and `WORKSPACE_ID` migration.

## Current State

- `ENTERPRISE_LICENSE` setting exists as a separate row containing a JWT token string
- `SYSTEM` setting contains: `{"authSecret": "...", "workspaceId": "..."}`
- Both are backend-only (in disallowed settings list)
- License is managed via `backend/enterprise/plugin/hub/hub.go`

## Proposed Changes

### 1. Database Migration

Create `backend/migrator/migration/3.13/0013##migrate_enterprise_license.sql`:

```sql
-- Migrate ENTERPRISE_LICENSE to SYSTEM setting
UPDATE setting
SET value = jsonb_set(
  value::jsonb,
  '{enterpriseLicense}',
  to_jsonb((SELECT value FROM setting WHERE name = 'ENTERPRISE_LICENSE'))
)::TEXT
WHERE name = 'SYSTEM';

DELETE FROM setting WHERE name = 'ENTERPRISE_LICENSE';
```

### 2. Proto Changes

Update `proto/store/store/setting.proto`:

**Add field to SystemSetting**:
```protobuf
message SystemSetting {
  string auth_secret = 1;
  string workspace_id = 2;
  string enterprise_license = 3;  // NEW
}
```

**Remove from enum**:
```protobuf
enum SettingName {
  SETTING_NAME_UNSPECIFIED = 0;
  SYSTEM = 1;
  WORKSPACE_PROFILE = 4;
  WORKSPACE_APPROVAL = 5;
  // 7 was ENTERPRISE_LICENSE, migrated to SYSTEM
  APP_IM = 8;
  AI = 10;
  DATA_CLASSIFICATION = 14;
  SEMANTIC_TYPES = 15;
  PASSWORD_RESTRICTION = 18;
  ENVIRONMENT = 19;
}
```

### 3. Store Layer Changes

Add convenience methods to `backend/store/setting.go`:

```go
func (s *Store) GetEnterpriseLicense(ctx context.Context) (string, error)
func (s *Store) UpsertEnterpriseLicense(ctx context.Context, license string) error
```

These methods handle marshaling/unmarshaling of the SYSTEM setting JSON.

### 4. License Service Changes

Update `backend/enterprise/plugin/hub/hub.go`:
- Replace direct `ENTERPRISE_LICENSE` setting access with new store methods
- `GetSettingV2(ENTERPRISE_LICENSE)` → `GetEnterpriseLicense()`
- `UpsertSettingV2(ENTERPRISE_LICENSE)` → `UpsertEnterpriseLicense()`

### 5. Schema Updates

Update `backend/migrator/migration/LATEST.sql`:
- Remove comment reference to `ENTERPRISE_LICENSE`
- Remove `INSERT INTO setting (name, value) VALUES ('ENTERPRISE_LICENSE', '');`
- Ensure SYSTEM setting initialization includes empty enterprise_license field

### 6. API Layer Cleanup

Update `backend/api/v1/setting_service.go`:
- Remove `ENTERPRISE_LICENSE` from `disallowedSettings` array (only SYSTEM remains)
- Update comment to note enterprise license is now in SYSTEM

Update `backend/api/v1/setting_service_converter.go`:
- Remove `storepb.SettingName_ENTERPRISE_LICENSE` case from converter

### 7. Migrator Test Update

Update `backend/migrator/migrator_test.go`:
- Update `TestLatestVersion` to reflect new migration version

## Testing Strategy

1. Test migration on database with existing ENTERPRISE_LICENSE data
2. Verify license JWT correctly moves to SYSTEM.enterpriseLicense
3. Test GetEnterpriseLicense() and UpsertEnterpriseLicense() methods
4. Run existing test suite
5. Verify license validation through SubscriptionService still works

## Build Commands

```bash
# Proto
cd proto && buf format -w && buf lint && buf generate

# Backend
gofmt -w <modified-files>
golangci-lint run --allow-parallel-runners  # Repeat until clean

# Tests
go test -v -count=1 github.com/bytebase/bytebase/backend/migrator
go test -v -count=1 github.com/bytebase/bytebase/backend/store
```

## Benefits

- Consolidates all internal system settings in one place
- Reduces number of setting rows in database
- Consistent with recent AUTH_SECRET/WORKSPACE_ID migration pattern
- Simpler to manage system-level configuration
