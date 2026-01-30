# Remove SystemBotUser Design

**Date:** 2026-01-29
**Status:** Approved

## Overview & Goals

### Current State

Bytebase currently uses a special "SystemBot" user (ID=1, email='support@bytebase.com') to represent system-generated actions. This includes:
- Task scheduler operations
- Automated issue comments (status changes)
- Fallback when real users can't be found
- Internal system operations

This SystemBot appears in the UI alongside real users, which confuses users. It's unclear whether "Bytebase <support@bytebase.com>" is a real user, a support account, or something else.

### Goal

Remove the SystemBot concept entirely. System-generated actions will be clearly distinguished from user actions:
- **Database:** `creator` fields will be `NULL` for system operations
- **Code:** Empty string `""` used as sentinel for system operations
- **UI:** Display "System" label for NULL creators
- **No special constants** or hardcoded IDs/emails

### Benefits

- Clearer separation: NULL = system, email = real user
- No confusing fake user in user lists
- Simpler code (fewer special cases)
- Better user experience (obvious what's system vs human)

## Database Schema Changes

### Affected Tables

Only two tables need schema changes to support NULL creators:
- `issue_comment` (creator) - System creates comments for status changes
- `task_run` (creator) - System scheduler creates/updates task runs

**All other tables with creator columns remain unchanged** - they only contain user-created records.

### Schema Modifications

```sql
-- Make issue_comment.creator nullable
ALTER TABLE issue_comment ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE issue_comment DROP CONSTRAINT issue_comment_creator_fkey;
ALTER TABLE issue_comment ADD CONSTRAINT issue_comment_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Make task_run.creator nullable
ALTER TABLE task_run ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE task_run DROP CONSTRAINT task_run_creator_fkey;
ALTER TABLE task_run ADD CONSTRAINT task_run_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;
```

**Note:** This minimal change reduces migration risk - we only modify tables that actually need it.

## Data Migration Strategy

### Migration Steps

```sql
-- Step 1: Make columns nullable and update FK constraints
ALTER TABLE issue_comment ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE issue_comment DROP CONSTRAINT issue_comment_creator_fkey;
ALTER TABLE issue_comment ADD CONSTRAINT issue_comment_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

ALTER TABLE task_run ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE task_run DROP CONSTRAINT task_run_creator_fkey;
ALTER TABLE task_run ADD CONSTRAINT task_run_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Step 2: Convert all existing SystemBot records to NULL
UPDATE issue_comment SET creator = NULL
  WHERE creator = 'support@bytebase.com';

UPDATE task_run SET creator = NULL
  WHERE creator = 'support@bytebase.com';

-- Step 3: Delete the SystemBot principal row
DELETE FROM principal WHERE id = 1 AND email = 'support@bytebase.com';
```

### Migration Files

- Create: `backend/migrator/migration/X.XX/0001##remove_system_bot.sql`
- Update: `backend/migrator/migration/LATEST.sql` (remove SystemBot INSERT, update table definitions)
- Update: `backend/migrator/migrator_test.go` TestLatestVersion

## Backend Code Changes

### Store Layer Changes

Update store methods to handle empty string as system sentinel:

```go
// CreateIssueComments - update to convert "" to NULL
func (s *Store) CreateIssueComments(ctx context.Context, creator string, create *IssueCommentMessage) (*IssueCommentMessage, error) {
    var creatorPtr *string
    if creator == "" {
        creatorPtr = nil  // NULL for system
    } else {
        creatorPtr = &creator
    }

    // Use creatorPtr in INSERT query
    // ...
}

// CreatePendingTaskRuns - update similarly
// UpdateTaskRunStatus - update TaskRunStatusPatch.Updater handling
```

### API Layer Changes

Replace all `common.SystemBotEmail` with `""`:

```go
// issue_service.go - Issue status change comments
s.store.CreateIssueComments(ctx, "", &store.IssueCommentMessage{...})

// rollout_service.go - Auto-rollout task runs
s.CreatePendingTaskRuns(ctx, "", create)
s.CreateIssueComments(ctx, "", &store.IssueCommentMessage{...})

// running_scheduler.go & pending_scheduler.go - Task run updates
s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
    ID:      taskRun.ID,
    Updater: "",  // System update
    Status:  storepb.TaskRun_AVAILABLE,
})
```

### Remove Special Cases

```go
// user_service.go - Remove SystemBot special case
// DELETE this entire block:
if email == common.SystemBotEmail {
    v1User, err := convertToUser(ctx, s.iamManager, store.SystemBotUser)
    // ...
}

// issue_service.go & approval/runner.go - Remove fallback to SystemBotUser
// When creator fetch fails, handle as deleted/unknown user, not SystemBot
```

### Delete Constants

```go
// backend/common/const.go - Remove:
// - SystemBotID
// - SystemBotEmail

// backend/store/principal.go - Remove:
// - SystemBotUser variable
// - UpdateUser SystemBotID check
```

## Frontend Changes

### Remove Constants

```typescript
// frontend/src/types/common.ts - DELETE:
export const SYSTEM_BOT_ID = 1;
export const SYSTEM_BOT_EMAIL = "support@bytebase.com";
```

### Update User Display Logic

Add helper function to display creator/user names:

```typescript
// frontend/src/utils/user.ts (or appropriate location)
export const displayUserName = (user: User | null | undefined): string => {
  if (!user || !user.email) {
    return t('common.system');  // Returns "System" (internationalized)
  }
  return user.name || user.email;
};
```

### Update i18n Files

```json
// frontend/src/locales/en-US.json
{
  "common": {
    "system": "System"
  }
}

// frontend/src/locales/zh-CN.json
{
  "common": {
    "system": "系统"
  }
}
```

### Remove Special Case Handling

```typescript
// frontend/src/components/Member/MemberDataTable/cells/UserOperationsCell.vue
// REMOVE check for SYSTEM_BOT_USER_NAME - it won't exist anymore

// Any other places checking SYSTEM_BOT_EMAIL or SYSTEM_BOT_ID:
// - Replace with null/undefined checks
// - Use displayUserName() helper
```

### Component Updates

Anywhere displaying creators/updaters (issue lists, task run lists, comments, etc.):
- Use `displayUserName()` helper
- Handle null/undefined gracefully
- Display "System" label with appropriate styling (possibly greyed out or with a system icon)

## Testing Considerations

### Unit Tests to Update

1. **Store tests** - Update any tests that reference SystemBotUser:
   - Tests for `CreateIssueComments` with empty string creator
   - Tests for `UpdateTaskRunStatus` with empty string updater
   - Verify NULL is correctly stored in database
   - Remove tests checking SystemBotID validation

2. **API tests** - Update tests that:
   - Check for SystemBot in user lists
   - Fetch SystemBot user by email
   - Verify issue comment creation with system creator

3. **Migration tests** - Verify:
   - `TestLatestVersion` passes with updated version
   - Schema correctly has nullable columns
   - SystemBot row is absent from fresh database

### Integration Tests to Add

1. **Task scheduler flow**:
   - Verify task runs created with NULL creator
   - Verify status updates use NULL updater
   - Check UI displays "System" correctly

2. **Issue status changes**:
   - Verify system comments have NULL creator
   - Check UI renders system comments correctly
   - Ensure webhook events handle NULL creator gracefully

3. **Migration verification**:
   - Run migration on test database with existing SystemBot data
   - Verify all `support@bytebase.com` converted to NULL
   - Verify SystemBot row deleted successfully
   - Verify foreign keys work correctly

### Manual Testing Checklist

- [ ] Create task run via scheduler → creator is NULL, displays "System"
- [ ] Change issue status → system comment created, displays "System"
- [ ] View existing migrated records → display "System" for old SystemBot records
- [ ] User list API → no SystemBot user returned
- [ ] GetUser API with old SystemBot email → returns 404 or not found

## Implementation Order

1. Database migration (schema + data)
2. Backend store layer changes (handle "" → NULL)
3. Backend API layer changes (replace SystemBotEmail with "")
4. Backend cleanup (remove constants and special cases)
5. Frontend changes (remove constants, add display logic)
6. Frontend i18n updates
7. Test updates
8. Integration testing
