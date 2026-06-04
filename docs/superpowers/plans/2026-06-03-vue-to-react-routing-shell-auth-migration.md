# Vue → React routing + shell + auth migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Vue host (vue-router + `App.vue` + `AuthContext.vue` + 3 layouts + 5 mount bridges + `ReactPageMount` + glob registry + Pinia auth store + guard + interceptor) with a single React root on react-router-dom v7, app store as the single session source of truth — fully deleting Vue.

**Architecture:** One `ReactDOM.createRoot` → `<RouterProvider router={createBrowserRouter(routes)}>` → `<RootLayout>` (providers + overlays) → `<AuthGate>` → `<Outlet/>`. Routes translate ~1:1 from vue-router; the guard is a verbatim port of `beforeEach` as a root-route `loader`; pages (already React) load via `lazy`.

**Tech Stack:** react-router-dom v7, the existing Zustand app store, react-i18next, esbuild react-tsx-transform.

**Reference:** `docs/superpowers/specs/2026-06-03-vue-to-react-routing-shell-auth-migration-design.md`. All commands run from repo root unless noted; frontend gates: `pnpm --dir frontend type-check`, `pnpm --dir frontend check`, `pnpm --dir frontend test`.

**Working branch:** `react-delete-vue-routing-shell-auth` (already created; spec committed).

**Verification gate after every task:** `pnpm --dir frontend type-check` exits 0 (ignore IDE `Cannot find module '@/...'` noise — only real `error TS` count). Run `pnpm --dir frontend test` after any task touching component/store/guard logic. Commit after each task.

---

## Phase 0 — Inventory & guardrails (read-only)

### Task 0.1: Capture the route table + guard as the porting source of truth
**Files (read):** `src/router/index.ts` (guard, 293 lines), `src/router/auth.ts`, `src/router/setup.ts`, `src/router/sqlEditor.ts`, `src/router/dashboard/{index,workspace,workspaceRoutes,workspaceSetting,projectV1,projectV1RouteHelpers,instance,environmentV1}.ts`.

- [ ] **Step 1:** Produce a flat table `routes.inventory.md` (scratch, not committed) listing every route: path, route-name constant, layout, the `page:"X"` string (or nested children), and any `meta`/`props`. Source: `grep -rnE "path:|name:|component:|page:|props:" src/router`.
- [ ] **Step 2:** Produce `bridges.inventory.md` (scratch): the 79 `useVueState` files (`grep -rln useVueState src/react --include=*.tsx --include=*.ts | grep -v .test.`), each tagged by what its getter reads (vue-router `useRoute`/`router`/`currentRoute`, Pinia auth, app-shell Pinia bridge, or other). This drives Phase 5.

No code change. No commit (scratch artifacts only).

---

## Phase 1 — React Router + route table (not yet wired to root)

### Task 1.1: Add react-router-dom
**Files:** `frontend/package.json`

- [ ] **Step 1:** `pnpm --dir frontend add react-router-dom@^7`
- [ ] **Step 2:** Verify it resolves: `pnpm --dir frontend type-check` → exit 0.
- [ ] **Step 3:** Commit: `chore(frontend): add react-router-dom v7`

### Task 1.2: Route-name handles module
**Files:** Create `src/react/router/handles.ts`

- [ ] **Step 1:** Re-export every route-name constant currently in `src/router/**` (92 constants: `*_MODULE`, `*_ROUTE_*`) from `@/react/router/handles` so React routes can attach `handle: { name }` without importing the Vue router files. For each, `export { AUTH_SIGNIN_MODULE } from "@/router/auth"` etc. (These constant modules are plain `.ts` with no Vue imports — verify with `grep -l "from \"vue\"" src/router/auth.ts` → empty.)
- [ ] **Step 2:** `type-check` → 0. Commit: `feat(frontend): re-export route-name handles for react router`

### Task 1.3: Translate the route table
**Files:** Create `src/react/router/routes.tsx` (+ split per-area files `routes/auth.tsx`, `routes/dashboard.tsx`, `routes/sqlEditor.tsx`, `routes/setup.tsx` if `routes.tsx` exceeds ~300 lines).

