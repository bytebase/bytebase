# Plan List Resizable Columns Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the project plan list (`/projects/<id>/plans`) support drag-to-resize column headers, mirroring the project database list.

**Architecture:** Adopt the column-config pattern already used by `DatabaseTableView`. Define a `PlanColumn[]` array with `defaultWidth`/`minWidth`/`resizable`/`render` inside `PlanTable`. Call the existing shared `useColumnWidths` hook for positional width state and the resize handler. Render `<Table className="table-fixed">` with a `<colgroup>` of `<col>` widths. Pass `resizable`/`onResizeStart` to each `TableHead`. The shared `Table` primitive already renders the `ColumnResizeHandle` on the right edge — no new UI components, no new hooks, no third-party libraries.

**Tech Stack:** React, TypeScript, Tailwind CSS v4, existing `useColumnWidths` hook, existing `Table` primitives (`Table`, `TableHead`, `TableRow`, `TableCell`, `TableBody`, `TableHeader`), existing `ColumnResizeHandle`.

**Spec:** `docs/superpowers/specs/2026-05-15-plan-list-resizable-columns-design.md`

---

## File Structure

- **Modify:** `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx` — rewrite the local `PlanTable` and `PlanRow` functions (roughly lines 376–615). No other section of this file changes.

No new files. No test files (the codebase has no unit tests for the plan list today; the shared `useColumnWidths` hook is already exercised by `DatabaseTableView`).

---

