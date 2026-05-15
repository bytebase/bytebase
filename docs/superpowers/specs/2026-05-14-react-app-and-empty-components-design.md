# Introduce React App + Empty `frontend/src/components/` — Design

Date: 2026-05-14
Status: design (pre-implementation)
Delivery: single PR

## Context

`frontend/src/components/` has been steadily emptied during the Vue→React migration. Only 10 files remain — a mix of three Vue files and seven non-component utility / type / constants files that ended up under `components/` for historical reasons. The directory's name no longer reflects what lives in it.

Two of the three remaining Vue files — `AgentWindowMount.vue` and `SessionExpiredSurfaceMount.vue` — exist as Vue shims that own a `<div>` and call `mountReactPage(container, ...)` from `src/react/mount.ts` in `onMounted`. Both render React components (`AgentWindow`, `SessionExpiredSurface`) that already `createPortal(..., getLayerRoot(...))` to `document.body`-level layer roots. Their visible DOM never lives inside the Vue subtree; the Vue mount is just a lifecycle anchor for the React root.

That means these shims are not load-bearing on their Vue parents. They are deletable today — if there is somewhere outside the Vue tree to hold the React root.

This design introduces that somewhere: a **sibling React app**, mounted at app boot alongside the existing Vue app, dedicated to React surfaces that already portal globally and don't need to be positioned inside the Vue layout DOM. Initial residents: `AgentWindow` and `SessionExpiredSurface`. The same PR also relocates the seven utility files to their natural homes and removes `frontend/src/components/` entirely.

## Goals

1. Stand up a long-lived React root in `index.html` at `<div id="react-app">`, mounted from `src/main.ts` after the Vue app mounts.
2. Move `AgentWindow` and `SessionExpiredSurface` into that React app. Delete both `*Mount.vue` shims.
3. Relocate every remaining file under `frontend/src/components/` to a natural home.
4. Delete `frontend/src/components/` entirely.

## Non-goals

- No visual or behavioral change to `AgentWindow` or `SessionExpiredSurface`.
- No migration of `OverlayStackManager.vue` to React — it remains Vue infrastructure for `BBModal.vue` (still Vue). It just relocates.
- No migration of `BodyLayout.vue` or `AuthContext.vue` to React. Both stay Vue; they just lose one child element each.
- No changes to existing `ReactPageMount.vue` / `mountReactPage` usage elsewhere. Those continue to be the right tool for React surfaces that must be positioned inside the Vue layout DOM.

## React app

### `index.html`

Add a sibling mount point:

```html
<div id="app"></div>
<div id="react-app"></div>          <!-- NEW -->
<div id="bb-toaster-root"></div>
```

`#react-app` has no visible DOM of its own — its children portal to `getLayerRoot("agent")` and `getLayerRoot("critical")`. Layout position is irrelevant.

### `src/react/app/`

Three new files:

**`ReactApp.tsx`** — top-level tree:

```tsx
export function ReactApp() {
  return (
    <I18nextProvider i18n={i18n}>
      <AgentWindow />
      <SessionExpiredSurfaceGate />
    </I18nextProvider>
  );
}
```

**`SessionExpiredSurfaceGate.tsx`** — the conditional + route-path wiring that previously lived in the Vue shim:

```tsx
export function SessionExpiredSurfaceGate() {
  const unauthenticatedOccurred = useVueState(
    () => useAuthStore().unauthenticatedOccurred
  );
  const currentPath = useVueState(() => router.currentRoute.value.fullPath);
  if (!unauthenticatedOccurred) return null;
  return <SessionExpiredSurface currentPath={currentPath} />;
}
```

`useAuthStore` and `router` are imported directly per the codebase convention.

**`mount.tsx`** — boot + Vue→React i18n sync (`.tsx` because it renders JSX):

```tsx
import { watch } from "vue";
import { createRoot } from "react-dom/client";
import { StrictMode } from "react";
import i18n, { i18nReady } from "@/react/i18n";
import { locale } from "@/plugins/i18n";
import { ReactApp } from "./ReactApp";

export async function mountReactApp(selector: string) {
  const container = document.querySelector(selector);
  if (!container) throw new Error(`Missing React app mount point: ${selector}`);
  await i18nReady;
  const root = createRoot(container);
  root.render(
    <StrictMode>
      <ReactApp />
    </StrictMode>
  );
  // One-time Vue→React locale sync. Replaces the per-shim watch(locale, ...).
  watch(locale, async (next) => {
    if (i18n.language !== next) await i18n.changeLanguage(next);
  });
  return root;
}
```

### `src/main.ts`

After the existing `createApp(App).mount("#app")`:

```ts
const { mountReactApp } = await import("./react/app/mount");
void mountReactApp("#react-app"); // fire-and-forget; boot continues
```

Fire-and-forget is intentional — `mountReactApp` resolves only after `i18nReady`, and we don't want to delay any subsequent Vue boot steps on it. The React app initializing in the background is fine because both its initial residents (`AgentWindow`, `SessionExpiredSurface`) are user-triggered or auth-state-triggered, not boot-time-visible.

Dynamic import keeps the React bundle off the Vue boot critical path.

### Knock-on edits