- [ ] **Step 1:** For each vue-router route, emit a react-router `RouteObject`:
  - `path` unchanged (vue `:param` → react `:param` is identical syntax).
  - layout route → `{ path, element: <SplashLayout/> | <BodyLayout/> | <DashboardLayout/>, children: [...] }` (React layouts built in Phase 3; for now import placeholders that render `<Outlet/>`).
  - leaf route `component: ReactPageMount, props:{page:"SigninPage"}` → `{ path, handle: { name: AUTH_SIGNIN_MODULE }, lazy: async () => ({ Component: (await import("@/react/pages/auth/SigninPage")).SigninPage }) }`. Map each of the 73 `page:"X"` strings to its `@/react/pages/.../X` import (the `import.meta.glob` patterns in `src/react/mount.ts` give the exact directories: `pages/settings`, `pages/project`, `pages/workspace`, `pages/auth`, `components/auth`, `components/sql-editor`, `components/*`).
  - named view `<router-view name="body">` → the child routes render into the layout's `<Outlet/>` (single outlet; drop the name).
- [ ] **Step 2:** Export `export const routes: RouteObject[]`. Do NOT call `createBrowserRouter` yet (wired in Phase 4).
- [ ] **Step 3:** `type-check` → 0 (placeholders compile). Commit: `feat(frontend): react-router route table translated from vue-router`

---

## Phase 2 — Auth slice + guard + interceptor

### Task 2.1: Port the Pinia auth store onto the app-store AuthSlice
**Files:** Modify `src/react/stores/app/auth.ts`, `src/react/stores/app/types.ts`. Source to port verbatim: `src/store/modules/v1/auth.ts` (281 lines).

- [ ] **Step 1:** Extend `AuthSlice` in `types.ts` with: `unauthenticatedOccurred: boolean`, `requireResetPassword: boolean` (getter via method `getRequireResetPassword()` OR derived), `authSessionKey: string`, `isSelfEmailUpdate: boolean`, `isLoggedIn: () => boolean`, `login`, `signup`, `logout`, `fetchCurrentUser`, `setRequireResetPassword`, `sendEmailLoginCode`, `updateCurrentUserNameForEmailChange`, `setUnauthenticatedOccurred: (v: boolean) => void`. Copy each method's exact signature from the Pinia store's return block.
- [ ] **Step 2:** Port each method body **verbatim** from `src/store/modules/v1/auth.ts`, with only these swaps: Vue `ref`/`computed` → slice state + `get()`/`set()`; `router.push(x)` / `router.replace(x)` → `appRouter.navigate(x)` (the react-router instance, imported from `@/react/router/instance` created in Phase 4 — until then, import lazily inside the method: `const { appRouter } = await import("@/react/router/instance")`). Keep the imported route-name constants and redirect targets identical.
- [ ] **Step 3:** `type-check` → 0; `pnpm --dir frontend test src/react/stores/app/index.test.ts` → passes (store constructs).
- [ ] **Step 4:** Commit: `feat(frontend): port the auth lifecycle onto the app-store AuthSlice`

### Task 2.2: Guard loader (verbatim port of beforeEach)
**Files:** Create `src/react/router/guard.ts`. Source: `src/router/index.ts:84-278` (the `beforeEach`).

