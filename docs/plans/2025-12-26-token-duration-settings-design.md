# Token Duration Settings Design

## Overview

Add configurable settings for access token and refresh token durations in workspace settings.

## Background

Bytebase previously supported only "access tokens" which behaved like refresh tokens (long-lived, 7 days default). The existing `token_duration` setting and `FEATURE_SIGN_IN_FREQUENCY_CONTROL` feature flag are legacy naming from this era.

This change properly separates token types with distinct settings:
- **Access token**: Short-lived (1 hour default)
- **Refresh token**: Long-lived (7 days default)

## Design

### Proto Changes

**Store proto** (`proto/store/store/setting.proto`):
```protobuf
message WorkspaceProfileSetting {
  // ... existing fields 1-3 ...

  // RENAME: token_duration → refresh_token_duration (same field number 4)
  google.protobuf.Duration refresh_token_duration = 4;

  // NEW: access_token_duration
  google.protobuf.Duration access_token_duration = 18;

  // ... rest unchanged ...
}
```

**V1 proto** (`proto/v1/v1/setting_service.proto`):
- Same field renames in the API-facing `WorkspaceProfileSetting` message

### Data Migration

SQL migration to rename existing JSON key in stored settings:

```sql
UPDATE setting
SET value = jsonb_set(
  value - 'tokenDuration',
  '{refreshTokenDuration}',
  value->'tokenDuration'
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value ? 'tokenDuration';
```

### Backend Changes

**`backend/api/auth/header.go`**:
- Rename/refactor `GetTokenDuration()` → `GetAccessTokenDuration()`
- Add `GetRefreshTokenDuration()` function
- Both gated by `FEATURE_SIGN_IN_FREQUENCY_CONTROL`

**`backend/api/v1/auth_service.go`**:
- Update token generation to use appropriate duration functions
- Refresh token creation uses `GetRefreshTokenDuration()`

### Frontend Changes

- Update locale files with new i18n keys
- Update settings UI to show both duration fields
- Rename existing field label, add new field

### Defaults

| Token Type | Default Duration |
|------------|------------------|
| Access Token | 1 hour |
| Refresh Token | 7 days |

### Constraints

- No validation limits on durations - admins have full control
- Feature gated by `FEATURE_SIGN_IN_FREQUENCY_CONTROL` (enterprise)

## Implementation Steps

1. Update store proto (`proto/store/store/setting.proto`)
2. Update v1 proto (`proto/v1/v1/setting_service.proto`)
3. Run `cd proto && buf generate`
4. Add database migration for JSON key rename
5. Update `backend/api/auth/header.go` with new duration functions
6. Update `backend/api/v1/auth_service.go` to use new functions
7. Update frontend locales
8. Update frontend settings UI
