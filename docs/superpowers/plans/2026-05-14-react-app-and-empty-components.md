# Introduce React App + Empty `frontend/src/components/` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up a sibling React app in `frontend/index.html` to host `AgentWindow` and `SessionExpiredSurface`, then relocate every remaining file under `frontend/src/components/` to its natural home and delete the directory.

**Architecture:** New `<div id="react-app">` in `index.html` mounted from `src/main.ts` after the Vue app. Three new files under `src/react/app/` (`ReactApp.tsx`, `SessionExpiredSurfaceGate.tsx`, `mount.tsx`). Both initial residents already `createPortal` to `getLayerRoot(...)`, so the React app has no visible DOM of its own. Validation is batched at the end per repo convention.

**Tech Stack:** Vue 3 (host), React 18, vue-i18n + react-i18next, Vite, Pinia, Base UI, Tailwind v4.

**Companion spec:** [`docs/superpowers/specs/2026-05-14-react-app-and-empty-components-design.md`](../specs/2026-05-14-react-app-and-empty-components-design.md)

**Validation cadence:** Per repo memory — agents skip per-file `check` / `type-check` / `test`. Validation runs once at the end of the PR (Task 18). Each commit just needs the diff to be coherent.

---

## File Map

**Create:**
- `frontend/src/react/app/ReactApp.tsx`
- `frontend/src/react/app/SessionExpiredSurfaceGate.tsx`
- `frontend/src/react/app/SessionExpiredSurfaceGate.test.tsx`
- `frontend/src/react/app/mount.tsx`
- `frontend/src/utils/v1/member.ts`
- `frontend/src/types/v1/member.ts`
- `frontend/src/utils/v1/databaseResource.ts`
- `frontend/src/utils/v1/issue/task.ts`
- `frontend/src/bbkit/OverlayStackManager.vue` (moved from `src/components/misc/`)

**Modify:**
- `frontend/index.html` — add `<div id="react-app">`
- `frontend/src/main.ts` — call `mountReactApp("#react-app")` after the Vue app mounts
- `frontend/src/App.vue` — update import path for `OverlayStackManager`
- `frontend/src/AuthContext.vue` — remove `<SessionExpiredSurfaceMount>` element + import
- `frontend/src/layouts/BodyLayout.vue` — remove `<AgentWindowMount />` element + import
- `frontend/src/layouts/layout-bridge.test.ts` — drop `vi.mock("@/components/AgentWindowMount.vue", ...)`
- `frontend/src/shell-bridge.test.ts` — update `vi.mock` path for `OverlayStackManager`
- `frontend/src/bbkit/BBModal.vue` — update import path for `useOverlayStack`
- `frontend/src/utils/v1/instance.ts` — append `defaultPortForEngine` + `EngineIconPath`
- `frontend/src/utils/v1/issue/rollout.ts` — update import path for `TASK_STATUS_FILTERS`
- `frontend/src/react/components/EngineIcon.tsx` — update import path
- `frontend/src/react/components/sql-editor/ResultPanel/BatchQuerySelect.tsx` — update import path
- `frontend/src/react/components/sql-editor/RequestQueryButton.tsx` — update import path
- `frontend/src/react/pages/settings/MembersPage.tsx` — update three import paths
- 8 `*.test.tsx` files under `frontend/src/react/components/sql-editor/` and `frontend/src/react/pages/project/export-center/` — update `vi.mock` paths for `InstanceForm/constants`

**Delete:**
- `frontend/src/components/AdvancedSearch/types.ts` (orphan)
- `frontend/src/components/AgentWindowMount.vue`
- `frontend/src/components/SessionExpiredSurfaceMount.vue`
- `frontend/src/components/misc/OverlayStackManager.vue` (after move)
- `frontend/src/components/InstanceForm/constants.ts` (after merge)
- `frontend/src/components/Member/utils.ts` (after merge)
- `frontend/src/components/Member/projectRoleBindings.ts` (after merge)
- `frontend/src/components/Member/types.ts` (after move)
- `frontend/src/components/RoleGrantPanel/DatabaseResourceForm/common.ts` (after move)
- `frontend/src/components/RolloutV1/constants/task.ts` (after move)
- `frontend/src/components/` (entire directory at the end)

---

## Task 1: Add the React app mount point to `index.html`

**Files:**
- Modify: `frontend/index.html`

- [ ] **Step 1: Add `<div id="react-app">` after `<div id="app">`**

Edit `frontend/index.html`. Find:

```html
    <div id="app"></div>
    <!-- React Toaster root: hosts the persistent toast renderer.
         Visually empty; toasts portal into getLayerRoot("overlay"). -->
    <div id="bb-toaster-root"></div>
```

Replace with:

```html
    <div id="app"></div>
    <!-- React App root: hosts long-lived React surfaces that portal globally
         (AgentWindow, SessionExpiredSurface). Mounted by mountReactApp() in
         src/main.ts. Visually empty; children portal into getLayerRoot(...). -->
    <div id="react-app"></div>
    <!-- React Toaster root: hosts the persistent toast renderer.
         Visually empty; toasts portal into getLayerRoot("overlay"). -->
    <div id="bb-toaster-root"></div>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/index.html
git commit -m "feat(frontend): add #react-app mount point for sibling React app"
```

