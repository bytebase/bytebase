# Dedicated Email Update API Design

**Date:** 2025-12-10
**Status:** Approved

## Overview

This design introduces a dedicated `UpdateEmail` RPC method under UserService with a new `bb.users.updateEmail` permission. This separates email update functionality from the general `UpdateUser` operation, simplifying the data model and providing clearer permission boundaries.

## Motivation

Allowing unrestricted email updates creates architectural complexity:
- Requires maintaining user ID to email mappings
- Adds read latency from metadata store lookups
- Complicates HA architecture design

Making email updates admin-only (via dedicated permission) simplifies the architecture while still allowing admins to handle legitimate email change requests.

## API Definition

### Proto Changes

**File:** `proto/v1/v1/user_service.proto`

Add new RPC method to UserService:

```proto
// Updates a user's email address.
// Permissions required: bb.users.updateEmail
rpc UpdateEmail(UpdateEmailRequest) returns (User) {
  option (google.api.http) = {
    post: "/v1/{name=users/*}:updateEmail"
    body: "*"
  };
  option (google.api.method_signature) = "name,email";
  option (bytebase.v1.permission) = "bb.users.updateEmail";
  option (bytebase.v1.auth_method) = IAM;
  option (bytebase.v1.audit) = true;
}
```

Add new request message:

```proto
message UpdateEmailRequest {
  // The name of the user whose email to update.
  // Format: users/{user}
  string name = 1 [
    (google.api.field_behavior) = REQUIRED,
    (google.api.resource_reference) = {type: "bytebase.com/User"}
  ];

  // The new email address.
  string email = 2 [(google.api.field_behavior) = REQUIRED];
}
```

### Permission Changes

**File:** `backend/component/iam/permission.go`

Add new permission constant:

```go
PermissionUsersUpdateEmail Permission = "bb.users.updateEmail"
```

Add to `allPermissions` array (after `PermissionUsersUpdate`).

### Role Configuration

**File:** `backend/component/iam/acl.yaml`

Add `bb.users.updateEmail` to the `workspaceAdmin` role permissions (after line 114):

```yaml
- bb.users.update
- bb.users.updateEmail
```

## Backend Implementation

### Validation Logic

The `UpdateEmail` method performs the following validations (consistent with `CreateUser` and `UpdateUser`):

1. Permission check - Verify caller has `bb.users.updateEmail` (handled by IAM middleware)
2. User existence - Verify target user exists and is not deleted
3. Email format validation - Validate email format using existing validation utility
4. Uniqueness check - Ensure new email doesn't already exist in the system
5. No-op check - Return error if new email matches current email
6. Workspace domain validation - If workspace has allowed domain restrictions, validate new email matches

### Database Update

- Single database transaction to update `email` field in user table
- Return updated User object

### Error Responses

- `INVALID_ARGUMENT` - Invalid email format or same as current email
- `ALREADY_EXISTS` - Email already in use by another user
- `FAILED_PRECONDITION` - Email domain not allowed by workspace policy
- `NOT_FOUND` - User not found
- `PERMISSION_DENIED` - Caller lacks `bb.users.updateEmail` permission

### Audit Logging

The method includes `option (bytebase.v1.audit) = true` to record all email changes in audit logs. No email notifications are sent (infrastructure not currently available).

## Frontend Integration

### ProfileDashboard Changes

**File:** `frontend/src/views/ProfileDashboard.vue`

1. Update `allowEditEmail` computed property to check `bb.users.updateEmail` instead of `bb.policies.update`

2. Modify `saveEdit()` function to:
   - Detect if email was modified
   - If email changed: Call `UpdateEmail` RPC with `{name, email}`
   - If other fields changed: Call `UpdateUser` RPC (without email in `update_mask`)
   - Handle both calls if both email and other fields changed

3. Add error handling for email-specific validation failures

### API Client Generation

Run `cd proto && buf generate` to generate TypeScript client with new `updateEmail` method.

## Testing Strategy

### Backend Unit Tests

- Valid email updates succeed
- Invalid email formats rejected
- Duplicate emails rejected
- Same email (no-op) rejected
- Workspace domain validation enforced
- Permission enforcement verified

### Integration Tests

- End-to-end: Admin updates user email via API
- Verify audit log entry created
- Verify user can login with new email
- Verify old email no longer works for login

### Frontend Tests

- Email field only editable by users with `bb.users.updateEmail`
- Save flow calls correct API
- Error messages display correctly

## Implementation Order

1. Proto changes (`proto/v1/v1/user_service.proto`)
2. Permission constant (`backend/component/iam/permission.go`)
3. Role configuration (`backend/component/iam/acl.yaml`)
4. Backend service implementation (UserService handler)
5. Run `buf generate` to generate Go and TypeScript clients
6. Backend tests
7. Frontend permission check update
8. Frontend API integration
9. Integration tests

## Backward Compatibility

This is a non-breaking, additive-only change:
- Existing `UpdateUser` RPC continues to work unchanged
- No data migration required
- New permission only required for new API endpoint

## Files to Modify

- `proto/v1/v1/user_service.proto` - Add RPC and message
- `backend/component/iam/permission.go` - Add permission constant
- `backend/component/iam/acl.yaml` - Add permission to workspaceAdmin
- Backend UserService implementation - Add UpdateEmail handler
- `frontend/src/views/ProfileDashboard.vue` - Update permission check and save logic