- [ ] **Step 1:** Write `export async function rootGuard({ request, matches }: LoaderFunctionArgs)`. Reproduce the 13 checks **in the same order** (see spec §2). Read the target route name from `matches.at(-1)?.handle?.name`. Read session from `useAppStore.getState()` (`isLoggedIn()`, `unauthenticatedOccurred`, `requireResetPassword`). Where the old guard called `next({ name, query, replace })`, `throw redirect(pathFor(name, query))`; where it called `next()`, `return null`; where `next(false)`, `throw redirect(currentPath)` (no-op equivalent). Port the relay_state/redirect param logic and `stripSigninQueryParams`/`SIGNIN_QUERY_PARAMS` verbatim. Port the auth-route store resets (`resetDatabases/resetInstances/resetProjects` + the AI conversation reset).
- [ ] **Step 2:** Unit test `src/react/router/guard.test.ts` covering each branch: not-logged-in→signin (with `redirect` query); logged-in-on-auth-route→`/`; 2FA enforce; password-reset enforce; 403/404 passthrough; OAuth/OIDC callback passthrough; OAuth2 consent passthrough. Mock `useAppStore.getState()` session + a fake `matches`.
- [ ] **Step 3:** `pnpm --dir frontend test src/react/router/guard.test.ts` → all pass.
- [ ] **Step 4:** Commit: `feat(frontend): port the router guard as a react-router root loader`

### Task 2.3: Repoint the auth interceptor to the app store
**Files:** Modify `src/connect/middlewares/authInterceptorMiddleware.ts`.

- [ ] **Step 1:** Replace `useAuthStore().unauthenticatedOccurred = true` with `useAppStore.getState().setUnauthenticatedOccurred(true)`, and `authStore.isLoggedIn` reads with `useAppStore.getState().isLoggedIn()`. Keep the silent/notification logic identical.
- [ ] **Step 2:** `type-check` → 0; `pnpm --dir frontend test` → green. Commit: `refactor(frontend): auth interceptor reads session from the app store`

### Task 2.4: AuthGate (replaces AuthContext.vue)
**Files:** Create `src/react/app/AuthGate.tsx`. Source behavior: `src/AuthContext.vue` (139 lines).

- [ ] **Step 1:** A component that: subscribes to `isLoggedIn`/`unauthenticatedOccurred` via `useAppStore`; on mount, if not logged in and not on an auth route, `void useAppStore.getState().fetchCurrentUser()`; reproduces `AuthContext.vue`'s render gating + the `currentUserName` watch + `isSelfEmailUpdate` reset. Renders `{children}` (the `<Outlet/>`).
- [ ] **Step 2:** Test `AuthGate.test.tsx`: renders children when logged in; triggers `fetchCurrentUser` when not loaded.
- [ ] **Step 3:** `type-check` + test → pass. Commit: `feat(frontend): AuthGate replaces AuthContext.vue`

---

## Phase 3 — React layouts + RootLayout

### Task 3.1: React layout route components
**Files:** Create `src/react/app/layouts/{SplashLayout,BodyLayout,DashboardLayout}.tsx`. Source: `src/layouts/*.vue`.

- [ ] **Step 1:** Each renders its chrome + `<Outlet/>` for children. `DashboardLayout` mounts the React sidebar (today `ReactSidebarMount.vue` → use the existing `src/react/mountSidebar.ts` React tree directly) + `<Outlet/>` for the body. `BodyLayout` (152 lines) and `SplashLayout` port their wrapper markup. Port any `useEnsureWorkspaceCommonData()` call that lived in the Vue layout into `DashboardLayout`.
- [ ] **Step 2:** Wire the real layouts into `routes.tsx` (replace Phase-1 placeholders).
- [ ] **Step 3:** `type-check` → 0. Commit: `feat(frontend): react layout routes (Splash/Body/Dashboard)`

### Task 3.2: RootLayout
**Files:** Create `src/react/app/RootLayout.tsx`.

- [ ] **Step 1:** Renders the global overlays from today's `ReactApp` (`Watermark`, `Toaster`, `AgentWindow`, `SessionExpiredSurfaceGate`) + `<AuthGate><Outlet/></AuthGate>`. This is the root route's `element`, with `loader: rootGuard`.
- [ ] **Step 2:** `type-check` → 0. Commit: `feat(frontend): RootLayout (providers + overlays + guard + AuthGate)`

---

## Phase 4 — Flip main.ts to the single React root

### Task 4.1: Router instance + RouterProvider entry
**Files:** Create `src/react/router/instance.ts` (exports `appRouter = createBrowserRouter([{ element: <RootLayout/>, loader: rootGuard, children: routes }])`) and `src/react/app/AppRoot.tsx` (`<RouterProvider router={appRouter}/>`).

