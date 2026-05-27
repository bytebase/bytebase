# Naive-UI Removal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove `naive-ui` and its peer deps (`vdirs`, `vueuc`) from the frontend in a single PR. The app must continue to build, type-check, lint, test, and run after the change.

**Architecture:** Twelve sequential tasks. Each task leaves the build green so we can stop and verify mid-way. Order is chosen so that earlier tasks cannot break later ones: type decoupling → dead-code deletion → live-consumer migration → global wiring removal → dep removal → verification.

**Tech Stack:** Vue 3, Vite 8, pnpm 11, TypeScript 6, vue-tsc, ESLint, Biome.

**Spec:** `docs/superpowers/specs/2026-05-25-naive-ui-removal-design.md`

---

## File Structure

**Files modified:**
- `frontend/src/types/v2-shared.ts` — decouple `ResourceSelectOption` from naive-ui's `SelectOption`
- `frontend/src/types/sqlEditor/tree.ts` — decouple `SQLEditorTreeNode` from `TreeOption`
- `frontend/src/utils/v1/databaseResource.ts` — decouple `DatabaseTreeOption` from `TreeOption`/`TransferOption`
- `frontend/src/bbkit/index.ts` — remove deleted-wrapper re-exports
- `frontend/src/AuthContext.vue` — inline CSS spinner, drop `BBSpin` import
- `frontend/src/App.vue` — drop `NConfigProvider` wrapper and theme-config import
- `frontend/src/main.ts` — drop `NaiveUI` plugin import and registration
- `frontend/src/assets/css/tailwind.css` — remove naive-ui focus-ring overrides
- `frontend/vite.config.ts` — drop `naive-ui` chunk entry from `manualChunks`
- `frontend/package.json` — remove `naive-ui`, `vdirs`, `vueuc` (via `pnpm remove`)
- `frontend/pnpm-lock.yaml` — auto-updated by `pnpm remove`

**Files deleted:**
- `frontend/src/bbkit/BBAlert.vue` (0 consumers)
- `frontend/src/bbkit/BBAttention.vue` (0 consumers)
- `frontend/src/bbkit/BBButtonConfirm.vue` (0 consumers)
- `frontend/src/bbkit/BBTextField.vue` (0 consumers)
- `frontend/src/bbkit/BBSpin.vue` (after Task 5 migrates AuthContext.vue)
- `frontend/src/utils/naive-ui.ts` (`useAutoHeightDataTable`, 0 consumers)
- `frontend/src/plugins/naive-ui.ts` (global registration plugin)
- `frontend/naive-ui.config.ts` (theme + locale config)

**Files created:** none.

---

## Task 1: Decouple `ResourceSelectOption` from naive-ui

**Files:**
- Modify: `frontend/src/types/v2-shared.ts`

Three React consumers (`CustomApproval/utils.ts`, `sensitive-data/components-utils.ts`, `database-group/utils.ts`) use this type as a plain object shape. They never access naive-ui-specific `SelectOption` fields. Replace the `extends` with a standalone interface.

- [ ] **Step 1: Replace the file contents**

```typescript
// frontend/src/types/v2-shared.ts
export type ResourceSelectOption<T> = {
  resource?: T;
  value: string;
  label: string;
  // Allow arbitrary extra fields the Vue-side select components may set;
  // React consumers only read value/label/resource.
  [key: string]: unknown;
};

export type SelectSize = "tiny" | "small" | "medium" | "large";
```

- [ ] **Step 2: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS (vue-tsc and tsc both clean).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/types/v2-shared.ts
git commit -m "refactor(types): decouple ResourceSelectOption from naive-ui"
```

---

## Task 2: Decouple `SQLEditorTreeNode` from naive-ui `TreeOption`

**Files:**
- Modify: `frontend/src/types/sqlEditor/tree.ts`

`SQLEditorTreeNode` extends naive-ui's `TreeOption`. Consumers (`stores/sqlEditor/`, `react/components/sql-editor/ConnectionPane/`) read `disabled`, `children`, `key`, plus the locally-added `meta` field. Define the minimal shape inline.

- [ ] **Step 1: Replace the `TreeOption` import and `SQLEditorTreeNode` declaration**

Edit `frontend/src/types/sqlEditor/tree.ts`:

Remove line 1:
```typescript
import type { TreeOption } from "naive-ui";
```

Replace the `SQLEditorTreeNode` declaration (lines 50-56) with:

```typescript
export type SQLEditorTreeNode<
  T extends SQLEditorTreeNodeType = SQLEditorTreeNodeType,
