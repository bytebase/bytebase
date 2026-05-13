# Phase A — Strip Legacy Primitives: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete 21 orphan `.vue` files in `frontend/src/components/` and add a CI guard preventing any future `.vue` import from React-side code under `frontend/src/react/`.

**Architecture:** Single PR. Every targeted Vue file has zero remaining React-side callers (audit verified) and either zero callers anywhere or only sibling `index.ts` re-exports. The CI guard is a new vitest test that globs `frontend/src/react/**/*.{ts,tsx}` and asserts no source contains a `.vue` import, with a small explicit allowlist for mount-bridges (deferred to Phase B). The plan is "delete + verify + commit" repeated by group; no behavioral changes.

**Tech Stack:** Vitest, TypeScript, ripgrep, pnpm.

**Spec:** [`docs/plans/2026-05-12-phase-a-legacy-primitives-design.md`](./2026-05-12-phase-a-legacy-primitives-design.md)

---

## Pre-work: Confirm baseline

Before starting any task, confirm the working tree is clean and the baseline checks pass.

- [ ] **Step 0.1:** `git status` — confirm clean tree (or only untracked plan docs).
- [ ] **Step 0.2:** `pnpm --dir frontend type-check` — confirm passes.
- [ ] **Step 0.3:** `pnpm --dir frontend test` — confirm passes (existing tests, including `no-legacy-vue-deps.test.ts`).

If any baseline check fails, stop and report — do not start deletions.

---

## Task 1: Add the React→Vue import guard

**Goal:** New vitest test that fails if any file under `frontend/src/react/**/*.{ts,tsx}` contains a `.vue` import, except a small allowlist of mount-bridges deferred to Phase B.

**Files:**
- Create: `frontend/src/react/no-react-to-vue-imports.test.ts`

- [ ] **Step 1.1: Write the test**

Create `frontend/src/react/no-react-to-vue-imports.test.ts`:

```typescript
import { describe, expect, test } from "vitest";

// Every file under frontend/src/react/ that is *.ts or *.tsx (the React layer).
// .vue files are skipped: by definition .vue is Vue-side and is allowed to import
// other .vue files.
const sources = import.meta.glob("./**/*.{ts,tsx}", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

// Mount-bridge Vue files that React code is permitted to import until Phase B
// retires the Vue app shell. Adding new entries here requires explicit review.
const allowedVueImports = new Set([
  "@/components/SessionExpiredSurfaceMount.vue",
  "@/components/AgentWindowMount.vue",
]);

const vueImportPattern = /from\s+["']([^"']+\.vue)["']/g;

describe("React layer must not import .vue files", () => {
  test("no .tsx or .ts file under frontend/src/react/ imports a .vue file", () => {
    const violations: string[] = [];
    for (const [file, source] of Object.entries(sources)) {
      // Don't scan this guard itself (it contains .vue strings as test data
      // in the allowlist above).
      if (file.endsWith("/no-react-to-vue-imports.test.ts")) continue;
      // Don't scan the sibling guard (same reason — it has .vue strings as
      // banned-import test data).
      if (file.endsWith("/no-legacy-vue-deps.test.ts")) continue;

      let match: RegExpExecArray | null;
      vueImportPattern.lastIndex = 0;
      while ((match = vueImportPattern.exec(source)) !== null) {
        const importPath = match[1];
        if (!allowedVueImports.has(importPath)) {
          violations.push(`${file}: ${importPath}`);
        }
      }
    }
    expect(violations).toEqual([]);
  });
});
```

- [ ] **Step 1.2: Run the test — verify it passes on the current tree**

Run: `pnpm --dir frontend exec vitest run src/react/no-react-to-vue-imports.test.ts`

Expected: 1 test passing. The only legitimate Vue import in the React layer is `SessionExpiredSurfaceMount.vue` from `SessionExpiredSurface.test.tsx`, which is allowlisted.

**If the test fails** with violations: stop. The audit missed a real React→Vue import. Report which file and import; do not proceed to deletions.

- [ ] **Step 1.3: Commit**

