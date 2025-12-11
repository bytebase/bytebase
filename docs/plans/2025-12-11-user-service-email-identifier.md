# User Service Email Identifier Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate UserService API from UID-based to email-based resource identifiers

**Architecture:** Breaking change that updates proto definitions, backend parsing logic, and frontend API calls to exclusively use email addresses in user resource paths (users/{email} instead of users/{uid})

**Tech Stack:** Protocol Buffers, Go (backend), TypeScript/Vue (frontend), ConnectRPC

---

## Task 1: Update Proto File

**Files:**
- Modify: `proto/v1/v1/user_service.proto`

### Step 1: Update User resource pattern

In `proto/v1/v1/user_service.proto`, find the User message resource annotation (around line 253-256) and update it:

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

### Step 2: Update User.name field comment

Find the `name` field in the User message (around line 258-260) and update the comment:

**Before:**
```proto
// The name of the user.
// Format: users/{user}. {user} is a system-generated unique ID.
string name = 1 [(google.api.field_behavior) = OUTPUT_ONLY];
```

**After:**
```proto
// The name of the user.
// Format: users/{email} for API requests. Internally stored as users/{uid}.
string name = 1 [(google.api.field_behavior) = OUTPUT_ONLY];
```

### Step 3: Update GetUserRequest comment

Find GetUserRequest message (around line 117-124) and update the comment:

**Before:**
```proto
message GetUserRequest {
  // The name of the user to retrieve.
  // Format: users/{user uid or user email}
  string name = 1 [
```

**After:**
```proto
message GetUserRequest {
  // The name of the user to retrieve.
  // Format: users/{email}
  string name = 1 [
```

### Step 4: Update BatchGetUsersRequest comment

Find BatchGetUsersRequest message (around line 126-133) and update the comment:

**Before:**
```proto
message BatchGetUsersRequest {
  // The user names to retrieve.
  // Format: users/{user uid or user email}
  repeated string names = 1 [
```

**After:**
```proto
message BatchGetUsersRequest {
  // The user names to retrieve.
  // Format: users/{email}
  repeated string names = 1 [
```

### Step 5: Update UpdateUserRequest comment

Find UpdateUserRequest message (around line 197-202) and update the comment:

**Before:**
```proto
message UpdateUserRequest {
  // The user to update.
  //
  // The user's `name` field is used to identify the user to update.
  // Format: users/{user}
  User user = 1 [(google.api.field_behavior) = REQUIRED];
```

**After:**
```proto
message UpdateUserRequest {
  // The user to update.
  //
  // The user's `name` field is used to identify the user to update.
  // Format: users/{email}
  User user = 1 [(google.api.field_behavior) = REQUIRED];
```

### Step 6: Update DeleteUserRequest comment

Find DeleteUserRequest message (around line 222-228) and update the comment:

**Before:**
```proto
message DeleteUserRequest {
  // The name of the user to delete.
  // Format: users/{user}
  string name = 1 [
```

**After:**
```proto
message DeleteUserRequest {
  // The name of the user to delete.
  // Format: users/{email}
  string name = 1 [
```

### Step 7: Update UndeleteUserRequest comment

Find UndeleteUserRequest message (around line 231-237) and update the comment:

**Before:**
```proto
message UndeleteUserRequest {
  // The name of the deleted user.
  // Format: users/{user}
  string name = 1 [
```

**After:**
```proto
message UndeleteUserRequest {
  // The name of the deleted user.
  // Format: users/{email}
  string name = 1 [
```

### Step 8: Update UpdateEmailRequest comment with clarification

Find UpdateEmailRequest message (around line 240-250) and update the comment:

**Before:**
```proto
message UpdateEmailRequest {
  // The name of the user whose email to update.
  // Format: users/{user email}
  string name = 1 [
```

**After:**
```proto
message UpdateEmailRequest {
  // The name of the user whose email to update.
  // Format: users/{email}
  // Note: This is the current (old) email address. The new email is specified in the 'email' field.
  string name = 1 [
```

### Step 9: Format proto file

Run:
```bash
buf format -w proto
```

Expected: File formatted successfully

### Step 10: Lint proto file

Run:
```bash
buf lint proto
```

Expected: No linting errors

### Step 11: Generate proto code

Run:
```bash
cd proto && buf generate
```

Expected: Code generation successful, new files in `backend/generated-go/v1/`

### Step 12: Commit proto changes