> = {
  meta: SQLEditorTreeNodeMeta<T>;
  key: string;
  label?: string;
  disabled?: boolean;
  isLeaf?: boolean;
  children?: SQLEditorTreeNode[];
};
```

- [ ] **Step 2: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS. If any consumer accesses a field not listed above (`prefix`, `suffix`, `checkboxDisabled`, etc.), the error will pinpoint the line — add the field to the standalone type with the correct optional signature.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/types/sqlEditor/tree.ts
git commit -m "refactor(sql-editor): decouple SQLEditorTreeNode from naive-ui"
```

---

## Task 3: Decouple `DatabaseTreeOption` from naive-ui

**Files:**
- Modify: `frontend/src/utils/v1/databaseResource.ts`

`DatabaseTreeOption` extends both `TreeOption` and `TransferOption`. Code in this file sets only `label`, `value`, `level`, `isLeaf`, and `children`. The only external consumer (`RequestQueryButton.tsx`) imports `parseStringToResource`, not the type.

- [ ] **Step 1: Remove the naive-ui import and rewrite the interface**

Edit `frontend/src/utils/v1/databaseResource.ts`:

Remove line 2:
```typescript
import type { TransferOption, TreeOption } from "naive-ui";
```

Replace the `DatabaseTreeOption` declaration (lines 26-33) with:

```typescript
export interface DatabaseTreeOption<L = DatabaseResourceType> {
  label: string;
  level: L;
  value: string;
  isLeaf?: boolean;
  disabled?: boolean;
  children?: DatabaseTreeOption[];
}
```

- [ ] **Step 2: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/utils/v1/databaseResource.ts
git commit -m "refactor(database): decouple DatabaseTreeOption from naive-ui"
```

---

## Task 4: Delete unused bbkit wrappers and update barrel

**Files:**
- Delete: `frontend/src/bbkit/BBAlert.vue`
- Delete: `frontend/src/bbkit/BBAttention.vue`
- Delete: `frontend/src/bbkit/BBButtonConfirm.vue`
- Delete: `frontend/src/bbkit/BBTextField.vue`
- Delete: `frontend/src/utils/naive-ui.ts`
- Modify: `frontend/src/bbkit/index.ts`

Each of these has zero consumers per the cascade audit. Delete the files and remove the re-exports.

- [ ] **Step 1: Verify zero consumers (sanity check before delete)**

Run from repo root:
```bash
cd /Users/steven/Projects/bytebase/bb/frontend && \
  grep -rn 'BBAlert\b\|BBAttention\b\|BBButtonConfirm\b\|BBTextField\b\|useAutoHeightDataTable\b' \
    src --include='*.ts' --include='*.tsx' --include='*.vue' \
    | grep -v 'src/bbkit/' | grep -v 'src/utils/naive-ui.ts'
```

Expected: no output (the only matches are the definitions themselves, which are excluded by the `grep -v` filters).

If any line is printed, **stop and investigate** — there is a consumer the audit missed. Add a migration step before proceeding.

- [ ] **Step 2: Delete the five files**

```bash
rm frontend/src/bbkit/BBAlert.vue
rm frontend/src/bbkit/BBAttention.vue
rm frontend/src/bbkit/BBButtonConfirm.vue
rm frontend/src/bbkit/BBTextField.vue
rm frontend/src/utils/naive-ui.ts
```

- [ ] **Step 3: Update the bbkit barrel**

Replace `frontend/src/bbkit/index.ts` with:

```typescript
import BBSpin from "./BBSpin.vue";

export * from "./types";

export { BBSpin };
```

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A frontend/src/bbkit frontend/src/utils/naive-ui.ts
git commit -m "chore(bbkit): delete unused naive-ui wrappers"
```

---

## Task 5: Inline CSS spinner in AuthContext.vue and delete BBSpin.vue

