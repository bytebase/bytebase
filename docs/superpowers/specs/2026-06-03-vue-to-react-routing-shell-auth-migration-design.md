# Vue → React: routing + app-shell + auth migration

**Goal:** Replace the Vue *host* (vue-router, `App.vue`, `AuthContext.vue`, the 3
layouts, the 5 React-mount bridge `.vue` files, `ReactPageMount` + the glob page
registry, the Pinia auth store, the router guard, the auth interceptor) with a
single React root running **react-router-dom v7**, with the app store as the
single source of truth for session state. This is the critical-path track to
**fully deleting Vue** from `frontend/`.

**Delivery:** one **big-bang PR** (decision). The pages are already React, so the
bulk is mechanical route-table translation + deleting Vue files. The PR is built
in an internal order (below) so the tree stays buildable/reviewable per commit.

**Hard constraints:**
- **No breaking changes.** Purely swapping the host; nothing user-visible changes.
- **The guard is a faithful, line-by-line port** of the existing `router/index.ts`
  `beforeEach` — same checks, same order, same redirects.

## Decisions (locked)
1. Strategy: **big-bang single PR** (no two-router coexistence).
2. Router: **react-router-dom v7** (`createBrowserRouter`, nested route objects,
   `lazy`, loaders) — closest 1:1 translation of the nested vue-router config.
3. Guard: **faithful port** of `beforeEach` as a root-route `loader`.
4. pev2: **replace the third-party Vue query-plan visualizer with React** in this
   PR, so the `vue` core dep is removed entirely (clean, complete deletion).

## §1 Target architecture (end state)
One React root owns everything. `main.ts` drops `createApp(App)` and the separate
`#react-app` overlay mount, and mounts a single tree:

```
ReactDOM.createRoot(#app).render(
  <RouterProvider router={createBrowserRouter(routes)} />
)
  └─ <RootLayout>          // replaces App.vue: providers + global overlays
       (Toaster, AgentWindow, Watermark, SessionExpiredSurfaceGate — today's ReactApp)
       └─ <AuthGate>       // replaces AuthContext.vue: gates render on app-store session
            └─ <Outlet/>   // matched layout + page
```

- **Routes:** `createBrowserRouter` with a nested route-object array translated
  ~1:1 from the vue-router config (~102 routes, 73 page names). The 3 Vue layouts
  (`Splash`/`Body`/`Dashboard`) become React **layout routes** with `<Outlet/>`;
  `DashboardLayout`'s named `<router-view name="body">` → a nested `<Outlet/>`.
  Each leaf swaps `component: ReactPageMount, props:{page:"X"}` for
  `lazy: () => import("@/react/pages/.../X")` — replacing the island mount + the
  `import.meta.glob` registry. One React root; pages render in-place.
- **Navigation:** `useNavigate` / `useParams` / `useLocation` / `<Navigate>`
  replace `router.push` / `useRoute`.
- **Route names:** existing name constants (`AUTH_SIGNIN_MODULE`, …) are kept and
  attached as `handle: { name }` so the guard, `useRecentVisit`, and module
  back-path tracking keep working unchanged.

## §2 Auth / session model + guard
**Session source of truth → app-store `AuthSlice`.** Port the Pinia auth store
1:1 onto the slice:
- State: `currentUser`, `unauthenticatedOccurred`, `requireResetPassword`,
  `authSessionKey`, `isSelfEmailUpdate`. Getter: `isLoggedIn`.
- Methods: `login`, `signup`, `logout`, `fetchCurrentUser`,
  `setRequireResetPassword`, `sendEmailLoginCode`,
  `updateCurrentUserNameForEmailChange` — bodies copied verbatim; only the
  navigation calls swapped (`router.push` → the React Router instance's
  `router.navigate(...)`; data routers expose `.navigate()`).

**Auth interceptor:** `authInterceptorMiddleware.ts` flips `unauthenticatedOccurred`
on a 401 — repointed from `useAuthStore()` to `useAppStore.getState()`.
`SessionExpiredSurface` already reacts to that flag (now reads the app store).

**`<AuthGate>` (replaces `AuthContext.vue`):** reads `isLoggedIn` /
`unauthenticatedOccurred` from the app store, calls `fetchCurrentUser()` on mount
when not loaded, and gates render exactly like `AuthContext.vue` (watch on
`currentUserName`, `isSelfEmailUpdate` handling).