## Task 1: Refactor `PlanTable` and `PlanRow` to use the column-config pattern

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx`

- [ ] **Step 1: Confirm baseline**

Read `frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx`. Expected: `PlanTable` (starts ~line 376) renders a `<Table className="min-w-[1000px]">` with six `<TableHead>` cells using Tailwind width classes (`min-w-80`, `w-50`, `w-35`, `w-65`, `w-38`, `w-38`). `PlanRow` (starts ~line 408) hand-renders six `<TableCell>` blocks in matching order: Title, Checks, Approval, Stages, Updated, Creator.

Also skim `frontend/src/react/components/database/DatabaseTableView.tsx` and `frontend/src/react/hooks/useColumnWidths.ts` to confirm the reference pattern. Do not modify those files.

- [ ] **Step 2: Add `useColumnWidths` to the imports**

In the import block at the top of `ProjectPlanDashboardPage.tsx`, add:

```ts
import { useColumnWidths } from "@/react/hooks/useColumnWidths";
```

Anywhere in the `@/react/hooks/*` group is fine — `pnpm --dir frontend fix` in Step 8 runs Biome `organizeImports`, which will sort it into the final position.

- [ ] **Step 3: Replace `PlanTable`**

Locate the existing `PlanTable` function (the entire function definition, roughly the block from `function PlanTable(` through its closing `}` plus the `// PlanRow` section divider that follows). Replace `PlanTable` with:

```tsx
function PlanTable({ plans, projectId }: { plans: Plan[]; projectId: string }) {
  const { t } = useTranslation();

  const columns = useMemo<PlanColumn[]>(
    () => [
      {
        key: "name",
        title: t("issue.table.name"),
        defaultWidth: 320,
        minWidth: 200,
        resizable: true,
        render: (plan, ctx) => (
          <div className="flex items-center gap-x-2 overflow-hidden">
            <span className="whitespace-nowrap text-control opacity-60">
              {extractPlanUID(plan.name)}
            </span>
            {plan.title ? (
              <span className="truncate normal-nums">{plan.title}</span>
            ) : (
              <span className="opacity-60 italic">{t("common.untitled")}</span>
            )}
            {ctx.isDeleted && (
              <span className="inline-flex items-center rounded-full bg-warning/10 text-warning px-2 py-0.5 text-xs shrink-0">
                {t("common.closed")}
              </span>
            )}
            {ctx.showDraftTag && !ctx.isDeleted && (
              <span className="inline-flex items-center rounded-full bg-control-bg text-control-light px-2 py-0.5 text-xs shrink-0">
                {t("common.draft")}
              </span>
            )}
          </div>
        ),
      },
      {
        key: "checks",
        title: t("plan.checks.self"),
        defaultWidth: 200,
        minWidth: 120,
        resizable: true,
        render: (_plan, ctx) =>
          ctx.hasAnyCheck ? (
            <div className="flex items-center gap-3 flex-wrap">
              {ctx.checkSummary.running > 0 && (
                <div className="flex items-center gap-1 text-control">
                  <Loader2 className="size-4 animate-spin" />
                  <span>{t("task.status.running")}</span>
                </div>
              )}
              {ctx.checkSummary.error > 0 && (
                <div className="flex items-center gap-1 text-error">
                  <XCircle className="size-4" />
                  <span>{ctx.checkSummary.error}</span>
                </div>
              )}
              {ctx.checkSummary.warning > 0 && (
                <div className="flex items-center gap-1 text-warning">
                  <AlertCircle className="size-4" />
                  <span>{ctx.checkSummary.warning}</span>
                </div>
              )}
              {ctx.checkSummary.success > 0 && (
                <div className="flex items-center gap-1 text-success">
                  <CheckCircle className="size-4" />
                  <span>{ctx.checkSummary.success}</span>
                </div>
              )}
            </div>
          ) : (
            <span className="text-control-light">-</span>
          ),
      },
      {
        key: "review",
        title: t("plan.navigator.review"),
        defaultWidth: 140,
        minWidth: 100,
        resizable: true,
        render: (_plan, ctx) =>
          ctx.approvalTag ? (
            <StatusTag
              label={ctx.approvalTag.label}
              variant={ctx.approvalTag.variant}
            />
          ) : (
            <span className="text-control-light">-</span>
          ),
      },
      {
        key: "stages",
        title: t("rollout.stage.self", { count: 2 }),
        defaultWidth: 260,
        minWidth: 140,
        resizable: true,
        render: (plan, ctx) =>
          plan.rolloutStageSummaries.length === 0 ? (
            <span className="text-control-light">-</span>
          ) : (
            <div className="flex items-center gap-1 flex-wrap">
              {plan.rolloutStageSummaries.map((summary, index) => {
                const envName = formatEnvironmentName(
                  extractStageUID(summary.stage)
                );
                const environment =
                  ctx.environmentStore.getEnvironmentByName(envName);
                const stageStatus = getRolloutStageStatus(summary);
                return (
                  <div key={summary.stage} className="flex items-center gap-1">
                    <div className="flex items-center gap-1">
                      <TaskStatusIcon status={stageStatus} />
                      <span className="text-sm">
                        {environment?.title || envName}
                      </span>
                    </div>
                    {index < plan.rolloutStageSummaries.length - 1 && (
                      <span className="mx-1 text-control-light">&rarr;</span>
                    )}
                  </div>
                );
              })}
            </div>
          ),
      },
      {
        key: "updated",
        title: t("issue.table.updated"),
        defaultWidth: 152,
        minWidth: 100,
        resizable: true,
        render: (_plan, ctx) => (
          <Tooltip content={formatAbsoluteDateTime(ctx.updateTimeTs * 1000)}>
            <span className="text-control-light whitespace-nowrap">
              {humanizeTs(ctx.updateTimeTs)}
            </span>
          </Tooltip>
        ),
      },
      {
        key: "creator",
        title: t("issue.table.creator"),
        defaultWidth: 152,
        minWidth: 100,
        resizable: true,
        render: (_plan, ctx) => (
          <div className="flex items-center gap-x-1.5">
            <span className="text-sm truncate">{ctx.creator.title}</span>
          </div>
        ),
      },
    ],
    [t]
  );

  const { widths, totalWidth, onResizeStart } = useColumnWidths(columns);

  // Keep `overflow-x-auto` only on the wrapper — no `border rounded-sm` here
  // even though `DatabaseTableView` uses one. The plan list never had a
  // visible border around the table and we are not adding one as part of
  // this change.
  return (
    <div className="overflow-x-auto">
      <Table className="table-fixed" style={{ minWidth: `${totalWidth}px` }}>
        <colgroup>
          {widths.map((w, i) => (
            <col key={columns[i].key} style={{ width: `${w}px` }} />
          ))}
        </colgroup>
        <TableHeader>
          <TableRow className="bg-control-bg">
            {columns.map((col, colIdx) => (
              <TableHead
                key={col.key}
                resizable={col.resizable}
                onResizeStart={
                  col.resizable ? (e) => onResizeStart(colIdx, e) : undefined
                }
              >
                {col.title}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {plans.map((plan) => (
            <PlanRow
              key={plan.name}
              plan={plan}
              projectId={projectId}
              columns={columns}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
```

- [ ] **Step 4: Add `PlanColumn` and `PlanRowContext` types**

Immediately before the `function PlanTable(...)` definition (and after the `// PlanTable` section comment block), insert:

```tsx
interface PlanColumn {
  key: string;
  title: React.ReactNode;
  defaultWidth: number;
  minWidth?: number;
  resizable?: boolean;
  cellClassName?: string;
  render: (plan: Plan, ctx: PlanRowContext) => React.ReactNode;
}

interface PlanRowContext {
  creator: { title: string; name: string };
  updateTimeTs: number;
  approvalTag:
    | { label: string; variant: "default" | "success" | "warning" | "info" }
    | undefined;
  checkSummary: {
    running: number;
    success: number;
    warning: number;
    error: number;
  };
  hasAnyCheck: boolean;
  isDeleted: boolean;
  showDraftTag: boolean;
  environmentStore: ReturnType<typeof useEnvironmentV1Store>;
}
```

The narrow `{ title; name }` shape is intentional — those are the only fields read from `creator` in any cell. If a future cell needs more user fields, widen to `ReturnType<typeof unknownUser>` (or similar).

- [ ] **Step 5: Replace `PlanRow`**

Locate the existing `PlanRow` function and replace it with:

```tsx
function PlanRow({
  plan,
  projectId,
  columns,
}: {
  plan: Plan;
  projectId: string;
  columns: PlanColumn[];
}) {
  const userStore = useUserStore();
  const environmentStore = useEnvironmentV1Store();
  const { t } = useTranslation();

  const isDeleted = plan.state === State.DELETED;
  const showDraftTag = plan.issue === "" && !plan.hasRollout;

  const creator =
    userStore.getUserByIdentifier(plan.creator) || unknownUser(plan.creator);

  const updateTimeTs = Math.floor(
    getTimeForPbTimestampProtoEs(plan.updateTime, 0) / 1000
  );

  const planUrl = useMemo(() => {
    return router.resolve({
      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
      params: {
        projectId,
        planId: extractPlanUID(plan.name),
      },
    }).fullPath;
  }, [plan.name, projectId]);

  const onRowClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.ctrlKey || e.metaKey) {
        window.open(planUrl, "_blank");
      } else {
        router.push(planUrl);
      }
    },
    [planUrl]
  );

  const approvalTag = useMemo(() => {
    if (plan.issue === "") return undefined;
    switch (plan.approvalStatus) {
      case ApprovalStatus.CHECKING:
        return { label: t("task.checking"), variant: "default" as const };
      case ApprovalStatus.APPROVED:
        return {
          label: t("issue.table.approved"),
          variant: "success" as const,
        };
      case ApprovalStatus.SKIPPED:
        return { label: t("common.skipped"), variant: "default" as const };
      case ApprovalStatus.REJECTED:
        return { label: t("common.rejected"), variant: "warning" as const };
      case ApprovalStatus.PENDING:
        return { label: t("common.under-review"), variant: "info" as const };
      default:
        return undefined;
    }
  }, [plan.approvalStatus, plan.issue, t]);

  const checkSummary = useMemo(() => {
    const statusCount = plan.planCheckRunStatusCount || {};
    const running = statusCount["RUNNING"] || 0;
    const success = statusCount["SUCCESS"] || 0;
    const warning = statusCount["WARNING"] || 0;
    const error = (statusCount["ERROR"] || 0) + (statusCount["FAILED"] || 0);
    return { running, success, warning, error };
  }, [plan.planCheckRunStatusCount]);

  const hasAnyCheck =
    checkSummary.running +
      checkSummary.success +
      checkSummary.warning +
      checkSummary.error >
    0;

  const ctx: PlanRowContext = {
    creator,
    updateTimeTs,
    approvalTag,
    checkSummary,
    hasAnyCheck,
    isDeleted,
    showDraftTag,
    environmentStore,
  };

  return (
    <TableRow
      className={cn("cursor-pointer", isDeleted && "opacity-60")}
      onClick={onRowClick}
    >
      {columns.map((col) => (
        <TableCell
          key={col.key}
          className={cn("overflow-hidden", col.cellClassName)}
        >
          {col.render(plan, ctx)}
        </TableCell>
      ))}
    </TableRow>
  );
}
```

- [ ] **Step 6: Move `getRolloutStageStatus` out of `PlanRow`**

`getRolloutStageStatus` was previously a local function inside `PlanRow`. It is now called from the `stages` column's `render`, which lives inside `PlanTable`'s `useMemo`. Move the helper to module scope (immediately before `PlanTable`, after the types from Step 4):

```ts
function getRolloutStageStatus(
  summary: Plan_RolloutStageSummary
): Task_Status {
  for (const status of TASK_STATUS_FILTERS) {
    if (summary.taskStatusCounts.some((item) => item.status === status)) {
      return status;
    }
  }
  return Task_Status.STATUS_UNSPECIFIED;
}
```

Confirm no copy of `getRolloutStageStatus` remains inside `PlanRow`.

- [ ] **Step 7: Run the type checker**

```bash
pnpm --dir frontend type-check
```

Expected: no errors. If `PlanRowContext.creator` fails to type-check, fall back to the simpler `creator: { title: string; name: string }` shape mentioned in Step 4.

- [ ] **Step 8: Run the linter and formatter**

```bash
pnpm --dir frontend fix
```

Expected: no errors after the auto-fix pass; the file may be reformatted. If anything in the auto-fixed diff looks structural (not cosmetic), inspect it.

- [ ] **Step 9: Manual test — drag-to-resize works**

Start the frontend dev server if not already running:

```bash
pnpm --dir frontend dev
```

In the browser, navigate to a project's plan list (`/projects/<id>/plans`). For each of the six columns (Name, Checks, Review, Stages, Updated, Creator):
1. Hover the right edge of the header. Confirm cursor changes to `col-resize`.
2. Drag right and left. Confirm the column resizes live, only that column changes width, and the cursor stays `col-resize` during the drag.
3. Release. Confirm cursor returns to default and the new width sticks.

Then verify constraints:
- Try to drag a column smaller than its `minWidth`. Confirm it stops at the minimum (200 / 120 / 100 / 140 / 100 / 100 respectively).
- Resize Name to ~600px so the total table exceeds the viewport. Confirm the surrounding container scrolls horizontally.
- Click a plan row. Confirm navigation to the plan detail page still works.
- Navigate away and back. Confirm widths reset to defaults (intentional — no persistence, matching the database list).
- **Unmount-mid-drag regression check:** start a resize drag on any column header and, while the mouse is still down, click a sidebar link to navigate away. After the page changes, confirm the body cursor is not stuck on `col-resize` and text selection works again (the existing `useColumnWidths` cleanup handles this; this is a regression check for our adoption, not a behavior we are adding).

- [ ] **Step 10: Manual test — visual parity**

Confirm nothing regressed in the cell rendering:
- The Name column shows `<UID> <title>` with the `Closed`/`Draft` tags in the right cases.
- The Checks column shows the running spinner / red error count / yellow warning count / green success count, with `flex-wrap` if a row has many states.
- The Review column shows the `StatusTag` (Under review, Approved, etc.) or `-`.
- The Stages column shows each stage with its task status icon and label, with `→` between stages, wrapping if needed.
- The Updated column shows the relative time with a tooltip for the absolute time.
- The Creator column shows the creator title, truncated if too long.

- [ ] **Step 11: Commit**

```bash
git add frontend/src/react/pages/project/ProjectPlanDashboardPage.tsx
git commit -m "$(cat <<'EOF'
feat(plan): make plan list columns resizable

Adopt the database list's column-config pattern: PlanTable now owns a
columns array with defaultWidth/minWidth/resizable/render, calls the
shared useColumnWidths hook, and renders <Table className="table-fixed">
with a <colgroup>. PlanRow drives its cells from the same columns array
via a PlanRowContext object carrying per-row derived state.

Widths are not persisted, matching the database list. Default widths
match the previous Tailwind classes (min-w-80 / w-50 / w-35 / w-65 /
w-38 / w-38).
EOF
)"
```

---

## Out of scope

- Column reorder (drag-and-drop headers).
- Row drag-to-reorder.
- Width persistence across remounts.
- Any change to `useColumnWidths`, `Table` primitives, `ColumnResizeHandle`, or `DatabaseTableView`.
- Any change to the page-level search, paged footer, drawer, or other helpers in `ProjectPlanDashboardPage.tsx`.
