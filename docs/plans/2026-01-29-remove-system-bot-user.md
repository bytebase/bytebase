# Remove SystemBotUser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove SystemBotUser concept entirely - use NULL for system-generated actions

**Architecture:** Database migration makes creator columns nullable for issue_comment and task_run tables. Backend uses empty string "" as sentinel that gets converted to NULL in store layer. Frontend displays "System" for null creators.

**Tech Stack:** PostgreSQL, Go, TypeScript/Vue, i18n

---

## Task 1: Create Database Migration

**Files:**
- Create: `backend/migrator/migration/3.15/0006##remove_system_bot.sql`

**Step 1: Create migration file**

Create the migration SQL file with schema changes and data cleanup:

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

-- Convert all existing SystemBot records to NULL
UPDATE issue_comment SET creator = NULL
  WHERE creator = 'support@bytebase.com';

UPDATE task_run SET creator = NULL
  WHERE creator = 'support@bytebase.com';

-- Delete the SystemBot principal row
DELETE FROM principal WHERE id = 1 AND email = 'support@bytebase.com';
```

**Step 2: Commit migration file**

```bash
git add backend/migrator/migration/3.15/0006##remove_system_bot.sql
git commit -m "chore: add migration to remove SystemBot user

- Make issue_comment.creator and task_run.creator nullable
- Convert existing SystemBot records to NULL
- Delete SystemBot principal row

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Update LATEST.sql Schema

**Files:**
- Modify: `backend/migrator/migration/LATEST.sql`

**Step 1: Update issue_comment table definition**

Find the issue_comment table definition (around line 374) and change:
```sql
-- FROM:
creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,

-- TO:
creator text REFERENCES principal(email) ON UPDATE CASCADE ON DELETE SET NULL,
```

**Step 2: Update task_run table definition**

Find the task_run table definition (around line 264) and change:
```sql
-- FROM:
creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,

-- TO:
creator text REFERENCES principal(email) ON UPDATE CASCADE ON DELETE SET NULL,
```

**Step 3: Remove SystemBot INSERT statement**

Find and delete the line (around line 593):
```sql
-- DELETE THIS LINE:
INSERT INTO principal (id, type, name, email, password_hash) VALUES (1, 'SYSTEM_BOT', 'Bytebase', 'support@bytebase.com', '');
```

**Step 4: Commit LATEST.sql changes**

```bash
git add backend/migrator/migration/LATEST.sql
git commit -m "chore: update LATEST.sql to reflect SystemBot removal

- Make creator columns nullable in issue_comment and task_run
- Remove SystemBot INSERT statement

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Update Migration Test

**Files:**
- Modify: `backend/migrator/migrator_test.go:15-16`

**Step 1: Update TestLatestVersion**

Change the test to expect the new version:
```go
// FROM:
require.Equal(t, semver.MustParse("3.15.5"), *files[len(files)-1].version)
require.Equal(t, "migration/3.15/0005##migrate_iam_binding_member_prefix.sql", files[len(files)-1].path)

// TO:
require.Equal(t, semver.MustParse("3.15.6"), *files[len(files)-1].version)
require.Equal(t, "migration/3.15/0006##remove_system_bot.sql", files[len(files)-1].path)
```

**Step 2: Run migration test**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator -run ^TestLatestVersion$`
Expected: PASS

**Step 3: Commit test update**

