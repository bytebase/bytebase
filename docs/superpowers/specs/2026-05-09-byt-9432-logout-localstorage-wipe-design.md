# BYT-9432 — Stop wiping user-scoped localStorage on logout

- **Status:** Approved (design)
- **Linear:** [BYT-9432](https://linear.app/bytebase/issue/BYT-9432/logout-wipes-user-scoped-localstorage-including-sql-editor)
- **Date:** 2026-05-09

## Problem

On logout, both the Vue store (`frontend/src/store/modules/v1/auth.ts`) and the React store (`frontend/src/react/stores/app/auth.ts`) scan `localStorage` and remove every key whose suffix matches `.${currentUserEmail}`. This wipes SQL Editor tabs/current-tab pointers, worksheet UI state, recent visits, quick access, and other productivity state that users rely on persisting across sessions.

Keys are already user-scoped by email, so different users on the same browser cannot read each other's data — there is no cross-user contamination problem to solve by wiping. Closing the browser does not wipe `localStorage`; logout being more destructive than that is surprising. The wipe was introduced in PR #19218 alongside the storage-key registry refactor without an explicit rationale.

## Goal

Remove the broad `*.<email>` cleanup from both logout paths so user-scoped state survives a logout/login round-trip on the same browser.

## Non-goals

- No changes to `storage-keys.ts` or `storage-migrate.ts`. The keys themselves are correct.
- No new "Sign out and clear local data" opt-in action. If shared-machine privacy becomes a real ask, that's a separate spec.
- No new tests pinning negative behavior (logout-must-not-touch-localStorage). Manual verification only.

## Changes

### 1. `frontend/src/store/modules/v1/auth.ts` (Vue store)

- Delete the `cleanupUserStorage` helper (defined ~line 186, 11 lines). It has no other callers in the codebase (verified by grep).
- Delete its invocation inside `logout()`'s `finally` block (~line 204).
- Preserve the rest of `finally` unchanged: `unauthenticatedOccurred.value = false` and the redirect to the signin route via `window.location.href`.

### 2. `frontend/src/react/stores/app/auth.ts` (React store)

- Delete the inlined cleanup block in `logout()` (the loop populating `keysToRemove` and the subsequent `forEach(localStorage.removeItem)`).
- Drop the `email` capture on line 29 since it becomes unused.
- Drop the `getCurrentUserEmail` import from `./utils` if it has no remaining uses in this file.
- Keep the `try`/`catch` around `authServiceClientConnect.logout({})` and the `window.location.href = signinUrl` redirect.

### What stays untouched

- `frontend/src/utils/storage-keys.ts` — unchanged.
- `frontend/src/utils/storage-migrate.ts` — unchanged.
- All callers and key builders — unchanged.

## Behavior after fix

| Scenario | Before | After |
|----------|--------|-------|
| Logout, log back in as same email | User-scoped keys wiped; SQL Editor tabs gone | Keys persist; tabs and unsaved statements restore |
| Logout, log in as different email on same browser | Previous user's keys wiped | Previous user's keys remain in storage, inaccessible to the new email (different suffix). Consistent with how every other browser-persisted UI state already behaves. |
| Closing browser without logout | Keys persist (already true) | Keys persist (unchanged) |

## Risk

Low. Two small deletions in two files. No API, proto, or schema changes. The only observable behavior change is that user-scoped keys persist across logout — the change being asked for.

The minor consideration is that ephemeral auth-flow nudges keyed by email (e.g., `bb.iam-remind.<email>`, `bb.reset-password.<email>`) will also persist. On a single-user browser this is harmless: the same email re-login should see the same nudge state. If this turns out to be a problem in practice, those specific keys can be cleared at the points where they're produced/consumed rather than via a blanket logout sweep.

## Testing

### Manual

1. Sign in to the app.
2. Open SQL Editor, create 2–3 tabs with unsaved SQL.
3. Open DevTools → Application → Local Storage; note keys with the `.<email>` suffix (`bb.sql-editor.tabs.<project>.<email>`, `bb.sql-editor.current-tab.<project>.<email>`, etc.).
4. Sign out via the app menu.
5. Confirm those keys are still present in Local Storage after the redirect to the signin page.
6. Sign back in as the same user.
7. Confirm SQL Editor tabs, current tab, and the unsaved SQL within them all restore.

### Automated gates

- `pnpm --dir frontend fix`
- `pnpm --dir frontend check`
- `pnpm --dir frontend type-check`
- `pnpm --dir frontend test`

No new unit tests planned — the change is a pure deletion, and there is no covering test today. A test asserting "logout does not call `localStorage.removeItem`" would be a low-value negative-behavior pin.

## Rollout

Standard frontend deploy. No feature flag needed — the behavior change is small and clearly an improvement.
