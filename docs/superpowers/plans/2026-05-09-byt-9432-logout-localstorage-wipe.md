# BYT-9432 — Stop wiping user-scoped localStorage on logout (Implementation Plan)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the broad `*.<email>`-suffix `localStorage` cleanup from both the Vue and React logout flows so SQL Editor tabs and other user-scoped UI state survive a logout/login cycle.

**Architecture:** Pure deletion. The Vue store has a private `cleanupUserStorage(email)` helper called from `logout()`'s `finally` block; we delete both. The React store inlines the same loop in its own `logout()`'s `finally`; we delete the loop and the now-unused `email` capture and import. No other files change. The fix is two surgical edits in two files, validated by manual logout/login round-trip plus the standard frontend lint/check/type-check/test gates.

**Tech Stack:** TypeScript, Vue 3 (Pinia), React, frontend monorepo gated by `pnpm --dir frontend {fix,check,type-check,test}`.

**Spec:** [`docs/superpowers/specs/2026-05-09-byt-9432-logout-localstorage-wipe-design.md`](../specs/2026-05-09-byt-9432-logout-localstorage-wipe-design.md)

---

## File Map

- **Modify:** `frontend/src/store/modules/v1/auth.ts` — delete the `cleanupUserStorage` helper (~lines 186–196) and remove its call site inside `logout()`'s `finally` block (~line 204).
- **Modify:** `frontend/src/react/stores/app/auth.ts` — delete the inlined cleanup loop inside `logout()`'s `finally`, drop the unused `email` capture, and drop the now-unused `getCurrentUserEmail` import.

No files created. No tests added (pure deletion of behavior; per the spec, a "logout doesn't touch localStorage" test would be a low-value negative-behavior pin).

---

## Task 1: Remove cleanup from the Vue auth store

**Files:**
- Modify: `frontend/src/store/modules/v1/auth.ts`

- [ ] **Step 1: Delete the `cleanupUserStorage` helper definition**

In `frontend/src/store/modules/v1/auth.ts`, remove the entire helper (and its trailing blank line) so it's gone from the file. The block to delete:

```ts
  const cleanupUserStorage = (email: string) => {
    if (!email) return;
    const keysToRemove: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.endsWith(`.${email}`)) {
        keysToRemove.push(key);
      }
    }
    keysToRemove.forEach((key) => localStorage.removeItem(key));
  };

```

Verify: `grep -n "cleanupUserStorage" frontend/src/store/modules/v1/auth.ts` returns no matches after the delete in this step is paired with Step 2 (the call site).

- [ ] **Step 2: Remove the call site inside `logout()`'s `finally` block**

Inside the `logout` function in the same file, the `finally` block currently reads:

```ts
    } finally {
      cleanupUserStorage(currentUserEmail.value);
      unauthenticatedOccurred.value = false;
      const pathname = location.pathname;
      // Replace and reload the page to clear frontend state directly.
      window.location.href = router.resolve({
        name: AUTH_SIGNIN_MODULE,
        query: {
          redirect:
            getRedirectQuery() ||
            (pathname.startsWith("/auth") ? undefined : pathname),
        },
      }).fullPath;
    }
```

Edit it to drop the `cleanupUserStorage(currentUserEmail.value);` line, leaving:

```ts
    } finally {
      unauthenticatedOccurred.value = false;
      const pathname = location.pathname;
      // Replace and reload the page to clear frontend state directly.
      window.location.href = router.resolve({
        name: AUTH_SIGNIN_MODULE,
        query: {
          redirect:
            getRedirectQuery() ||
            (pathname.startsWith("/auth") ? undefined : pathname),
        },
      }).fullPath;
    }
```

- [ ] **Step 3: Verify no stale references remain**

Run: `grep -n "cleanupUserStorage" frontend/src/store/modules/v1/auth.ts frontend/src/react/stores/app/auth.ts`
Expected: no output.

Run also: `grep -rn "cleanupUserStorage" frontend/src`
Expected: no output (helper was only referenced by name in this one file).

---

## Task 2: Remove cleanup from the React auth store

**Files:**
- Modify: `frontend/src/react/stores/app/auth.ts`

- [ ] **Step 1: Delete the inlined cleanup block and the unused `email` capture**

Open `frontend/src/react/stores/app/auth.ts`. The current `logout` reads:

```ts
  logout: async (signinUrl) => {
    const email = getCurrentUserEmail(get);
    try {
      await authServiceClientConnect.logout({});
    } catch {
      // Ignore logout errors and clear the local session by redirecting anyway.
    } finally {
      if (email) {
        const keysToRemove: string[] = [];
        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i);
          if (key?.endsWith(`.${email}`)) {
            keysToRemove.push(key);
          }
        }
        keysToRemove.forEach((key) => localStorage.removeItem(key));
      }
      window.location.href = signinUrl;
    }
  },
```

Replace the entire function with:

```ts
  logout: async (signinUrl) => {
    try {
      await authServiceClientConnect.logout({});
    } catch {
      // Ignore logout errors and clear the local session by redirecting anyway.
    } finally {
      window.location.href = signinUrl;
    }
  },
```

