# SQL Editor React Migration — Stage 9 Design

**Date:** 2026-04-21
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate the AccessPane subtree (3 Vue files, ~800 lines) to React with full feature parity. Add an `"execute-sql"` event to the Stage 8 Emittery bridge with a Vue listener in `StandardPanel.vue` so the React `AccessPane` can request SQL execution without porting the 364-line `useExecuteSQL` composable.

**Non-goals:**
- Porting `useExecuteSQL.ts` to React — deferred to a dedicated stage.
- Migrating any other aside panel subtree (SchemaPane, WorksheetPane, TabList).
- AI plugin context bridge or OpenAIButton migration.
- FolderForm / tree primitive work.
- Any Vue file beyond the 3 AccessPane files and the targeted modifications to events.ts / StandardPanel.vue / AsidePanel.vue.

## 2. Files involved

**Create (React):**
- `frontend/src/react/components/sql-editor/AccessPane.tsx` (~350 lines)
- `frontend/src/react/components/sql-editor/AccessPane.test.tsx`
- `frontend/src/react/components/sql-editor/AccessGrantItem.tsx` (~250 lines)
- `frontend/src/react/components/sql-editor/AccessGrantItem.test.tsx`
- `frontend/src/react/components/sql-editor/AccessGrantRequestDrawer.tsx` (~350 lines)
- `frontend/src/react/components/sql-editor/AccessGrantRequestDrawer.test.tsx`

**Modify:**
- `frontend/src/views/sql-editor/events.ts` — add `"execute-sql"` event type
- `frontend/src/views/sql-editor/EditorPanel/StandardPanel/StandardPanel.vue` — subscribe to `execute-sql` event via `useEmitteryEventListener`, call existing `execute` handler
- `frontend/src/views/sql-editor/AsidePanel/AsidePanel.vue:58` — swap `<AccessPane v-if="asidePanelTab === 'ACCESS'" />` to `<ReactPageMount v-if="asidePanelTab === 'ACCESS'" page="AccessPane" />` and drop the `AccessPane` import
- React locales (5 files) — add missing i18n keys with Vue `{var}` → React `{{var}}` conversion

**Delete:** `frontend/src/views/sql-editor/AsidePanel/AccessPane/` (3 files + any `index.ts` after zero-caller verification)

## 3. `execute-sql` event bridge

### 3.1 Event type

Add to `frontend/src/views/sql-editor/events.ts`'s `SQLEditorEvents` union:

```ts
"execute-sql": {
  connection: SQLEditorConnection;
  statement: string;
  batchQueryContext?: BatchQueryContext;
};
```

Import types as needed (`SQLEditorConnection`, `BatchQueryContext`) from `@/types`.

### 3.2 Vue handler in StandardPanel.vue

Adds a listener inside the existing `<script setup>` block using the existing Vue composable pattern:

```ts
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useDatabaseV1Store } from "@/store";
import { sqlEditorEvents } from "@/views/sql-editor/events";
// (existing imports — keep)

// After `const { execute } = useExecuteSQL();` (or wherever execute is acquired):
useEmitteryEventListener(
  sqlEditorEvents,
  "execute-sql",
  async ({ connection, statement, batchQueryContext }) => {
    await useDatabaseV1Store().getOrFetchDatabaseByName(connection.database);
    const tab = tabStore.addTab(
      { connection, statement, batchQueryContext },
      /* beside */ true
    );
    nextTick(() => {
      const instance = getInstanceResource(tab.connection); // import from @/utils if not already
      execute({
        connection: { ...tab.connection },
        statement,
        engine: instance.engine,
        explain: false,
        selection: null,
        limit: 0,
      });
    });
  }
);
```

### 3.3 React emit point

Inside `AccessPane.tsx`'s Run handler:

```ts
import { sqlEditorEvents } from "@/views/sql-editor/events";

const handleRun = async (grant: AccessGrant) => {
  const database = grant.targets[0] ?? "";
  const instanceName = database.replace(/\/databases\/.*$/, "");
  await sqlEditorEvents.emit("execute-sql", {
    connection: { instance: instanceName, database },
    statement: grant.query,
    batchQueryContext: { databases: grant.targets },
  });
};
```

Note: this keeps the 364-line `useExecuteSQL` composable Vue-side. Future stages may port it; Stage 9 bridges via event.

## 4. React files — feature parity map

### 4.1 `AccessPane.tsx`

Container + search + grants list + empty/loading states.

