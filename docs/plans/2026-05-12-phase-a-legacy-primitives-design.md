# Phase A — Strip Legacy Primitives: Design

**Date:** 2026-05-12
**Parent doc:** [2026-05-12-react-migration-status-and-plan.md](./2026-05-12-react-migration-status-and-plan.md)
**Playbook:** [2026-04-08-react-migration-playbook.md](./2026-04-08-react-migration-playbook.md)

---

## Goal

**No `.tsx` file imports any `.vue` file.**

After Phase A, every remaining `.vue` file in `frontend/src/components/` is imported only by other `.vue` files. The Vue and React layers are cleanly separated. Phase B (app shell, router, state) can then delete whole subtrees of Vue at a time without cross-framework concerns.

## Strategy: defer-until-callers-die

Each `.vue` file in `frontend/src/components/` is classified by who imports it:

| Bucket | Vue callers | React callers | Phase A action |
|---|---|---|---|
| **DEAD** | 0 | 0 | Delete. No migration. |
| **REACT-ONLY** | 0 | ≥1 | Swap React callers to a React equivalent; delete the Vue file in the same change. |
| **MIXED** | ≥1 | ≥1 | Out of scope — handled opportunistically. The Vue file stays for its Vue callers. |
| **VUE-ONLY** | ≥1 | 0 | Out of scope — defer to Phase B (will retire with its Vue callers). |

Phase A acts on DEAD and REACT-ONLY only. MIXED files become eligible the moment their React callers drop to zero — those are handled as one-off cleanup PRs as engineers touch the area, not as part of Phase A.

## Scope: one PR

A single PR — "Phase A: drop React→Vue cross-framework imports" — covering:

### 1. Dead-code deletion (4 files)

`git rm` only, no replacements:

- `frontend/src/components/RequiredStar.vue`
- `frontend/src/components/EditEnvironmentDrawer.vue`
- `frontend/src/components/Permission/NoPermissionPlaceholder.vue`
- `frontend/src/components/misc/MaskSpinner.vue`

### 2. REACT-ONLY swaps (~20 files)

For each, update every React caller to import the existing React equivalent (or remove the obsolete re-export), then delete the Vue file. All replacements already exist in `frontend/src/react/components/` or `frontend/src/react/components/ui/`.

| Vue file(s) to delete | React callers | Replacement |
|---|---|---|
| `EllipsisText.vue` | 13 | `react/components/ui/ellipsis-text.tsx` |
| `FeatureGuard/FeatureAttention.vue` | 20 | `react/components/FeatureAttention.tsx` |
| `AdvancedSearch/*` (5 files) | 21 (re-exports) | `react/components/AdvancedSearch.tsx` |
| `DatabaseInfo.vue` | 4 | React component / inline call sites |
| `Instance/InstanceSyncButton.vue` | 3 | `react/components/instance/InstanceSyncButton.tsx` |
| `DatabaseDetail/SyncDatabaseButton.vue` | 1 | React equivalent in `sql-editor/ResultView` |
| `RoleGrantPanel/MaxRowCountSelect.vue` | 2 | `react/components/MaxRowCountSelect.tsx` |
| `misc/SQLUploadButton.vue` | 2 | `react/components/sql-editor/StandardPanel/SQLUploadButton.tsx` |
| `v2/Container/*` (2 files) | 4 | Sheet / drawer primitives in `react/components/ui/` |
| `v2/TabFilter.vue` | 3 | React equivalent / re-export cleanup |
| `SQLReview/RuleConfigComponents/*` (5 files) | 2 (re-exports) | Re-export cleanup |

**Total: ~24 Vue files deleted.**

### 3. CI guard

Extend `no-legacy-vue-deps.test.ts` (or add a sibling test) to fail on any `.tsx` file that imports a `.vue` file. This locks in the win and prevents regression while MIXED files are picked off opportunistically.

The guard's allowlist starts empty — every MIXED file (`LearnMoreLink`, `FeatureBadge`, `FeatureModal`, `UserAvatar`, `MonacoEditor/*`, `Icon/*`, `v2/Button/*`, `v2/Form/*`, `v2/Select/*`, `v2/Model/*`, plus the long tail) needs an explicit entry until its React callers are migrated off. As each MIXED file becomes REACT-ONLY (or DEAD), its allowlist entry is removed and the file deleted.

## Out of scope for Phase A

These stay; each is its own future PR or waits for Phase B:

**MIXED — opportunistic cleanup PRs** (one per primitive when an engineer touches the area):

- `LearnMoreLink.vue` (17 React callers)
- `FeatureGuard/FeatureModal.vue` (12), `FeatureBadge.vue` (29)
- `UserAvatar.vue` (9)
- `FileContentPreviewModal.vue` (2), `HighlightCodeBlock.vue` (1)
- `MonacoEditor/*` (21 — many likely re-exports; verify when picked up)
- `Icon/*` (81 — likely many re-exports; verify when picked up)
- `v2/Button/*` (232 — count includes transitive matches; needs filtering)
- `v2/Form/*` (44), `v2/Select/*` (85), `v2/Model/*` (4)
- Long tail: `PermissionGuardWrapper`, `InputWithTemplate`, `SpannerQueryPlan/*`, `ErrorList`, `YouTag`

**VUE-ONLY — defer to Phase B** (no React callers; retires when its Vue caller dies):

- `ReleaseRemindModal.vue` (called from `BodyLayout.vue`)
- `misc/OverlayStackManager.vue` (called from `App.vue`, `BBModal.vue`)
- `misc/AccountTag.vue`
- `User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue`
- `Member/MemberDataTable/cells/RoleCell.vue`, `UserRolesCell.vue`
- `InputWithTemplate/AutoWidthInput.vue`

## Validation

Per the playbook:

1. `pnpm --dir frontend fix`
2. `pnpm --dir frontend check`
3. `pnpm --dir frontend type-check`
4. `pnpm --dir frontend test` — includes the new `.tsx`→`.vue` guard
5. Manual smoke: open every page whose React caller list was touched (settings pages, project pages, instance detail) and verify nothing renders broken.

## Done when

- `rg -l '\.vue["\x27]' frontend/src/**/*.tsx` returns only the MIXED files listed in the guard's allowlist.
- The four DEAD files are gone.
- The ~20 REACT-ONLY files are gone and their callers reference the React replacement.
- The CI guard exists and passes.

## Rough cost

One PR, touching ~24 `.vue` deletions + ~80 React caller updates (mostly mechanical import path swaps) + 1 test file. Estimated 1–2 days of focused work, including manual smoke verification.
