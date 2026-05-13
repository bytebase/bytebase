# Vue → React Migration: Status & Remaining Plan

**Date:** 2026-05-12
**Goal:** Full Vue removal — every `.vue` file deleted, `pnpm remove vue vue-router pinia` ships, single-framework codebase.
**Companion doc:** [2026-04-08-react-migration-playbook.md](./2026-04-08-react-migration-playbook.md) (process rules, deletion safety, state preferences).

---

## Part 1 — Current State

### Counts

- **`.tsx` files:** 681
- **`.vue` files:** 154 (outside `frontend/src/react/`)
- **React pages:** 167 (auth 11, project 100, settings 53, workspace 3)
- **Shared React UI primitives:** 46 in `frontend/src/react/components/ui/`
- **React SQL Editor surface:** 223 `.tsx` files under `frontend/src/react/components/sql-editor/`

### Routing — fully bridged

Every route file under `frontend/src/router/` already delegates to React via `ReactPageMount.vue` or `ReactRouteShellBridge.vue`:

- `auth.ts` — all auth/consent routes
- `setup.ts` — first-run setup
- `dashboard/index.ts`, `dashboard/workspace.ts`, `dashboard/instance.ts`, `dashboard/workspaceSetting.ts`, `dashboard/projectV1.ts` — every leaf
- `sqlEditor.ts` — parent is a one-line Vue render-only shim that mounts `<ReactPageMount page="SQLEditorLayout" />`; children are `NoopRouteComponent` (the React layout reads `useCurrentRoute()` and decides what to show). All panel content is React.

### Cross-layer bridges

- `frontend/src/react/hooks/useVueState.ts` — React subscribes to Vue reactive state (Pinia stores, refs, computed) via `useSyncExternalStore`.
- `frontend/src/react/shell-bridge.ts` — custom events for `bb.react-locale-change`, `bb.react-notification`, `bb.react-quickstart-reset`.
- `frontend/src/react/router/` — `useCurrentRoute()` / `useNavigate()` wrap vue-router for React consumers.

### What's migrated

| Area | Status |
|---|---|
| Auth (signin, signup, OAuth/OIDC, password reset, 2FA, profile setup, consent) | ✅ React |
| Workspace dashboard (Projects, Instances, Databases, Environments, MyIssues, 403/404) | ✅ React |
| Workspace settings (53 pages: members, roles, users, instances, environments, groups, IDPs, approvals, SQL review, semantic types, classifications, masking, risk, audit logs, subscription, general, profile, service accounts, workload identities, MCP) | ✅ React |
| Project pages (100 pages: issue detail, plan detail, release detail, database detail, changelog, revisions, data export, webhooks, audit logs, database groups, GitOps, masking exemptions, access grants) | ✅ React |
| **SQL Editor (all panels: editor, results, schema, connection, tabs, worksheets, history, diagram, terminal, access, masking, request drawers, save/upload, compact editor)** | **✅ React** |
| Shared UI primitives (46 Base UI wrappers: input, button, dialog, sheet, popover, dropdown, table, tabs, tree, select, combobox, switch, tooltip, …) | ✅ React |

### What's left

**1. `frontend/src/components/` — 154 `.vue` files:**

| Subdirectory | Files | Notes |
|---|---|---|
| `v2/` | 52 | Button, Container, Form, Model, Select, TabFilter — foundational primitives; React equivalents already exist in `react/components/ui/` |
| `Icon/` | 18 | Thin wrappers over icon libs |
| `misc/` | 6 | |
| `MonacoEditor/` | 5 | |
| `SQLReview/` | 5 | |
| `AdvancedSearch/` | 5 | |
| `FeatureGuard/` | 3 | |
| `User/`, `Member/`, `Permission/`, `InputWithTemplate/`, `SpannerQueryPlan/` | 2 each (10 total) | |
| `DatabaseDetail/`, `Instance/`, `RoleGrantPanel/` | 1 each | |
| Top-level singletons (`DatabaseInfo`, `SessionExpiredSurfaceMount`, `LearnMoreLink`, `ReleaseRemindModal`, `RequiredStar`, `HighlightCodeBlock`, `FileContentPreviewModal`, `EditEnvironmentDrawer`, `EllipsisText`, `AgentWindowMount`) | 10 | |

**2. App shell & framework:**

- `frontend/src/App.vue`, `AuthContext.vue`, `NotificationContext.vue`
- `frontend/src/layouts/BodyLayout.vue`, `DashboardLayout.vue`
- `frontend/src/mountSidebar.ts`, `mountProjectSidebar.ts`
- `frontend/src/router/` — vue-router driving the URL
- `frontend/src/store/` — Pinia stores (still the source of truth for some domains; read from React via `useVueState`)
- `frontend/src/main.ts`, `init.ts` — Vue bootstrap
- The one-line `SQLEditorLayoutComponent` Vue shim in `router/sqlEditor.ts` (retires when Vue Router is replaced)

**3. React-native stores already in place:** `frontend/src/react/stores/app/` contains 10 stores (auth, workspace, project, preferences, notification, iam, …). Future store migrations land here.

### Rough completion read

- **Routed page surface:** 100% React (only the SQL Editor parent route remains a thin Vue shim that mounts React)
- **Feature components:** ~78% React (~681 `.tsx` vs 154 `.vue`)
- **Foundation (shell + router + state):** 0% — still entirely Vue

---

## Part 2 — Migration Plan

