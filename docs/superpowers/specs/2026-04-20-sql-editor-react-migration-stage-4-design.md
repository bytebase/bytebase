# SQL Editor React Migration — Stage 4 Design

**Date:** 2026-04-20
**Author:** d@bytebase.com
**Status:** Draft

## 1. Goal & non-goals

**Goal:** (a) Delete two dead Vue files (cleanup); (b) migrate `EditorCommon/AdminModeButton.vue` to React as a single-leaf spike to learn what blockers exist before committing to a multi-stage `EditorAction.vue` migration roadmap.

**Non-goals (Stage 4):**
- Migrating `EditorAction.vue`, `OpenAIButton`, `DatabaseChooser`, `SchemaChooser`, `ContainerChooser`, `QueryContextSettingPopover`, `SharePopover`, or any other EditorAction child.
- Building Vue-in-React mount infrastructure.
- Bridging the Emittery `events` bus.
- Bridging the AI plugin context.
- Decision on Q1 (top-down) vs Q2 (cascade-up) for Stage 5+ — that decision is informed by what we learn from this spike.

## 2. Cleanup tasks

- **Delete** `frontend/src/views/sql-editor/EditorCommon/DisconnectedIcon.vue` (3 lines, zero live callers — only present in the barrel).
- **Delete** `frontend/src/views/sql-editor/EditorCommon/ReadonlyDatasourceHint.vue` (66 lines, zero callers anywhere).
- **Update** `frontend/src/views/sql-editor/EditorCommon/index.ts` — remove the `DisconnectedIcon` import and named export.

Pre-deletion verification: `rg "DisconnectedIcon|ReadonlyDatasourceHint" frontend/src/` confirms zero live consumers (already verified — only the barrel and the files themselves).

## 3. Spike: migrate `AdminModeButton`

### 3.1 New React files

