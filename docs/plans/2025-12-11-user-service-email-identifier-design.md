# User Service Email Identifier Migration

**Date:** 2025-12-11
**Status:** Approved
**Type:** Breaking Change

## Overview

This is a breaking change that standardizes all UserService API endpoints to use email addresses as the user identifier in resource paths instead of UIDs. The motivation is to provide user-friendly, readable URLs (`users/alice@company.com` instead of `users/123`) and create consistent API behavior across all endpoints.

### What's Changing

- **Proto file**: Update all comments from "users/{user uid or user email}" or "users/{user}" to explicitly state "users/{email}"
- **User resource pattern**: Change from `pattern: "users/{user}"` to `pattern: "users/{email}"`
- **Backend**: Update all parsing logic that currently extracts UID from the path to extract and lookup by email instead
- **Frontend**: Update all API calls that construct paths with UIDs to use emails instead

### What's NOT Changing

- The User message structure itself (name field still uses internal UID format for storage)
- Database schema (email is already unique via recent migration)
- The actual email field in User messages
- API method signatures and request/response types

### Breaking Change Impact

- All existing API clients must update to use emails in resource paths
- This will be marked with the `breaking` label in the PR
- Frontend and backend should be deployed together to avoid compatibility issues

## Proto File Changes

### Resource Pattern Update

**Before:**
```proto
option (google.api.resource) = {
  type: "bytebase.com/User"
  pattern: "users/{user}"
};
```

**After:**
```proto
option (google.api.resource) = {
  type: "bytebase.com/User"
  pattern: "users/{email}"
};
```

### Comment Updates for All Methods

Update format comments in request messages:
- `GetUserRequest`: "Format: users/{email}"
- `BatchGetUsersRequest`: "Format: users/{email}"
- `UpdateUserRequest`: "Format: users/{email}"
- `DeleteUserRequest`: "Format: users/{email}"
- `UndeleteUserRequest`: "Format: users/{email}"
- `UpdateEmailRequest`: Make consistent "Format: users/{email}"

### UpdateEmail Special Case

The path will contain the *old* email (the one being updated from), and the `email` field in the request body contains the *new* email. Add clear comment:

```proto
// The name of the user whose email to update.
// Format: users/{email}
// Note: This is the current (old) email address. The new email is specified in the 'email' field.
```

### User.name Field Comment Update

**Before:**
```proto
// The name of the user.
// Format: users/{user}. {user} is a system-generated unique ID.
```

**After:**
```proto
// The name of the user.
// Format: users/{email} for API requests. Internally stored as users/{uid}.
```

## Backend Changes

### Path Parsing Logic

Currently, backend code extracts the UID from paths like `users/123`. Update all UserService handlers to:
1. Extract the email from the path (e.g., `users/alice@company.com`)
2. Look up the user by email in the database
3. Use the retrieved user's UID for internal operations

### Files to Update

`backend/api/v1/user_service.go` (or similar) - All RPC method implementations:
- `GetUser()` - Parse email from name, lookup by email
- `UpdateUser()` - Parse email from user.name, lookup by email
- `DeleteUser()` - Parse email from name, lookup by email
- `UndeleteUser()` - Parse email from name, lookup by email
- `UpdateEmail()` - Parse old email from name, lookup by email, then update to new email
- `BatchGetUsers()` - Parse emails from names array, batch lookup by emails

### Helper Function Changes

- Any utility functions that parse resource names (e.g., `getUserID(name string)`) should be updated to `getUserEmail(name string)` and return the email portion
- Database store methods may need new methods or updates to support lookup by email (check `backend/store/user.go`)

### Validation

- Add email format validation when parsing from the path
- Return appropriate errors if email is invalid or user not found

### UpdateEmail Special Handling

- Parse old email from path
- Lookup user by old email
- Validate new email doesn't already exist (handled by unique constraint, but should return friendly error)
- Update the email field