| Vue feature | React mapping |
|---|---|
| `AdvancedSearch` (Vue) | React `AdvancedSearch` from `@/react/components/AdvancedSearch` |
| `FeatureBadge` for JIT gating | React `FeatureBadge` from `@/react/components/FeatureBadge` |
| `PermissionGuardWrapper` | React `PermissionGuard` render-prop pattern (see `@/react/components/PermissionGuard`) |
| `MaskSpinner` loading overlay | `Loader2` + `animate-spin` (Stage 8 pattern) |
| `hasFeature(PlanFeature.FEATURE_JIT)` | Same — imported from `@/store` |
| Search params / `AccessFilter` / `getValueFromSearchParams` / `getValuesFromSearchParams` | Same — pure utilities, work in React |
| Responsive small-layout when `containerWidth < 250` | React `useRef<HTMLDivElement>` + `ResizeObserver` or inline `useLayoutEffect` pattern |
| `fetchAccessGrants` + `fetchIssuesForPendingGrants` | Same logic; `useState` + `useEffect` |
| Project/filter change refetch | `useEffect` with appropriate deps |
| `highlightAccessGrantName` auto-clear after 3s | Read via `useVueState(() => uiStore.highlightAccessGrantName)` from `useSQLEditorUIStore`; use `setTimeout` + direct store mutation to clear |
| Run grant | Emits `execute-sql` event (§3.3) |
| Request pre-fill | Local state → pass props to `<AccessGrantRequestDrawer>` |
| Open drawer | `useState<boolean>` for `showDrawer` |

**Props:** zero (mounted via `<ReactPageMount page="AccessPane" />`).

**Stores at top-level:** `useProjectV1Store`, `useSQLEditorStore`, `useSQLEditorTabStore`, `useAccessGrantStore`, `useIssueV1Store`, `useSQLEditorUIStore`. Get instance via `useConnectionOfCurrentSQLEditorTab` (composable — verify React usage works; may need a `useVueState` wrapper).

### 4.2 `AccessGrantItem.tsx`

Individual grant row.

