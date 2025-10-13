# Plan Header Actions Documentation

This document describes all available actions in the Plan Header section, including visibility rules, permissions, and disabled states.

## Overview

Actions are displayed in the Plan Header based on the current state of the plan, issue, and rollout. Actions are divided into:

- **Primary Actions**: Displayed as prominent buttons (usually blue or green)
- **Secondary Actions**: Displayed in the dropdown menu (three-dot menu)

## Action Types

### Plan Actions

#### `ISSUE_CREATE` (Primary)
**Label**: "Ready for review"

**Description**: Creates an issue from the plan and initiates the approval workflow.

**Visibility**:
- Plan has no issue attached (`plan.issue === ""`)
- Plan has no rollout attached (`plan.rollout === ""`)
- Plan is in ACTIVE state
- User has `bb.issues.create` permission

**Disabled When**:
- Any spec has an empty statement
- Plan checks are currently running
- Plan checks failed and SQL review is enforced (`project.enforceSqlReview`)

**Disabled Tooltip**:
- "Missing statement"
- "Plan checks are running"
- "Some task checks didn't pass" (when SQL review enforced)

---

#### `PLAN_CLOSE` (Secondary)
**Label**: "Close"

**Description**: Closes (archives) the plan.

**Visibility**:
- Plan has no issue attached (`plan.issue === ""`)
- Plan has no rollout attached (`plan.rollout === ""`)
- Plan is in ACTIVE state
- User is the plan creator OR has `bb.plans.update` permission

---

#### `PLAN_REOPEN` (Primary)
**Label**: "Reopen"

**Description**: Reopens a closed (archived) plan.

**Visibility**:
- Plan has no issue attached (`plan.issue === ""`)
- Plan has no rollout attached (`plan.rollout === ""`)
- Plan is in DELETED state
- User is the plan creator OR has `bb.plans.update` permission

---

### Issue Review Actions

#### `ISSUE_REVIEW_APPROVE` (Primary)
**Label**: "Approve"

**Description**: Approves the current approval step.

**Visibility**:
- Issue exists and is OPEN
- Issue approval status is PENDING or REJECTED (not yet approved)
- User is in the current approval step's candidate list

**Notes**:
- When issue has been rejected, only APPROVE action is shown (no REJECT)
- Approval flow is determined by the approval template roles

---

#### `ISSUE_REVIEW_REJECT` (Secondary)
**Label**: "Send back"

**Description**: Rejects the current approval step and sends the issue back.

**Visibility**:
- Issue exists and is OPEN
- Issue approval status is PENDING (not yet approved)
- Issue has NOT been rejected yet
- User is in the current approval step's candidate list

---

#### `ISSUE_REVIEW_RE_REQUEST` (Primary)
**Label**: "Re-request review"

**Description**: Re-requests approval after the issue was rejected.

**Visibility**:
- Issue exists and is OPEN
- Issue has been rejected (at least one approver has REJECTED status)
- User is the issue creator

---

### Issue Status Actions

#### `ISSUE_STATUS_RESOLVE` (Primary)
**Label**: "Resolve"

**Description**: Marks the issue as resolved (DONE).

**Visibility**:
- Issue exists and is OPEN
- Issue approval status is APPROVED or SKIPPED
- All tasks in the rollout are finished (DONE or SKIPPED)
- User has `bb.issues.update` permission

---

#### `ISSUE_STATUS_CLOSE` (Secondary)
**Label**: "Close"

**Description**: Closes the issue without resolving it (CANCELED).

**Visibility**:
- Issue exists and is OPEN
- User has `bb.issues.update` permission

---

#### `ISSUE_STATUS_REOPEN` (Primary)
**Label**: "Reopen"

**Description**: Reopens a closed or resolved issue.

**Visibility**:
- Issue exists and is CANCELED or DONE
- User has `bb.issues.update` permission

---

### Rollout Actions

#### `ROLLOUT_START` (Primary)
**Label**:
- "Export" (for export data issues)
- "Rollout" (for other issues)

**Description**: Starts or retries tasks in the rollout.