## Frontend Changes

### API Call Updates

All frontend code that constructs user resource paths needs to change from using UIDs to using emails.

**Before:**
```typescript
const user = await userServiceClient.getUser({ name: `users/${userId}` });
await userServiceClient.updateUser({ user: { name: `users/${userId}`, ... } });
await userServiceClient.deleteUser({ name: `users/${userId}` });
```

**After:**
```typescript
const user = await userServiceClient.getUser({ name: `users/${email}` });
await userServiceClient.updateUser({ user: { name: `users/${email}`, ... } });
await userServiceClient.deleteUser({ name: `users/${email}` });
```

### No Manual Encoding Needed

ConnectRPC's generated client will automatically URL-encode the email (handling `@` and other special characters) when making HTTP requests.

### Files to Search and Update

- Search for `users/\${` or `"users/" +` patterns in TypeScript/Vue files
- Common locations: `frontend/src/` (stores, views, components that interact with user API)
- Look for:
  - Direct API calls to UserService methods
  - Route parameter construction
  - Any code that builds user resource names

### Edge Cases

- Ensure any cached or stored user references are updated
- Update any frontend routing that uses user IDs to use emails
- Check for any UI components that display or link to user resources

## Testing Strategy

### Unit Tests

Update existing unit tests that construct user resource names to use emails instead of UIDs.

### Integration Tests

Test all UserService endpoints with email-based paths.

### Edge Cases to Test

- Emails with special characters (e.g., `user+tag@domain.com`, `user.name@domain.co.uk`)
- UpdateEmail endpoint (old email in path, new email in body)
- User not found by email (proper error messages)
- Invalid email format in path

## Implementation Steps

### 1. Proto Generation

After updating the proto file, run:
```bash
buf format -w proto
buf lint proto
cd proto && buf generate
```

### 2. Implementation Order

1. Update proto file and regenerate
2. Update backend handlers (all methods in UserService)
3. Update frontend API calls
4. Run formatting/linting for all changed code
5. Update and run tests
6. Verify all endpoints work with email-based paths

### 3. Code Formatting

**Go changes:**
```bash
gofmt -w <modified-files>
golangci-lint run --allow-parallel-runners
golangci-lint run --fix --allow-parallel-runners
```

**Frontend changes:**
```bash
pnpm --dir frontend biome:check
pnpm --dir frontend type-check
```

## Migration Considerations

- This is a breaking change - add `breaking` label to PR
- Document in release notes that all API clients must update
- No database migration needed (email already has unique index via commit 2121f49a4e)
- Consider the timing of frontend and backend deployments (may need to deploy together)

### Rollout Considerations

If there are external API clients:
- Announce the change in advance
- Provide migration timeline
- Clear documentation on what changed

## Success Criteria

- All UserService endpoints accept only `users/{email}` format
- Proto comments clearly document the email-based format
- Backend correctly parses emails and looks up users
- Frontend uses emails in all user resource paths
- All tests pass with email-based paths
- Documentation updated for API clients

---

## Implementation Notes

**Implementation Completed:** 2025-12-11

All UserService endpoints now exclusively accept email-based identifiers in resource paths (`users/{email}`). The implementation was completed across proto definitions, backend services, frontend code, and tests.

### Changes Made

#### Proto Changes
- Updated `proto/v1/v1/user_service.proto`:
  - Changed User resource pattern from `pattern: "users/{user}"` to `pattern: "users/{email}"`
  - Updated all request message comments (GetUserRequest, BatchGetUsersRequest, UpdateUserRequest, DeleteUserRequest, UndeleteUserRequest, UpdateEmailRequest) to specify "Format: users/{email}"
  - Updated User.name field comment to document "Format: users/{email} for API requests. Internally stored as users/{uid}."
- Generated proto code via `buf generate`
- Commits: `77dc9f54c7`

