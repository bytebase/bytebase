# Remove naive-ui from the frontend

**Date:** 2026-05-25
**Scope:** Single PR
**Owner:** TBD

## Context

The frontend is mid-migration from Vue to React. React UI is built on Base UI
(`@base-ui/react`) wrapped in shadcn-style primitives under
`frontend/src/react/components/ui/`. The Vue surface still imports `naive-ui`
(v2.44.1) as its component library, but the actual usage has shrunk
dramatically as pages have moved to React.

A cascade audit shows naive-ui usage is now confined to:

- 11 files under `frontend/src/` that directly import from `naive-ui`
- 1 root-level theme config: `frontend/naive-ui.config.ts`
- 3 lines of CSS overrides in `frontend/src/assets/css/tailwind.css`
- 3 package-json deps: `naive-ui`, `vdirs`, `vueuc`

Of the 11 direct importers, the live downstream cascade is exactly
**three files**: `App.vue`, `AuthContext.vue`, and `main.ts`. Everything else
is unused, type-only, or already decoupled from naive-ui's runtime.

The React side has zero naive-ui imports — it only consumes a handful of
type-only re-exports and a pure utility function from files that *happen* to
also export naive-ui-flavored types.

## Goal

Remove `naive-ui` and its peer deps (`vdirs`, `vueuc`) from `package.json`
and delete or decouple every file that imports them. The app must still
build, type-check, lint, test, and run after the change.

## Non-goals

- **Not migrating router, app shell, or Pinia stores.** vue-router, the
  Vue layout shells (`BodyLayout.vue`, `DashboardLayout.vue`,
  `SplashLayout.vue`), Pinia, and vue-i18n stay. This PR strictly removes
  naive-ui; the larger Vue → React extraction is out of scope.
- **Not introducing a replacement Vue component library.** The eight
  Vue files that survive after naive-ui removal are pure layouts and
  React-mount bridges; none of them needs UI primitives.
- **Not touching React UI.** The React Base UI stack is already canonical.

## Approach

Seven sequential steps inside a single branch. Each step leaves the build
green so we can stop and verify mid-way if needed.

### Step 1 — Decouple types from naive-ui

Three files re-export types that extend naive-ui types. Replace the
`extends` with standalone interfaces containing only the fields actually
consumed.

- `frontend/src/types/v2-shared.ts` — `ResourceSelectOption<T>` currently
  extends `SelectOption` from naive-ui. Rewrite as a standalone interface.
  Three React utility files consume this type:
  - `frontend/src/react/components/CustomApproval/utils.ts`
  - `frontend/src/react/lib/sensitive-data/components-utils.ts`
  - `frontend/src/react/lib/database-group/utils.ts`
- `frontend/src/types/sqlEditor/tree.ts` — drop `extends TreeOption`;
  inline the fields the tree code uses.
- `frontend/src/utils/v1/databaseResource.ts` — `DatabaseTreeOption`
  drops its naive-ui `TreeOption` extension. The lone React consumer
  (`RequestQueryButton.tsx`) only uses `parseStringToResource`, not the
  type, so this is safe.

### Step 2 — Delete dead wrappers

These files have zero consumers in the codebase:

- `frontend/src/bbkit/BBAttention.vue`
- `frontend/src/bbkit/BBButtonConfirm.vue`
- `frontend/src/bbkit/BBTextField.vue`
- `frontend/src/bbkit/BBAlert.vue`
- `frontend/src/utils/naive-ui.ts` (`useAutoHeightDataTable` composable)

Update `frontend/src/bbkit/index.ts` to drop the corresponding re-exports.

### Step 3 — Migrate the single live BBSpin consumer

`frontend/src/AuthContext.vue` is the only file that imports `BBSpin` from
`@/bbkit`. Inline a CSS-only spinner in `AuthContext.vue` (no Vue
component, no external dep — a div with a CSS rotation animation),
then delete `frontend/src/bbkit/BBSpin.vue` and the corresponding
re-export in `frontend/src/bbkit/index.ts`.

### Step 4 — Strip the global naive-ui wiring

- `frontend/src/App.vue` — remove the `<NConfigProvider>` wrapper. Theme
  tokens are already CSS custom properties (`--color-accent`, `--color-error`,
  etc.) defined in `tailwind.css`, so removing the provider is a no-op for
  visual rendering.
- `frontend/src/main.ts` — remove the `app.use(NaiveUI)` registration call
  and the corresponding import.