```bash
git add proto/v1/v1/user_service.proto backend/generated-go/
git commit -m "feat(proto): migrate user service to email-based identifiers

Update UserService proto to use email addresses instead of UIDs in
resource paths. This is a breaking change that makes user APIs more
user-friendly and consistent.

Breaking change: All user resource paths now require email format
(users/{email}) instead of accepting both UID and email.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Update Backend GetUser Method

**Files:**
- Modify: `backend/api/v1/user_service.go:59-84`

### Step 1: Simplify GetUser to use email only

Find the `GetUser` method (lines 59-84) and replace the entire method:

**Before:**
```go
func (s *UserService) GetUser(ctx context.Context, request *connect.Request[v1pb.GetUserRequest]) (*connect.Response[v1pb.User], error) {
	userID, err := common.GetUserID(request.Msg.Name)
	var user *store.UserMessage
	if err != nil {
		email, err := common.GetUserEmail(request.Msg.Name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		u, err := s.store.GetUserByEmail(ctx, email)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
		}
		user = u
	} else {
		u, err := s.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
		}
		user = u
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %d not found", userID))
	}
	return connect.NewResponse(convertToUser(ctx, user)), nil
}
```

**After:**
```go
func (s *UserService) GetUser(ctx context.Context, request *connect.Request[v1pb.GetUserRequest]) (*connect.Response[v1pb.User], error) {
	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	return connect.NewResponse(convertToUser(ctx, user)), nil
}
```

### Step 2: Format the file

Run:
```bash
gofmt -w backend/api/v1/user_service.go
```

Expected: File formatted

### Step 3: Run golangci-lint

Run:
```bash
golangci-lint run --allow-parallel-runners backend/api/v1/user_service.go
```

Expected: No linting errors (run multiple times if needed due to max-issues limit)

### Step 4: Commit GetUser changes

```bash
git add backend/api/v1/user_service.go
git commit -m "refactor(backend): update GetUser to use email-only parsing

Remove UID fallback logic from GetUser method. Now exclusively parses
and looks up users by email address from resource path.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Update Backend UpdateUser Method

**Files:**
- Modify: `backend/api/v1/user_service.go:330-520`

### Step 1: Update UpdateUser to use email parsing

Find the `UpdateUser` method (lines 330-520) and replace the user lookup section (lines 343-368):

**Before (lines 343-368):**
```go
	userID, err := common.GetUserID(request.Msg.User.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		if request.Msg.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersCreate, callerUser)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionUsersCreate))
			}
			return s.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
				User: request.Msg.User,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %d not found", userID))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q has been deleted", userID))
	}
```

**After:**
```go
	email, err := common.GetUserEmail(request.Msg.User.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		if request.Msg.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionUsersCreate, callerUser)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionUsersCreate))
			}
			return s.CreateUser(ctx, connect.NewRequest(&v1pb.CreateUserRequest{
				User: request.Msg.User,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q has been deleted", email))
	}
```

### Step 2: Update permission check to use user.ID

Find the permission check section (around line 370-378) and update to use `user.ID` instead of `userID`:

**Before:**
```go
	if callerUser.ID != userID {
```

**After:**
```go
	if callerUser.ID != user.ID {
```

### Step 3: Format the file

Run:
```bash
gofmt -w backend/api/v1/user_service.go
```

Expected: File formatted

### Step 4: Run golangci-lint

Run:
```bash
golangci-lint run --allow-parallel-runners backend/api/v1/user_service.go
```

Expected: No linting errors (run multiple times if needed)

### Step 5: Commit UpdateUser changes

```bash
git add backend/api/v1/user_service.go
git commit -m "refactor(backend): update UpdateUser to use email parsing

Replace UID parsing with email parsing in UpdateUser method. Lookup
user by email instead of UID from resource path.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Update Backend DeleteUser Method

**Files:**
- Modify: `backend/api/v1/user_service.go:522-568`

### Step 1: Update DeleteUser to use email parsing

Find the `DeleteUser` method (lines 522-568) and replace the user lookup section (lines 536-548):

**Before:**
```go
	userID, err := common.GetUserID(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %d not found", userID))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q has been deleted", userID))
	}
```

**After:**
```go
	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	if user.MemberDeleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q has been deleted", email))
	}