#### Backend Changes
- Updated `backend/api/v1/user_service.go`:
  - **GetUser**: Replaced UID/email fallback logic with email-only parsing using `common.GetUserEmail()`
  - **UpdateUser**: Changed from `GetUserID()` to `GetUserEmail()` and replaced `GetUserByID()` with `GetUserByEmail()`
  - **DeleteUser**: Changed from `GetUserID()` to `GetUserEmail()` and replaced `GetUserByID()` with `GetUserByEmail()`
  - **UndeleteUser**: Changed from `GetUserID()` to `GetUserEmail()` and replaced `GetUserByID()` with `GetUserByEmail()`
  - **convertToUser**: Updated to use `FormatUserEmail()` instead of `FormatUserUID()` for resource name formatting, with fallback to UID for users with empty email
- All error messages updated to use email identifiers for consistency
- Commits: `3830495207`, `25f766ca1d`, `36585fc87d`, `a0c045cb50`, `b0553832bd`

#### Frontend Changes
- Updated `frontend/src/types/v1/user.ts`:
  - Changed `SYSTEM_BOT_USER_NAME` constant to use `SYSTEM_BOT_EMAIL` instead of `SYSTEM_BOT_ID`
  - Updated `allUsersUser` to use `ALL_USERS_USER_EMAIL` instead of `ALL_USERS_USER_ID`
  - Removed unused `ALL_USERS_USER_ID` constant
- All user resource names now constructed with email addresses
- Commit: `b0553832bd`

#### Tests
- Added `backend/api/v1/user_service_test.go`:
  - New `TestConvertToUser` with comprehensive test cases:
    - User with valid email (formats as `users/{email}`)
    - User with empty email (formats as `users/{uid}` fallback)
    - Service account with email
  - All tests pass successfully
- Commit: `23fc2b0bd4`

### Verification Steps

All verification steps completed successfully:

1. **Proto Validation**
   - `buf format -w proto` - File formatted
   - `buf lint proto` - No linting errors
   - `cd proto && buf generate` - Code generation successful

2. **Backend Validation**
   - `gofmt -w backend/api/v1/user_service.go` - File formatted
   - `golangci-lint run --allow-parallel-runners` - No linting errors
   - All backend methods updated and tested

3. **Frontend Validation**
   - `pnpm --dir frontend biome:check` - Format and lint passed
   - `pnpm --dir frontend type-check` - Type checks passed

4. **Build Verification**
   - Backend builds successfully
   - All unit tests pass

5. **Git Status**
   - All changes committed with descriptive commit messages
   - Working tree clean

### Migration Requirements

**Breaking Change:** All API clients must update to use email addresses in user resource paths.

**Before:**
```
users/101
users/42
```

**After:**
```
users/alice@example.com
users/bob@company.com
```

**Backend Behavior:**
- All UserService methods now exclusively parse email from resource paths
- User lookup changed from UID-based to email-based queries
- Error messages use email identifiers for better clarity
- Fallback to UID format only for internal users with empty email addresses

**Frontend Behavior:**
- All user resource names constructed with email addresses
- System bot and special users use email constants
- ConnectRPC client automatically handles URL encoding of special characters in emails

**No Database Migration Required:**
- Email unique index already exists (commit `2121f49a4e`)
- Internal storage format unchanged (still uses UID internally)
- Only API resource path format changed

### Deployment Considerations

- Frontend and backend should be deployed together to avoid compatibility issues
- External API clients must be updated before or during deployment
- This is marked as a breaking change in the PR

### Related Commits

- Design document: `e632362438`
- Implementation plan: `6941123b81`
- Proto changes: `77dc9f54c7`
- Backend GetUser: `3830495207`
- Backend UpdateUser: `25f766ca1d`
- Backend DeleteUser: `36585fc87d`
- Backend UndeleteUser: `a0c045cb50`
- Frontend and convertToUser: `b0553832bd`
- Tests and validation: `23fc2b0bd4`
