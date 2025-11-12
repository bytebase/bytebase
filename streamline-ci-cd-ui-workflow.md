# Streamlining the CI/CD UI Workflow

## Key Concepts

- **Plan**: A template that specifies what database operations to perform (schema changes, data migrations, database creation, or data export)
- **Issue**: An operational wrapper that links a plan to an execution context with approval workflows and status tracking
- **Rollout**: The execution instance that splits a plan into stages (by environment) and tasks (by database/operation)

```
Project
├─ Plan (template/specification)
│  ├─ Issue (approval flow wrapper) [optional]
│  └─ Rollout (execution)
│     └─ Stages (environment-based)
│        └─ Tasks (database-based)
│           └─ TaskRuns (execution attempts)
```

## Problems

### 1. Too Many Navigations and Clicks

Related Linear issue: BYT-8054

#### Current Experience

- **Issue Review Workflow**

  1. Click link → lands on Issue page `/issues/456` (Overview tab)
  2. Sees title, description, labels and approval status
  3. ❓ "What SQL am I approving?"
  4. Click **Changes** tab
  5. Scroll through SQL changes
  6. Click back to **Overview** tab
  7. Click "Approve" button
  8. Total: 3 tab switches, 5+ clicks

* **Plan to Issue to Rollout Navigation**

  1. Create plan at `/plans/123`
  2. Click "Ready for Review" → redirected to `/issues/456`
  3. After approval → redirected to `/rollouts/789`
  4. Total: 3 pages, 3 redirects, context lost each time

#### Root Causes

- **Redirect-Heavy Workflow**: View plans/issues forces navigation away from current context
- **Tab-Based Critical Information**: SQL changes hidden in separate tab from approval actions

#### Impact

- Users lose track of "Where am I in the workflow? What's next?"
- Batch review of 10 issues = 30+ tab switches

### 2. Information Needed for Decision-Making Is on Different Pages

Related Linear issue: BYT-7911

#### Current Experience

- **Rollout Page Missing SQL Context**

  1. Releaser monitors tasks at `/rollouts/789`
  2. Sees: Stage progress, task status, execution logs
  3. ❓ "What SQL is running?"
  4. Must navigate to task detail page `/rollouts/789/stages/prod/tasks/100` to see SQL statements
  5. Navigate back to `/rollouts/789` to continue monitoring
  6. Problem: Can't see what you're running while monitoring task

- **Plan Page Missing Approval Status**

  1. DBA views plan at `/plans/123`
  2. Sees: SQL specs, plan checks
  3. ❓ "Who has approved this? Is it ready to run?"
  4. Must navigate to `/issues/456` to see approval status
  5. Problem: Can't see approval progress from plan page

#### Root Causes

- **Missing Status Context**: Plan/Issue pages don't show rollout status
- **No Cross-Page Context**: Each page only shows its own entity, not related information

#### Impact

- Context loss from constant redirects between plan/issue/rollout
- Releaser can't quickly verify SQL being executed

## Solution Approach

### Approach 1: Unified Plan/Issue Page

Use same UI components for plan and issue pages. Remove tabs, use consistent layout.

#### Key Changes

- **Shared components**: Plan and issue pages use same layout components
- **No tabs**: Replace tabs with scrollable sections
- **Consistent experience**: Same visual structure whether viewing plan or issue

#### Page Organization

**Always show:**

- Header: Title, status badge, contextual actions
- Progress indicator (see Approach 2)

**Section priority by state:**

1. **Draft**: SQL changes → Checks
2. **Review**: SQL changes → Checks -> Approval status -> Comments
3. **Rolling Out**: Task status → SQL (collapsible) → Logs
4. **Done**: Execution summary → SQL (collapsible) → Logs

#### Key Principle

Put most important information first based on what user needs to do:

- Drafting: See and edit SQL
- Reviewing: See SQL to approve
- Approved: See task status

#### Benefits

- No tab switching to see SQL while approving
- Clear visual progression through workflow

### Approach 2: Workflow Progress Indicators

Add visual progress bar showing current workflow stage.

#### Key Changes

**Add progress bar to plan/issue/rollout pages at the top:**

```
Draft → Review (2/3 approved) → Rollout → Done
        ↑ Current stage
```

**Show for each stage:**

- Status (not started, in progress, completed, failed)
- Progress detail (e.g., "2 of 3 approved", "Stage 2/3: 67%")
- Blockers if any (e.g., "Waiting approval", "Blocked: SQL check failed")

**Interactions:**

- Click stage → Jump to relevant page section or navigate to corresponding page
- Hover → Show detailed tooltip
- Real-time updates without page refresh

#### Benefits

- Always know current stage and what's next
- Immediately see what's blocking progress
- Quick navigation to relevant sections
- Consistent experience across all pages

### Approach 3: Show SQL Context on Rollout Page

Group tasks by SQL statement in stage view. Show statement snippets in task tables.

#### Key Changes

**1. Stage View - Group by SQL Statement:**

In each stage, group tasks that execute the same SQL statement:

```
Stage: Production
├─ SQL: ALTER TABLE users ADD COLUMN email VARCHAR(255)
│  ├─ Task 1: prod-db-01 ✓ Done
│  ├─ Task 2: prod-db-02 ▶ Running (5s)
│  └─ Task 3: prod-db-03 ○ Pending
│
└─ SQL: CREATE INDEX idx_email ON users(email)
   ├─ Task 4: prod-db-01 ○ Pending
   └─ Task 5: prod-db-02 ○ Pending
```

**2. Task Tables - Show Statement Snippets:**

When viewing tasks in table/list format, show SQL snippet (first line or truncated):

```
| Task      | Database    | Statement                        | Status   | Duration |
|-----------|-------------|----------------------------------|----------|----------|
| Task 1    | prod-db-01  | ALTER TABLE users ADD COLUMN...  | ✓ Done   | 2s       |
| Task 2    | prod-db-02  | ALTER TABLE users ADD COLUMN...  | ▶ Running| 5s       |
| Task 3    | prod-db-03  | CREATE INDEX idx_email ON...     | ○ Pending| -        |
```

Click snippet → Show full SQL in side panel or expand inline

#### Benefits

- Group-level visibility (see how statement is progressing across databases)
- No repeated SQL statements (cleaner UI)
- Quick scanning with snippets in task tables
- Full context available on click
- Easier to correlate failures across tasks executing same SQL

## Implementation Strategy

### Phase 1: Workflow Progress Indicators

- Create progress indicator component
- Add to plan, issue, and rollout page headers
- Show 4 stages: Draft → Review → Rollout → Done
- Display current stage, progress details, and blockers

### Phase 2: SQL Context on Rollout

- Group tasks by SQL statement in stage view
- Add statement column to task tables with snippets
- Implement SQL detail panel for full view

### Phase 3: Unified Plan/Issue Layout

Most complex, requires component refactoring, affects multiple pages.

- Create shared layout components for plan and issue pages
- Remove tabs, implement scrollable sections
- Ensure consistency between plan and issue pages