| Vue feature | React mapping |
|---|---|
| `NTag` status badge, unmask badge | shadcn `Badge` with variant per status tag color |
| `NEllipsis` for expiration | `truncate` + Stage 6 `Tooltip` on overflow |
| `NTooltip` query-preview popover | React `Tooltip` primitive with pre block |
| Status display utilities (`getAccessGrantDisplayStatus`, `getAccessGrantStatusTagType`, `getAccessGrantDisplayStatusText`, `getAccessGrantExpirationText`, `getAccessGrantExpireTimeMs`) | Same — imported from `@/utils/accessGrant` |
| Database names display (2 + "and N more") | Same logic; use i18n with `{{n}}` |
| `Run` / `Re-request` / `View Issue` buttons | shadcn `Button` with `size="sm"` variants |
| `highlight-pulse` CSS keyframe animation | Inline `<style>` scoped or use `@keyframes` in component-local stylesheet OR reuse a utility class from tailwind config (implementation's choice) |

**Props:** `{ grant: AccessGrant; highlight?: boolean; issue?: Issue; onRun: (grant) => void; onRequest: (grant) => void }`

### 4.3 `AccessGrantRequestDrawer.tsx`

Request form drawer.

| Vue feature | React mapping |
|---|---|
| `Drawer` + `DrawerContent` (Vue v2) | React `Sheet` + `SheetContent` with `width="wide"` (832px) |
| `BBAttention type="info"` | React `Alert` (info variant) |
| `DatabaseSelect` (Vue, multiple) | React `DatabaseSelect` at `@/react/components/DatabaseSelect` |
| `MonacoEditor` (Vue) | React `MonacoEditor` at `@/react/components/monaco/MonacoEditor` |
| `NCheckbox` | Use native `<input type="checkbox">` styled via Tailwind OR `Switch` primitive if more shadcn-idiomatic. Check project conventions. |
| `NSelect` for duration (1h/4h/1d/7d/custom) | shadcn `Combobox` or `Select` |
| `NDatePicker` for custom expire | React `ExpirationPicker` from `@/react/components/ui/expiration-picker` |
| `NInput` textarea for reason | shadcn `Textarea` |
| `RequiredStar` | Inline `<span className="text-error">*</span>` |
| Access grant API call | Same `accessGrantServiceClientConnect.createAccessGrant` — protos framework-agnostic |
| On success: open issue OR set `asidePanelTab + highlightAccessGrantName` | Use `useSQLEditorUIStore()` for writes |

**Props:** `{ targets?: string[]; query?: string; unmask?: boolean; onClose: () => void }` — matches Vue's defineProps + emit close.

**Duration mapping:** `duration === -1` → custom datetime via `ExpirationPicker`. Other values → TTL seconds = `duration * 3600` (hours → seconds).

## 5. i18n keys

Verify these in all 5 React locales; add missing with byte-exact Vue values (convert `{var}` → `{{var}}` for react-i18next):

- `sql-editor.request-access`
- `sql-editor.no-access-requests`
- `sql-editor.grant-type-unmask`
- `sql-editor.access-type-unmask`
- `sql-editor.expire-at` (`{time}` → `{{time}}`)
- `sql-editor.expire-in` (`{time}` → `{{time}}`)
- `sql-editor.re-request`
- `sql-editor.view-issue`
- `sql-editor.request-data-access`
- `sql-editor.only-select-allowed`
- `sql-editor.duration-hours` (`{hours}` → `{{hours}}`)
- `sql-editor.duration-day` (`{days}` → `{{days}}`)
- `sql-editor.duration-days` (`{days}` → `{{days}}`)
- `sql-editor.and-n-more-databases` (`{n}` → `{{n}}`)
- `common.custom`, `common.run`, `common.submit`, `common.created`, `common.databases`, `common.statement`, `common.expiration`, `common.reason`
- `issue.access-grant.expired-at`
- `issue.advanced-search.filter`

Most `common.*` keys likely already exist. Priority verification: the 4 interpolation keys (`expire-at`, `expire-in`, `duration-*`, `and-n-more-databases`).

## 6. Verification

### 6.1 Per-stage

- `pnpm fix && check && type-check && test` all green
- ~15 new tests across 3 component test files
- No pre-existing flaky regressions

### 6.2 Manual UX (critical full-parity checklist)

1. Open aside panel → ACCESS tab → empty state shows when no grants, or filtered grants list shows when present
2. AdvancedSearch filters by status (default: ACTIVE + PENDING) and database; changes trigger refetch
3. JIT feature gating: `FeatureBadge` visible when plan lacks JIT feature; Request Access button disabled
4. Missing permission: Request Access button disabled with PermissionGuard tooltip
5. Click Request Access → drawer opens; DatabaseSelect populated; Monaco SQL editor functional; duration picker works; custom datetime picker shows when duration=Custom
6. Submit → access grant created → "Created" toast; if PENDING+issue, opens new tab to issue; otherwise sets `asidePanelTab=ACCESS` + highlights new grant
7. Click Run on ACTIVE grant → emits `execute-sql` event → new tab opens + query executes (via Vue handler in StandardPanel)
8. Click Re-request on rejected/canceled grant → drawer opens with grant.query, grant.unmask, grant.targets pre-filled
9. Click View Issue → opens issue detail in new tab
10. Highlight pulse animation fires for 3s when `highlightAccessGrantName` is set
11. Pagination "Load more" works

### 6.3 Bridge integrity

- Existing Vue flows unchanged: save-sheet (Ctrl+S), auto-save, project switch, query execution from run button
- New event flow: React AccessPane → `execute-sql` → StandardPanel Vue listener → `execute()` → tab opens + query runs

## 7. Practical checklist

- [ ] `events.ts` adds `execute-sql` event type
- [ ] `StandardPanel.vue` subscribes via `useEmitteryEventListener`
- [ ] `AccessPane.tsx` + test
- [ ] `AccessGrantItem.tsx` + test
- [ ] `AccessGrantRequestDrawer.tsx` + test
- [ ] i18n keys verified/added (with `{var}` → `{{var}}` conversion)
- [ ] `AsidePanel.vue:58` swapped + `AccessPane` import removed
- [ ] Vue `AsidePanel/AccessPane/` directory deleted
- [ ] `pnpm fix && check && type-check && test` all pass
- [ ] Manual UX from §6.2 verified

## 8. Out of scope (deferred)

- `useExecuteSQL` React port (keeps 364-line Vue composable; event-bridged per §3)
- Vue v2 component ports (CopyButton, BBAttention as a separate primitive, etc.) — Alert + inline copy patterns used instead
- Other aside panel subtrees (SchemaPane, WorksheetPane, TabList)
- AI plugin bridge, OpenAIButton, FolderForm