```bash
git add backend/migrator/migrator_test.go
git commit -m "test: update TestLatestVersion for SystemBot removal

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Update Store Layer - issue_comment.go

**Files:**
- Modify: `backend/store/issue_comment.go:124-145`

**Step 1: Update CreateIssueComments to handle empty creator**

Modify the CreateIssueComments function:
```go
func (s *Store) CreateIssueComments(ctx context.Context, creator string, creates ...*IssueCommentMessage) (*IssueCommentMessage, error) {
	if len(creates) == 0 {
		return nil, nil
	}

	// Prepare all payloads.
	issueIDs := make([]int, 0, len(creates))
	payloads := make([][]byte, 0, len(creates))
	for _, create := range creates {
		payload, err := protojson.Marshal(create.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		issueIDs = append(issueIDs, create.IssueUID)
		payloads = append(payloads, payload)
	}

	// Convert empty string to NULL for system-generated comments
	var creatorPtr any
	if creator == "" {
		creatorPtr = nil
	} else {
		creatorPtr = creator
	}

	// Use UNNEST to insert all comments in one query.
	q := qb.Q().Space(`
		INSERT INTO issue_comment (creator, issue_id, payload)
		SELECT ?, unnest(?::INT[]), unnest(?::JSONB[])
	`, creatorPtr, issueIDs, payloads)

	// Rest of function remains unchanged...
```

**Step 2: Update ListIssueComment to handle NULL creator**

The SELECT already handles this, but ensure CreatorEmail can be NULL:
```go
// Around line 100 in the Scan, CreatorEmail already uses sql.NullString or similar
// Verify the scan handles NULL properly - if it scans directly to string, change to:
var creatorEmail sql.NullString
if err := rows.Scan(
	&ic.UID,
	&creatorEmail,  // Changed from &ic.CreatorEmail
	&ic.CreatedAt,
	&ic.UpdatedAt,
	&ic.IssueUID,
	&p,
); err != nil {
	return nil, err
}
if creatorEmail.Valid {
	ic.CreatorEmail = creatorEmail.String
}
// Continue with rest of scan logic...
```

**Step 3: Run affected tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store -run IssueComment`
Expected: PASS

**Step 4: Commit store changes**

```bash
git add backend/store/issue_comment.go
git commit -m "feat: handle NULL creator in issue comments

- Convert empty string to NULL when creating comments
- Handle NULL when scanning creator from database

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Update Store Layer - task_run.go

**Files:**
- Modify: `backend/store/task_run.go:192-250,294-350`

**Step 1: Update patchTaskRunStatusImpl to handle empty updater**

Find patchTaskRunStatusImpl function (check around line 220-250) and update to convert empty string to NULL:
```go
func (s *Store) patchTaskRunStatusImpl(ctx context.Context, tx *sql.Tx, patch *TaskRunStatusPatch) (*TaskRunMessage, error) {
	set := qb.Q()
	set.Comma("updated_at = ?", time.Now())

	// Convert empty string to NULL for system updates
	var updaterPtr any
	if patch.Updater == "" {
		updaterPtr = nil
	} else {
		updaterPtr = patch.Updater
	}
	set.Comma("updater = ?", updaterPtr)

	// Rest of function...
```

**Step 2: Update CreatePendingTaskRuns to handle empty creator**

Modify around line 306-320:
```go
func (s *Store) CreatePendingTaskRuns(ctx context.Context, creator string, creates ...*TaskRunMessage) error {
	if len(creates) == 0 {
		return nil
	}

	var taskUIDs []int
	var runAts []*time.Time
	for _, create := range creates {
		taskUIDs = append(taskUIDs, create.TaskUID)
		runAts = append(runAts, create.RunAt)
	}

	// Convert empty string to NULL for system-created task runs
	var creatorPtr any
	if creator == "" {
		creatorPtr = nil
	} else {
		creatorPtr = creator
	}

	// Single query that:
	// 1. Filters out tasks with existing PENDING/RUNNING/DONE task runs (idempotent)
	// 2. Calculates next attempt for each remaining task
	// 3. Inserts task runs
	// 4. Uses ON CONFLICT DO NOTHING to handle race conditions
	q := qb.Q().Space(`
		INSERT INTO task_run (
			creator,
			task_id,
			// ... rest of fields
		)
		// ... rest of query, use creatorPtr instead of creator
	`, creatorPtr, /* other params */)

	// Rest of function...
```

**Step 3: Update ListTaskRuns scan to handle NULL creator**

Around line 124-125, ensure CreatorEmail handles NULL:
```go
var creatorEmail sql.NullString
if err := rows.Scan(
	&taskRun.ID,
	&creatorEmail,  // Changed from &taskRun.CreatorEmail
	&taskRun.CreatedAt,
	&taskRun.UpdatedAt,
	&taskRun.TaskUID,
	&statusString,
	&startedAt,
	&runAt,
	&resultJSON,
	&payloadJSON,
	&taskRun.PlanUID,
	&taskRun.Environment,
	&taskRun.ProjectID,
); err != nil {
	return nil, err
}
if creatorEmail.Valid {
	taskRun.CreatorEmail = creatorEmail.String
}
```

**Step 4: Run affected tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store -run TaskRun`
Expected: PASS

**Step 5: Commit store changes**

```bash
git add backend/store/task_run.go
git commit -m "feat: handle NULL creator in task runs

- Convert empty string to NULL when creating/updating task runs
- Handle NULL when scanning creator from database

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Update API Layer - Replace SystemBotEmail

**Files:**
- Modify: `backend/api/v1/issue_service.go:1267`
- Modify: `backend/api/v1/rollout_service.go:404,459`
- Modify: `backend/runner/taskrun/pending_scheduler.go:167`
- Modify: `backend/runner/taskrun/running_scheduler.go:121,143,188`

**Step 1: Update issue_service.go**

Replace SystemBotEmail with empty string:
```go
// Around line 1267
// FROM:
if _, err := s.store.CreateIssueComments(ctx, common.SystemBotEmail, &store.IssueCommentMessage{

// TO:
if _, err := s.store.CreateIssueComments(ctx, "", &store.IssueCommentMessage{
```

**Step 2: Update rollout_service.go**

Replace both occurrences:
```go
// Around line 404
// FROM:
if _, err := s.CreateIssueComments(ctx, common.SystemBotEmail, &store.IssueCommentMessage{

// TO:
if _, err := s.CreateIssueComments(ctx, "", &store.IssueCommentMessage{

// Around line 459
// FROM:
if err := s.CreatePendingTaskRuns(ctx, common.SystemBotEmail, create); err != nil {

// TO:
if err := s.CreatePendingTaskRuns(ctx, "", create); err != nil {
```

**Step 3: Update pending_scheduler.go**

```go
// Around line 167
// FROM:
if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
	ID:      taskRun.ID,
	Updater: common.SystemBotEmail,

// TO:
if _, err := s.store.UpdateTaskRunStatus(ctx, &store.TaskRunStatusPatch{
	ID:      taskRun.ID,
	Updater: "",
```

**Step 4: Update running_scheduler.go (3 occurrences)**

```go
// Around line 121
taskRunStatusPatch := &store.TaskRunStatusPatch{
	ID:          taskRunUID,
	Updater:     "",  // FROM: common.SystemBotEmail
	Status:      storepb.TaskRun_CANCELED,

// Around line 143
taskRunStatusPatch := &store.TaskRunStatusPatch{
	ID:      taskRunUID,
	Updater: "",  // FROM: common.SystemBotEmail
	Status:  storepb.TaskRun_FAILED,

// Around line 188
taskRunStatusPatch := &store.TaskRunStatusPatch{
	ID:          taskRunUID,
	Updater:     "",  // FROM: common.SystemBotEmail
	Status:      storepb.TaskRun_DONE,
```

**Step 5: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/... backend/runner/...`
Expected: No issues (may need multiple runs)

**Step 6: Commit API layer changes**

```bash
git add backend/api/v1/issue_service.go backend/api/v1/rollout_service.go backend/runner/taskrun/pending_scheduler.go backend/runner/taskrun/running_scheduler.go
git commit -m "refactor: replace SystemBotEmail with empty string

Use empty string as sentinel for system-generated actions.
Store layer converts to NULL in database.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Remove SystemBot Special Cases

**Files:**
- Modify: `backend/api/v1/user_service.go:62-71`
- Modify: `backend/api/v1/issue_service.go:754-758`
- Modify: `backend/runner/approval/runner.go:898-902`

**Step 1: Remove GetUser SystemBot special case**

In user_service.go, delete the entire special case block:
```go
// DELETE lines 62-71:
// Special case for SYSTEM_BOT user which is a built-in resource.
// SYSTEM_BOT is stored in principal table with type='SYSTEM_BOT', but GetEndUserByEmail
// only queries END_USER type. We use the static SystemBotUser here to avoid mixing user types.
if email == common.SystemBotEmail {
	v1User, err := convertToUser(ctx, s.iamManager, store.SystemBotUser)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert user"))
	}
	return connect.NewResponse(v1User), nil
}
```

**Step 2: Remove SystemBotUser fallback in issue_service.go**

```go
// Around line 754-758, change:
// FROM:
creator, err := s.store.GetPrincipalByEmail(ctx, issue.CreatorEmail)
if err != nil {
	slog.Warn("failed to get issue creator, using system bot", log.BBError(err))
	creator = store.SystemBotUser
}

// TO:
creator, err := s.store.GetPrincipalByEmail(ctx, issue.CreatorEmail)
if err != nil {
	slog.Warn("failed to get issue creator", log.BBError(err))
	// Skip webhook if creator not found
	return nil
}
```

**Step 3: Remove SystemBotUser fallback in approval/runner.go**

```go
// Around line 898-902, change:
// FROM:
creator, err := stores.GetPrincipalByEmail(ctx, issue.CreatorEmail)
if err != nil {
	slog.Warn("failed to get issue creator, using system bot", log.BBError(err))
	creator = store.SystemBotUser
}

// TO:
creator, err := stores.GetPrincipalByEmail(ctx, issue.CreatorEmail)
if err != nil {
	slog.Warn("failed to get issue creator", log.BBError(err))
	// Use default/unknown user for approval checks
	return errors.Wrap(err, "creator not found")
}
```

**Step 4: Run golangci-lint**

Run: `golangci-lint run --allow-parallel-runners backend/api/v1/... backend/runner/...`
Expected: No issues

**Step 5: Commit special case removal**

```bash
git add backend/api/v1/user_service.go backend/api/v1/issue_service.go backend/runner/approval/runner.go
git commit -m "refactor: remove SystemBot special case handling

- Remove GetUser special case for SystemBotEmail
- Remove fallback to SystemBotUser when creator not found
- Handle missing creators as errors instead

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: Delete SystemBot Constants

**Files:**
- Modify: `backend/common/const.go:5-9`
- Modify: `backend/store/principal.go:18-24,575-577`

**Step 1: Delete constants from common/const.go**

Delete lines defining SystemBot constants:
```go
// DELETE lines 5-9:
// SystemBotID is the ID of the system robot.
SystemBotID = 1

// SystemBotEmail is the email of the system robot.
SystemBotEmail = "support@bytebase.com"
```

**Step 2: Delete SystemBotUser variable from store/principal.go**

Delete the variable definition:
```go
// DELETE lines 18-24:
// SystemBotUser is the static system bot user.
var SystemBotUser = &UserMessage{
	ID:      common.SystemBotID,
	Name:    "Bytebase",
	Email:   "support@bytebase.com",
	Type:    storepb.PrincipalType_SYSTEM_BOT,
	Profile: &storepb.UserProfile{},
}
```

**Step 3: Remove UpdateUser SystemBotID check**

Delete the check in UpdateUser function (around line 575-577):
```go
// DELETE:
if currentUser.ID == common.SystemBotID {
	return nil, errors.Errorf("cannot update system bot")
}
```

**Step 4: Run golangci-lint repeatedly**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No issues (run multiple times until clean)

**Step 5: Commit constant deletion**

```bash
git add backend/common/const.go backend/store/principal.go
git commit -m "refactor: delete SystemBot constants and variables

- Remove SystemBotID and SystemBotEmail constants
- Remove SystemBotUser variable
- Remove UpdateUser SystemBot check

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Update Frontend - Remove Constants

**Files:**
- Modify: `frontend/src/types/common.ts:1-4`

**Step 1: Delete SYSTEM_BOT constants**

Delete the constant definitions:
```typescript
// DELETE lines 1-4:
// System bot ID
export const SYSTEM_BOT_ID = 1;
// System bot email
export const SYSTEM_BOT_EMAIL = "support@bytebase.com";
```

**Step 2: Search for usage**

Run: `grep -r "SYSTEM_BOT_ID\|SYSTEM_BOT_EMAIL" frontend/src/`
Expected: Find usages to replace in next tasks

**Step 3: Commit constant deletion**

```bash
git add frontend/src/types/common.ts
git commit -m "refactor: remove SYSTEM_BOT constants from frontend

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Add i18n Translations

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json`
- Modify: `frontend/src/locales/es-ES.json` (if exists)
- Modify: `frontend/src/locales/ja-JP.json` (if exists)

**Step 1: Add "system" to en-US.json**

Find the "common" section and add:
```json
{
  "common": {
    ...existing keys...,
    "system": "System"
  }
}
```

**Step 2: Add "system" to zh-CN.json**

```json
{
  "common": {
    ...existing keys...,
    "system": "系统"
  }
}
```

**Step 3: Add to other locale files if they exist**

Check and update other locale files with appropriate translations.

**Step 4: Run frontend linter**

Run: `pnpm --dir frontend fix`
Expected: No errors

**Step 5: Commit i18n changes**

```bash
git add frontend/src/locales/
git commit -m "feat: add 'system' translation for NULL creators

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 11: Remove Frontend Special Case Handling

**Files:**
- Modify: `frontend/src/components/Member/MemberDataTable/cells/UserOperationsCell.vue`

**Step 1: Find and read UserOperationsCell.vue**

Read the file to find the SystemBot check (around line 69-71).

**Step 2: Remove SYSTEM_BOT_USER_NAME check**

```typescript
// FROM (around line 68-72):
const user = props.binding.user ?? unknownUser();
if (user.name === SYSTEM_BOT_USER_NAME) {
  // Cannot edit the member binding for support@bytebase.com, but can edit allUsers
  return false;
}

// TO:
const user = props.binding.user ?? unknownUser();
// System user is no longer in the database, no special check needed
```

**Step 3: Search for other SYSTEM_BOT references**

Run: `grep -r "SYSTEM_BOT" frontend/src/`
Expected: No remaining references (or handle them)

**Step 4: Run frontend checks**

Run: `pnpm --dir frontend fix && pnpm --dir frontend type-check`
Expected: PASS

**Step 5: Commit frontend cleanup**

```bash
git add frontend/src/components/
git commit -m "refactor: remove SystemBot special case checks

SystemBot no longer exists in database.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 12: Update Frontend User Display Logic

**Files:**
- Search and update components displaying creators/users

**Step 1: Find components displaying creator/user**

Run: `grep -r "creator\|\.user" frontend/src/components/ | grep -i "\.name\|\.email"`
Expected: List of files to review

**Step 2: Update to handle null/undefined users**

For each component found, ensure it handles null/undefined:
```typescript
// Example pattern:
// FROM:
{{ user.name || user.email }}

// TO:
{{ user?.name || user?.email || $t('common.system') }}
```

**Step 3: Run frontend checks**

Run: `pnpm --dir frontend fix && pnpm --dir frontend type-check`
Expected: PASS

**Step 4: Commit display logic updates**

```bash
git add frontend/src/components/
git commit -m "feat: display 'System' for null/undefined users

Use i18n common.system when user is null or undefined.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 13: Run Backend Tests

**Files:**
- Test: All backend tests

**Step 1: Run store tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/store/...`
Expected: PASS (fix any failures)

**Step 2: Run API tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/api/...`
Expected: PASS (fix any failures)

**Step 3: Run migration tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/migrator/...`
Expected: PASS

**Step 4: Run all backend tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/...`
Expected: PASS

**Step 5: If tests fail, fix and commit**

If tests need updates:
```bash
git add <test files>
git commit -m "test: update tests for SystemBot removal

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 14: Run Frontend Tests

**Files:**
- Test: All frontend tests

**Step 1: Run type check**

Run: `pnpm --dir frontend type-check`
Expected: PASS

**Step 2: Run linter**

Run: `pnpm --dir frontend check`
Expected: PASS

**Step 3: Run unit tests**

Run: `pnpm --dir frontend test`
Expected: PASS (or fix failures)

**Step 4: If tests fail, fix and commit**

```bash
git add frontend/
git commit -m "test: update frontend tests for SystemBot removal

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 15: Build and Manual Verification

**Files:**
- Build: Backend and frontend

**Step 1: Build backend**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Success

**Step 2: Run golangci-lint final check**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No issues (run multiple times if needed)

**Step 3: Build frontend**

Run: `pnpm --dir frontend fix`
Expected: Success

**Step 4: Start application and verify**

Manual testing:
- Start application
- Trigger task run → verify creator is NULL in DB, displays "System" in UI
- Change issue status → verify comment creator is NULL, displays "System"
- List users → verify no SystemBot user appears

**Step 5: Final commit if any fixes needed**

```bash
git add .
git commit -m "fix: final adjustments for SystemBot removal

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Verification Checklist

After all tasks complete:

- [ ] Migration file created and passes TestLatestVersion
- [ ] LATEST.sql updated (nullable columns, no SystemBot INSERT)
- [ ] Store layer converts "" to NULL for issue_comment and task_run
- [ ] API layer uses "" instead of common.SystemBotEmail
- [ ] SystemBot constants deleted from backend
- [ ] SystemBot special cases removed
- [ ] Frontend constants deleted
- [ ] i18n translations added for "System"
- [ ] Frontend displays "System" for null users
- [ ] All backend tests pass
- [ ] All frontend tests pass
- [ ] Build succeeds
- [ ] Manual verification confirms NULL creators display as "System"

---

## Notes

- Use @superpowers:verification-before-completion before claiming tasks complete
- Run golangci-lint multiple times due to max-issues limit
- Commit frequently (after each task)
- Test after each layer (store, API, frontend)