---

## Task 2: Create `SessionExpiredSurfaceGate.tsx` with a test

**Files:**
- Create: `frontend/src/react/app/SessionExpiredSurfaceGate.tsx`
- Create: `frontend/src/react/app/SessionExpiredSurfaceGate.test.tsx`

The gate replaces the `v-if="authStore.unauthenticatedOccurred"` guard that lives in `AuthContext.vue` today, and supplies the `currentPath` prop that `SessionExpiredSurfaceMount.vue` derived from Vue's `useRoute().fullPath`.

- [ ] **Step 1: Write the failing test**

Create `frontend/src/react/app/SessionExpiredSurfaceGate.test.tsx`:

```tsx
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { ref } from "vue";

// useVueState wraps Vue reactive sources for React. Mock it so the test
// drives the gate's branching purely from refs we control.
const unauthenticatedOccurredRef = ref(false);
const fullPathRef = ref("/");

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: <T,>(getter: () => T) => {
    // Subscribe via a tiny effect-free pull each render. The gate only re-runs
    // when React re-renders, which is enough for these assertions.
    return getter();
  },
}));

vi.mock("@/store", () => ({
  useAuthStore: () => ({
    get unauthenticatedOccurred() {
      return unauthenticatedOccurredRef.value;
    },
  }),
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: {
      get value() {
        return { fullPath: fullPathRef.value };
      },
    },
  },
}));

vi.mock("@/react/components/auth/SessionExpiredSurface", () => ({
  SessionExpiredSurface: ({ currentPath }: { currentPath: string }) => (
    <div data-testid="surface" data-path={currentPath} />
  ),
}));

import { SessionExpiredSurfaceGate } from "./SessionExpiredSurfaceGate";

describe("SessionExpiredSurfaceGate", () => {
  let container: HTMLDivElement;
  let root: Root;

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    unauthenticatedOccurredRef.value = false;
    fullPathRef.value = "/";
  });

  test("renders nothing when unauthenticatedOccurred is false", () => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    act(() => root.render(<SessionExpiredSurfaceGate />));
    expect(container.querySelector("[data-testid='surface']")).toBeNull();
  });

  test("renders SessionExpiredSurface with current path when true", () => {
    unauthenticatedOccurredRef.value = true;
    fullPathRef.value = "/projects/sample/plans/123";
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    act(() => root.render(<SessionExpiredSurfaceGate />));
    const surface = container.querySelector("[data-testid='surface']");
    expect(surface).not.toBeNull();
    expect(surface?.getAttribute("data-path")).toBe("/projects/sample/plans/123");
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run from the repo root:

```bash
pnpm --dir frontend test -- src/react/app/SessionExpiredSurfaceGate.test.tsx
```

Expected: FAIL — `Cannot find module './SessionExpiredSurfaceGate'`.

- [ ] **Step 3: Create the gate component**

Create `frontend/src/react/app/SessionExpiredSurfaceGate.tsx`:

```tsx
import { SessionExpiredSurface } from "@/react/components/auth/SessionExpiredSurface";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { useAuthStore } from "@/store";