(The `const email = getCurrentUserEmail(get);` line and the entire `if (email) { ... }` block are gone.)

- [ ] **Step 2: Drop the now-unused `getCurrentUserEmail` import**

At the top of the same file, the imports currently read:

```ts
import { authServiceClientConnect, userServiceClientConnect } from "@/connect";
import type { AppSliceCreator, AuthSlice } from "./types";
import { getCurrentUserEmail } from "./utils";
```

`getCurrentUserEmail` was only referenced inside the deleted `logout` body. Remove the import line so the imports become:

```ts
import { authServiceClientConnect, userServiceClientConnect } from "@/connect";
import type { AppSliceCreator, AuthSlice } from "./types";
```

If, for any reason, `getCurrentUserEmail` turns out to still be referenced elsewhere in this file (it should not be), keep the import. Verify with: `grep -n "getCurrentUserEmail" frontend/src/react/stores/app/auth.ts` — expected: no output after the edit.

- [ ] **Step 3: Verify no stale references remain in this file**

Run: `grep -n "localStorage\|cleanupUserStorage\|getCurrentUserEmail" frontend/src/react/stores/app/auth.ts`
Expected: no output.

---

## Task 3: Run frontend validation gates

**Files:** none (validation only)

- [ ] **Step 1: Auto-fix formatter / linter / import order**

Run: `pnpm --dir frontend fix`
Expected: completes without error. May produce no diff, or only trivial formatting changes — review any diff.

- [ ] **Step 2: CI-equivalent check**

Run: `pnpm --dir frontend check`
Expected: passes with no errors.

- [ ] **Step 3: Type check**

Run: `pnpm --dir frontend type-check`
Expected: passes. If the `getCurrentUserEmail` import was correctly removed in Task 2 / Step 2, there should be no "unused import" or "cannot find name" errors.

- [ ] **Step 4: Unit tests**

Run: `pnpm --dir frontend test`
Expected: existing tests pass. There are no new tests in this plan.

---

## Task 4: Manual browser verification

**Files:** none (runtime verification)

- [ ] **Step 1: Start the backend and frontend dev servers**

In one terminal:
`PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug`

In another:
`pnpm --dir frontend dev`

- [ ] **Step 2: Sign in and create SQL Editor state**

1. Open the app in a browser and sign in as a real user.
2. Navigate to SQL Editor.
3. Create 2–3 tabs in any project, type a different SQL statement into each (do not save them as worksheets — leave them as unsaved tab content).

- [ ] **Step 3: Snapshot the user-scoped localStorage keys**

Open DevTools → Application → Local Storage → the dev-server origin.

Note the presence of keys whose name ends with `.<your-email>`, especially:
- `bb.sql-editor.tabs.<project>.<your-email>`
- `bb.sql-editor.current-tab.<project>.<your-email>`

- [ ] **Step 4: Sign out and confirm keys persist**

1. Click sign out from the app menu.
2. After the redirect to the signin page, refresh the DevTools Local Storage view for the same origin.
3. Confirm the `*.<your-email>` keys from Step 3 are **still present**.

If any of those keys are missing after logout, the cleanup deletion is incomplete — go back and re-check Tasks 1 and 2.

- [ ] **Step 5: Sign back in and confirm tabs restore**

1. Sign back in as the same user.
2. Navigate to SQL Editor.
3. Confirm the tabs from Step 2 are restored, including the unsaved SQL text in each tab.
4. Confirm the previously-active tab is still active (driven by `bb.sql-editor.current-tab.<project>.<email>`).

---

## Task 5: Commit

**Files:** the two modified files from Tasks 1 and 2.

- [ ] **Step 1: Stage and commit**

Run:

```bash
git add frontend/src/store/modules/v1/auth.ts frontend/src/react/stores/app/auth.ts
git commit -m "$(cat <<'EOF'
fix(frontend): stop wiping user-scoped localStorage on logout

Both the Vue and React logout flows scanned localStorage and removed
every key ending in `.<email>`. That wiped SQL Editor tabs, recent
visits, and other user-scoped UI state — productivity state users
expect to persist across sessions on the same browser.

Keys are already user-scoped, so cross-user contamination was not a
concern. Drop the cleanup so same-email re-login restores the prior
SQL Editor tabs and other UI state.

Fixes BYT-9432

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds. If a pre-commit hook fails, fix the underlying issue and create a new commit (do not `--amend`).

- [ ] **Step 2: Verify the commit**

Run: `git log -1 --stat`
Expected: shows two files modified (`frontend/src/store/modules/v1/auth.ts`, `frontend/src/react/stores/app/auth.ts`) with a small net-negative line count.

---

## Done criteria

- `cleanupUserStorage` no longer exists in the codebase (`grep -rn "cleanupUserStorage" frontend/src` returns nothing).
- Neither auth store calls `localStorage.removeItem` from `logout()`.
- Frontend lint, check, type-check, and test gates pass.
- Manual round-trip (Task 4) confirms SQL Editor tabs survive a logout/login cycle on the same email.
- Single commit on the branch with both file changes and a descriptive message linking BYT-9432.