- Delete `frontend/src/plugins/naive-ui.ts` (the 29-component global
  registration plugin and style-tag re-append logic).
- Delete `frontend/naive-ui.config.ts` (theme overrides + zhCN locale
  config). Before deletion, confirm that any locale strings worth keeping
  already exist in `frontend/src/locales/zh-CN.json` — the `dateZhCN`
  binding from naive-ui isn't a translation, it's date-formatting wiring
  for naive-ui's own date components, so nothing to port.

### Step 5 — Clean up CSS overrides

In `frontend/src/assets/css/tailwind.css` (around lines 421–426), remove
the focus-ring resets targeting `.n-base-selection-input`,
`.n-base-selection-input-tag__input`, and `.n-input__input-el`. These
selectors become dead once naive-ui is gone.

### Step 6 — Drop dependencies

```
pnpm --dir frontend remove naive-ui vdirs vueuc
```

Then update `frontend/vite.config.ts`: remove `naive-ui` from the
`ui-framework` entry of `build.rollupOptions.output.manualChunks` (the
chunk can remain for other UI libs in that group, or collapse if naive-ui
was its only member — confirm during implementation).

### Step 7 — Verify

Run, in order:

1. `pnpm --dir frontend fix` — ESLint + Biome auto-fix + i18n sort
2. `pnpm --dir frontend check` — ESLint + Biome CI + react-i18n check +
   react-layering scanner + i18n sort check
3. `pnpm --dir frontend type-check` — vue-tsc (still covers remaining
   Vue files) + tsc (React)
4. `pnpm --dir frontend test`
5. `pnpm --dir frontend dev` — manual smoke test of the auth flow, since
   `AuthContext.vue` is the only file with a meaningful behavioral
   change (the spinner)

## Risks

1. **Hidden auto-registered usage.** The audit found no
   `unplugin-vue-components` naive-ui resolver, and all naive-ui
   components are explicitly registered in `plugins/naive-ui.ts`. If a
   template somewhere uses `<n-foo>` without an import, type-check
   won't catch it (Vue templates are loosely typed for global
   components). Mitigation: after dropping the deps, run `grep -r '<n-'
   frontend/src/` to catch any stragglers.

2. **Theme regression in remaining Vue surfaces.** `NConfigProvider`
   injected naive-ui's theme. The eight surviving Vue files
   (`BodyLayout.vue`, `DashboardLayout.vue`, `SplashLayout.vue`, and
   five React-mount bridges) don't render naive-ui components, so this
   should be a no-op. Confirm visually in the smoke test.

3. **vite chunking change.** Removing naive-ui from `manualChunks`
   slightly reshuffles the bundle. This is cosmetic — the chunk just
   becomes smaller. If the `ui-framework` entry had naive-ui as its only
   member, drop the entry entirely.

4. **dateZhCN behavioral change.** `naive-ui.config.ts` wires `dateZhCN`
   into NConfigProvider, which only affects naive-ui's own
   date-picker / calendar components. Since none of those are rendered
   in the surviving Vue surfaces, removal is a no-op. No app-level i18n
   change is needed.

## Verification checklist

Before opening the PR:

- [ ] `grep -r 'naive-ui' frontend/src/` returns zero results
- [ ] `grep -r 'from "vdirs"' frontend/src/` and `from "vueuc"` return zero
- [ ] `grep -r '<n-' frontend/src/` returns zero (template auto-reg check)
- [ ] `pnpm --dir frontend check` passes
- [ ] `pnpm --dir frontend type-check` passes
- [ ] `pnpm --dir frontend test` passes
- [ ] `pnpm --dir frontend dev` boots; signin → dashboard flow works;
      `AuthContext` spinner shows during auth bootstrap
- [ ] `package.json` no longer lists `naive-ui`, `vdirs`, or `vueuc`
- [ ] `pnpm-lock.yaml` updated

## Out-of-band follow-ups (not in this PR)

- The eight surviving Vue files (`App.vue`, `AuthContext.vue`,
  `BodyLayout.vue`, `DashboardLayout.vue`, `SplashLayout.vue`, and four
  React-mount bridges) remain. Phase B of the broader Vue → React
  migration (router swap, shell port, Pinia → Zustand) covers their
  eventual removal.
- The `bbkit/` directory will be left with `BBUtil.ts` (a pure
  `hashCode()` utility) and `index.ts`. Consider moving `hashCode` to
  a shared utility location in a follow-up so `bbkit/` can be deleted
  entirely.