- [ ] **Step 1:** Create both. Resolve the Phase-2 lazy `appRouter` import in the auth slice to the static `@/react/router/instance`.
- [ ] **Step 2:** `type-check` → 0. Commit: `feat(frontend): react-router browser router instance + AppRoot`

### Task 4.2: Rewrite main.ts
**Files:** Modify `src/main.ts` (76 lines).

- [ ] **Step 1:** Remove `createApp(App)`, `app.use(pinia/router/highlight/i18n)`, `app.mount("#app")`, and the separate `mountReactApp("#react-app")` block. Keep the side-effect imports (`./init`, `regenerator-runtime`, css, `./react/lib/toast`, `migrateStorageKeys`). Replace the bootstrap with: `const currentUser = await useAppStore.getState().fetchCurrentUser();` then the same `fetchServerInfo`/`fetchSubscription`/`fetchWorkspaceList` init (now via app store), then `ReactDOM.createRoot(document.getElementById("app")!).render(<AppRoot/>)`. Keep `highlight`/`i18n`/`dayjs` plugin init where still needed (highlight is non-Vue; i18n → react-i18next is already initialized in `@/react/i18n`).
- [ ] **Step 2:** `index.html`: ensure a single `#app` root div (remove `#react-app` if present).
- [ ] **Step 3:** `pnpm --dir frontend build` (or `pnpm --dir frontend dev` smoke) compiles; `type-check` → 0; `test` → green.
- [ ] **Step 4:** Commit: `feat(frontend): boot a single React root via RouterProvider`

> ⚠️ After Task 4.2 the app runs on React Router. **Manual QA checkpoint** (spec verification): login, logout, deep-link to a protected route (→signin→back), 404, OAuth callback. Fix regressions before proceeding.

---

## Phase 5 — Sweep the 79 useVueState bridges

### Task 5.1..5.N: per-area bridge sweeps
**Files:** the 79 `useVueState` files (Phase 0.1 inventory). Batches: `pages/project` (47), `pages/settings` (16), `pages/auth` (4) + `hooks`/`components` (8).

- [ ] For each file, apply the transformation rule by the getter's source:
  - vue-router (`useRoute()`, `router`, `currentRoute`) → `useParams()` / `useLocation()` / `useNavigate()`.
  - Pinia auth read → `useAppStore((s) => s.<field>)` / `s.isLoggedIn()`.
  - app-shell Pinia-bridge read (`useAppFeature`, the `DashboardFrameShell` legacy bootstrap, `syncSubscriptionToPinia`) → direct app-store selector; delete the now-orphan bridge.
  - Replace `useVueState(() => X)` with the equivalent React hook/selector; remove the `useVueState` import.
- [ ] After each batch: `type-check` → 0; `test` → green; commit `refactor(frontend): drop useVueState bridges in <area> (<n>/79)`.

### Task 5.last: delete the bridge infra
**Files:** Delete `src/react/hooks/useVueState.ts`; remove `effectScope` usage in `src/react/components/ProjectSidebar.tsx`; delete `src/react/ReactPageMount.vue`, `ReactRouteShellBridge.vue`, `ReactSidebarMount.vue`, `ReactProjectSidebarMount.vue`, `LandingPageMount.vue`, `src/react/mount.ts`, `src/react/app/mount.ts` glob registry.

- [ ] `grep -rn useVueState src/react --include=*.tsx --include=*.ts | grep -v .test.` → empty. `type-check` + `test` → green. Commit.

---

## Phase 6 — Replace pev2 with a React plan visualizer

### Task 6.1: React Postgres plan view
**Files:** Rewrite `src/react/explain-visualizer/PostgresPlanView.tsx` (currently `createApp(pev2)`); siblings `MSSQLPlanView.tsx` + `SpannerQueryPlan.tsx` are already React for parity.