```bash
git add frontend/src/react/no-react-to-vue-imports.test.ts
git commit -m "test(frontend): add CI guard blocking React-layer imports of .vue files

Mount-bridges (SessionExpiredSurfaceMount, AgentWindowMount) are
allowlisted until Phase B retires the Vue app shell.
"
```

---

## Task 2: Delete files with zero callers anywhere

**Goal:** Delete 9 Vue files that have zero importers (no Vue callers, no React callers, no `index.ts` re-exports).

**Files to delete:**
- `frontend/src/components/RequiredStar.vue`
- `frontend/src/components/EditEnvironmentDrawer.vue`
- `frontend/src/components/Permission/NoPermissionPlaceholder.vue`
- `frontend/src/components/misc/MaskSpinner.vue`
- `frontend/src/components/DatabaseInfo.vue`
- `frontend/src/components/Instance/InstanceSyncButton.vue`
- `frontend/src/components/DatabaseDetail/SyncDatabaseButton.vue`
- `frontend/src/components/RoleGrantPanel/MaxRowCountSelect.vue`
- `frontend/src/components/misc/SQLUploadButton.vue`

- [ ] **Step 2.1: Re-verify each file has zero callers**

For each file in the list, run:

```bash
for f in RequiredStar EditEnvironmentDrawer NoPermissionPlaceholder MaskSpinner DatabaseInfo InstanceSyncButton SyncDatabaseButton MaxRowCountSelect SQLUploadButton; do
  echo "=== $f.vue ==="
  rg -l --type-add 'tsx:*.tsx' --type-add 'vue:*.vue' -tts -tvue -ttsx "${f}\.vue" frontend/src/ 2>/dev/null \
    | grep -v "^frontend/src/components/.*${f}\.vue$"
done
```

Expected: each file shows zero callers (lines may appear only in `.tsx` files that contain the basename in a code comment like `// Inline replacement for SyncDatabaseButton.vue` — manually verify these are comments, not imports).

The two known comment-only hits are:
- `frontend/src/react/components/sql-editor/ResultView/ResultView.tsx` references `SyncDatabaseButton.vue` in a comment
- `frontend/src/react/components/sql-editor/StandardPanel/SQLUploadButton.tsx` references `SQLUploadButton.vue` in a comment

If any file shows a real `import ... from "...<file>.vue"` line, stop and report — that file belongs in a different bucket.

- [ ] **Step 2.2: Delete the files**

```bash
git rm \
  frontend/src/components/RequiredStar.vue \
  frontend/src/components/EditEnvironmentDrawer.vue \
  frontend/src/components/Permission/NoPermissionPlaceholder.vue \
  frontend/src/components/misc/MaskSpinner.vue \
  frontend/src/components/DatabaseInfo.vue \
  frontend/src/components/Instance/InstanceSyncButton.vue \
  frontend/src/components/DatabaseDetail/SyncDatabaseButton.vue \
  frontend/src/components/RoleGrantPanel/MaxRowCountSelect.vue \
  frontend/src/components/misc/SQLUploadButton.vue
```

- [ ] **Step 2.3: Check if any now-empty parent directories should be removed**

After deletion, check directories that might be empty:

```bash
for dir in frontend/src/components/Permission frontend/src/components/Instance frontend/src/components/DatabaseDetail frontend/src/components/RoleGrantPanel; do
  if [ -d "$dir" ]; then
    contents=$(ls "$dir" 2>/dev/null)
    if [ -z "$contents" ]; then
      echo "EMPTY: $dir — remove"
      rmdir "$dir"
    else
      echo "$dir has: $contents"
    fi
  fi
done
```

For each remaining directory: if every file in it is also being deleted in this PR or has zero callers, plan to delete in a later task. Otherwise leave alone.

Expected as of audit:
- `Permission/` — `PermissionGuardWrapper.vue` remains (has callers); leave dir.
- `Instance/` — should be empty after deletion; `rmdir`.
- `DatabaseDetail/` — should be empty; `rmdir`.
- `RoleGrantPanel/` — should be empty; `rmdir`.

- [ ] **Step 2.4: Verify build & guard still pass**

```bash
pnpm --dir frontend type-check
pnpm --dir frontend exec vitest run src/react/no-react-to-vue-imports.test.ts src/react/no-legacy-vue-deps.test.ts
```

