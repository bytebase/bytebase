# Plan Header Actions Documentation

This document describes all available actions in the Plan Header section, including visibility rules, permissions, and disabled states.

## Architecture

The action system uses a **Registry Pattern** for maintainability and type safety.

### Core Components

```
Actions/
├── registry/                    # Action Registry (source of truth)
│   ├── types.ts                 # ActionContext, ActionDefinition interfaces
│   ├── context.ts               # buildActionContext() - context builder
│   ├── useActionRegistry.ts     # Main composable
│   ├── actions/
│   │   ├── plan.ts              # PLAN_CLOSE, PLAN_REOPEN
│   │   ├── issue.ts             # ISSUE_CREATE, ISSUE_REVIEW, ISSUE_STATUS_*
│   │   └── rollout.ts           # ROLLOUT_CREATE, ROLLOUT_START, ROLLOUT_CANCEL, EXPORT_DOWNLOAD
│   └── components/
│       ├── ActionButton.vue     # Single action button
│       ├── ActionDropdown.vue   # Secondary actions dropdown
│       └── IssueReviewButton.vue # Review popover with approve/reject/comment
└── Actions.vue                  # Main orchestrator
```

### Key Concepts

**ActionContext**: Single reactive object containing all state needed for action decisions:
- Entities: plan, issue, rollout, project
- Derived flags: isIssueOnly, isExportPlan, hasDeferredRollout, isCreator
- Permissions: updatePlan, createIssue, updateIssue, createRollout, runTasks
- Validation: hasEmptySpec, planChecksRunning, planChecksFailed

**hasDeferredRollout**: Plans where rollout is created on-demand (export, create database). For these plans:
- ROLLOUT_CREATE is hidden; ROLLOUT_START creates rollout and runs tasks in one step
- ISSUE_STATUS_RESOLVE is hidden (issues auto-resolve when task completes)

**ActionDefinition**: Declarative action with pure functions:
```typescript
interface ActionDefinition {
  id: UnifiedActionType;
  label: (ctx: ActionContext) => string;
  buttonType: "primary" | "success" | "default";
  category: ActionCategory | ((ctx: ActionContext) => ActionCategory); // Can be dynamic
  priority: number;  // Lower = higher priority
  isVisible: (ctx: ActionContext) => boolean;
  isDisabled: (ctx: ActionContext) => boolean;
  disabledReason: (ctx: ActionContext) => string | undefined;
  executeType: ExecuteType;
}
```

---

## Action Types

### Plan Actions

#### `ISSUE_CREATE` (Primary, Priority: 5)
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

---

#### `PLAN_CLOSE` (Secondary, Priority: 100)
**Label**: "Close"

**Description**: Closes (archives) the plan.

**Visibility**:
- Plan has no issue attached
- Plan has no rollout attached
- Plan is in ACTIVE state
- User is the plan creator OR has `bb.plans.update` permission

---

#### `PLAN_REOPEN` (Primary, Priority: 10)
**Label**: "Reopen"

**Description**: Reopens a closed (archived) plan.

**Visibility**:
- Plan has no issue attached
- Plan has no rollout attached
- Plan is in DELETED state
- User is the plan creator OR has `bb.plans.update` permission

---

### Issue Review Action

#### `ISSUE_REVIEW` (Primary, Priority: 30)
**Label**: "Review"

**Description**: Opens the unified review popover for approve/reject/comment.

**Visibility**:
- Issue exists and is OPEN
- Issue approval status is not APPROVED or SKIPPED
- User is a candidate for the current approval step (canApprove or canReject)

---

### Issue Status Actions

#### `ISSUE_STATUS_RESOLVE` (Primary, Priority: 50)
**Label**: "Resolve"

**Description**: Marks the issue as resolved (DONE).

**Visibility**:
- Issue exists and is OPEN
- Issue approval status is APPROVED or SKIPPED
- All tasks in the rollout are finished (DONE or SKIPPED)
- Rollout exists (`plan.hasRollout`)
- Not a deferred rollout plan (export/create database plans auto-resolve when task completes)
- User has `bb.issues.update` permission

---

#### `ISSUE_STATUS_CLOSE` (Secondary, Priority: 90)
**Label**: "Close"

**Description**: Closes the issue without resolving it (CANCELED).