Two phases, sequenced **user-value first**: visible improvements first, foundation last. (Previous "Phase A — finish SQL Editor" is complete and folded into Phase B's layout retirement.)

### Phase A — Strip legacy primitives & one-offs

**Strategy:** delete-driven. Each PR migrates a primitive *and* updates all its callers in the same change. No long-lived dual implementations. Most React replacements already exist in `react/components/ui/` (46 components) — this is largely find-and-replace.

**Order (cheap → load-bearing):**

1. **Top-level singletons (1–2 PRs)** — `RequiredStar`, `LearnMoreLink`, `EllipsisText`, `HighlightCodeBlock`, `FileContentPreviewModal`, `ReleaseRemindModal`, `DatabaseInfo`, `EditEnvironmentDrawer`. Defer `SessionExpiredSurfaceMount` and `AgentWindowMount` to Phase B (they are mount-bridges that disappear with the shell).
2. **`Icon/` (1 PR)** — 18 wrappers; bulk replace with React icon equivalents.
3. **Small feature dirs (5 PRs)** — `MonacoEditor/`, `SQLReview/`, `AdvancedSearch/`, `FeatureGuard/`, `misc/`. One PR per dir.
4. **`components/v2/` (6–7 PRs)** — Button, Container, Form, Model, Select, TabFilter. One PR per subdir; final cleanup PR deletes the `v2/` tree.
5. **Other small dirs (1–2 PRs)** — User, Permission, Member, Instance, InputWithTemplate, RoleGrantPanel, DatabaseDetail, SpannerQueryPlan.

**Done when:** `frontend/src/components/` contains only files that will be deleted alongside their layouts in Phase B.

**Estimated:** 10–15 PRs.

### Phase B — App shell, router, Vue extraction

The longest and riskiest phase. Each step forces a decision that the bridge has so far deferred.

#### B1. Router migration (vue-router → React Router DOM)

The hardest single piece. Introduce React Router DOM at the *root* and have it own the URL. Port routes from `frontend/src/router/{auth,setup,sqlEditor,dashboard/*}.ts` into React route trees.

The `useCurrentRoute()` / `useNavigate()` hooks already abstract route access — swap their *implementation* (vue-router refs → RR DOM hooks) without touching consumers.

**Risk:** param/query/hash semantics differ between vue-router and RR DOM (trailing slashes, optional params, `RouteLocationNormalized` shape). Plan for a careful dev-verification window before merge. Hard cutover; no feature flag (router cannot meaningfully dual-stack at runtime).

#### B2. App shell + layouts

Convert `App.vue`, `AuthContext.vue`, `NotificationContext.vue`, `layouts/BodyLayout.vue`, `layouts/DashboardLayout.vue`, `mountSidebar.ts`, `mountProjectSidebar.ts`, and `shell-bridge.ts` event consumers.

The shell currently *hosts* React pages; after this step React hosts everything. `useVueState`, `shell-bridge.ts`, and `ReactPageMount.vue` retire — once no reactive state is Vue-owned, the bridges have no consumers. `SessionExpiredSurfaceMount.vue` and `AgentWindowMount.vue` disappear here. The `SQLEditorLayoutComponent` Vue shim in `router/sqlEditor.ts` also retires.

#### B3. State migration (Pinia → React state)

Per the playbook, deferred until concrete problem. After B2, Pinia stores have no Vue component consumers — only React via `useVueState`. Options:

- **Keep Pinia** as a pure data layer. `createPinia()` works without Vue components.
- **Port store-by-store** to the existing pattern in `react/stores/app/` (already 10 stores in place).

Recommendation: port. Pinia stays a transitive dep of `vue` and blocks B4 otherwise. Use the same delete-driven pattern as Phase A — one store per PR, update all `useVueState` callers in the same change.

#### B4. Final extraction

- `pnpm remove vue vue-router pinia @vue/* vue-i18n vue-tsc @vitejs/plugin-vue`
- Delete `frontend/src/{App.vue,AuthContext.vue,NotificationContext.vue,init.ts,mount.ts,layouts,store,router}` and Vue shims (`shims-vue-*.d.ts`)
- Move `frontend/src/react/*` up one level (drop the `react/` namespace prefix)
- Switch Vite plugins: drop `@vitejs/plugin-vue`; promote `react-tsx-transform` to the standard React plugin
- Collapse `tsconfig.json` + `tsconfig.react.json` into one
- Retire `no-legacy-vue-deps.test.ts` enforcement
- Merge `frontend/src/react/locales/` into a single locales tree; drop `vue-i18n` callers
- Move framework-neutral `frontend/src/views/sql-editor/` utilities (events, hooks, types — no `.vue` files remain) to wherever they best fit in the unified layout

**Estimated Phase B:** 8–12 PRs, sequenced B1 → B2 → B3 → B4. B1 is the riskiest single PR in the entire migration.

---

## Part 3 — Cross-Cutting Rules

- **One PR, one replacement.** Every migration PR deletes the Vue file(s) it replaces. No dual-stack components.
- **Locales.** New strings land in `frontend/src/react/locales/`. Each phase opportunistically removes unused keys from `frontend/src/locales/`. Final merge happens in B4.
- **i18n compatibility.** `vue-i18n` and `react-i18next` stay parallel until B4.
- **State.** Stick with Pinia + `useVueState` until B3. Don't introduce new Zustand stores during Phase A unless a concrete problem demands it (playbook rule).
- **Shared UI.** Always check `frontend/src/react/components/ui/` before hand-rolling a control (AGENTS.md rule).
- **Composite-PK tests.** Backend tests are out of scope for this migration; no expected impact.
- **Testing.** Existing unit tests + manual QA per surface. No new E2E gate is proposed unless coverage gaps surface during Phase B1.
