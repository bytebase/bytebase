# Project Settings Page React Migration

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the project settings page (`/projects/:projectId/settings`) from Vue to React and remove all unused Vue code.

**Architecture:** Single `ProjectSettingsPage.tsx` React page with inline section components, mounted via `ProjectSettingsPageMount.vue`. The page uses `useVueState` to access Pinia stores and computes `allowEdit` internally. All i18n keys are added to the React locale files. Old Vue components are deleted after migration.

**Tech Stack:** React, Base UI, Tailwind CSS v4, react-i18next, Pinia stores via `useVueState`

---

## File Structure

### New Files
- `frontend/src/react/pages/project/ProjectSettingsPage.tsx` — Main page with all four sections (General, Security & Policy, Issue-Related, Danger Zone) and save/cancel bar
- `frontend/src/react/ProjectSettingsPageMount.vue` — Vue wrapper that mounts the React page

### Modified Files
- `frontend/src/router/dashboard/projectV1.ts:366-374` — Route to point at mount component
- `frontend/src/react/mount.ts:4` — Add `ProjectSettingsPage` to project page loaders (already covered by glob)
- `frontend/src/react/locales/en-US.json` — Add all needed translation keys
- `frontend/src/react/locales/zh-CN.json` — Add translation keys (copy English as placeholder)
- `frontend/src/react/locales/es-ES.json` — Add translation keys (copy English as placeholder)
- `frontend/src/react/locales/ja-JP.json` �� Add translation keys (copy English as placeholder)
- `frontend/src/react/locales/vi-VN.json` — Add translation keys (copy English as placeholder)

### Deleted Files (after migration is verified working)
- `frontend/src/views/project/ProjectSettingPanel.vue`
- `frontend/src/components/ProjectSettingPanel.vue`
- `frontend/src/components/Project/Settings/ProjectGeneralSettingPanel.vue`
- `frontend/src/components/Project/Settings/ProjectSecuritySettingPanel.vue`
- `frontend/src/components/Project/Settings/ProjectIssueRelatedSettingPanel.vue`
- `frontend/src/components/Project/Settings/ApprovalFlowIndicator.vue`
- `frontend/src/components/SQLReview/components/SQLReviewForResource.vue` — Only if no other consumers remain
- `frontend/src/components/SQLReview/components/SQLReviewPolicySelectPanel.vue` — Only if no other consumers remain

---

## Task 1: Add i18n Translation Keys

All user-facing text must come from locale files. Copy the needed keys from the Vue locale structure into the React locale files.

**Files:**
- Modify: `frontend/src/react/locales/en-US.json`
- Modify: `frontend/src/react/locales/zh-CN.json`
- Modify: `frontend/src/react/locales/es-ES.json`
- Modify: `frontend/src/react/locales/ja-JP.json`
- Modify: `frontend/src/react/locales/vi-VN.json`

**Keys to add** (nested under appropriate namespaces — mirror the Vue `frontend/src/locales/en-US.json` structure):