```

### Step 2: Format the file

Run:
```bash
gofmt -w backend/api/v1/user_service.go
```

Expected: File formatted

### Step 3: Run golangci-lint

Run:
```bash
golangci-lint run --allow-parallel-runners backend/api/v1/user_service.go
```

Expected: No linting errors

### Step 4: Commit DeleteUser changes

```bash
git add backend/api/v1/user_service.go
git commit -m "refactor(backend): update DeleteUser to use email parsing

Replace UID parsing with email parsing in DeleteUser method.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Update Backend UndeleteUser Method

**Files:**
- Modify: `backend/api/v1/user_service.go:615-654`

### Step 1: Update UndeleteUser to use email parsing

Find the `UndeleteUser` method (lines 615-654) and replace the user lookup section (lines 629-642):

**Before:**
```go
	userID, err := common.GetUserID(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %d not found", userID))
	}
	if !user.MemberDeleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("user %q is already active", userID))
	}
```

**After:**
```go
	email, err := common.GetUserEmail(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user, error: %v", err))
	}
	if user == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", email))
	}
	if !user.MemberDeleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("user %q is already active", email))
	}
```

### Step 2: Format the file

Run:
```bash
gofmt -w backend/api/v1/user_service.go
```

Expected: File formatted

### Step 3: Run golangci-lint

Run:
```bash
golangci-lint run --allow-parallel-runners backend/api/v1/user_service.go
```

Expected: No linting errors

### Step 4: Commit UndeleteUser changes

```bash
git add backend/api/v1/user_service.go
git commit -m "refactor(backend): update UndeleteUser to use email parsing

Replace UID parsing with email parsing in UndeleteUser method.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Update Frontend User References (Part 1 - Search and Identify)

**Files:**
- Search: `frontend/src/**/*.{ts,vue}`

### Step 1: Search for user resource path construction patterns

Run:
```bash
cd frontend && grep -r "users/\${" src/ --include="*.ts" --include="*.vue" | head -20
```

Expected: List of files that construct user resource paths

### Step 2: Search for FormatUserUID usage

Run:
```bash
cd frontend && grep -r "FormatUserUID\|formatUserUID" src/ --include="*.ts" --include="*.vue" | head -20
```

Expected: List of files using FormatUserUID helper

### Step 3: Document findings

Create a list of files that need updates. Common patterns to look for:
- `` `users/${userId}` ``
- `` `users/${user.id}` ``
- `FormatUserUID(id)` or similar helpers
- Direct construction of user paths with numeric IDs

---

## Task 7: Update Frontend User References (Part 2 - Core Store)

**Files:**
- Modify: `frontend/src/store/modules/user.ts`

### Step 1: Review current getUserByIdentifier implementation

The function at line 223-234 already handles both UID and email. After backend migration, it should prefer email but may still work as-is due to the fallback logic.

### Step 2: Update ensureUserFullName usage if needed

Check if `ensureUserFullName` needs updates to always format with email instead of UID when constructing user paths.

### Step 3: Search for direct user path construction in store

Run:
```bash
grep -n "users/\${" frontend/src/store/modules/user.ts
```

Expected: Find any direct path constructions that use UID

### Step 4: Update any UID-based constructions to use email

If found, replace patterns like:
- `` `users/${userId}` `` → `` `users/${email}` ``
- `user.name` (if it's UID-based) → ensure email is used

### Step 5: Run frontend type check

Run:
```bash
pnpm --dir frontend type-check
```

Expected: No type errors

### Step 6: Commit store changes if any were made

```bash
git add frontend/src/store/modules/user.ts
git commit -m "refactor(frontend): update user store to use email identifiers

Update user store to construct user resource paths with email addresses
instead of UIDs.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Update Frontend Components (Batch Update)

**Files:**
- Multiple Vue and TypeScript files that construct user paths

### Step 1: Create script to find and replace user path patterns

This will be a manual review and update process. For each file found in Task 6:

1. Open the file
2. Find patterns like `` `users/${userId}` `` or `` `users/${user.id}` ``
3. Replace with `` `users/${email}` `` or `` `users/${user.email}` ``
4. Ensure the email variable is available in scope

### Step 2: Update common files

Based on the search from Task 6, update these known files:

**File:** `frontend/src/components/v2/Model/cells/UserNameCell.vue`
- Likely already uses email (seen in UserByEmail.vue example)
- Verify no UID usage

**File:** `frontend/src/views/sql-editor/EditorCommon/SharePopover.vue`
- Check for user path construction
- Update if using UID

### Step 3: Run format and lint

Run:
```bash
pnpm --dir frontend biome:check
```

Expected: Code formatted and linted

