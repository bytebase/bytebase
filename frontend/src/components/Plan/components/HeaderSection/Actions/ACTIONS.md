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
│   │   └── rollout.ts           # ROLLOUT_CREATE, ROLLOUT_START, ROLLOUT_CANCEL
│   └── components/
│       ├── ActionButton.vue     # Single action button
│       └── ActionDropdown.vue   # Secondary actions dropdown
├── unified/                     # Legacy components (compatibility layer)
│   ├── UnifiedActionGroup.vue   # Button group with dropdown
│   └── IssueReviewButton.vue    # Special review popover
└── Actions.vue                  # Main orchestrator
```

### Key Concepts

**ActionContext**: Single reactive object containing all state needed for action decisions:
- Entities: plan, issue, rollout, project
- Derived flags: isIssueOnly, isExportPlan, isCreator
- Permissions: updatePlan, createIssue, updateIssue, createRollout, runTasks
- Validation: hasEmptySpec, planChecksRunning, planChecksFailed

**ActionDefinition**: Declarative action with pure functions:
```typescript
interface ActionDefinition {
  id: UnifiedActionType;
  label: (ctx: ActionContext) => string;
  buttonType: "primary" | "success" | "default";
  category: "primary" | "secondary";
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

**Description**: Reopens a closed or resolved issue.

**Visibility**:
- Issue exists and is CANCELED or DONE
- User has `bb.issues.update` permission

---

### Rollout Actions

#### `ROLLOUT_CREATE` (Primary, Priority: 55)
**Label**: "Create Rollout"

**Description**: Creates a rollout from the approved issue.

**Visibility**:
- Plan has no rollout attached
- Issue exists
- User has `bb.rollouts.create` permission
- Rollout preconditions met (approval status, plan checks)

---

#### `ROLLOUT_START` (Primary, Priority: 60)
**Label**:
- "Export" (for export data issues)
- "Rollout" (for other issues)

**Description**: Starts or retries tasks in the rollout.

**Visibility**:
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

- **registry/**: Action registry (source of truth)
- **Actions.vue**: Main orchestrator
- **unified/**: Legacy components (compatibility layer)