**Visibility**:
- Issue exists and is OPEN
- Issue approval status is APPROVED or SKIPPED
- Rollout has at least one DATABASE_CREATE or DATABASE_EXPORT task
- At least one task is in NOT_STARTED, FAILED, or CANCELED status
- **Permission**:
  - For DATABASE_EXPORT issues: User must be the issue creator
  - For other issues: User must have `bb.taskRuns.create` permission OR match roles in environment rollout policy

**Notes**:
- For DATABASE_EXPORT issues, only the creator can export data (security restriction)
- For other issues, the two-tier permission check applies:
  1. Users with `bb.taskRuns.create` permission can always rollout
  2. OR users with matching roles in the environment rollout policy can rollout

---

#### `ROLLOUT_CANCEL` (Secondary)
**Label**: "Cancel"

**Description**: Cancels running or pending tasks.

**Visibility**:
- Issue exists and is OPEN
- Issue approval status is APPROVED or SKIPPED
- At least one task is in PENDING or RUNNING status
- **Permission**: Same as ROLLOUT_START

---

## Global Disabled States

All actions are disabled when:
- Editor is in editing mode (`editorState.isEditing === true`)
- **Tooltip**: "Save changes before continuing"

---

## Action Priority Order

When multiple actions are available, they are prioritized as follows:

### Primary Actions (first match wins):
1. `ISSUE_CREATE` - Create issue from plan
2. `PLAN_REOPEN` - Reopen deleted plan
3. `ISSUE_STATUS_REOPEN` - Reopen closed/done issue
4. `ISSUE_REVIEW_APPROVE` - Approve current step
5. `ISSUE_REVIEW_RE_REQUEST` - Re-request review after rejection
6. `ISSUE_STATUS_RESOLVE` - Resolve completed issue
7. `ROLLOUT_START` - Start/retry tasks

### Secondary Actions (all matching actions shown):
1. `ISSUE_REVIEW_REJECT` - Reject current approval step
2. `ROLLOUT_CANCEL` - Cancel running tasks
3. `ISSUE_STATUS_CLOSE` - Close issue
4. `PLAN_CLOSE` - Close plan

---

## Permission Summary

### Plan Permissions
- `bb.plans.update`: Close or reopen plans
- `bb.issues.create`: Create issues from plans

### Issue Permissions
- `bb.issues.update`: Close, reopen, or resolve issues

### Rollout Permissions
- `bb.taskRuns.create`: Run rollout tasks (for non-export issues)
- **Issue creator**: Run export tasks (for DATABASE_EXPORT issues)
- **Environment rollout policy roles**: Run tasks based on role matching

### Approval Permissions
- **Approval template roles**: Approve or reject based on current approval step

---

## Special Cases

### Plan Actions Without Issue/Rollout
Plan actions (PLAN_CLOSE, PLAN_REOPEN, ISSUE_CREATE) are only available when:
- No issue is attached to the plan
- No rollout is attached to the plan

**Rationale**: Once an issue or rollout is created, the plan enters a workflow state where plan-level actions are no longer appropriate. Users should interact with the issue/rollout instead.

### Export Data Issues
- Export button is labeled "Export" instead of "Rollout"
- Only the issue creator can start the export (security restriction)
- Export button shows only when approval is APPROVED or SKIPPED (not during review)

### Environment Rollout Policy
When a user lacks `bb.taskRuns.create` permission, the system checks the environment rollout policy:
1. Gets the database's effective environment
2. Checks if a rollout policy exists for that environment
3. If no policy exists: Allow rollout (no restrictions)
4. If policy exists: Check if user has any of the required roles in the policy

This allows fine-grained control over who can rollout tasks in specific environments.

---

## Implementation Files

- **Actions.vue**: Main action logic and visibility rules (frontend/src/components/Plan/components/HeaderSection/Actions/Actions.vue)
- **UnifiedActionGroup.vue**: Action button rendering (frontend/src/components/Plan/components/HeaderSection/Actions/unified/UnifiedActionGroup.vue)
- **taskPermissions.ts**: Rollout permission checks (frontend/src/components/Plan/components/RolloutView/taskPermissions.ts)