Expected: both pass.

- [ ] **Step 2.5: Commit**

```bash
git add -A frontend/src/components/
git commit -m "chore(frontend): drop orphan Vue files with zero callers

Removes 9 .vue files whose call sites already migrated to React or
were never used. Empty parent directories cleaned up.
"
```

---

## Task 3: Delete the v2/TabFilter directory

**Goal:** Remove `v2/TabFilter/TabFilter.vue` and its `index.ts` re-export plumbing. Update `v2/index.ts` to stop re-exporting from `./TabFilter`.

**Files:**
- Delete: `frontend/src/components/v2/TabFilter/TabFilter.vue`
- Delete: `frontend/src/components/v2/TabFilter/index.ts`
- Delete: `frontend/src/components/v2/TabFilter/types.ts` (if no external callers — verify in step 3.1)
- Modify: `frontend/src/components/v2/index.ts` — remove `export * from "./TabFilter";`

- [ ] **Step 3.1: Verify no external caller imports TabFilter or its types**

```bash
echo "=== Imports of TabFilter (outside the dir itself) ==="
rg -n --type-add 'tsx:*.tsx' --type-add 'vue:*.vue' -tts -tvue -ttsx 'TabFilter' frontend/src/ 2>/dev/null \
  | grep -v '^frontend/src/components/v2/TabFilter/'

echo ""
echo "=== Imports of v2/TabFilter/types ==="
rg -n --type-add 'tsx:*.tsx' --type-add 'vue:*.vue' -tts -tvue -ttsx 'v2/TabFilter/types' frontend/src/ 2>/dev/null
```

Expected: only `frontend/src/components/v2/index.ts:2:export * from "./TabFilter";` (which we will remove) and zero types imports.

If any other file imports `TabFilter` (e.g., `from "@/components/v2"`), stop and report — TabFilter has hidden consumers and cannot be deleted in this PR.

- [ ] **Step 3.2: Remove the export line from v2/index.ts**

Edit `frontend/src/components/v2/index.ts`:

```typescript
// Before
export * from "./Select";
export * from "./TabFilter";
export * from "./Model";
export * from "./Form";
export * from "./Button";
export * from "./Container";

// After
export * from "./Select";
export * from "./Model";
export * from "./Form";
export * from "./Button";
export * from "./Container";
```

- [ ] **Step 3.3: Delete the directory**

```bash
git rm -r frontend/src/components/v2/TabFilter/
```

- [ ] **Step 3.4: Verify build still passes**

```bash
pnpm --dir frontend type-check
```

Expected: pass.

- [ ] **Step 3.5: Commit**

```bash
git add frontend/src/components/v2/index.ts
git commit -m "chore(frontend): drop unused v2/TabFilter

TabFilter had no callers outside its own re-export plumbing in v2/index.ts.
"
```

---

## Task 4: Delete FeatureGuard/FeatureAttention.vue

**Goal:** Delete the orphan `FeatureAttention.vue` (React callers already use `@/react/components/FeatureAttention`). Update `FeatureGuard/index.ts` to stop re-exporting it. `FeatureBadge.vue` and `FeatureModal.vue` stay — they have Vue callers (`RoleSelect.vue`, `DatabaseView.vue`).

**Files:**
- Delete: `frontend/src/components/FeatureGuard/FeatureAttention.vue`
- Modify: `frontend/src/components/FeatureGuard/index.ts`

- [ ] **Step 4.1: Verify FeatureAttention has no callers**

```bash
rg -n --type-add 'tsx:*.tsx' --type-add 'vue:*.vue' -tts -tvue -ttsx 'FeatureAttention' frontend/src/ 2>/dev/null \
  | grep -v '^frontend/src/components/FeatureGuard/FeatureAttention\.vue:' \
  | grep -v '^frontend/src/react/components/FeatureAttention\.tsx:'
```

Expected: only `frontend/src/components/FeatureGuard/index.ts` matches (the re-export we will remove) and React-side files importing `@/react/components/FeatureAttention` (the React replacement, not the Vue file).

If a Vue file (`.vue`) other than `index.ts` matches, stop and report.