```json
{
  "common": {
    "general": "General",
    "settings": "Settings",
    "name": "Name",
    "project": "Project",
    "cancel": "Cancel",
    "confirm-and-update": "Confirm and update",
    "archive": "Archive",
    "archived": "Archived",
    "restore": "Restore",
    "restored": "Restored",
    "delete": "Delete",
    "deleted": "Deleted",
    "danger-zone": "Danger Zone",
    "archive-resource": "Archive {{type}}",
    "archive-description": "Mark \"{{name}}\" as archived and read-only.",
    "confirm-archive": "Archive Resource?",
    "delete-resource": "Delete {{type}}",
    "delete-resource-description": "Once you delete \"{{name}}\", there is no going back. Please be certain.",
    "confirm-delete": "Delete Resource?",
    "leave-without-saving": "Leave without saving?",
    "required": "Required"
  },
  "settings": {
    "sidebar": {
      "security-and-policy": "Security & Policy"
    },
    "general": {
      "workspace": {
        "maximum-sql-result": {
          "rows": {
            "self": "Maximum SQL result rows",
            "description": "Maximum number of rows returned by SQL queries.",
            "rows": "rows"
          }
        },
        "no-limit": "No limit (0)"
      }
    }
  },
  "sql-review": {
    "title": "SQL Review",
    "configure-policy": "Configure policy",
    "tooltip-for-resource": "It's also configurable at the {{scope}} level. Bytebase will prioritize the environment-level configuration.",
    "policy-updated": "SQL review policy updated"
  },
  "project": {
    "settings": {
      "success-updated": "Project settings updated successfully.",
      "update-failed": "Failed to update the project setting.",
      "confirm-archive-project": "Archiving \"{{name}}\" will make it read-only and hide it from active project lists. All issues, databases, and settings will be preserved. You can restore it later from the archived projects list.",
      "confirm-delete-project": "Deleting \"{{name}}\" will permanently remove the project and all associated data including issues, configurations, and history. All databases in this project will be moved to the default project. This action cannot be undone.",
      "restore": {
        "title": "Restore project",
        "btn-text": "Restore this project"
      },
      "project-labels": {
        "self": "Project Labels",
        "description": "Organize projects with key-value labels (max 64)"
      },
      "issue-related": {
        "self": "Issue Related",
        "labels": {
          "self": "Issue Labels",
          "description": "Labels can be attached in the issues for management",
          "placeholder": "Press Enter to add tag",
          "force-issue-labels": {
            "self": "Require labels for all issues",
            "description": "Require at least one label when creating issues.",
            "warning": "Require at least configured one label"
          }
        },
        "enforce-issue-title": {
          "self": "Require manual title",
          "description": "Require titles for issues and plans to be created by user instead of auto-generated."
        },
        "enforce-sql-review": {
          "self": "Enforce SQL review",
          "description": "Require SQL review checks to pass without errors before allowing issue creation."
        },
        "allow-self-approval": {
          "self": "Allow self approval",
          "description": "Allow the issue creator to approve the issue."
        },
        "allow-request-role": {
          "self": "Allow request role",
          "description": "Allow project members to request roles."
        },
        "allow-jit": {
          "self": "Just-In-Time access",
          "description": "Allow project members to request just-in-time (JIT) access."
        },
        "postgres-database-tenant-mode": {
          "self": "Postgres database tenant mode",
          "description": "The tenant mode for Postgres database. If enable, the issue will be run by the database OWNER. Otherwise, the issue will be run by the instance connection user."
        },
        "max-retries": {
          "self": "Maximum retries count for lock timeout",
          "description": "The maximum retries count for lock timeout when running the task. The default value is 0. Only applicable to Postgres."
        },
        "ci-sampling-size": {
          "self": "CI Data Sampling Size",
          "description": "The maximum number of databases to sample during CI data validation. When disabled (zero), full validation will be performed."
        },
        "parallel_tasks_per_rollout": {
          "self": "Parallel tasks per rollout",
          "description": "Control the number of parallel tasks per rollout. Setting this to zero disables any throttling."
        },
        "require-issue-approval": {
          "self": "Require issue approval before creating rollout",
          "description": "Issue must be approved before creating rollout."
        },
        "require-plan-check-no-error": {
          "self": "Require plan checks to pass before creating rollout",
          "description": "Plan checks must pass without error before creating rollout."
        },
        "approval-flow-configured": "Approval flow configured",
        "approval-flow-fallback": "Using fallback approval flow",
        "approval-flow-not-configured": "No approval flow configured",
        "view-approval-flow": "View approval flow"
      }
    }
  }
}
```

- [ ] **Step 1:** Read the current `frontend/src/react/locales/en-US.json` to see existing keys and avoid duplicates
- [ ] **Step 2:** Add the missing keys from the JSON above to `en-US.json`, merging into existing namespaces. Note: React i18next uses `{{variable}}` interpolation (not `{variable}` like Vue i18n)
- [ ] **Step 3:** Copy the same keys to `zh-CN.json`, `es-ES.json`, `ja-JP.json`, `vi-VN.json` (use English as placeholder). Where a translation already exists in the Vue locale file (`frontend/src/locales/<lang>.json`), use that translation instead
- [ ] **Step 4:** Verify no JSON syntax errors: `node -e "require('./frontend/src/react/locales/en-US.json')"`