**Files:**
- Modify: `frontend/src/AuthContext.vue`
- Delete: `frontend/src/bbkit/BBSpin.vue`
- Modify: `frontend/src/bbkit/index.ts`

`AuthContext.vue` is the sole live consumer of `BBSpin`. The current usage is `<BBSpin />` (no size override) inside a centered container during auth bootstrap. Replace with a self-contained CSS spinner — a 20×20 div with a 2-px border, three sides in `currentColor`, one side transparent, rotating 360° infinitely.

The current BBSpin renders `NSpin` at small size with stroke-width 25 — a ~20-px circular spinner using the theme's accent color. The inline replacement matches that visual.

- [ ] **Step 1: Update AuthContext.vue**

Edit `frontend/src/AuthContext.vue`:

Replace the template (lines 1-17) with:

```vue
<template>
  <slot v-if="ready"></slot>
  <div v-else class="flex items-center justify-center h-screen">
    <span class="bb-spinner" aria-label="Loading" role="status"></span>
  </div>
  <template v-if="!isAuthRoute && authStore.isLoggedIn">
    <!-- Session-expired surface lives in the React app
         (src/react/app/SessionExpiredSurfaceGate.tsx).
         InactiveRemindModal stays mounted here because it reads Vue router
         state via the bridged provide/inject context, which the sibling
         React app doesn't carry. -->
    <ReactPageMount
      v-if="!authStore.unauthenticatedOccurred"
      page="InactiveRemindModal"
    />
  </template>
</template>
```

Remove line 22:
```typescript
import { BBSpin } from "@/bbkit";
```

Append a scoped `<style>` block to the end of the file (after the closing `</script>`):

```vue
<style scoped>
.bb-spinner {
  display: inline-block;
  width: 20px;
  height: 20px;
  border: 2px solid var(--color-accent, currentcolor);
  border-top-color: transparent;
  border-radius: 50%;
  animation: bb-spinner-rotate 0.8s linear infinite;
}

@keyframes bb-spinner-rotate {
  to {
    transform: rotate(360deg);
  }
}
</style>
```

- [ ] **Step 2: Delete BBSpin.vue**

```bash
rm frontend/src/bbkit/BBSpin.vue
```

- [ ] **Step 3: Update the bbkit barrel to remove the last BB* export**

Replace `frontend/src/bbkit/index.ts` with:

```typescript
export * from "./types";
```

- [ ] **Step 4: Verify nothing else imports BBSpin**

Run:
```bash
grep -rn 'BBSpin\b' frontend/src --include='*.ts' --include='*.tsx' --include='*.vue'
```

Expected: no output.

- [ ] **Step 5: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/AuthContext.vue frontend/src/bbkit/BBSpin.vue frontend/src/bbkit/index.ts
git commit -m "refactor(auth): inline CSS spinner, delete BBSpin"
```

---

## Task 6: Strip NConfigProvider from App.vue

**Files:**
- Modify: `frontend/src/App.vue`

`NConfigProvider` injects naive-ui's theme. Theme tokens are already CSS custom properties in `tailwind.css`, so removing the wrapper has no visual effect on app-level styling. After Task 5, no surviving file renders a naive-ui component, so the provider is a dead wrapper.

- [ ] **Step 1: Update the template**

Edit `frontend/src/App.vue`. Replace the template block (lines 1-11) with:

```vue
<template>
  <AuthContext>
    <router-view />
  </AuthContext>
</template>
```

- [ ] **Step 2: Remove the naive-ui import and theme-config import**

In the `<script setup>` block, remove:

```typescript
import { NConfigProvider } from "naive-ui";
```

And remove:

```typescript
import { dateLang, generalLang, themeOverrides } from "../naive-ui.config";
```

- [ ] **Step 3: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS. `dateLang`, `generalLang`, and `themeOverrides` are no longer referenced anywhere.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/App.vue
git commit -m "refactor(app): drop NConfigProvider wrapper from App.vue"
```

---

## Task 7: Remove naive-ui plugin wiring from main.ts and delete the plugin

**Files:**
- Modify: `frontend/src/main.ts`
- Delete: `frontend/src/plugins/naive-ui.ts`
- Delete: `frontend/naive-ui.config.ts`