**Visibility**:
- Issue exists and is OPEN
- No rollout exists (can't close after rollout starts)
- User has `bb.issues.update` permission

---

#### `ISSUE_STATUS_REOPEN` (Primary, Priority: 20)
**Label**: "Reopen"

**Description**: Reopens a canceled issue.

**Visibility**:
- Issue exists and is CANCELED (not DONE - resolved issues cannot be reopened)
- User has `bb.issues.update` permission

---

### Rollout Actions

#### `ROLLOUT_CREATE` (Primary/Secondary, Priority: 55)
**Label**: "Create Rollout"

**Description**: Creates a rollout from the issue. Shows as primary action when ready, moves to dropdown when conditions aren't met.

**Visibility**:
- Not a deferred rollout plan (export/create database plans use ROLLOUT_START instead)
- Not an issue-only plan
- Plan has no rollout attached
- Issue exists
- User has `bb.rollouts.create` permission
- If `requireIssueApproval=true` AND issue not APPROVED/SKIPPED → hidden
- If `requirePlanCheckNoError=true` AND plan checks failed → hidden

**Category Logic**:
- **Primary** (main button): When all conditions are met (no warnings)
- **Secondary** (dropdown): When warnings exist (require_*=false but conditions not met)

**Warning Panel** (shown when clicking from dropdown):
- `requireIssueApproval=false` AND approval not APPROVED/SKIPPED → shows approval flow section
- Plan checks are running → shows running message
- `requirePlanCheckNoError=false` AND plan checks failed → shows plan check status

When warnings exist, user must check "Bypass stage requirements" checkbox to proceed.

---

#### `ROLLOUT_START` (Primary, Priority: 60)
**Label**:
- "Export" (for export data issues)
- "Rollout" (for other issues)

**Description**: Starts or retries tasks in the rollout. For deferred rollout plans (export/create database), this action creates the rollout and runs all tasks in one step.

**Visibility**:
For deferred rollout plans (export/create database):
- Issue exists and is APPROVED or SKIPPED
- No rollout yet (will create it) OR has startable tasks
- User has `bb.taskRuns.create` permission

For regular plans:
- Rollout exists
- Issue approval status is APPROVED or SKIPPED
- Has DATABASE_CREATE or DATABASE_EXPORT tasks
- At least one task is startable (NOT_STARTED, FAILED, or CANCELED)
- **Permission**: User has `bb.taskRuns.create` permission OR is creator (for exports)

---

#### `ROLLOUT_CANCEL` (Secondary, Priority: 80)
**Label**: "Cancel"

**Description**: Cancels running or pending tasks.

**Visibility**:
- Rollout exists
- Issue approval status is APPROVED or SKIPPED
- At least one task is in PENDING or RUNNING status
- **Permission**: Same as ROLLOUT_START

---

#### `EXPORT_DOWNLOAD` (Primary, Priority: 0)
**Label**: "Download"

**Description**: Downloads completed export archive.

**Visibility**:
- Plan is an export plan
- Export archive is ready
- User is the issue creator

---

## Warning Behavior

### Rollout Creation Category and Warnings

The ROLLOUT_CREATE button visibility and category depend on project settings and current state:

| Condition | Result |
|-----------|--------|
| `requireIssueApproval=true` + not approved | **Hidden** |
| `requirePlanCheckNoError=true` + checks failed | **Hidden** |
| `requireIssueApproval=false` + not approved | Secondary (dropdown) with warning |
| `requirePlanCheckNoError=false` + checks failed | Secondary (dropdown) with warning |
| Plan checks running | Secondary (dropdown) with warning |
| All conditions met | Primary (main button) |

### Task Execution Warnings

Task execution (RUN/SKIP/CANCEL) always shows **warnings** for approval and plan check issues, never errors. Users can bypass these warnings to proceed.

---

## Global Disabled States

All actions are disabled when:
- Editor is in editing mode (`editorState.isEditing === true`)
- **Tooltip**: "Save changes before continuing"

---

## Action Priority Order

Actions are sorted by priority (lower number = higher priority). The first visible primary action is shown as the main button.

| Priority | Action | Category |
|----------|--------|----------|
| 0 | EXPORT_DOWNLOAD | primary |
| 5 | ISSUE_CREATE | primary |
| 10 | PLAN_REOPEN | primary |
| 20 | ISSUE_STATUS_REOPEN | primary |
| 30 | ISSUE_REVIEW | primary |
| 50 | ISSUE_STATUS_RESOLVE | primary |
| 55 | ROLLOUT_CREATE | primary |
| 60 | ROLLOUT_START | primary |
| 80 | ROLLOUT_CANCEL | secondary |
| 90 | ISSUE_STATUS_CLOSE | secondary |
| 100 | PLAN_CLOSE | secondary |

---

## Permission Summary

### Plan Permissions
- `bb.plans.update`: Close or reopen plans
- `bb.issues.create`: Create issues from plans

### Issue Permissions
- `bb.issues.update`: Close, reopen, or resolve issues

### Rollout Permissions
- `bb.rollouts.create`: Create rollouts
- `bb.taskRuns.create`: Run rollout tasks
- **Issue creator**: Run export tasks (for DATABASE_EXPORT issues)

### Approval Permissions
- **Approval template roles**: Approve or reject based on current approval step

---

## Usage

### Using the Action Registry

```typescript
import { useActionRegistry } from "./registry";

const {
  context,           // Reactive ActionContext
  primaryAction,     // First visible primary action
  secondaryActions,  // All visible secondary actions
  isActionDisabled,  // Check if action is disabled
  getDisabledReason, // Get disabled tooltip
  executeAction,     // Execute action by ID
} = useActionRegistry();
```

### Adding a New Action

1. Define the action in the appropriate file (`registry/actions/*.ts`):
```typescript
export const MY_NEW_ACTION: ActionDefinition = {
  id: "MY_NEW_ACTION",
  label: () => t("my-action.label"),
  buttonType: "primary",
  category: "primary",
  priority: 45,
  isVisible: (ctx) => /* visibility logic */,
  isDisabled: (ctx) => /* disabled logic */,
  disabledReason: (ctx) => /* reason or undefined */,
  executeType: "immediate",
};
```

2. Add to the actions array in the same file
3. Add the type to `registry/types.ts`
4. Handle execution in `Actions.vue` if needed

---

## Implementation Files

- **registry/**: Action registry (source of truth for all actions and components)
- **Actions.vue**: Main orchestrator