- [ ] **Step 1:** Replace the `createApp(pev2)` Vue mount with a React plan-tree component rendering the same parsed-plan data (`@/utils/pev2` `readExplainFromToken` output): node tree with per-node cost/rows/time, expand/collapse, and the cost heat coloring pev2 provided. Reuse `MSSQLPlanView`/`SpannerQueryPlan` patterns for the tree UI.
- [ ] **Step 2:** Remove `import "pev2/dist/pev2.css"` and the pev2 Font-Awesome shim in `main.tsx`; drop the `pev2` dep from `package.json`.
- [ ] **Step 3:** Test `PostgresPlanView.test.tsx` (render a sample plan, assert node rows). `type-check` + `test` → green. **Screenshot-compare** against the old pev2 output.
- [ ] **Step 4:** Commit: `feat(frontend): react postgres query-plan visualizer (replaces pev2)`

---

## Phase 7 — Delete Vue + dependency teardown

### Task 7.1: Delete the Vue shell + vue-router + Pinia auth
**Files:** Delete `src/App.vue`, `src/AuthContext.vue`, `src/layouts/*.vue`, `src/router/index.ts` (guard — superseded), `src/store/modules/v1/auth.ts` + its barrel export, `src/react/app/ReactApp.tsx` (folded into RootLayout). Keep the route-name constant modules under `src/router/**` (re-exported by `handles.ts`) OR relocate them into `src/react/router/` and delete `src/router/**`.

- [ ] `grep -rln "useAuthStore\|from \"vue-router\"\|\.vue\"" src --include=*.ts --include=*.tsx` → only the constant modules / none. `type-check` + `test` → green. Commit.

### Task 7.2: vue-i18n → react-i18next
**Files:** the 5 `vue-i18n` importers (`grep -rln vue-i18n src`).

- [ ] Replace each with `react-i18next` `useTranslation()` / the `@/react/i18n` instance. `type-check` + `test` → green. Commit.

### Task 7.3: Remove Vue dependencies + build config
**Files:** `frontend/package.json`, `frontend/vite.config.ts`, `frontend/tsconfig*.json`, `frontend/scripts/`.

- [ ] **Step 1:** `pnpm --dir frontend remove vue vue-router pinia vue-i18n @vueuse/core @vitejs/plugin-vue @vue/compiler-sfc @vue/test-utils @vue/tsconfig vue-tsc`.
- [ ] **Step 2:** Remove the Vue plugin from `vite.config.ts`; drop `vue-tsc` from the type-check script (esbuild react-tsx-transform + `tsconfig.react.json` remain).
- [ ] **Step 3:** Add `frontend/scripts/check-react-no-pinia.mjs` (fails CI if `src/react/**` imports `@/store` Pinia hooks) and wire into `pnpm check`.
- [ ] **Step 4:** `pnpm --dir frontend i` clean; `type-check` + `check` + `test` + `build` → all green. Commit: `chore(frontend): remove Vue, vue-router, pinia, vue-i18n; add no-pinia guard`

### Task 7.4: Final verification + PR
- [ ] `find src -name '*.vue'` → empty. `grep -rn "from \"vue\"" src --include=*.ts --include=*.tsx` → empty.
- [ ] Full manual QA pass (spec §Verification): login, logout, session-expiry (401→surface), OAuth/OIDC callback, 2FA setup, password reset, deep-link redirect, workspace switch, SQL-editor entry, Postgres EXPLAIN viz.
- [ ] Open the PR with the spec + this plan linked.

---

## Self-review notes
- **Spec coverage:** §1 architecture → Phases 3,4; §2 auth/guard → Phase 2; §3 routes/bridges/pev2 → Phases 1,5,6; §4 build order → phase numbering matches the spec's 7 steps; verification → per-task gates + the two manual-QA checkpoints (after Phase 4 and Task 7.4).
- **Risk ordering:** the app runs end-to-end after Phase 4 (root flip), so Phases 5–7 are incremental cleanups each behind green gates.
- **Verbatim ports** (auth slice, guard) cite exact source files + the only allowed transformations (no behavior change) — precise, not placeholder.