After Task 6, nothing references the plugin or the config file.

- [ ] **Step 1: Update main.ts**

Edit `frontend/src/main.ts`:

Remove line 11:
```typescript
import NaiveUI from "./plugins/naive-ui";
```

Change line 64 from:
```typescript
app.use(router).use(highlight).use(i18n).use(NaiveUI);
```

To:
```typescript
app.use(router).use(highlight).use(i18n);
```

- [ ] **Step 2: Delete the plugin and config files**

```bash
rm frontend/src/plugins/naive-ui.ts
rm frontend/naive-ui.config.ts
```

- [ ] **Step 3: Verify no remaining references**

Run:
```bash
grep -rn 'plugins/naive-ui\|naive-ui\.config' frontend/src frontend/vite.config.ts frontend/tsconfig*.json 2>/dev/null
```

Expected: no output.

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/main.ts frontend/src/plugins/naive-ui.ts frontend/naive-ui.config.ts
git commit -m "chore(plugins): remove naive-ui plugin and theme config"
```

---

## Task 8: Remove naive-ui CSS overrides from tailwind.css

**Files:**
- Modify: `frontend/src/assets/css/tailwind.css`

The focus-ring overrides at lines 421-426 target `.n-base-selection-input`, `.n-base-selection-input-tag__input`, and `.n-input__input-el` — all naive-ui-specific selectors that are now dead.

- [ ] **Step 1: Remove the override block**

Edit `frontend/src/assets/css/tailwind.css`. Delete lines 421-426 (the comment `/* compatibility fixes for tailwindcss and naive-ui */` through the closing `}` of the `.n-input__input-el:focus` rule):

```css
/* compatibility fixes for tailwindcss and naive-ui */
.n-base-selection-input:focus,
.n-base-selection-input-tag__input:focus,
.n-input__input-el:focus {
  box-shadow: 0 0 0 0;
}
```

- [ ] **Step 2: Verify no other naive-ui CSS selectors remain**

Run:
```bash
grep -rn '\.n-[a-z]' frontend/src/assets frontend/src --include='*.css' --include='*.scss' --include='*.vue'
```

Expected: no output (any remaining `.n-*` selectors would also be dead).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/assets/css/tailwind.css
git commit -m "style(css): drop naive-ui focus-ring overrides"
```

---

## Task 9: Update vite.config.ts manualChunks

**Files:**
- Modify: `frontend/vite.config.ts`

The `ui-framework` chunk currently contains only naive-ui (per the audit). Remove the entry.

- [ ] **Step 1: Remove the naive-ui chunk entry**

Edit `frontend/vite.config.ts`. Delete lines 98-101 (the comment and the `if` block):

```typescript
          // UI framework
          if (id.includes("naive-ui")) {
            return "ui-framework";
          }
```

- [ ] **Step 2: Verify build config still parses**

Run: `pnpm --dir frontend type-check`
Expected: PASS (the tsconfig covers vite.config.ts).

- [ ] **Step 3: Commit**

```bash
git add frontend/vite.config.ts
git commit -m "build(vite): drop naive-ui manualChunks entry"
```

---