---

## Task 2: Create ProjectSettingsPageMount.vue

Vue wrapper component following the `ProjectDatabasesPageMount.vue` pattern.

**Files:**
- Create: `frontend/src/react/ProjectSettingsPageMount.vue`

- [ ] **Step 1:** Create the mount component

```vue
<template>
  <div ref="container" class="h-full" />
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const { locale } = useI18n();
const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

async function render() {
  if (!container.value) return;
  const [{ mountReactPage, updateReactPage }, i18nModule] = await Promise.all([
    import("./mount"),
    import("./i18n"),
  ]);
  if (i18nModule.default.language !== locale.value) {
    await i18nModule.default.changeLanguage(locale.value);
  }
  const pageProps = { projectId: props.projectId };
  if (!root) {
    root = await mountReactPage(
      container.value,
      "ProjectSettingsPage",
      pageProps
    );
  } else {
    await updateReactPage(root, "ProjectSettingsPage", pageProps);
  }
}

onMounted(() => render());
watch(locale, () => render());
watch(
  () => props.projectId,
  () => render()
);
onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
```

Note: `allowEdit` is NOT passed as a prop — the React page computes it internally from the project state and permissions.

- [ ] **Step 2:** Update the route in `frontend/src/router/dashboard/projectV1.ts` to use the mount component. Change lines 371-373:

```typescript
// Before:
component: () => import("@/views/project/ProjectSettingPanel.vue"),
// After:
component: () => import("@/react/ProjectSettingsPageMount.vue"),
```

---

## Task 3: Create ProjectSettingsPage.tsx — General Settings Section

Build the main page skeleton and the General Settings section (project name + project labels).

**Files:**
- Create: `frontend/src/react/pages/project/ProjectSettingsPage.tsx`

**Reference:** `frontend/src/components/Project/Settings/ProjectGeneralSettingPanel.vue`

- [ ] **Step 1:** Create `ProjectSettingsPage.tsx` with the page skeleton and General Settings section

The page component must:
1. Accept `{ projectId: string }` as props
2. Compute `allowEdit` from project state + permissions (matching `ProjectV1Layout.vue:116-122`)
3. Track dirty state across all sections with a single save/cancel bar
4. Implement General Settings: project name input + project labels (key-value editor with validation)

```tsx
// Skeleton structure:
export function ProjectSettingsPage({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  // allowEdit computed from project state + permissions
  const allowEdit = project.state !== State.DELETED &&
    hasProjectPermissionV2(project, "bb.projects.update");

  // General settings state
  const [title, setTitle] = useState(project.title);
  const [labelKVList, setLabelKVList] = useState(...);

  // Dirty tracking
  const isDirty = title !== project.title || labelsChanged;

  // Save handler — calls projectStore.updateProject with updateMask
  // Revert handler — resets state to project values

  return (
    <div className="w-full flex flex-col gap-y-0 pt-4 px-4">
      <div className="divide-y divide-block-border">
        {/* General section */}
        {/* Security section (Task 4) */}
        {/* Issue-related section (Task 5) */}
        {/* Danger zone section (Task 6) */}
        {/* Save/Cancel bar */}
      </div>
    </div>
  );
}
```

For project labels, implement a simple key-value list editor inline:
- Each label has `key` and `value` fields
- Validation: no duplicate keys, no empty keys, value max 63 chars
- Add/remove buttons
- Uses `convertLabelsToKVList` and `convertKVListToLabels` from `@/utils`

- [ ] **Step 2:** Verify the page loads by starting the dev server and navigating to `/projects/<id>/settings`

---

## Task 4: Security & Policy Section

