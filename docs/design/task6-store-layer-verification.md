# Task 6: Store Layer WorkloadIdentityConfig Verification

## Summary

The store layer at `/Users/rebeliceyang/Github/bytebase/backend/store/principal.go` **already properly handles WorkloadIdentityConfig** for WORKLOAD_IDENTITY type principals without requiring any code modifications.

## Key Findings

### 1. UserMessage Struct (Lines 51-68)
**Status: ✅ No changes needed**

The `UserMessage` struct has a `Profile *storepb.UserProfile` field. Since `UserProfile` was extended in Task 7 to include `workload_identity_config` (line 55 of `proto/store/store/user.proto`), the WorkloadIdentityConfig is automatically accessible via:

```go
user.Profile.WorkloadIdentityConfig
```

### 2. listUserImpl Function (Lines 211-350)
**Status: ✅ No changes needed**

The function reads and unmarshals the profile JSONB column:
- Line 278: Scans `principal.profile` into `profileBytes`
- Lines 337-341: Unmarshals to `storepb.UserProfile` using protojson

```go
profile := storepb.UserProfile{}
if err := common.ProtojsonUnmarshaler.Unmarshal(profileBytes, &profile); err != nil {
    return nil, err
}
userMessage.Profile = &profile
```

Since `UserProfile` proto now includes `workload_identity_config` field, the unmarshaling automatically includes WorkloadIdentityConfig without any code changes.

### 3. CreateUser Function (Lines 353-415)
**Status: ✅ No changes needed**

The function marshals the profile to JSONB:
- Lines 366-368: Initializes profile if nil
- Lines 369-372: Marshals using protojson
- Line 381: Stores in database

```go
if create.Profile == nil {
    create.Profile = &storepb.UserProfile{}
}
profileBytes, err := protojson.Marshal(create.Profile)
```

When creating a workload identity user with `Profile.WorkloadIdentityConfig` set, the marshaling automatically includes it in the JSONB without code changes.

### 4. UpdateUser Function (Lines 418-491)
**Status: ✅ No changes needed**

The function updates the profile:
- Lines 450-456: Marshals and updates if profile is provided

```go
if v := patch.Profile; v != nil {
    profileBytes, err := protojson.Marshal(v)
    if err != nil {
        return nil, err
    }
    set.Comma("profile = ?", profileBytes)
}
```

Updating WorkloadIdentityConfig works automatically through the profile update mechanism.

## How It Works

The store layer uses Protocol Buffers' JSON marshaling (`protojson`) to serialize/deserialize the `UserProfile` message to/from PostgreSQL's JSONB column. When `UserProfile` was extended to include `workload_identity_config` in Task 7:

```protobuf
message UserProfile {
  google.protobuf.Timestamp last_login_time = 1;
  google.protobuf.Timestamp last_change_password_time = 2;
  string source = 3;
  WorkloadIdentityConfig workload_identity_config = 4;  // Added in Task 7
}
```

The protojson marshaler/unmarshaler automatically handles the new field without requiring any changes to the store layer code.

## Database Storage Format

The WorkloadIdentityConfig is stored in the `principal.profile` JSONB column with the following structure:

```json
{
  "lastLoginTime": "2024-12-11T10:00:00Z",
  "lastChangePasswordTime": "2024-12-11T09:00:00Z",
  "source": "",
  "workloadIdentityConfig": {
    "providerType": "PROVIDER_GITHUB",
    "issuerUrl": "https://token.actions.githubusercontent.com",
    "allowedAudiences": ["https://github.com/myorg"],
    "subjectPattern": "repo:myorg/myrepo:ref:refs/heads/main"
  }
}
```

Note: protojson uses camelCase for field names (e.g., `workloadIdentityConfig` instead of `workload_identity_config`).

## Testing

Added comprehensive tests in `/Users/rebeliceyang/Github/bytebase/backend/store/principal_workload_identity_test.go`:

1. **TestWorkloadIdentityConfigInUserMessage**: Verifies WorkloadIdentityConfig is accessible through UserMessage.Profile
2. **TestCreateUserMessageWithWorkloadIdentityConfig**: Verifies CreateUser can handle WorkloadIdentityConfig
3. **TestUpdateUserMessageWithWorkloadIdentityConfig**: Verifies UpdateUser can handle WorkloadIdentityConfig updates

All tests pass successfully:
```
=== RUN   TestWorkloadIdentityConfigInUserMessage
--- PASS: TestWorkloadIdentityConfigInUserMessage (0.00s)
PASS
ok  	github.com/bytebase/bytebase/backend/store	1.562s
```

## Verification

1. ✅ Code compiles without errors
2. ✅ All tests pass
3. ✅ golangci-lint reports 0 issues
4. ✅ gofmt formatting applied

## Conclusion

**No modifications were needed to the store layer.** The existing profile handling through protojson marshaling/unmarshaling automatically supports WorkloadIdentityConfig. The store layer is ready to support WORKLOAD_IDENTITY type principals with their configuration.

## Files Changed

- ✅ `/Users/rebeliceyang/Github/bytebase/backend/store/principal_workload_identity_test.go` (created)
- ✅ `/Users/rebeliceyang/Github/bytebase/backend/store/principal.go` (no changes - verification only)

## Next Steps

The store layer is complete. Proceed to Task 8 (UserService updates) to handle workload identity CRUD operations at the API layer.