- [ ] **Step 4.2: Update FeatureGuard/index.ts**

Edit `frontend/src/components/FeatureGuard/index.ts`:

```typescript
// Before
import FeatureAttention from "./FeatureAttention.vue";
import FeatureBadge from "./FeatureBadge.vue";
import FeatureModal from "./FeatureModal.vue";

export { FeatureAttention, FeatureBadge, FeatureModal };

// After
import FeatureBadge from "./FeatureBadge.vue";
import FeatureModal from "./FeatureModal.vue";

export { FeatureBadge, FeatureModal };
```

- [ ] **Step 4.3: Delete the Vue file**

```bash
git rm frontend/src/components/FeatureGuard/FeatureAttention.vue
```

- [ ] **Step 4.4: Verify build still passes**

```bash
pnpm --dir frontend type-check
```

Expected: pass.

- [ ] **Step 4.5: Commit**

```bash
git add frontend/src/components/FeatureGuard/index.ts
git commit -m "chore(frontend): drop orphan FeatureAttention.vue

React callers already use @/react/components/FeatureAttention.
FeatureBadge/FeatureModal stay — Vue callers remain.
"
```

---

## Task 5: Delete SQLReview RuleConfigComponents

**Goal:** The five `*Component.vue` files plus the `index.ts` that re-exports them have zero callers. Delete the whole set. `types.ts` and `utils.ts` in the same dir may also be orphaned — verify and delete if so.

**Files:**
- Delete: `frontend/src/components/SQLReview/components/RuleConfigComponents/BooleanComponent.vue`
- Delete: `frontend/src/components/SQLReview/components/RuleConfigComponents/NumberComponent.vue`
- Delete: `frontend/src/components/SQLReview/components/RuleConfigComponents/StringArrayComponent.vue`
- Delete: `frontend/src/components/SQLReview/components/RuleConfigComponents/StringComponent.vue`
- Delete: `frontend/src/components/SQLReview/components/RuleConfigComponents/TemplateComponent.vue`
- Delete: `frontend/src/components/SQLReview/components/RuleConfigComponents/index.ts`
- Conditionally delete: `types.ts` and `utils.ts` in same dir (verify in step 5.1)

- [ ] **Step 5.1: Verify the whole directory is orphaned**

```bash
echo "=== Any imports of RuleConfigComponents (subpath) ==="
rg -n --type-add 'tsx:*.tsx' --type-add 'vue:*.vue' -tts -tvue -ttsx 'RuleConfigComponents' frontend/src/ 2>/dev/null \
  | grep -v '^frontend/src/components/SQLReview/components/RuleConfigComponents/'

echo ""
echo "=== Imports of the dir's types.ts or utils.ts (by path) ==="
rg -n --type-add 'tsx:*.tsx' --type-add 'vue:*.vue' -tts -tvue -ttsx '(RuleConfigComponents/types|RuleConfigComponents/utils)' frontend/src/ 2>/dev/null
```

Expected: zero results for both. If anything matches outside the dir, only delete the `.vue` files + `index.ts` and leave `types.ts`/`utils.ts`.

- [ ] **Step 5.2: Delete the directory if fully orphaned, else partial delete**

If step 5.1 returned **zero external imports**:

```bash
git rm -r frontend/src/components/SQLReview/components/RuleConfigComponents/
```

Otherwise, delete only the Vue files + index.ts:

```bash
git rm \
  frontend/src/components/SQLReview/components/RuleConfigComponents/BooleanComponent.vue \
  frontend/src/components/SQLReview/components/RuleConfigComponents/NumberComponent.vue \
  frontend/src/components/SQLReview/components/RuleConfigComponents/StringArrayComponent.vue \
  frontend/src/components/SQLReview/components/RuleConfigComponents/StringComponent.vue \
  frontend/src/components/SQLReview/components/RuleConfigComponents/TemplateComponent.vue \
  frontend/src/components/SQLReview/components/RuleConfigComponents/index.ts
```

- [ ] **Step 5.3: Verify build still passes**

```bash
pnpm --dir frontend type-check
```

Expected: pass.

- [ ] **Step 5.4: Commit**