Add the Security & Policy section to `ProjectSettingsPage.tsx`.

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectSettingsPage.tsx`

**Reference:** `frontend/src/components/Project/Settings/ProjectSecuritySettingPanel.vue`

This section contains:
1. **SQL Review** — policy selection with enable/disable toggle. Uses `useSQLReviewStore` and `useReviewPolicyByResource` composable
2. **Maximum SQL Result Rows** — numeric input with feature badge. Uses `usePolicyV1Store` for DATA_QUERY policy
3. **Allow Request Role** — toggle with approval flow indicator
4. **Allow Just-In-Time Access** — toggle with approval flow indicator

- [ ] **Step 1:** Add state variables for security settings

```tsx
// SQL Review state
const reviewStore = useSQLReviewStore();
const sqlReviewPolicy = useVueState(() =>
  reviewStore.getReviewPolicyByResouce(projectName)
);
const [pendingReviewPolicy, setPendingReviewPolicy] = useState<SQLReviewPolicy | undefined>();
const [enforceSQLReview, setEnforceSQLReview] = useState(false);
const [showReviewSelectPanel, setShowReviewSelectPanel] = useState(false);

// Max rows state
const policyStore = usePolicyV1Store();
const queryDataPolicy = useVueState(() =>
  policyStore.getQueryDataPolicyByParent(projectName)
);
const [maximumResultRows, setMaximumResultRows] = useState(0);

// Project toggles
const [allowRequestRole, setAllowRequestRole] = useState(project.allowRequestRole);
const [allowJustInTimeAccess, setAllowJustInTimeAccess] = useState(project.allowJustInTimeAccess);
```

- [ ] **Step 2:** Implement the SQL Review sub-section

Display the currently attached SQL review policy with an enforce toggle and a remove button. If no policy is attached, show a "Configure policy" button that opens a selection panel.

For the SQL review policy select panel, implement as a `Dialog` that lists available policies from `reviewStore.reviewPolicyList` and lets the user pick one.

- [ ] **Step 3:** Implement maximum result rows with `FeatureBadge`

```tsx
import { FeatureBadge } from "@/react/components/FeatureBadge";
```

Use a plain `<input type="number">` styled with Tailwind (or the `Input` UI component). Gate editing on `hasQueryPolicyFeature` from `useSubscriptionV1Store`.

- [ ] **Step 4:** Implement Allow Request Role and Allow JIT Access toggles

Each toggle has:
- A `Switch` component from `@/react/components/ui/switch`
- A label
- An `ApprovalFlowIndicator` — implement inline as a tooltip icon that checks `useWorkspaceApprovalSettingStore` for configured approval rules

```tsx
function ApprovalFlowIndicator({ source }: { source: number }) {
  const approvalStore = useWorkspaceApprovalSettingStore();
  const [ready, setReady] = useState(false);
  useEffect(() => {
    if (hasWorkspacePermissionV2("bb.settings.get")) {
      approvalStore.fetchConfig().then(() => setReady(true));
    }
  }, [approvalStore]);
  if (!ready) return null;
  const hasSource = approvalStore.getRulesBySource(source).length > 0;
  const hasFallback = approvalStore.getRulesBySource(0).length > 0;
  const configured = hasSource || hasFallback;
  // Render ShieldCheck (green) or TriangleAlert (yellow) icon with tooltip
}
```

- [ ] **Step 5:** Wire dirty tracking for security section into the top-level `isDirty`

---

## Task 5: Issue-Related Section

Add the Issue-Related section to `ProjectSettingsPage.tsx`.

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectSettingsPage.tsx`

**Reference:** `frontend/src/components/Project/Settings/ProjectIssueRelatedSettingPanel.vue`

This section contains:
1. **Issue Labels** — dynamic tag input with color picker
2. **8 boolean toggles** — force issue labels, enforce issue title, enforce SQL review, allow self-approval, require issue approval, require plan check no error, postgres database tenant mode
3. **3 numeric inputs** — max retries, CI sampling size, parallel tasks per rollout

- [ ] **Step 1:** Add state variables for all issue-related settings