## Task 10: Drop naive-ui, vdirs, vueuc from package.json

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/pnpm-lock.yaml`

After Task 9, nothing imports `naive-ui`, `vdirs`, or `vueuc`. Remove them.

- [ ] **Step 1: Final source scan before dep removal**

Run:
```bash
grep -rn '\bnaive-ui\b\|from "vdirs"\|from "vueuc"' frontend/src frontend/vite.config.ts frontend/*.ts frontend/*.json 2>/dev/null
```

Expected: no output. If anything matches, **stop** — there is still a live reference. Resolve it before continuing (likely an earlier task missed a file).

- [ ] **Step 2: Run pnpm remove**

```bash
pnpm --dir frontend remove naive-ui vdirs vueuc
```

Expected: command succeeds; `package.json` no longer lists the three deps; `pnpm-lock.yaml` is updated.

- [ ] **Step 3: Verify package.json**

Run:
```bash
grep -E '"naive-ui"|"vdirs"|"vueuc"' frontend/package.json
```

Expected: no output.

- [ ] **Step 4: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS. If a residual import survives, vue-tsc or tsc will fail with `Cannot find module 'naive-ui'`. Fix the import and re-run.

- [ ] **Step 5: Commit**

```bash
git add frontend/package.json frontend/pnpm-lock.yaml
git commit -m "build(deps): drop naive-ui, vdirs, vueuc"
```

---

## Task 11: Full verification pass

**Files:** none modified.

Run every gate end-to-end. This task does not commit.

- [ ] **Step 1: Source grep — nothing references naive-ui**

```bash
grep -rn '\bnaive-ui\b' frontend/src frontend/vite.config.ts frontend/*.ts frontend/*.json 2>/dev/null
```

Expected: no output.

- [ ] **Step 2: Source grep — no vdirs/vueuc imports**

```bash
grep -rn 'from "vdirs"\|from "vueuc"\|from '\''vdirs'\''\|from '\''vueuc'\''' frontend/src 2>/dev/null
```

Expected: no output.

- [ ] **Step 3: Source grep — no auto-registered naive-ui templates**

```bash
grep -rn '<n-[a-z]' frontend/src --include='*.vue'
```

Expected: no output.

- [ ] **Step 4: Source grep — no .n-* CSS selectors**

```bash
grep -rn '\.n-[a-z]' frontend/src --include='*.css' --include='*.scss' --include='*.vue'
```

Expected: no output.

- [ ] **Step 5: Lint and format**

Run: `pnpm --dir frontend fix`
Expected: command exits 0. If files are modified, re-stage and `git commit -m "chore: apply lint fixes after naive-ui removal"`.

- [ ] **Step 6: CI-mode check**

Run: `pnpm --dir frontend check`
Expected: PASS (ESLint + Biome CI + react-i18n + react-layering + i18n sort all green).

- [ ] **Step 7: Type-check**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 8: Unit tests**

Run: `pnpm --dir frontend test`
Expected: PASS.

---

## Task 12: Manual smoke test of auth flow

**Files:** none modified.

The only behavioral change is the spinner in `AuthContext.vue`. Manually verify the auth bootstrap renders correctly.

- [ ] **Step 1: Start the dev server**

In one terminal:
```bash
pnpm --dir frontend dev
```

In another terminal, ensure the backend is running on port 8080:
```bash
PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug
```

- [ ] **Step 2: Verify the spinner during signin**

Open the dev URL (typically `http://localhost:3000`). On a cold load (clear cookies / hard-reload), the spinner should appear briefly during auth bootstrap, centered on the page, rotating smoothly.

If the spinner does not appear or does not animate, inspect the DOM for the `.bb-spinner` element and confirm the `<style scoped>` block in `AuthContext.vue` is present.

- [ ] **Step 3: Verify the dashboard renders**

After login, confirm the dashboard loads with no visual regressions. Sidebar, header, footer all render correctly (these are still Vue, but had no naive-ui dependency — should be visually identical).

- [ ] **Step 4: Verify a few key pages**

Click into:
- Projects list → a project → Issue list → an issue
- A database detail page (overview tab)
- SQL Editor

These are all React pages; nothing should have changed visually. The intent of the smoke test is to confirm the global wiring removal did not break React mount points.

- [ ] **Step 5: Check the browser console**

Open DevTools → Console. Expected: no errors related to `naive-ui`, `NConfigProvider`, `vdirs`, `vueuc`, or undefined Vue components.

- [ ] **Step 6: If smoke test passes, the PR is ready**

No commit needed for this task — verification only.

---

## Out-of-band follow-ups (not in this PR)

- The eight surviving Vue files (`App.vue`, `AuthContext.vue`, `BodyLayout.vue`, `DashboardLayout.vue`, `SplashLayout.vue`, and four React-mount bridges) remain. Phase B of the broader Vue → React migration (router swap, shell port, Pinia → Zustand) covers their eventual removal.
- The `bbkit/` directory will be left with `BBUtil.ts` (a pure `hashCode()` utility) and `index.ts` re-exporting `./types`. Consider moving `hashCode` to a shared utility location in a follow-up so `bbkit/` can be deleted entirely.