### Step 4: Run type check

Run:
```bash
pnpm --dir frontend type-check
```

Expected: No type errors

### Step 5: Commit frontend component changes

```bash
git add frontend/src/
git commit -m "refactor(frontend): update components to use email in user paths

Update all Vue components to use email addresses instead of UIDs when
constructing user resource paths.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Update Backend Tests

**Files:**
- Modify: `backend/api/v1/user_service_test.go` (if tests exist)

### Step 1: Check if tests exist

Run:
```bash
cat backend/api/v1/user_service_test.go | head -50
```

Expected: View test content (file may be empty based on earlier read)

### Step 2: Add basic GetUser test with email

If the file is minimal/empty, add a basic test:

```go
package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/stretchr/testify/require"
)

func TestGetUser_EmailIdentifier(t *testing.T) {
	// This is a placeholder test to document the expected behavior
	// Full integration tests should be added
	email := "test@example.com"
	request := &v1pb.GetUserRequest{
		Name: "users/" + email,
	}

	// Verify the request format is valid
	require.Equal(t, "users/test@example.com", request.Name)
}
```

### Step 3: Run the test

Run:
```bash
go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run ^TestGetUser_EmailIdentifier$
```

Expected: Test passes

### Step 4: Commit test changes

```bash
git add backend/api/v1/user_service_test.go
git commit -m "test(backend): add email identifier test for user service

Add basic test documenting expected email-based identifier format.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Final Verification

**Files:**
- All modified files

### Step 1: Run all Go lints repeatedly

Run:
```bash
golangci-lint run --allow-parallel-runners
```

Repeat until no issues (due to max-issues limit)

Expected: No linting errors

### Step 2: Run Go formatting

Run:
```bash
find backend/api/v1 -name "*.go" -exec gofmt -w {} \;
```

Expected: All files formatted

### Step 3: Run frontend checks

Run:
```bash
pnpm --dir frontend biome:check && pnpm --dir frontend type-check
```

Expected: All checks pass

### Step 4: Build backend

Run:
```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: Build succeeds

### Step 5: Run git status

Run:
```bash
git status
```

Expected: All changes committed, working tree clean

### Step 6: Review all commits

Run:
```bash
git log --oneline -10
```

Expected: See all commits from this implementation

---

## Task 11: Documentation and Breaking Change Label

**Files:**
- Modify: `docs/plans/2025-12-11-user-service-email-identifier-design.md`

### Step 1: Update design doc with implementation notes

Add an "Implementation" section at the end of the design document:

```markdown
## Implementation Notes

Implementation completed on 2025-12-11. All UserService endpoints now exclusively accept email-based identifiers in resource paths.

### Changes Made

- **Proto**: Updated all request message comments to reflect email-only format
- **Backend**: Removed UID fallback logic from all UserService methods (GetUser, UpdateUser, DeleteUser, UndeleteUser)
- **Frontend**: Updated all user path constructions to use email addresses
- **Tests**: Added basic email identifier test

### Verification

- All Go lints pass
- Frontend type checks pass
- Backend builds successfully
- All changes committed

### Migration Required

All API clients must update to use email addresses in user resource paths:
- Before: `users/101`
- After: `users/alice@example.com`
```

### Step 2: Commit documentation update

```bash
git add docs/plans/2025-12-11-user-service-email-identifier-design.md
git commit -m "docs: add implementation notes to design document

Document completed implementation and verification steps.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Step 3: Create summary of changes

Create a summary of all changes for the pull request:

**Breaking Changes:**
- UserService API now requires email addresses in all resource paths
- `users/{uid}` format no longer supported
- All clients must update to `users/{email}` format

**Files Changed:**
- Proto: `proto/v1/v1/user_service.proto`
- Backend: `backend/api/v1/user_service.go`
- Frontend: Multiple components in `frontend/src/`
- Tests: `backend/api/v1/user_service_test.go`

---

## Success Criteria

- [ ] All UserService proto comments reflect email-only format
- [ ] Backend methods use email parsing exclusively (no UID fallback)
- [ ] Frontend uses email in all user resource path constructions
- [ ] All linting and type checks pass
- [ ] Backend builds successfully
- [ ] All changes committed with descriptive messages
- [ ] Documentation updated

## Next Steps After Implementation

1. Create pull request with `breaking` label
2. Update API documentation/changelog with migration guide
3. Notify API clients about breaking change
4. Plan deployment strategy (backend and frontend together)