```tsx
const [issueLabels, setIssueLabels] = useState<Label[]>(cloneDeep(project.issueLabels));
const [forceIssueLabels, setForceIssueLabels] = useState(project.forceIssueLabels);
const [enforceIssueTitle, setEnforceIssueTitle] = useState(project.enforceIssueTitle);
const [enforceSqlReview, setEnforceSqlReview] = useState(project.enforceSqlReview);
const [allowSelfApproval, setAllowSelfApproval] = useState(project.allowSelfApproval);
const [requireIssueApproval, setRequireIssueApproval] = useState(project.requireIssueApproval);
const [requirePlanCheckNoError, setRequirePlanCheckNoError] = useState(project.requirePlanCheckNoError);
const [postgresDatabaseTenantMode, setPostgresDatabaseTenantMode] = useState(project.postgresDatabaseTenantMode);
const [maxRetries, setMaxRetries] = useState(project.executionRetryPolicy?.maximumRetries ?? 0);
const [ciSamplingSize, setCiSamplingSize] = useState(project.ciSamplingSize);
const [parallelTasksPerRollout, setParallelTasksPerRollout] = useState(project.parallelTasksPerRollout);
```

- [ ] **Step 2:** Implement the Issue Labels sub-section

Build a simple tag input:
- Display existing labels as colored badges with an X to remove
- Each badge has a small color swatch (clickable to open a color picker — use a simple `<input type="color">` element)
- Text input at the end to add new labels (on Enter)
- When all labels are removed, auto-disable force issue labels

- [ ] **Step 3:** Implement all boolean toggles

Each toggle follows the same pattern:
```tsx
<div>
  <div className="flex items-center gap-x-2">
    <Switch
      checked={value}
      onCheckedChange={setValue}
      disabled={!allowEdit || loading}
    />
    <span className="textlabel">{t("project.settings.issue-related.xxx.self")}</span>
  </div>
  <div className="mt-1 text-sm text-gray-400">
    {t("project.settings.issue-related.xxx.description")}
  </div>
</div>
```

Gate all toggles on `allowEdit && hasProjectPermissionV2(project, "bb.projects.update")`.

- [ ] **Step 4:** Implement the 3 numeric inputs (max retries, CI sampling size, parallel tasks per rollout)

Use plain `<input type="number" min={0} step={1}>` styled with Tailwind, matching the `w-60` width from the Vue version.

- [ ] **Step 5:** Wire dirty tracking for all issue-related fields into the top-level `isDirty` and build the `updateMask` computation

The update mask for issue-related fields maps to these API field names:
- `issue_labels`, `force_issue_labels`, `enforce_issue_title`, `enforce_sql_review`
- `allow_self_approval`, `require_issue_approval`, `require_plan_check_no_error`
- `postgres_database_tenant_mode`, `execution_retry_policy`
- `ci_sampling_size`, `parallel_tasks_per_rollout`

---

## Task 6: Danger Zone Section

Add the Danger Zone section with archive/restore/delete functionality.

**Files:**
- Modify: `frontend/src/react/pages/project/ProjectSettingsPage.tsx`

**Reference:** `frontend/src/components/ProjectSettingPanel.vue:57-167`

- [ ] **Step 1:** Implement the Danger Zone section

Three actions based on project state:
- **Archive** (when `project.state === State.ACTIVE`) — archives the project
- **Restore** (when `project.state === State.DELETED`) — restores the project
- **Delete** (always shown) — permanently deletes the project

Each action uses a confirmation dialog (`Dialog` from `@/react/components/ui/dialog`).

```tsx
// Pattern for each action:
const [showArchiveConfirm, setShowArchiveConfirm] = useState(false);

const handleArchive = useCallback(async () => {
  setExecuting(true);
  try {
    await projectStore.archiveProject(project);
    pushNotification({ module: "bytebase", style: "SUCCESS",
      title: `${project.title || project.name} ${t("common.archived")}` });
    router.push({ name: PROJECT_V1_ROUTE_DASHBOARD });
  } finally {
    setExecuting(false);
  }
}, [project, projectStore, t]);
```