- `src/layouts/BodyLayout.vue:43` — remove `<AgentWindowMount />` from template; remove the `import AgentWindowMount from "@/components/AgentWindowMount.vue"`.
- `src/AuthContext.vue:8` — remove `<SessionExpiredSurfaceMount v-if="authStore.unauthenticatedOccurred" />`; remove the import.
- `src/layouts/layout-bridge.test.ts:118` — drop the `vi.mock("@/components/AgentWindowMount.vue", ...)` block; the surface is no longer mounted by `BodyLayout`.
- `src/react/components/auth/SessionExpiredSurface.test.tsx` — update import path if it references the old shim (component itself is unchanged).
- `src/react/mount.ts` — `pageLoaders` no longer needs `AgentWindow.tsx` or `SessionExpiredSurface.tsx` for the `mountReactPage` path. Verify and prune the glob entries if unused.

## File relocations

| Source | Destination | Notes |
|---|---|---|
| `AdvancedSearch/types.ts` | 🗑️ delete | 0 importers; orphan |
| `AgentWindowMount.vue` | 🗑️ delete | Replaced by React app |
| `SessionExpiredSurfaceMount.vue` | 🗑️ delete | Replaced by `SessionExpiredSurfaceGate` |
| `misc/OverlayStackManager.vue` | `src/bbkit/OverlayStackManager.vue` | Vue infra for `BBModal`; one importer |
| `InstanceForm/constants.ts` | merge into `src/utils/v1/instance.ts` | 10 importers; engine port/icon helpers |
| `Member/utils.ts` | `src/utils/v1/member.ts` (new) | 1 importer (`MembersPage.tsx`) |
| `Member/projectRoleBindings.ts` | append to `src/utils/v1/member.ts` | 1 importer; combined for density |
| `Member/types.ts` | `src/types/v1/member.ts` (new) | 1 importer; matches existing `src/types/v1/{database,project,user}.ts` pattern (the existing top-level `src/types/member.ts` is unrelated — it holds `DatabaseResource`) |
| `RoleGrantPanel/DatabaseResourceForm/common.ts` | `src/utils/v1/databaseResource.ts` (new) | 2 importers; pure helpers |
| `RolloutV1/constants/task.ts` | `src/utils/v1/issue/task.ts` (new) | 1 importer; sibling `rollout.ts` exists |

All import sites are updated in the same PR. Total importer count across the seven .ts files: ~18 import statements to rewrite.

### Destination rationale

- **`src/utils/v1/instance.ts`** already exists and contains `supportedEngineV1List` and related helpers. The engine port / icon constants belong with them.
- **`src/types/v1/member.ts`** is new; matches the sibling pattern of `src/types/v1/{database,project,user}.ts`. The existing top-level `src/types/member.ts` only holds `DatabaseResource` (unrelated) — not extended in this PR.
- **`src/utils/v1/member.ts`** is new but mirrors the existing sibling pattern (`src/utils/v1/{database,user,project}.ts`).
- **`src/utils/v1/databaseResource.ts`** is new; `parseStringToResource` and friends are pure DatabaseResource helpers with no UI dependency.
- **`src/utils/v1/issue/task.ts`** is new but `src/utils/v1/issue/rollout.ts` already exists in the same folder; task-status constants belong alongside.
- **`src/bbkit/OverlayStackManager.vue`** sits with the other `BB*` Vue primitives. The component's internal `name` is already `BBOverlayStack`.

## Validation

- `pnpm --dir frontend check` — ESLint + Biome + import sort
- `pnpm --dir frontend type-check` — both Vue (vue-tsc) and React (tsconfig.react.json)
- `pnpm --dir frontend test` — including `layout-bridge.test.ts` (updated), `SessionExpiredSurface.test.tsx`, `no-react-to-vue-imports.test.ts`, `no-legacy-vue-deps.test.ts`
- Manual smoke: open the app, confirm AgentWindow toggles via `Cmd/Ctrl+Shift+A`, confirm SessionExpired surface appears on forced 401 (e.g. revoke token in DevTools).
- Confirm the `frontend/src/components/` directory is absent at the end.

## Risks

- **Boot order race.** `mountReactApp` is awaited after Vue mounts; until it resolves, `AgentWindow` and `SessionExpiredSurface` are not present. AgentWindow is user-triggered so latency is invisible; SessionExpiredSurface only matters once an authenticated request fails, which happens long after boot. Acceptable.
- **Vue→React i18n sync coverage.** The existing shims each ran their own `watch(locale, ...)`. The new app runs one. If a future React-app resident needs to react to locale beyond `i18next.changeLanguage`, it must subscribe via `useTranslation()` (already standard).
- **`useVueState` re-render churn.** `SessionExpiredSurfaceGate` subscribes to two Vue reactive sources. Both update infrequently. No expected perf issue.
- **Layer ordering.** The React app's residents continue using `getLayerRoot("agent" | "critical")`. Layer ordering is unaffected because the layer roots live at `document.body`, not inside `#react-app` or `#app`.

## Out of scope, but enabled

Future React surfaces that already portal globally can move into `ReactApp.tsx` and drop their Vue mount shims. Candidates to consider in follow-up work: any future agent-related panels, body-level toasts that don't depend on Vue notification state, global keyboard-shortcut overlays. None of these are in this PR.
