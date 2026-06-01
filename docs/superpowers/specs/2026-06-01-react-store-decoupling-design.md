# React Store Decoupling — Design

Date: 2026-06-01
Status: Approved (design)

## Background

The Vue→React migration's view layer is effectively complete: 722 `.tsx` files,
0 `naive-ui` imports, and all 86 dashboard/auth/setup/sqlEditor routes mount React
through bridge components. The remaining Vue footprint is the runtime substrate —
the Vue Router config, the app shell (`App.vue`, `AuthContext.vue`, 3 layouts, 5
mount bridges), and the Pinia store layer (~40 modules, ~7,300 lines using Vue
reactivity).

The end goal is to **fully delete Vue**. Routing is the keystone that collapses
the 10 remaining `.vue` files, but it is high-risk. The lower-risk first track is
the **store layer**, and that work is already ~80% done on the React side: a
Zustand `useAppStore` (`src/react/stores/app/index.ts:68`) is live and consumed by
168 React files across ~26 slices (auth, project, instance, database, user, role,
workspace, iam, etc.).

What remains is a concentrated tail of React code still reaching back into Pinia.

## Goal

Sever React's remaining dependency on Pinia / Vue reactivity. Concretely:

- `frontend/src/react/**` imports **zero** Pinia store modules (`@/store/...`).
- `frontend/src/react/**` has **zero** `useVueState` reads that resolve to a Pinia
  store.

## Scope

### In scope

The ~12 Pinia stores still consumed from `react/` (~68 call sites). Counts are
`useXxxStore()` call sites within `react/` at design time:

| Pinia store | React call sites | Zustand slice today |
|---|---|---|
| `useDBGroupStore` | 19 | partial (`app/dbGroup.ts`) |
| `useAuthStore` | 18 | exists (`app/auth.ts`) |
| `useSQLReviewStore` | 9 | none |
| `useUIStateStore` | 7 | none |
| `useIssueCommentStore` | 4 | none |
| `useWorkspaceApprovalSettingStore` | 3 | none |
| `useStorageStore` | 2 | none |
| `useProjectIamPolicyStore` | 2 | maybe (`app/iam.ts`) |
| `useSQLStore` | 1 | exists (`app/sql.ts`) |
| `useSQLEditorStore` | 1 | exists (`react/stores/sqlEditor`) |
| `useRolloutStore` | 1 | none |
| `usePlanStore` | 1 | none |

Even stores with an existing slice need an **API-parity step** — the slices are
partial. Example: the dbGroup slice has `fetchDBGroup` / `listDBGroupsForProject`,
but the 19 consumers call `getOrFetchDBGroupByName` (11), `getDBGroupByName` (8),
`create/update/deleteDatabaseGroup`, and `fetchDatabaseGroupMatchList` — none of
which exist on the slice yet.

### Explicitly NOT in scope (deferred to the later routing/shell phase)

- **Deleting the Pinia store files.** They keep their Vue-side consumers (router,
  the 10 `.vue` files, plugins, and 22 inter-store imports). They are deleted when
  routing flips to React, not in this increment.
- Vue Router, `App.vue`, `AuthContext.vue`, layouts, mount bridges.
- Migrating store internals that have **no** React consumers (e.g. `setting`,
  `subscription`, `environment`) — untouched until routing.

## Approach

**Store-by-store, one PR per store** (chosen over a single mega-PR and over a
file-by-file `useVueState` sweep). Each PR is independently reviewable and
verifiable, ships a fully-decoupled store, and front-loads the highest-value cases.

### Per-store recipe (applied in each PR)

1. **Parity.** Ensure the Zustand slice in `react/stores/app/<slice>.ts` exposes
   every method the React consumers actually call. Newly-ported stores
   (`sqlReview`, `uistate`, `issueComment`, `workspaceApprovalSetting`, `storage`,
   `rollout`, `plan`) are added as **new slices inside `createAppStore`**,
   consistent with the existing 26 slices. Port logic from the Pinia method
   verbatim and reuse the same Connect client calls.
2. **Cut consumers.** Replace `useVueState(() => useXStore().foo)` reactive reads
   with Zustand selectors (`useAppStore(s => s.foo)`); replace plain
   `useXStore().method()` calls with `useAppStore.getState().method()` or a
   selector. Remove the now-dead `useVueState` wrappers.
3. **Verify.** Confirm `react/` no longer imports that Pinia store, then run the
   full frontend verification suite.

### PR order

1. `dbGroup` (19) — partial slice; validates the parity+cutover recipe at high value.
2. `auth` (18) — existing slice; high count.
3. `sqlReview` (9)
4. `uistate` (7)
5. `issueComment` (4)
6. `workspaceApprovalSetting` (3)
7. `storage` (2)
8. `projectIamPolicy` (2)
9. `sql` (1)
10. `sqlEditor` (1)
11. `rollout` (1)
12. `plan` (1)
13. **Guard PR** — add `frontend/scripts/check-react-no-pinia.mjs`, wired into
    `pnpm --dir frontend check`, matching the `check-react-i18n.mjs` /
    `check-react-layering.mjs` convention. It fails if anything under `react/**`
    imports a `@/store` Pinia module or calls a Pinia `useXxxStore()`. This locks
    the gain so the decoupling cannot regress.

## Verification

Per the migration playbook, every PR runs:

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test`

Parity ports add focused tests where the Pinia store had non-trivial behavior worth
pinning (e.g. dbGroup's cache-with-fallback logic). The guard script lands last and
makes the zero-Pinia-in-React invariant enforceable in CI.

## Risks & mitigations

- **Partial-parity surprises.** A slice may silently differ from the Pinia method
  (caching, in-place mutation, error handling). Mitigation: port verbatim and add a
  focused test before cutting consumers.
- **Reactivity semantics drift.** `useVueState` surfaces in-place proxy mutations;
  Zustand selectors compare by reference. Ports must produce new references on
  update (immutable updates) so selectors re-render correctly.
- **Hidden Vue consumers.** A method removed from Pinia could still be called by
  Vue code. Mitigation: parity step **adds** to the slice and leaves the Pinia
  store intact; nothing is removed from Pinia in this increment.

## Out-of-increment follow-up

After all 13 PRs, React is Pinia-free. The Pinia layer then has only Vue-side
consumers and is deleted as part of the subsequent routing/app-shell migration —
the keystone that also removes the 10 remaining `.vue` files.