Permission checks:
- Archive/Delete: `bb.projects.delete`
- Restore: `bb.projects.undelete`

Style the danger zone with: `border border-error-alpha bg-error-alpha rounded-lg divide-y divide-error-alpha`

- [ ] **Step 2:** Implement the save/cancel sticky bar at the bottom

```tsx
{allowEdit && isDirty && (
  <div className="sticky bottom-0 z-10">
    <div className="flex justify-between w-full py-4 border-t border-block-border bg-white">
      <Button variant="outline" onClick={handleRevert}>{t("common.cancel")}</Button>
      <Button onClick={handleSave}>{t("common.confirm-and-update")}</Button>
    </div>
  </div>
)}
```

The `handleSave` function must:
1. Update project fields via `projectStore.updateProject(projectPatch, updateMask)`
2. Update SQL review policy if changed (via `reviewStore.upsertReviewConfigTag` and `reviewStore.upsertReviewPolicy`)
3. Update max rows policy if changed (via `policyStore.upsertPolicy`)
4. Show success/error notification

The `handleRevert` function resets all state variables to current project values.

- [ ] **Step 3:** Add `beforeunload` event listener for unsaved changes protection

```tsx
useEffect(() => {
  if (!isDirty) return;
  const handler = (e: BeforeUnloadEvent) => { e.preventDefault(); };
  window.addEventListener("beforeunload", handler);
  return () => window.removeEventListener("beforeunload", handler);
}, [isDirty]);
```

---

## Task 7: Delete Old Vue Components

Remove all Vue components that are no longer used after the migration.

**Files to delete:**
- `frontend/src/views/project/ProjectSettingPanel.vue`
- `frontend/src/components/ProjectSettingPanel.vue`
- `frontend/src/components/Project/Settings/ProjectGeneralSettingPanel.vue`
- `frontend/src/components/Project/Settings/ProjectSecuritySettingPanel.vue`
- `frontend/src/components/Project/Settings/ProjectIssueRelatedSettingPanel.vue`
- `frontend/src/components/Project/Settings/ApprovalFlowIndicator.vue`

- [ ] **Step 1:** Check for any remaining imports of these components

```bash
grep -r "ProjectSettingPanel\|ProjectGeneralSettingPanel\|ProjectSecuritySettingPanel\|ProjectIssueRelatedSettingPanel\|ApprovalFlowIndicator" frontend/src/ --include="*.vue" --include="*.ts" --include="*.tsx" -l
```

Only proceed with deletion if the only references are:
- The files themselves
- The route (already updated in Task 2)
- Index files that re-export them

- [ ] **Step 2:** Check if `SQLReviewForResource.vue` and `SQLReviewPolicySelectPanel.vue` have other consumers

```bash
grep -r "SQLReviewForResource\|SQLReviewPolicySelectPanel" frontend/src/ --include="*.vue" --include="*.ts" --include="*.tsx" -l
```

If only consumed by the deleted settings panel, delete them too. If consumed elsewhere, leave them.

- [ ] **Step 3:** Delete the identified files and update any index files (e.g., `frontend/src/components/Project/Settings/index.ts`) that re-export them

- [ ] **Step 4:** Clean up empty directories if any

---

## Task 8: Lint, Type-check, and Verify

- [ ] **Step 1:** Run frontend lint fix: `pnpm --dir frontend fix`
- [ ] **Step 2:** Run frontend check: `pnpm --dir frontend check`
- [ ] **Step 3:** Run type-check: `pnpm --dir frontend type-check`
- [ ] **Step 4:** Run tests: `pnpm --dir frontend test`
- [ ] **Step 5:** Manual verification — start dev server and test:
  - Navigate to `/projects/<id>/settings`
  - Verify all four sections render correctly
  - Change project name → verify save/cancel bar appears
  - Toggle settings → save → verify persistence
  - Test archive/restore/delete flows
  - Test browser back with unsaved changes (beforeunload)
  - Verify i18n works (switch language if possible)