| File | Purpose |
|---|---|
| `frontend/src/react/components/sql-editor/AdminModeButton.tsx` | Function component. Props: `size?: "sm" \| "default"` (defaults to "default"; "sm" maps to shadcn Button's `size="sm"`) and `hideText?: boolean` (default false). Reads `allowAdmin`, `currentTab.mode`, and `isDisconnected` via `useVueState` from Pinia. Writes `tabStore.updateCurrentTab({ mode: "ADMIN" })` on click. Uses `WrenchIcon` from `lucide-react`. Renders nothing when `showButton` is false. |
| `frontend/src/react/components/sql-editor/AdminModeButton.test.tsx` | 4 vitest cases: hidden when `!allowAdmin`; hidden when current tab mode is not WORKSHEET; disabled when disconnected; click writes `mode: "ADMIN"`. |

### 3.2 Vue caller swap

`EditorAction.vue:54–65` currently:

```vue
<NPopover placement="bottom">
  <template #trigger>
    <AdminModeButton
      size="small"
      :hide-text="true"
      style="--n-padding: 0 5px"
    />
  </template>
  <template #default>
    <span>{{ $t("sql-editor.admin-mode.self") }}</span>
  </template>
</NPopover>
```

Swap to:

```vue
<ReactPageMount page="AdminModeButton" size="sm" :hideText="true" />
```

The Vue `<NPopover>` is removed; the React `AdminModeButton` will internally wrap its button with the React `Tooltip` primitive (label = `t("sql-editor.admin-mode.self")`) so the tooltip-on-hover behavior is preserved within the React leaf.

Drop the `import AdminModeButton from "./AdminModeButton.vue";` import from `EditorAction.vue`. Add `import ReactPageMount from "@/react/ReactPageMount.vue";` if not already present (it isn't yet for this file).

### 3.3 Vue file deletion — DEFERRED (spike learning)

Original assumption: `AdminModeButton.vue` had only one caller (`EditorAction.vue`), so the Vue file would be deletable after the swap. **Reality discovered during implementation:** the Vue component has 3 callers, not 1:

1. `EditorCommon/EditorAction.vue` — migrated to React in this stage.
2. `EditorCommon/ExecuteHint.vue:25,36` — uses `<AdminModeButton @enter="$emit('close')" />` (Vue caller, uses the `enter` emit which the React port doesn't yet expose).
3. `EditorPanel/ReadonlyModeNotSupported.vue:13` — `<AdminModeButton />` zero-prop usage (Vue caller).

The Vue `AdminModeButton.vue` is RETAINED for these 2 remaining Vue callers. The React version co-exists. This is the expected bottom-up migration pattern: shared leaves stay Vue until ALL callers are React, then the Vue version is deleted.

**Key spike learning:** Stage 5+ planning must `rg`-verify ALL callers of any candidate file BEFORE assuming it's a clean leaf migration. Multi-caller Vue children aren't deletable in a single stage.

### 3.4 i18n

`sql-editor.admin-mode.self` — verify in React locales. Add to all 5 locales if missing, using the Vue locale values byte-exact. If `admin-mode.exit` and other admin-mode keys are also unused in React but referenced by the broader EditorAction (still Vue), defer those — only this stage's React consumer matters now.

## 4. Verification

### 4.1 Per-leaf

- `pnpm --dir frontend fix && check && type-check && test` — all green.
- 4 new tests on `AdminModeButton.test.tsx`, all pass.
- React i18n consistency check passes (no missing/unused for `sql-editor.admin-mode.self`).

### 4.2 Manual UX (in dev server)

1. Open SQL Editor as a user with `allowAdmin` permission. In WORKSHEET mode, the admin-mode button (wrench icon) is visible in the action toolbar. In ADMIN mode (or any non-WORKSHEET mode), the button is hidden.
2. Open as a user without `allowAdmin` → button hidden in all modes.
3. Hover the button → tooltip "Admin mode" appears (label from `sql-editor.admin-mode.self`).
4. Click the button → tab switches to ADMIN mode (UI changes immediately).
5. When connection is lost (`isDisconnected=true`), the button renders disabled (visually muted, no click effect).

### 4.3 Side-by-side visual parity

The naive-ui `NButton type="warning" ghost` renders with a warning-orange ghost outline. The React shadcn `Button` lacks a "warning" variant out of the box. The React migration will use `variant="outline"` with a `className` override of `border-warning text-warning hover:bg-warning/5` for visual parity. Confirm visually during manual UX.

## 5. Out of scope (deferred)

- All other `EditorAction.vue` children (per §1 non-goals).
- Vue-in-React mount infrastructure (the spike doesn't need it).
- Emittery events bridge.
- AI plugin context bridge.

## 6. Stage 5 informed by this spike

After Stage 4, we'll have one Vue child of `EditorAction.vue` migrated. We use that to evaluate:
- Did visual parity feel right (NPopover → Tooltip)?
- Was the prop bridge fine for `size` and `hideText`?
- Did any unexpected dependency surface?
- Is there appetite for migrating each remaining child the same way (cascade-up, Q2), or would Q1's top-down approach now feel justified?

Stage 5 brainstorm picks the next leaf or the EditorAction shell itself with this information in hand.

## 7. Practical checklist

- [ ] `frontend/src/views/sql-editor/EditorCommon/DisconnectedIcon.vue` deleted.
- [ ] `frontend/src/views/sql-editor/EditorCommon/ReadonlyDatasourceHint.vue` deleted.
- [ ] `frontend/src/views/sql-editor/EditorCommon/index.ts` — `DisconnectedIcon` import + export removed.
- [ ] `sql-editor.admin-mode.self` added to React locales (if missing).
- [ ] `react/components/sql-editor/AdminModeButton.tsx` + 4 tests created.
- [ ] `EditorAction.vue` swapped: `<NPopover>...<AdminModeButton>...</NPopover>` → `<ReactPageMount page="AdminModeButton" size="sm" :hideText="true" />`. `AdminModeButton` import removed; `ReactPageMount` import added.
- [ ] `EditorCommon/AdminModeButton.vue` deleted after `rg` confirms zero remaining callers.
- [ ] `pnpm fix && check && type-check && test` all pass.
- [ ] Manual UX verified for the 5 states above.
