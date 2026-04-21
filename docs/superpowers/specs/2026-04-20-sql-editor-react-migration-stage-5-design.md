# SQL Editor React Migration — Stage 5 Design

**Date:** 2026-04-20
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** Migrate the entire `<NButtonGroup>` of 3 connection choosers in `EditorAction.vue` to React as one unit: `DatabaseChooser`, `SchemaChooser`, `ContainerChooser`, plus their shared private primitive `ConnectChooser`. Single React mount preserves the visual button-group behavior; all 4 Vue files become deletable in this stage.

**Non-goals (Stage 5):**
- Migrating `EditorAction.vue` itself — it stays Vue; just one of its child slots is swapped.
- Migrating `OpenAIButton`, `QueryContextSettingPopover`, `SharePopover`, `SaveSheetModal`.
- Building Vue-in-React mount infrastructure.
- Bridging the Emittery `events` bus.
- Porting the custom `SchemaIcon.vue` and `DatabaseIcon.vue` Vue icons — use `lucide-react` substitutes (`Network` for SchemaIcon, `Database` for DatabaseIcon).

## 2. Architecture

`EditorAction.vue:125-129`'s `<NButtonGroup>` block:

```vue
<NButtonGroup>
  <DatabaseChooser />
  <SchemaChooser />
  <ContainerChooser />
</NButtonGroup>
```

Becomes a single React mount:

```vue
<ReactPageMount page="ChooserGroup" />
```

The React `ChooserGroup` component owns the visual grouping (flex + connected-border styling matching NButtonGroup's appearance) and renders 3 React children: `DatabaseChooser`, `SchemaChooser`, `ContainerChooser`. The shared `ConnectChooser` primitive is the styled chooser-button-with-popover used by SchemaChooser and ContainerChooser.

## 3. New React files

| File | Lines | Purpose |
|---|---|---|
| `react/components/sql-editor/ConnectChooser.tsx` | ~60 | Shared chooser primitive. Shadcn `Combobox` wrapped in a styled trigger button with `Network` icon. Props: `value`, `onChange`, `options`, `placeholder`, `isChosen`. Replaces NPopselect+NButton. Base UI's positioner handles auto-flip and viewport-aware sizing — drops the manual `useElementBounding` math. |
| `react/components/sql-editor/SchemaChooser.tsx` | ~50 | Wraps ConnectChooser. Reads `useConnectionOfCurrentSQLEditorTab` for instance/database, `useDBSchemaV1Store` for metadata, `useSQLEditorTabStore` for current tab. Computes `show = instanceAllowsSchemaScopedQuery(engine)`, options from `databaseMetadata.schemas`, value from `tab.connection.schema` with `"-1"` sentinel for unspecified. Watches `router.currentRoute.value.query.schema` to seed via `useEffect`. |
| `react/components/sql-editor/ContainerChooser.tsx` | ~55 | Same pattern as SchemaChooser but for tables. `show = engine === COSMOSDB`, options from `databaseMetadata.schemas[].tables`, value from `tab.connection.table`. Watches `router.currentRoute.value.query.table`. |
| `react/components/sql-editor/DatabaseChooser.tsx` | ~80 | The breadcrumb-style chooser. Reads multiple stores. Renders Environment > Instance > Database via React `EnvironmentLabel` (already exists) + `<img>` with `EngineIconPath` (existing pattern from TransferProjectSheet). Click sets `useSQLEditorUIStore().showConnectionPanel = true`. Has a tooltip popover for batch-mode indicator. Uses lucide `SquareStack` for batch icon, `ChevronRight` for separators, `Database` for the database icon. |
| `react/components/sql-editor/ChooserGroup.tsx` | ~30 | Flex wrapper with connected-border styling. Renders the 3 React choosers in sequence. The mount entry point referenced by `ReactPageMount`. |

Each gets a `*.test.tsx` file with focused vitest cases (3-6 tests per).

## 4. Vue caller swap

`EditorAction.vue:125-129` — replace the `<NButtonGroup>` block (5 lines) with `<ReactPageMount page="ChooserGroup" />` (1 line).

Imports to update at the top of `EditorAction.vue`:
- Remove `DatabaseChooser`, `SchemaChooser`, `ContainerChooser` Vue imports
- Remove `NButtonGroup` from the `naive-ui` named import (verify it's not used elsewhere — visual inspection of `EditorAction.vue` shows another `<NButtonGroup>` at line 23 wrapping the run-query button + popover; KEEP `NButtonGroup` in the import).
- `ReactPageMount` is already imported (added in Stage 4).

## 5. Vue file deletions

After the swap, these become orphaned (verified single-caller in audit):
- `EditorCommon/DatabaseChooser.vue`
- `EditorCommon/SchemaChooser.vue`
- `EditorCommon/ContainerChooser.vue`
- `EditorCommon/ConnectChooser.vue`

`rg`-verify zero remaining callers before deletion. Expected: only the React replacement files match.

## 6. i18n keys

Verify these in React locales; add any missing using values from Vue locales:
- `database.schema.select` (SchemaChooser placeholder)
- `database.schema.unspecified` (Schema/Container "all schemas" option)
- `db.schema.default` (default-schema fallback label)
- `database.table.select` (ContainerChooser placeholder)

DatabaseChooser also uses:
- `sql-editor.batch-query.batch` (batch-mode tooltip)
- `sql-editor.select-a-database-to-start` (placeholder when disconnected)

## 7. Verification

- `pnpm fix && check && type-check && test` all green
- New tests: ~20 across 5 component test files
- Manual UX: open SQL Editor, verify the 3-button toolbar group renders identically (visual inspection vs. Vue), schema/container choosers gate correctly per engine, database chooser breadcrumb renders correctly, click on database chooser opens connection panel, schema/table selection persists to tab connection state, route-query seeding still works.

## 8. Visual styling notes

- The Vue `NButtonGroup` produces a row of buttons with shared/joined borders. The React `ChooserGroup` should replicate this with `flex` + `[&>*:not(:first-child)]:border-l-0` + `[&>*:first-child]:rounded-r-none` etc., OR use a simpler `flex gap-0` and have each chooser button render with `rounded-none border-r-0` except the last. Pick whichever is cleaner during implementation.
- The Vue chooser button uses `type="primary" ghost` styling (accent border, accent text, transparent fill) similar to ConnectionHolder — reuse the same className pattern: `border-accent text-accent hover:bg-accent/5`.
- DatabaseChooser's "select a database" disconnected state has different styling — verify visually.

## 9. Out of scope (deferred)

- `SchemaIcon.vue` port to React — used by 6 callers, out of scope for this stage. Use lucide `Network` substitute.
- `DatabaseIcon.vue` port — used by ~20 callers. Use lucide `Database`.
- Migrating `EditorAction.vue` itself.
- Vue-in-React mount infrastructure.
- Emittery events bridge.

## 10. Practical checklist

- [ ] React `ConnectChooser.tsx` + test
- [ ] React `SchemaChooser.tsx` + test
- [ ] React `ContainerChooser.tsx` + test
- [ ] React `DatabaseChooser.tsx` + test
- [ ] React `ChooserGroup.tsx` + test (or smoke test only)
- [ ] i18n keys verified/added in 5 React locales
- [ ] `EditorAction.vue:125-129` swapped to `<ReactPageMount page="ChooserGroup" />`; Vue chooser imports removed
- [ ] 4 Vue files deleted after `rg` confirms no remaining callers
- [ ] `pnpm fix && check && type-check && test` all pass
- [ ] Manual UX parity verified