```bash
git add -A frontend/src/components/SQLReview/
git commit -m "chore(frontend): drop unused SQLReview RuleConfigComponents

Whole subtree had no callers. SQL review UI is fully React-side."
```

---

## Task 6: Delete AdvancedSearch Vue files

**Goal:** Delete the five `.vue` files and the `index.ts` that re-exports them. Keep `types.ts` and `useCommonSearchScopeOptions.ts` — `frontend/src/utils/accessGrant.ts` imports from `@/components/AdvancedSearch/types`.

**Files:**
- Delete: `frontend/src/components/AdvancedSearch/AdvancedSearch.vue`
- Delete: `frontend/src/components/AdvancedSearch/ScopeMenu.vue`
- Delete: `frontend/src/components/AdvancedSearch/ScopeTags.vue`
- Delete: `frontend/src/components/AdvancedSearch/TimeRange.vue`
- Delete: `frontend/src/components/AdvancedSearch/ValueMenu.vue`
- Delete: `frontend/src/components/AdvancedSearch/index.ts`
- Keep: `types.ts`, `useCommonSearchScopeOptions.ts`

- [ ] **Step 6.1: Verify the Vue files have no callers outside the directory itself**

```bash
echo "=== Imports of @/components/AdvancedSearch (any subpath) ==="
rg -n --type-add 'tsx:*.tsx' --type-add 'vue:*.vue' -tts -tvue -ttsx '@/components/AdvancedSearch' frontend/src/ 2>/dev/null
```

Expected output (this is the only allowed external import):
```
frontend/src/utils/accessGrant.ts: ... } from "@/components/AdvancedSearch/types";
```

If anything else matches (e.g., a default import of `@/components/AdvancedSearch`), stop and report.

- [ ] **Step 6.2: Delete the Vue files and index.ts**

```bash
git rm \
  frontend/src/components/AdvancedSearch/AdvancedSearch.vue \
  frontend/src/components/AdvancedSearch/ScopeMenu.vue \
  frontend/src/components/AdvancedSearch/ScopeTags.vue \
  frontend/src/components/AdvancedSearch/TimeRange.vue \
  frontend/src/components/AdvancedSearch/ValueMenu.vue \
  frontend/src/components/AdvancedSearch/index.ts
```

- [ ] **Step 6.3: Verify `types.ts` is still importable**

```bash
rg -n 'AdvancedSearch/types' frontend/src/utils/accessGrant.ts
pnpm --dir frontend type-check
```

Expected: the import line shows; type-check passes.

- [ ] **Step 6.4: Verify `useCommonSearchScopeOptions.ts` is still used**

```bash
rg -n 'useCommonSearchScopeOptions' frontend/src/ 2>/dev/null \
  | grep -v '^frontend/src/components/AdvancedSearch/useCommonSearchScopeOptions\.ts:'
```

If zero results: the file is now orphaned — flag for the next opportunistic cleanup PR but leave it in this PR (deleting Vue-helper TS files that *might* still be referenced via dynamic patterns is risky for a delete-driven PR).

- [ ] **Step 6.5: Commit**

```bash
git add -A frontend/src/components/AdvancedSearch/
git commit -m "chore(frontend): drop Vue AdvancedSearch components

All consumers now use @/react/components/AdvancedSearch.
types.ts kept (still imported by utils/accessGrant.ts).
"
```

---

## Task 7: Final validation

**Goal:** Confirm the full PR is clean: builds, lints, type-checks, all tests including the new guard, no regressions.