**The guard (replaces `beforeEach`):** a single **root-route `loader`** that runs
on every navigation and reproduces the existing guard step-for-step:
1. same-route loop guard → 2. 403/404 direct access → 3. OAuth/OIDC callbacks
direct → 4. OAuth2 consent → 5. allow 2FA-setup / password-reset / profile-setup
when logged in → 6. module back-path tracking → 7. logged-in-on-auth-route
redirect (relay_state/redirect) → 8. auth-route store resets
(resetDatabases/instances/projects) → 9. not-logged-in → signin redirect (with
`redirect` query) → 10. enforce 2FA → 11. enforce password reset → 12.
allowed-route-patterns → 13. 404 fallback. It reads the matched route's
`handle.name` (mirroring vue route names), reads session from the app store, and
`throw redirect()` where the old guard called `next({name})`.

## §3 Route table, the 79 bridges, reverse islands
- **Route table:** mechanical translation of the ~102 routes; layouts → layout
  routes; leaf routes → `lazy` imports; name constants → `handle.name`. The
  SQL-editor "layout as a page" island collapses into a normal layout route.
- **The 79 `useVueState` bridges:** each reads a Vue-reactive source that is being
  deleted, so each is rewritten — vue-router reads (`useRoute`/`router`/
  `currentRoute`) → `useParams`/`useLocation`/`useNavigate`; Pinia-auth reads →
  app-store auth selectors; app-shell Pinia-bridge reads (the `DashboardFrameShell`
  legacy bootstrap + `syncSubscriptionToPinia`) → those bridges are *deleted* once
  the Vue shell that needed them is gone, so they become direct app-store
  selectors. Then `useVueState.ts`, the Vue reactivity primitives (`effectScope`
  in `ProjectSidebar`), and the page-mount/glob registry are removed. This ~79-file
  sweep is the single largest mechanical part and the main reason the PR is big.
- **vue-i18n (5 importers) → react-i18next** (React already uses it); the
  `ReactPageMount` locale-sync watcher goes away.
- **pev2 reverse island:** re-implement the Postgres query-plan visualizer
  (`PostgresPlanView.tsx`, currently `createApp(pev2)`) in React so `vue` is fully
  removed. The one piece of genuinely net-new UI; isolated in its own commit.

## §4 Internal build order (within the one PR)
1. Add `react-router-dom`; author the nested route table + `handle.name` + `lazy`
   imports (not wired to the root yet).
2. Port the auth slice (verbatim bodies) + the guard loader + `<AuthGate>` +
   repoint the interceptor.
3. Build `<RootLayout>` + the 3 React layout routes.
4. Flip `main.ts` to the single React root (`RouterProvider`); drop
   `createApp(App)` + `#react-app`.
5. Sweep the 79 `useVueState` bridges → React Router hooks / app-store selectors;
   delete `useVueState` + Vue reactivity primitives + glob registry.
6. Replace pev2 with a React plan visualizer.
7. Delete the Vue shell (`App.vue`, `AuthContext.vue`, 3 layouts, 5 mount bridges,
   `ReactPageMount`, Pinia auth store, old guard) + remove `vue`/`vue-router`/
   `pinia`/`vue-i18n`/`@vueuse`/`@vitejs/plugin-vue`/`vue-tsc` + clean the Vite
   config + add `check-react-no-pinia.mjs`.

## Verification ("no breaking changes" gate)
- **Guard unit tests** mirroring every `beforeEach` branch (not-logged-in→signin,
  logged-in-on-auth-route→home, 2FA enforce, password-reset enforce, 403/404,
  OAuth callback passthrough, consent).
- Full `type-check` / `check` / `test` (the existing 2308-test suite; pages are
  unchanged React, so their tests stand).
- **Manual QA of critical flows:** login, logout, session-expiry (401→surface),
  OAuth/OIDC callback, 2FA setup, password reset, deep-link to a protected route
  (→signin→back), workspace switch, SQL-editor entry.
- History/back-button parity: `createBrowserRouter` ↔ the old `createWebHistory`.

## Risks & mitigations
- **Guard** (highest risk): verbatim port + unit tests + auth-flow QA.
- **pev2 React rebuild:** the only net-new UI; isolated commit, screenshot-compare.
- **79-file sweep:** mechanical; caught by type-check + existing page tests.
- **Two-stage rollback safety:** built in internal order so each commit type-checks.

## Out of scope
- `projectIamPolicy` Pinia store: its bridge stays until the permission/workspace
  layer migrates (separate, pre-existing).
- Any page-level UI/behavior change — this is a host swap only.
