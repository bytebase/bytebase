# System Setting Migration Design

## Overview

Migrate the existing string-based settings `AUTH_SECRET` and `WORKSPACE_ID` into a new structured `SystemSetting` message. This consolidates system-level configuration (infrastructure concerns set during initialization) separate from workspace-level configuration (user-configurable settings in `WorkspaceProfileSetting`).

## Motivation

- **Semantic clarity**: `AUTH_SECRET` and `WORKSPACE_ID` are system-level, infrastructure concerns that differ from user-configurable workspace settings
- **Consistency**: Follows the established pattern of consolidating related settings into structured messages
- **Organization**: Clear separation between system-level (immutable, infrastructure) and workspace-level (user-configurable) settings
- **Future-proofing**: Provides a natural location for future system-level configs (deployment IDs, instance metadata, etc.)

## Proto Changes

**Location:** `proto/store/store/setting.proto`

### Add SystemSetting Message

```protobuf
message SystemSetting {
  // Authentication secret for token signing (32-character random string).
  string auth_secret = 1;
  // Unique workspace identifier (UUID).
  string workspace_id = 2;
}
```

### Update SettingName Enum

Remove `AUTH_SECRET = 1` and `WORKSPACE_ID = 3`, add `SYSTEM = 1`:

```protobuf
enum SettingName {
  SETTING_NAME_UNSPECIFIED = 0;
  SYSTEM = 1;
  WORKSPACE_PROFILE = 4;
  WORKSPACE_APPROVAL = 5;
  ENTERPRISE_LICENSE = 7;
  APP_IM = 8;
  AI = 10;
  DATA_CLASSIFICATION = 14;
  SEMANTIC_TYPES = 15;
  PASSWORD_RESTRICTION = 18;
  ENVIRONMENT = 19;
}
```

## Backend Changes

### Add Store Method

**Location:** `backend/store/setting.go`

Add new getter following the pattern of `GetWorkspaceGeneralSetting`:

```go
func (s *Store) GetSystemSetting(ctx context.Context) (*storepb.SystemSetting, error) {
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_SYSTEM)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_SYSTEM)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_SYSTEM)
	}

	payload := new(storepb.SystemSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), payload); err != nil {
		return nil, err
	}
	return payload, nil
}
```

### Update Call Sites

Update all call sites of `GetWorkspaceID()` and `GetSecret()` to use `GetSystemSetting()`:

```go
// Old pattern:
workspaceID, err := s.store.GetWorkspaceID(ctx)

// New pattern:
systemSetting, err := s.store.GetSystemSetting(ctx)
if err != nil {
    return err
}
workspaceID := systemSetting.WorkspaceId
```

```go
// Old pattern:
secret, err := s.store.GetSecret(ctx)

// New pattern:
systemSetting, err := s.store.GetSystemSetting(ctx)
if err != nil {
    return err
}
secret := systemSetting.AuthSecret
```

After updating all call sites, remove the `GetWorkspaceID()` and `GetSecret()` methods from `backend/store/setting.go`.

### Update SaaS Feature Control

**Location:** `backend/component/config/profile.go`

Update the feature control map:

```go
var saasFeatureControlMap = map[string]bool{
	storepb.SettingName_SYSTEM.String(): true,
	storepb.SettingName_AI.String():      true,
}
```

### Update API/Converter Files

**Location:** `backend/api/v1/setting_service.go` and `backend/api/v1/setting_service_converter.go`

Remove any special handling for `AUTH_SECRET` or `WORKSPACE_ID` enum values and replace with `SYSTEM` handling where needed.

## Database Changes

### Migration SQL

**Location:** `backend/migrator/prod/<<version>>/<<timestamp>>##migrate_system_setting.sql`

```sql
-- Migrate AUTH_SECRET and WORKSPACE_ID to SYSTEM setting
INSERT INTO setting (name, value)
SELECT
  'SYSTEM',
  json_build_object(
    'authSecret', (SELECT value FROM setting WHERE name = 'AUTH_SECRET'),
    'workspaceId', (SELECT value FROM setting WHERE name = 'WORKSPACE_ID')
  )::TEXT;

DELETE FROM setting WHERE name IN ('AUTH_SECRET', 'WORKSPACE_ID');
```

### Update LATEST.sql

**Location:** `backend/migrator/migration/LATEST.sql`

Replace the current AUTH_SECRET and WORKSPACE_ID initialization (lines 566-571):

```sql
-- Initialize SYSTEM setting with auth_secret and workspace_id
INSERT INTO setting (name, value)
VALUES (
  'SYSTEM',
  json_build_object(
    'authSecret', (SELECT string_agg(substr('0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ', floor(random() * 62 + 1)::int, 1), '')
     FROM generate_series(1, 32)),
    'workspaceId', gen_random_uuid()::text
  )::TEXT
);
```

Update the table comment (line 36):

```sql
    -- name: SYSTEM, WORKSPACE_PROFILE, WORKSPACE_APPROVAL,
    -- ENTERPRISE_LICENSE, APP_IM, AI,
    -- DATA_CLASSIFICATION, SEMANTIC_TYPES, PASSWORD_RESTRICTION, ENVIRONMENT
```

## Additional Tasks

1. Update `TestLatestVersion` in `backend/migrator/migrator_test.go` to increment expected version
2. Run `cd proto && buf generate` to regenerate Go code
3. Run `gofmt -w` on modified Go files
4. Run `golangci-lint run --allow-parallel-runners` repeatedly until clean

## Files Affected

- `proto/store/store/setting.proto`
- `backend/store/setting.go`
- `backend/api/v1/setting_service.go`
- `backend/api/v1/setting_service_converter.go`
- `backend/component/config/profile.go`
- `backend/migrator/prod/<<version>>/<<timestamp>>##migrate_system_setting.sql` (new)
- `backend/migrator/migration/LATEST.sql`
- `backend/migrator/migrator_test.go`
- Generated proto files (via `buf generate`)

## Testing Strategy

1. Unit tests for `GetSystemSetting()`
2. Integration test for migration SQL (verify old data migrates correctly)
3. Manual testing: verify existing deployments can upgrade and system continues to function
4. Verify all existing tests pass after changes