- [ ] **Step 7.1: Run the full frontend check chain**

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test
```

Expected: every command exits 0.

If `fix` modifies any files (e.g., auto-formats remaining files after the deletions changed imports), stage and amend those into the most recent relevant commit — or land a small follow-up `style:` commit. Do not skip this step.

- [ ] **Step 7.2: Build sanity check**

```bash
pnpm --dir frontend build
```

Expected: build succeeds. (Heavy step — only required if any of the earlier steps modified `.ts`/`.tsx` files in `frontend/src/components/v2/index.ts` or `FeatureGuard/index.ts`. If only `.vue` files were deleted, type-check is sufficient.)

- [ ] **Step 7.3: Verify file count delta**

```bash
echo "Vue files remaining outside frontend/src/react/:"
fd -e vue . frontend/src 2>/dev/null | grep -v '/react/' | wc -l
```

Expected: **133** (was 154, minus 21 deleted in this PR).

- [ ] **Step 7.4: Spot manual smoke**

The deleted files have no live callers, but as a final safety check, start the dev server and click through 3–4 high-traffic React pages that previously referenced near-neighbors of these files:

```bash
PG_URL=postgresql://bbdev@localhost/bbdev pnpm --dir frontend dev
```

Visit and verify no broken UI/console errors on:
- A project issue list (uses `AdvancedSearch.tsx` — confirm search filters still work)
- A workspace settings page with subscription gating (uses `FeatureAttention.tsx` — confirm gating renders)
- Instance detail page (used `InstanceSyncButton` — confirm sync button renders)
- SQL Editor home (uses `MaxRowCountSelect`, `SQLUploadButton` — confirm both controls work)

Report any console errors before proceeding to PR.

---

## Task 8: PR preparation

- [ ] **Step 8.1: Review commit log**

```bash
git log --oneline main..HEAD
```

Expected: 6 commits (one per task 1–6).

- [ ] **Step 8.2: Push & open PR**

```bash
git push -u origin <branch>
gh pr create --title "chore(frontend): drop 21 orphan Vue primitives + add React→Vue import guard" --body "$(cat <<'EOF'
## Summary
Phase A of the Vue→React migration ([status doc](docs/plans/2026-05-12-react-migration-status-and-plan.md), [design](docs/plans/2026-05-12-phase-a-legacy-primitives-design.md)).

- Deletes 21 `.vue` files in `frontend/src/components/` whose call sites already migrated to React or were never used.
- Adds a new vitest guard (`no-react-to-vue-imports.test.ts`) that fails CI if any `frontend/src/react/**/*.{ts,tsx}` file imports a `.vue` file, with a small explicit allowlist for mount-bridges (`SessionExpiredSurfaceMount`, `AgentWindowMount`) deferred to Phase B.

## Test plan
- [ ] `pnpm --dir frontend type-check` passes
- [ ] `pnpm --dir frontend test` passes (includes new guard)
- [ ] `pnpm --dir frontend check` passes
- [ ] `pnpm --dir frontend build` succeeds
- [ ] Manual smoke: AdvancedSearch, FeatureAttention, InstanceSyncButton, MaxRowCountSelect, SQLUploadButton call sites render without errors
EOF
)"
```

---

## Self-review notes

- **Spec coverage:** Task 1 implements the CI guard requirement. Tasks 2–6 cover all 21 files listed in the spec's "Easy-delete sweep" section. Task 7 covers the spec's "Validation" subsection. The spec's "Out of scope" list (MIXED files and VUE-ONLY files) is honored — no plan task touches `LearnMoreLink`, `FeatureBadge`, `FeatureModal`, `UserAvatar`, `MonacoEditor/*`, `Icon/*`, `v2/Button|Form|Select|Model`, `ReleaseRemindModal`, `OverlayStackManager`, etc.
- **EllipsisText.vue deviation:** The spec listed `EllipsisText.vue` in the easy-delete sweep (13 React callers). Re-verification during planning revealed those 13 callers already import the React `ellipsis-text.tsx`; the only remaining caller is `frontend/src/components/v2/Select/RemoteResourceSelector/utils.tsx`, which is a Vue-JSX file (not React — Vue 3 supports `.tsx` via `@vitejs/plugin-vue-jsx`). EllipsisText.vue is therefore VUE-ONLY and deferred to Phase B. The CI guard's scope to `frontend/src/react/**` correctly excludes the Vue-JSX caller.
- **v2/Container/Drawer** deviation: The spec listed `v2/Container/*` in the easy-delete sweep. The audit found `frontend/src/plugins/ai/components/HistoryPanel/HistoryPanel.vue` still imports it. Deferred to Phase B. Not included in this plan.
- **Final delete count:** 21 `.vue` files + 4 `index.ts` edits (delete or modify) + 1 new test file. Matches the spec's "~24 Vue files" target.
