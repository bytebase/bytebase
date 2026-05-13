# Notification System Migration — Vue → React (Base UI Toast)

**Date:** 2026-05-13
**Companion docs:**
- [2026-04-08-react-migration-playbook.md](./2026-04-08-react-migration-playbook.md)
- [2026-05-12-react-migration-status-and-plan.md](./2026-05-12-react-migration-status-and-plan.md)

## Goal

Replace the Vue-rendered notification system (`NotificationContext.vue` + Naive UI's `NNotificationProvider` + `useNotification()`) with a React-rendered one built on Base UI's Toast primitive. After this work, **React owns toast rendering for the entire app**; Vue retains only the `pushNotification()` API for backwards compatibility with its 17 remaining callers, which now route through a window event into the React renderer.

## Non-goals

- Migrate `App.vue` itself to React — that's Phase B2.
- Migrate the Pinia `notificationStore` to Zustand — that's Phase B3. The Pinia store stays as a thin pass-through during this PR.
- Add new toast variants (action toasts, promise toasts, etc.). Parity with the current Naive UI surface only.

## Current architecture

| Path | Renderer |
|---|---|
| Vue caller → `useNotificationStore().pushNotification(...)` → `watchEffect` in `NotificationContext.vue` → `useNotification().create()` | Naive UI |
| React caller → `useAppStore().notify(...)` (Zustand slice) → emits `window` `CustomEvent('bb.react-notification')` → `NotificationContext.vue` catches → `useNotification().create()` | Naive UI (via Vue bridge) |

149 total `pushNotification` consumers: 17 Vue, 132 React.

## Target architecture

```
Vue caller ──┐
             ▼
        pushNotification()  ─┐                                   ┌─► Base UI <Toaster />
                              ▼                                   │   (React tree)
                    Pinia notificationStore                       │
                              │ emit CustomEvent('bb.vue-notification')
                              ▼                                   │
                    window listener (react/lib/toast.ts)          │
                              │                                   │
                              ▼                                   │
                       toastManager.add()  ─────────────────────►─┘
                              ▲
                              │
React caller ──► notify() ────┘
```

Single renderer (React/Base UI). Vue callers keep their API; the Pinia store now publishes a window event instead of running its own render loop. React callers bypass the event entirely and call `toastManager.add()` straight from the Zustand slice.

## File layout

### Created

| File | Purpose |
|---|---|
| `frontend/src/react/components/ui/toast.tsx` | shadcn-style styled wrappers around `@base-ui/react/toast` parts (`Title`, `Description`, `Action`, `Close`, `Arrow`, `Root`). `cva` variants for `type: success / info / warn / critical`. |
| `frontend/src/react/components/ui/toaster.tsx` | The `<Toaster />` mount: `Toast.Provider` + `Toast.Portal` (`container={getLayerRoot("overlay")}`) + `Toast.Viewport`. Subscribes to `toastManager` via Base UI's hook to render items. |
| `frontend/src/react/lib/toast.ts` | Module-level `toastManager = createToastManager()` (no React tree dependency). Exports `pushReactNotification(item: NotificationCreate)` which maps the existing `NotificationCreate` shape to a Base UI Toast. Registers the `bb.vue-notification` window listener at module-eval time (top-level side effect). |

### Modified

| File | Change |
|---|---|
| `frontend/src/react/stores/app/notification.ts` | `notify()` calls `toastManager.add()` directly. Drop `emitReactNotification` import. Remove the unused `notifications` state array (no consumer reads it). |
| `frontend/src/react/shell-bridge.ts` | Remove `notification: "bb.react-notification"` event + `emitReactNotification()`. Direction is flipped now. |
| `frontend/src/store/modules/notification.ts` (Pinia) | Replace internal queue/`tryPopNotification` consumption with: emit `CustomEvent('bb.vue-notification', { detail: item })` on every `pushNotification(...)`. Keep the public method signatures unchanged so all 17 Vue callers compile. |
| `frontend/src/App.vue` | Remove `<NNotificationProvider>` and `<NotificationContext>` wrappers; drop their imports. |
| `frontend/src/react/main.tsx` (or wherever the React shell root mounts) | Render `<Toaster />` once at the React root. |
| `frontend/src/shell-bridge.test.ts` | Drop the `NotificationContext.vue` mock entry. |

### Deleted

| File | |
|---|---|
| `frontend/src/NotificationContext.vue` | The Vue render loop is gone. |

## API parity (`NotificationCreate` mapping)

The existing shape is preserved:

```ts
{
  module: "bytebase",
  style: "SUCCESS" | "INFO" | "WARN" | "CRITICAL",
  title: string,
  description?: string | (() => unknown),
  link?: string,
  linkTitle?: string,
  manualHide?: boolean,
}
```

Mapping in `pushReactNotification`:

| Field | Maps to |
|---|---|
| `style` → `type` | `SUCCESS`→`"success"`, `INFO`→`"info"`, `WARN`→`"warning"`, `CRITICAL`→`"error"`. |
| `title` | `<Toast.Title>` |
| `description: string` | `<Toast.Description>` |
| `description: () => VNode` | **Not supported.** Function-typed descriptions must be plain strings; if grep finds any caller using a function, rewrite that caller as part of this PR (audit before merge). |
| `link + linkTitle` | `<Toast.Action>` rendered inside the toast body, opens link in new tab. |
| `manualHide: true` | `Toast.Root` with `timeout={null}` (or equivalent — confirm against Base UI's API at implementation time). |
| Default duration | 6000ms. |
| `style: "CRITICAL"` duration | 10000ms. |
| `module: "bytebase"` filter | Applied at the window listener level before calling `toastManager.add()`. |

## Accessibility

Base UI Toast provides ARIA live region semantics, focus management, swipe-to-dismiss, and keyboard navigation (Tab/Shift+Tab between toasts, Esc to dismiss focused one) out of the box. Set `priority="high"` (or equivalent assertive politeness) on `CRITICAL` toasts; `priority="low"` on others.

`<Toast.Close>` button: `aria-label={t("common.close")}`.

## Layering

`<Toast.Portal container={getLayerRoot("overlay")}>` — same overlay family as dialogs, sheets, dropdowns. Position: bottom-right (matches the current Naive UI default). No raw `z-index` per `frontend/AGENTS.md`'s overlay policy.

## i18n

No new keys required. Existing `pushNotification` callers already pass localized strings. The close-button `aria-label` reuses `common.close`.

## Risks & mitigations

1. **Listener registration order.** The `bb.vue-notification` window listener must be registered before any `pushNotification` fires. Achieved by registering it at module-eval time (top-level side effect in `react/lib/toast.ts`); the React entry point imports the module before any rendering or Vue bootstrap completes.
2. **Duplicate toasts during transition.** If `NotificationContext.vue` is still mounted when the new path lights up, every notification renders twice. Mitigation: deletion of `NotificationContext.vue` lands in the **same commit** that flips the Pinia store to emit events. Atomic switch.
3. **Function-typed `description`.** The existing API permits `description: () => VNode` for Vue render functions. React can't render Vue VNodes. Pre-implementation audit: `grep -rn "description:.*=>" frontend/src` to find function uses; rewrite as plain strings (expected count: ~0–3). If a caller genuinely needs richness, use a `Trans` component string with markup placeholders instead.
4. **Test mocks.** `shell-bridge.test.ts` and `layout-bridge.test.ts` mock `NotificationContext.vue`. Remove those entries when the file is deleted.
5. **Pinia store consumers.** Any code that depends on `notificationStore.tryPopNotification` (the queue-pop API) breaks. Audit: `grep -rn "tryPopNotification" frontend/src`. The only known consumer is `NotificationContext.vue` itself (also being deleted), but verify.

## Validation gates

| Gate | When |
|---|---|
| `pnpm --dir frontend fix` | After every commit |
| `pnpm --dir frontend type-check` | After every commit |
| `pnpm --dir frontend test` | After commits 3 + 5 (state plumbing changes) |
| Manual smoke | After commit 5 — see below |

### Manual smoke checklist

- Trigger a SUCCESS toast (e.g. successful save) — green, auto-dismisses at 6s.
- Trigger an INFO toast — blue, auto-dismisses at 6s.
- Trigger a WARN toast — yellow.
- Trigger a CRITICAL toast (e.g. SQL execution error) — red, auto-dismisses at 10s.
- Trigger a toast with `link + linkTitle` — action button renders, opens link in new tab.
- Trigger a toast with `manualHide: true` — stays until the close button is clicked.
- Trigger multiple toasts rapidly — they stack with proper animations.
- Hover a toast — pause auto-dismiss; mouse out — resume.
- Tab into toasts — focus moves between them; Esc dismisses focused one.
- Swipe gesture on touch — dismisses.
- Toast renders above open dialogs and sheets (same `overlay` layer).
- Toast renders below `critical` layer (session-expired surface).
- Auth flow with `pushNotification` from middlewares (e.g. trigger an auth interceptor error) — toast renders correctly.

## Estimated commits

1. **Add Base UI Toast wrappers.** Create `react/components/ui/toast.tsx` + `react/components/ui/toaster.tsx`. No mounts yet, no wiring. Static; just shadcn-style component definitions.
2. **Add the toast manager + window listener.** Create `react/lib/toast.ts` with `createToastManager()` and `pushReactNotification()`. Wire the `bb.vue-notification` listener.
3. **Mount `<Toaster />` + flip the Pinia store.** Render `<Toaster />` in the React root. Rewrite `store/modules/notification.ts` to dispatch the window event instead of running its own loop. Atomic with commit 5.
4. **Swap the React slice.** Update `react/stores/app/notification.ts` to call `toastManager.add()` directly. Drop the `emitReactNotification` path from `react/shell-bridge.ts`.
5. **Delete the Vue path.** Remove `NotificationContext.vue`, unwrap `App.vue`, drop test mocks. Same commit as 3 if we want true atomicity (recommended).

Practically, commits 3 and 5 should be a single commit — otherwise the app double-renders or no-renders for the duration between. Final shape: **4 commits**.

## Out-of-scope follow-ups

- Phase B3: Pinia `notificationStore` → Zustand. The pass-through becomes a thin Zustand slice; the window-event bridge collapses to a direct function call.
- Phase B2: App.vue migration. When `App.vue` becomes React, the `bb.vue-notification` event bridge can be deleted entirely — only Vue store callers will still need it, and the Vue store itself dies in B3.