export function SessionExpiredSurfaceGate() {
  const unauthenticatedOccurred = useVueState(
    () => useAuthStore().unauthenticatedOccurred
  );
  const currentPath = useVueState(() => router.currentRoute.value.fullPath);
  if (!unauthenticatedOccurred) return null;
  return <SessionExpiredSurface currentPath={currentPath} />;
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:

```bash
pnpm --dir frontend test -- src/react/app/SessionExpiredSurfaceGate.test.tsx
```

Expected: PASS — both test cases green.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/app/SessionExpiredSurfaceGate.tsx \
        frontend/src/react/app/SessionExpiredSurfaceGate.test.tsx
git commit -m "feat(react): add SessionExpiredSurfaceGate for the React app"
```

---

## Task 3: Create `ReactApp.tsx`

**Files:**
- Create: `frontend/src/react/app/ReactApp.tsx`

`ReactApp` is the top-level tree. It does not own an `I18nextProvider` itself — `mount.tsx` (Task 4) wraps it, so tests of `ReactApp` can render it without setting up i18n.

- [ ] **Step 1: Create the component**

Create `frontend/src/react/app/ReactApp.tsx`:

```tsx
import { AgentWindow } from "@/react/plugins/agent/components/AgentWindow";
import { SessionExpiredSurfaceGate } from "./SessionExpiredSurfaceGate";

export function ReactApp() {
  return (
    <>
      <AgentWindow />
      <SessionExpiredSurfaceGate />
    </>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/react/app/ReactApp.tsx
git commit -m "feat(react): add ReactApp top-level tree (AgentWindow + session-expired gate)"
```

---

## Task 4: Create `mount.tsx`

**Files:**
- Create: `frontend/src/react/app/mount.tsx`

This file is `.tsx` (not `.ts`) because it renders JSX. It owns the `<I18nextProvider>` wrapper and the one-time Vue→React locale sync that replaces the per-shim `watch(locale, ...)` blocks.

- [ ] **Step 1: Create the mount entrypoint**

Create `frontend/src/react/app/mount.tsx`:

```tsx
import { StrictMode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { I18nextProvider } from "react-i18next";
import { watch } from "vue";
import { locale } from "@/plugins/i18n";
import i18n, { i18nReady } from "@/react/i18n";
import { ReactApp } from "./ReactApp";

export async function mountReactApp(selector: string): Promise<Root> {
  const container = document.querySelector(selector);
  if (!container) {
    throw new Error(`mountReactApp: missing container ${selector}`);
  }
  await i18nReady;
  // Sync initial locale before first paint.
  if (i18n.language !== locale.value) {
    await i18n.changeLanguage(locale.value);
  }
  const root = createRoot(container);
  root.render(
    <StrictMode>
      <I18nextProvider i18n={i18n}>
        <ReactApp />
      </I18nextProvider>
    </StrictMode>
  );
  // One-time Vue→React locale sync. Replaces the per-shim watch(locale, ...).
  watch(locale, async (next) => {
    if (i18n.language !== next) {
      await i18n.changeLanguage(next);
    }
  });
  return root;
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/react/app/mount.tsx
git commit -m "feat(react): add mountReactApp entrypoint"
```

---

## Task 5: Wire `main.ts` to mount the React app

**Files:**
- Modify: `frontend/src/main.ts`

Boot order: Vue app mounts first, React app is fire-and-forget after. Use dynamic `import()` so the React-app chunk stays off the Vue boot critical path.

- [ ] **Step 1: Read the existing `main.ts` to locate the Vue mount call**

Run:

```bash
grep -n "app.mount\|createApp" frontend/src/main.ts
```

Expected: finds the `app.mount("#app")` call near the end of the bootstrap IIFE.

- [ ] **Step 2: Add the React app mount after the Vue mount**

Edit `frontend/src/main.ts`. Find the `app.mount("#app");` line and add a new statement immediately after it:

```ts
  app.mount("#app");

  // Mount the sibling React app for surfaces that portal globally
  // (AgentWindow, SessionExpiredSurface). Fire-and-forget so any boot work
  // that follows is not blocked on i18nReady.
  void (async () => {
    const { mountReactApp } = await import("./react/app/mount");
    await mountReactApp("#react-app");
  })();
```

If `app.mount(...)` is the last statement before the IIFE closes, the new block is the last statement; either way the order is "Vue first, then React".

- [ ] **Step 3: Commit**

```bash
git add frontend/src/main.ts
git commit -m "feat(frontend): boot React app after the Vue app in main.ts"
```

---

## Task 6: Remove `<AgentWindowMount />` from `BodyLayout.vue`

**Files:**
- Modify: `frontend/src/layouts/BodyLayout.vue`
- Modify: `frontend/src/layouts/layout-bridge.test.ts`

- [ ] **Step 1: Remove the element and its import from `BodyLayout.vue`**

Edit `frontend/src/layouts/BodyLayout.vue`. Delete line 43 (the `<AgentWindowMount />` element) and the corresponding import:

```vue
import AgentWindowMount from "@/components/AgentWindowMount.vue";
```

The agent keyboard shortcut handler (`agentShortcutHandler` and its `onMounted`/`onUnmounted` registrations) stays — it just toggles the agent store. The agent is now rendered by the React app rather than by this layout's child element.

- [ ] **Step 2: Drop the `vi.mock` for `AgentWindowMount` in `layout-bridge.test.ts`**

Edit `frontend/src/layouts/layout-bridge.test.ts`. Delete this block (currently at lines 118–128):

```ts
vi.mock("@/components/AgentWindowMount.vue", async () => {
  const { defineComponent, h } = await import("vue");
  return {
    default: defineComponent({
      name: "MockAgentWindowMount",
      setup() {
        return () => h("div", { "data-testid": "agent-window" });
      },
    }),
  };
});
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/layouts/BodyLayout.vue frontend/src/layouts/layout-bridge.test.ts
git commit -m "refactor(layouts): drop AgentWindowMount from BodyLayout (React app hosts it)"
```

---

## Task 7: Remove `<SessionExpiredSurfaceMount />` from `AuthContext.vue`

**Files:**
- Modify: `frontend/src/AuthContext.vue`

The previous template was:

```vue
<template v-if="!isAuthRoute && authStore.isLoggedIn">
  <SessionExpiredSurfaceMount v-if="authStore.unauthenticatedOccurred" />
  <ReactPageMount v-else page="InactiveRemindModal" />
</template>
```

The session-expired branch now lives in `SessionExpiredSurfaceGate` inside the React app. `InactiveRemindModal` stays here because it still mounts via `ReactPageMount` and depends on Vue-side route state.

- [ ] **Step 1: Update the template**

Edit `frontend/src/AuthContext.vue`. Replace lines 6–10 with:

```vue
<template v-if="!isAuthRoute && authStore.isLoggedIn">
  <!-- Session-expired surface is rendered by the React app
       (see src/react/app/SessionExpiredSurfaceGate.tsx). -->
  <ReactPageMount
    v-if="!authStore.unauthenticatedOccurred"
    page="InactiveRemindModal"
  />
</template>
```

- [ ] **Step 2: Remove the import**

In the same file, delete:

```vue
import SessionExpiredSurfaceMount from "@/components/SessionExpiredSurfaceMount.vue";
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/AuthContext.vue
git commit -m "refactor(auth): drop SessionExpiredSurfaceMount from AuthContext (React app hosts it)"
```

---

## Task 8: Delete the two `*Mount.vue` shim files and their shim-only tests

**Files:**
- Delete: `frontend/src/components/AgentWindowMount.vue`
- Delete: `frontend/src/components/SessionExpiredSurfaceMount.vue`
- Modify: `frontend/src/react/components/auth/SessionExpiredSurface.test.tsx` — delete shim-only test cases
- Modify (optional): `frontend/src/react/mount.ts` — prune unused glob entries

The first `grep` in Step 1 will still match `SessionExpiredSurface.test.tsx` until that file's shim-specific tests are removed. Handle that here, then re-run the grep.

- [ ] **Step 1: Snapshot remaining importers**

Run:

```bash
grep -rnE "AgentWindowMount|SessionExpiredSurfaceMount" frontend/src 2>/dev/null
```

Expected matches (after Tasks 6 + 7):
- `frontend/src/react/components/auth/SessionExpiredSurface.test.tsx` — five test cases that mount `SessionExpiredSurfaceMount` directly to verify the shim's lifecycle. These tests no longer apply (the shim is going away) and must be deleted.

If anything else still matches, halt and triage — the shim is still load-bearing.

- [ ] **Step 2: Remove the shim-only tests from `SessionExpiredSurface.test.tsx`**

Tests 1–4 in this file (`mounts into the critical root`, `moves focus into the critical dialog`, `does not let Escape dismiss lower-layer dialogs`, `keeps the agent layer inert while the critical surface is open`) render `<SessionExpiredSurface currentPath="..." />` directly and exercise the React component itself — keep these.

Tests 5–9 (`syncs React i18n before the initial mount`, `reconciles the latest route after async mount resolves`, `unmounts late-mounted roots when the Vue bridge is already gone`, `keeps the newest route when async syncs finish out of order`, `keeps the newest locale when async locale syncs finish out of order`) all call `mount(SessionExpiredSurfaceMount)` and validate the shim's async mount + locale/route reconciliation. Delete them. The new architecture (single long-lived React root + `useVueState` subscription) has no shim to reconcile against, so the behaviors they test cannot exist.

Concretely, edit `frontend/src/react/components/auth/SessionExpiredSurface.test.tsx`:

1. Delete line 81: `import SessionExpiredSurfaceMount from "@/components/SessionExpiredSurfaceMount.vue";`
2. Delete the entire `test(...)` blocks for the five shim tests (currently lines 230–416 inclusive — the `test("syncs React i18n before the initial mount", ...)` through the closing brace of `test("keeps the newest locale when async locale syncs finish out of order", ...)`).
3. Delete the `vi.mock` blocks that are now unused by the four remaining tests:
   - The `vi.hoisted` `mountMocks` block (lines 23–46), plus the `vi.mock("@/react/i18n", ...)` and `vi.mock("@/react/mount", ...)` that reference it.
   - `vi.mock("vue-i18n", ...)` (lines 53–62) and `vi.mock("vue-router", ...)` (lines 64–79).
4. Delete the `import { flushPromises, mount } from "@vue/test-utils";` and `import { nextTick } from "vue";` imports at the top — only the shim tests used them.
5. Remove `mountMocks.changeLanguage.mockClear()`, `mountMocks.mountReactPage.mockClear()`, `mountMocks.updateReactPage.mockClear()`, the `mountMocks.locale!.value = "zh-CN"` reset, the `mountMocks.reactI18nLanguage.value = "en-US"` reset, and the `routePath` reset from the `afterEach` block (lines 90–102). What remains in `afterEach` should be:

```ts
afterEach(() => {
  document.body.innerHTML = "";
});
```

After these edits, the file's surviving content is: imports, the `vi.mock("@/react/pages/auth/SigninPage", ...)` mock, the `vi.mock("@/store", ...)` mock, the `vi.mock("react-i18next", ...)` mock, the import of `Dialog` / `DialogContent` / `SessionExpiredSurface`, the `IS_REACT_ACT_ENVIRONMENT` flag, and the four React-component tests (1–4).

- [ ] **Step 3: Re-verify no remaining importers of the shims**

```bash
grep -rnE "AgentWindowMount|SessionExpiredSurfaceMount" frontend/src 2>/dev/null
```

Expected: no matches.

- [ ] **Step 4: Delete the shim files**

```bash
git rm frontend/src/components/AgentWindowMount.vue \
       frontend/src/components/SessionExpiredSurfaceMount.vue
```

- [ ] **Step 5 (optional): Prune unused glob entries in `mount.ts`**

After the shims are gone, `mountReactPage` is no longer used to load `AgentWindow` or `SessionExpiredSurface`. They are now imported statically by `ReactApp.tsx` and `SessionExpiredSurfaceGate.tsx`.

Verify:

```bash
grep -nE "AgentWindow|SessionExpiredSurface" frontend/src/react/mount.ts
```

If the only matches are inside `import.meta.glob([...])` arrays (no other code references `"AgentWindow"` or `"SessionExpiredSurface"` as `mountReactPage` page names), edit `frontend/src/react/mount.ts`:

- Delete the entire `const pluginComponentLoaders = import.meta.glob("./plugins/agent/components/AgentWindow.tsx");` declaration.
- Remove `"./components/auth/SessionExpiredSurface.tsx"` from the `authComponentLoaders` array (but keep `InactiveRemindModal.tsx` — `AuthContext.vue` still mounts it via `ReactPageMount`).
- Remove `...pluginComponentLoaders` from the `pageLoaders` spread.
- Remove `"./plugins/agent/components"` from the `pageDirs` array.

Skip this step if a grep elsewhere shows a remaining `mountReactPage(..., "AgentWindow"|"SessionExpiredSurface", ...)` call site — leave the glob entries in that case.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/components/auth/SessionExpiredSurface.test.tsx \
        frontend/src/react/mount.ts
git commit -m "chore(frontend): delete *Mount.vue shims and their shim-only tests"
```

---

## Task 9: Delete the orphan `AdvancedSearch/types.ts`

**Files:**
- Delete: `frontend/src/components/AdvancedSearch/types.ts`

- [ ] **Step 1: Re-confirm zero importers**

Run:

```bash
grep -rnE "@/components/AdvancedSearch" frontend/src 2>/dev/null
```

Expected: no matches.

- [ ] **Step 2: Delete the file**

```bash
git rm frontend/src/components/AdvancedSearch/types.ts
```

- [ ] **Step 3: Commit**

```bash
git commit -m "chore(frontend): drop orphan AdvancedSearch/types.ts"
```

---

## Task 10: Move `OverlayStackManager.vue` to `src/bbkit/`

**Files:**
- Create: `frontend/src/bbkit/OverlayStackManager.vue`
- Delete: `frontend/src/components/misc/OverlayStackManager.vue`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/bbkit/BBModal.vue`
- Modify: `frontend/src/shell-bridge.test.ts`

- [ ] **Step 1: Move the file (preserves git history)**

```bash
git mv frontend/src/components/misc/OverlayStackManager.vue \
       frontend/src/bbkit/OverlayStackManager.vue
```

- [ ] **Step 2: Update `App.vue` import**

Edit `frontend/src/App.vue` line 32. Replace:

```ts
import OverlayStackManager from "./components/misc/OverlayStackManager.vue";
```

with:

```ts
import OverlayStackManager from "./bbkit/OverlayStackManager.vue";
```

- [ ] **Step 3: Update `BBModal.vue` import**

Edit `frontend/src/bbkit/BBModal.vue` line 55. Replace:

```ts
import { useOverlayStack } from "@/components/misc/OverlayStackManager.vue";
```

with:

```ts
import { useOverlayStack } from "@/bbkit/OverlayStackManager.vue";
```

- [ ] **Step 4: Update `shell-bridge.test.ts` mock path**

Edit `frontend/src/shell-bridge.test.ts` line 78. Replace:

```ts
vi.mock("./components/misc/OverlayStackManager.vue", async () => {
```

with:

```ts
vi.mock("./bbkit/OverlayStackManager.vue", async () => {
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/bbkit/OverlayStackManager.vue \
        frontend/src/App.vue \
        frontend/src/bbkit/BBModal.vue \
        frontend/src/shell-bridge.test.ts
git commit -m "refactor(frontend): move OverlayStackManager.vue from components/misc to bbkit"
```

---

## Task 11: Merge `InstanceForm/constants.ts` into `src/utils/v1/instance.ts`

**Files:**
- Modify: `frontend/src/utils/v1/instance.ts`
- Delete: `frontend/src/components/InstanceForm/constants.ts`
- Modify: `frontend/src/react/components/EngineIcon.tsx`
- Modify: `frontend/src/react/components/sql-editor/ResultPanel/BatchQuerySelect.tsx`
- Modify (8 test files): update `vi.mock` paths

- [ ] **Step 1: Read the full content of `InstanceForm/constants.ts`**

Run:

```bash
cat frontend/src/components/InstanceForm/constants.ts
```

Confirm the file exports `defaultPortForEngine` (function) and `EngineIconPath` (const). Note its imports — they must merge cleanly with `instance.ts`'s existing imports (`computed`, `Engine`, `supportedEngineV1List`).

- [ ] **Step 2: Append both exports to `src/utils/v1/instance.ts`**

Edit `frontend/src/utils/v1/instance.ts`. Append the entire body of `InstanceForm/constants.ts` to the end of the file, EXCEPT for its import block (its imports — `computed` from "vue", `Engine` from proto-es, `supportedEngineV1List` from `@/utils` — are already present in `instance.ts`; verify and only add any missing import).

- [ ] **Step 3: Delete the source file**

```bash
git rm frontend/src/components/InstanceForm/constants.ts
```

- [ ] **Step 4: Update the two real importers**

Edit `frontend/src/react/components/EngineIcon.tsx` line 2. Replace:

```tsx
import { EngineIconPath } from "@/components/InstanceForm/constants";
```

with:

```tsx
import { EngineIconPath } from "@/utils/v1/instance";
```

Edit `frontend/src/react/components/sql-editor/ResultPanel/BatchQuerySelect.tsx` line 7. Replace:

```tsx
import { EngineIconPath } from "@/components/InstanceForm/constants";
```

with:

```tsx
import { EngineIconPath } from "@/utils/v1/instance";
```

- [ ] **Step 5: Update the 8 test mock paths**

Each of the following files contains `vi.mock("@/components/InstanceForm/constants", ...)`. Replace that string in each with `vi.mock("@/utils/v1/instance", ...)`. If a file mocks more than `EngineIconPath` (e.g. unrelated `instance.ts` exports), only the mocked exports need to be defined — but check that the `vi.mock` factory still returns every export the test reads. If `@/utils/v1/instance` already had a real export used elsewhere in the test, hoist it into the mock as well.

Files to update:
- `frontend/src/react/components/sql-editor/ReadonlyModeNotSupported.test.tsx:48`
- `frontend/src/react/components/sql-editor/DatabaseChooser.test.tsx:51`
- `frontend/src/react/components/sql-editor/SheetConnectionIcon.test.tsx:25`
- `frontend/src/react/components/sql-editor/ConnectionPane/TreeNode/DatabaseNode.test.tsx:28`
- `frontend/src/react/components/sql-editor/ConnectionPane/ConnectionPane.test.tsx:185`
- `frontend/src/react/components/sql-editor/ConnectionPane/TreeNode/InstanceNode.test.tsx:19`
- `frontend/src/react/components/sql-editor/SchemaPane/SchemaPane.test.tsx:197`
- `frontend/src/react/pages/project/export-center/DataExportPrepSheet.test.tsx:139`

For each file, run `grep -n "vi.mock(\"@/components/InstanceForm/constants\"" <file>` to locate the exact line, then change only the path string. The mock factory body remains unchanged.

- [ ] **Step 6: Verify no remaining importers**

```bash
grep -rnE "@/components/InstanceForm" frontend/src 2>/dev/null
```

Expected: no matches.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/utils/v1/instance.ts \
        frontend/src/react/components/EngineIcon.tsx \
        frontend/src/react/components/sql-editor/ResultPanel/BatchQuerySelect.tsx \
        frontend/src/react/components/sql-editor/ReadonlyModeNotSupported.test.tsx \
        frontend/src/react/components/sql-editor/DatabaseChooser.test.tsx \
        frontend/src/react/components/sql-editor/SheetConnectionIcon.test.tsx \
        frontend/src/react/components/sql-editor/ConnectionPane/TreeNode/DatabaseNode.test.tsx \
        frontend/src/react/components/sql-editor/ConnectionPane/ConnectionPane.test.tsx \
        frontend/src/react/components/sql-editor/ConnectionPane/TreeNode/InstanceNode.test.tsx \
        frontend/src/react/components/sql-editor/SchemaPane/SchemaPane.test.tsx \
        frontend/src/react/pages/project/export-center/DataExportPrepSheet.test.tsx
git commit -m "refactor(frontend): merge InstanceForm/constants into utils/v1/instance"
```

---

## Task 12: Move Member binding types to `src/types/v1/member.ts`

**Files:**
- Create: `frontend/src/types/v1/member.ts`
- Delete: `frontend/src/components/Member/types.ts`
- Modify: `frontend/src/react/pages/settings/MembersPage.tsx`

- [ ] **Step 1: Create the new types file**

Create `frontend/src/types/v1/member.ts` with the full body of the source file (no edits to the type contents):

```ts
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { type User } from "@/types/proto-es/v1/user_service_pb";

export interface MemberRole {
  workspaceLevelRoles: Set<string>;
  projectRoleBindings: Binding[];
}

export interface GroupBinding extends Group {
  deleted?: boolean;
}

export interface MemberBinding extends MemberRole {
  title: string;
  // binidng is the fullname for binding member,
  // like user:{email} or group:{email}
  binding: string;
  type: "users" | "groups";
  user?: User;
  group?: GroupBinding;
  // True when the email is in the IAM policy but has no principal (user hasn't
  // signed up yet). Only set when the current user has permission to list/get
  // users — otherwise we can't tell and this stays undefined.
  pending?: boolean;
}
```

- [ ] **Step 2: Delete the source file**

```bash
git rm frontend/src/components/Member/types.ts
```

- [ ] **Step 3: Update the importer in `MembersPage.tsx`**

Edit `frontend/src/react/pages/settings/MembersPage.tsx` line 22. Replace:

```tsx
import type { GroupBinding, MemberBinding } from "@/components/Member/types";
```

with:

```tsx
import type { GroupBinding, MemberBinding } from "@/types/v1/member";
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/v1/member.ts \
        frontend/src/react/pages/settings/MembersPage.tsx
git commit -m "refactor(types): move Member binding types to types/v1/member"
```

---

## Task 13: Combine `Member/utils.ts` + `Member/projectRoleBindings.ts` into `src/utils/v1/member.ts`

**Files:**
- Create: `frontend/src/utils/v1/member.ts`
- Delete: `frontend/src/components/Member/utils.ts`
- Delete: `frontend/src/components/Member/projectRoleBindings.ts`
- Modify: `frontend/src/react/pages/settings/MembersPage.tsx`

The two source files share no helpers and import disjoint sets — combine by concatenation, with `utils.ts` content first, then `projectRoleBindings.ts` content. Both files reference types from `@/types/v1/member` (after Task 12) — verify the imports.

- [ ] **Step 1: Read both source files in full**

```bash
cat frontend/src/components/Member/utils.ts
echo "---"
cat frontend/src/components/Member/projectRoleBindings.ts
```

Confirm:
- `utils.ts` exports `getMemberBindings` and any internal helpers (export-only what callers use).
- `projectRoleBindings.ts` exports `ProjectRoleBindingGroup` (type), `getProjectRoleBindingKey`, and `groupProjectRoleBindings`.

- [ ] **Step 2: Create the combined file**

Create `frontend/src/utils/v1/member.ts`. Write the combined content:

1. Merge imports (deduplicate). Any reference inside the bodies to local types (`MemberRole`, `GroupBinding`, `MemberBinding`) that previously came from `./types` must now come from `@/types/v1/member`.
2. Paste the export bodies from `utils.ts` first, then `projectRoleBindings.ts`, in that order. Preserve all comments.
3. Do not rename any exported symbol.

If `utils.ts` had a local relative import like `from "./types"`, change it to `from "@/types/v1/member"` in the combined file.

- [ ] **Step 3: Delete the source files**

```bash
git rm frontend/src/components/Member/utils.ts \
       frontend/src/components/Member/projectRoleBindings.ts
```

- [ ] **Step 4: Update `MembersPage.tsx`**

Edit `frontend/src/react/pages/settings/MembersPage.tsx`. Replace both lines:

```tsx
import { groupProjectRoleBindings } from "@/components/Member/projectRoleBindings";
import { getMemberBindings } from "@/components/Member/utils";
```

with a single combined import:

```tsx
import {
  getMemberBindings,
  groupProjectRoleBindings,
} from "@/utils/v1/member";
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/utils/v1/member.ts \
        frontend/src/react/pages/settings/MembersPage.tsx
git commit -m "refactor(utils): combine Member helpers into utils/v1/member"
```

---

## Task 14: Move `RoleGrantPanel/DatabaseResourceForm/common.ts` to `src/utils/v1/databaseResource.ts`

**Files:**
- Create: `frontend/src/utils/v1/databaseResource.ts`
- Delete: `frontend/src/components/RoleGrantPanel/DatabaseResourceForm/common.ts`
- Modify: `frontend/src/react/components/sql-editor/RequestQueryButton.tsx`

The second importer was `MembersPage.tsx`'s reference to a different file path — re-verify before editing.

- [ ] **Step 1: Re-confirm the importer set**

Run:

```bash
grep -rnE "@/components/RoleGrantPanel" frontend/src 2>/dev/null
```

Expected: matches in `frontend/src/react/components/sql-editor/RequestQueryButton.tsx` and (if it exists) any other location. The earlier survey showed two — confirm them here.

- [ ] **Step 2: Move the file with git mv**

```bash
git mv frontend/src/components/RoleGrantPanel/DatabaseResourceForm/common.ts \
       frontend/src/utils/v1/databaseResource.ts
```

The contents do not need to change — its existing imports (`@/store`, `@/types`, `@/utils`, proto-es) all resolve identically from the new location.

- [ ] **Step 3: Update every importer found in Step 1**

For each importer file, replace:

```ts
from "@/components/RoleGrantPanel/DatabaseResourceForm/common"
```

with:

```ts
from "@/utils/v1/databaseResource"
```

Known importer: `frontend/src/react/components/sql-editor/RequestQueryButton.tsx:4`.

- [ ] **Step 4: Re-grep to confirm no stale paths**

```bash
grep -rnE "@/components/RoleGrantPanel" frontend/src 2>/dev/null
```

Expected: no matches.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/utils/v1/databaseResource.ts \
        frontend/src/react/components/sql-editor/RequestQueryButton.tsx
git commit -m "refactor(utils): move DatabaseResourceForm/common to utils/v1/databaseResource"
```

---

## Task 15: Move `RolloutV1/constants/task.ts` to `src/utils/v1/issue/task.ts`

**Files:**
- Create: `frontend/src/utils/v1/issue/task.ts`
- Delete: `frontend/src/components/RolloutV1/constants/task.ts`
- Modify: `frontend/src/utils/v1/issue/rollout.ts`

- [ ] **Step 1: Move the file**

```bash
git mv frontend/src/components/RolloutV1/constants/task.ts \
       frontend/src/utils/v1/issue/task.ts
```

Contents do not change.

- [ ] **Step 2: Update the importer**

Edit `frontend/src/utils/v1/issue/rollout.ts` line 1. Replace:

```ts
import { TASK_STATUS_FILTERS } from "@/components/RolloutV1/constants/task";
```

with:

```ts
import { TASK_STATUS_FILTERS } from "./task";
```

- [ ] **Step 3: Confirm no other importers**

```bash
grep -rnE "@/components/RolloutV1" frontend/src 2>/dev/null
```

Expected: no matches.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/utils/v1/issue/task.ts \
        frontend/src/utils/v1/issue/rollout.ts
git commit -m "refactor(utils): move RolloutV1/constants/task to utils/v1/issue/task"
```

---

## Task 16: Delete the now-empty `frontend/src/components/` directory

**Files:**
- Delete: `frontend/src/components/` (directory)

- [ ] **Step 1: List what's left**

```bash
find frontend/src/components -type f 2>/dev/null
```

Expected: no files (every individual file was deleted or moved in prior tasks).

- [ ] **Step 2: Confirm no remaining references anywhere**

```bash
grep -rnE "@/components/|\"./components/|'\\./components/" frontend/src 2>/dev/null
```

Expected: no matches.

- [ ] **Step 3: Remove the directory**

```bash
rm -rf frontend/src/components
```

(Empty directories don't appear in git; once the contents were `git rm`'d in earlier commits, the directory ceases to exist in the working tree as well. `rm -rf` is a no-op safety net.)

- [ ] **Step 4: Commit (if there is any tracked change)**

```bash
git status frontend/src/components 2>&1 | head -5
```

If the prior task commits already removed the last tracked file in `components/`, there is nothing to commit here — proceed to Task 17. Otherwise:

```bash
git add -A frontend/src/components
git commit -m "chore(frontend): remove empty components directory"
```

---

## Task 17: Validate the full PR

Per repo memory, all per-file checks were skipped during implementation; validation runs once now.

- [ ] **Step 1: Auto-fix lint + format**

```bash
pnpm --dir frontend fix
```

Expected: completes cleanly; if it modifies any file, stage the result.

- [ ] **Step 2: Run the static check (no modifications, CI-equivalent)**

```bash
pnpm --dir frontend check
```

Expected: PASS — ESLint + Biome + import-sort + the layering scanner all green. No new lint warnings introduced.

- [ ] **Step 3: Run the type checks**

```bash
pnpm --dir frontend type-check
```

Expected: PASS — both vue-tsc and the React tsconfig succeed. Common cause of failure here: an import path that pointed at the old `@/components/...` location was missed. If so, the type error message includes the bad path; fix and re-run.

- [ ] **Step 4: Run the test suite**

```bash
pnpm --dir frontend test
```

Expected: PASS. Critical tests to monitor:

- `src/react/app/SessionExpiredSurfaceGate.test.tsx` — the new gate test (Task 2)
- `src/layouts/layout-bridge.test.ts` — no longer mocks `AgentWindowMount`; the rendered BodyLayout should no longer contain `data-testid="agent-window"` (that assertion, if present, must be removed in Task 6)
- `src/react/components/auth/SessionExpiredSurface.test.tsx` — 4 component tests remain after Task 8 surgery
- `src/shell-bridge.test.ts` — `vi.mock` path updated in Task 10
- Every `vi.mock("@/utils/v1/instance", ...)` test from Task 11
- `src/react/no-react-to-vue-imports.test.ts` and `src/react/no-legacy-vue-deps.test.ts` — cross-framework import guards. The new `mount.tsx` imports `watch` from "vue" and `locale` from `@/plugins/i18n` (Vue side). These guards specifically allow boot/lifecycle code under `src/react/app/` to bridge into Vue — verify by reading the guard's allowlist. If `src/react/app/` is not allowlisted, the guard test will fail and you must add the directory to the allowlist (one-line edit) as part of this PR.

- [ ] **Step 5: Fix anything that breaks**

If any of the above fail:
- Type errors: read the path in the message; if it's a stale `@/components/...` reference, repoint to the new home from this plan's File Map.
- Test failures: the most common cause is a `vi.mock` path still pointing at `@/components/...`. Re-run `grep -rnE "@/components/" frontend/src` to find stragglers.
- Layering scanner: this PR does not add any `z-index` or portal — should not trigger.

After fixing, re-run all four commands in order until each is green. Commit the fixes:

```bash
git add -A frontend/src
git commit -m "chore(frontend): post-relocation validation fixes"
```

(Skip this commit if no fixes were needed.)

- [ ] **Step 6: Manual smoke test**

Start the dev server:

```bash
pnpm --dir frontend dev
```

Open the app in a browser. Verify:

1. The app boots without console errors.
2. `Cmd+Shift+A` (or `Ctrl+Shift+A`) opens the AgentWindow.
3. The AgentWindow renders at the bottom-right floating position and behaves identically to before (chat list, input, minimize/close).
4. In DevTools → Application → Storage, clear the auth cookie/token, then trigger any API call (navigate to another page). The `SessionExpiredSurface` modal should appear with the current path preserved as `currentPath`.
5. The `<div id="react-app">` element in DOM is present but visually empty; the agent UI lives inside `#bb-react-layer-agent`.

If anything misbehaves, halt and diagnose before opening the PR. The most likely failure mode is the Vue→React i18n sync — confirm by switching locale via the Vue UI and verifying the AgentWindow's `t(...)` strings update.

- [ ] **Step 7: Open the PR**

Follow the project's standard PR flow (see `docs/pre-pr-checklist.md`). Suggested PR title:

> `refactor(frontend): introduce sibling React app, empty /components/ directory`

The PR description should link to both the spec and this plan, summarize the React-app introduction (with the rationale that AgentWindow + SessionExpiredSurface already portal globally), and list the seven relocations.
