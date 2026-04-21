## References

1. [React Migration Playbook (repo-internal)](../2026-04-08-react-migration-playbook.md) — authoritative for Bytebase-specific patterns: migration ordering, Pinia-from-React, `useVueState`, test requirements, Monaco pitfalls, shared-component rules, deletion rules. Verified via `Read`.
2. [SQL Editor React Migration Stage 1 Design (repo-internal)](../../superpowers/specs/2026-04-20-sql-editor-react-migration-stage-1-design.md) — precedent for the React-island-inside-Vue pattern; documents the `mount.ts` glob registration, `ReactPageMount.vue` prop-callback bridge, and per-leaf verification checklist. Verified via `Read`.
3. [Schema Editor React Migration Design (repo-internal)](../2026-04-16-schema-editor-react-migration/design.md) — precedent for a larger multi-phase migration with shared TS layers; establishes the phased-delivery format and "feature parity with Vue, zero regression in Vue callers" goal structure. Verified via `Read`.
4. [Martin Fowler — Strangler Fig Application](https://martinfowler.com/bliki/StranglerFigApplication.html) — canonical reference for incremental legacy-subsystem replacement; frames the Vue-route-shell-wraps-React-body approach as necessary transitional architecture. Verified via `WebFetch`.
5. [React.dev — `createRoot` for partial adoption](https://react.dev/reference/react-dom/client/createRoot) — official guidance for mounting React into a non-React host page with one root per island, and for `root.unmount()` cleanup when the host removes the container. Verified via `WebFetch`.
6. [Vue to React Migration Strategy (index.dev)](https://www.index.dev/blog/vue-to-react-migration-strategy) — industry write-up of incremental Vue→React migration; calls out that router migration should come last and that state synchronization across frameworks is the primary risk. Verified via `WebFetch`.

## Industry Baseline

**Strangler Fig for framework migration**: Fowler's Strangler Fig pattern is the dominant reference for modernizing live systems. Build new work adjacent to the legacy code and shift functionality piece-by-piece behind a stable facade. Temporary integration code — the "strangler fig" itself — is explicitly expected and not considered waste. (Martin Fowler — Strangler Fig Application)

**React islands in a host page**: React's official docs describe `createRoot` as the mechanism for partial adoption: one root per island, explicit `unmount()` on host-driven teardown, and no constraint on the surrounding framework owning the rest of the DOM. This is the exact shape of `frontend/src/react/ReactPageMount.vue` and `frontend/src/react/mount.ts:120`. (React.dev — `createRoot`)

**Migrate the router last**: The only outside source that addresses Vue→React migration order specifically recommends leaving the router for last and keeping state synchronization explicit during the transition. (Vue to React Migration Strategy)

**Repo precedent**: Two prior Bytebase migrations established the working shape: the Schema Editor plan models phased delivery with per-phase entry/exit criteria; the SQL Editor Stage 1 plan models the Vue-shell-hosts-React-leaf pattern with a one-line `<ReactPageMount page="..." />` swap. The Two-Factor Setup page is already live production evidence that the Vue-shell-wraps-React-page pattern works for an auth-adjacent route. (React Migration Playbook; SQL Editor Stage 1; Schema Editor Design)

**Trade-offs**: The Strangler Fig approach pays a duplication cost — the React version and the Vue version coexist until the last caller is gone, and shared components (e.g., `BytebaseLogo`) may exist in both frameworks for weeks. The alternative (big-bang route flip) has a larger blast radius, cannot be validated incrementally, and collides with the Vue router guard surface in `frontend/src/router/index.ts:85-276` that the rest of the app still depends on.

## Research Summary

1. **The Vue-shell-hosts-React-body pattern is already production-proven for one auth route.** `frontend/src/views/TwoFactorRequired.vue:22-28` calls `mountReactPage(container, "TwoFactorSetupPage", { cancelAction })`. No router-level change was needed. This is the shape every remaining auth/setup page should follow. (Repo evidence; SQL Editor Stage 1 design)

2. **Router guards do not need to move.** `frontend/src/router/index.ts:85-276` reads `route.name` (`AUTH_SIGNIN_MODULE` et al.) and Pinia store state. Swapping the component a route renders does not affect either. Leaving the router intact is also the outside-recommended ordering. (Vue to React Migration Strategy)

3. **Pinia and Connect RPC stay; React consumes via `useVueState`.** The playbook and the SQL Editor stage 1 both commit to the React-reads-Pinia pattern rather than porting stores. `authStore.login()`, `authStore.logout()`, and `authStore.requireResetPassword` are called from React the same way they are called from Vue today.

4. **SigninBridge must flip in lockstep with `Signin.vue` deletion.** `frontend/src/react/components/auth/SigninBridge.tsx:9` imports `@/views/auth/Signin.vue`. The migration must resolve this — either the React signin page is embeddable inside `SessionExpiredSurface` (preferred), or the bridge temporarily keeps a trimmed Vue signin for the dialog. Strangler Fig explicitly expects this kind of transitional coupling. (Martin Fowler — Strangler Fig)

5. **One React root per page, explicit unmount.** `ReactPageMount.vue` already does this correctly (`onUnmounted(() => root?.unmount())`). No new infrastructure needed. (React.dev — `createRoot`)

6. **Shared auth subcomponents have zero non-auth callers.** `PasswordSigninForm.vue`, `EmailCodeSigninForm.vue`, and `DemoSigninForm.vue` are only imported by `Signin.vue`. They port and delete in the same phase as `Signin.vue` with no risk to other Vue callers. (`rg` on repo)

## Design Goals

1. **Zero router-guard changes.** `frontend/src/router/index.ts:85-276` and the route-name constants in `frontend/src/router/auth.ts` / `frontend/src/router/setup.ts` are not modified by this migration. Verifiable: `git diff main...HEAD -- frontend/src/router/` shows only component-import swaps, not guard changes.
2. **Page-at-a-time shippability.** Every phase produces an independently shippable PR that leaves auth/setup fully functional. Verifiable: each phase can be reverted without forcing a revert of later phases; each PR runs `fix`/`check`/`type-check`/`test` green.
3. **No new state libraries.** No TanStack Query, no zustand, no new Pinia stores introduced for this surface. Verifiable: `package.json` diff adds no runtime dependencies.
4. **`SessionExpiredSurface` never regresses during migration.** The session-expired relogin dialog must keep working in every intermediate state. Verifiable: `frontend/src/react/components/auth/SessionExpiredSurface.test.tsx` passes on every PR; manual path: force token expiration, confirm relogin surface appears and succeeds.
5. **Vue deletions are verified.** Every phase deletes the Vue counterpart it replaces, verified by `rg` for remaining importers. Verifiable: each PR includes an `rg` log in the description showing zero live callers for deleted files. (Playbook §Deletion Rule)
6. **UX parity.** Every migrated page visually matches the Vue counterpart at shipping time (same copy, same form fields, same error states, same redirect behavior). Verifiable: manual side-by-side for each phase; i18n keys reused, not duplicated.

## Non-Goals

Inherits all non-goals from `definition.md` §Non-Goals. Additions discovered during research:

- **No React Router under `/auth/*`.** Even at the end of the migration, Vue Router continues to own URL-to-page dispatch. A future BYT may flip the whole app to React Router; it is explicitly not this effort.
- **No server-side rendering or pre-render** for auth pages. Existing Vite build pipeline is unchanged.
- **No consolidation of `/auth`, `/auth/signin`, `/auth/admin`** into a single unified React `SigninPage` with variants. Routes and route names stay 1:1 with today.
- **No rework of `openWindowForSSO` or the OAuth state-token protocol.** `frontend/src/utils/sso.ts` is consumed as-is from React.

## Proposed Design

### Overall shape

Each migrated route keeps its Vue registration in `frontend/src/router/auth.ts` or `frontend/src/router/setup.ts`. The registered component changes from a Vue SFC with a `<template>` body to a thin Vue wrapper that mounts a React page via `ReactPageMount.vue`. The global router guard in `frontend/src/router/index.ts` is not touched. This is the pattern `TwoFactorRequired.vue` already demonstrates and is consistent with the Strangler Fig precedent of preserving the facade while replacing the interior. (Traces to Design Goals 1 and 2.)

Ordering: shared primitives first, then leaf pages, then the critical-path Signin page (which forces the `SigninBridge` rewrite), then Setup. Every phase ships independently.

### Route-file pattern after migration

Given a route today registered as:

```
{ path: "password-forgot", name: AUTH_PASSWORD_FORGOT_MODULE,
  component: () => import("@/views/auth/PasswordForgot.vue") }
```

Post-migration, the same route record points at a new Vue shell `frontend/src/views/auth/PasswordForgotPage.vue` whose template is one line — `<ReactPageMount page="PasswordForgotPage" />` — analogous to `TwoFactorRequired.vue:1-10`. Route name and metadata are unchanged.

If a page needs to call `authStore.logout()` or another Vue-context action that the React side should only trigger indirectly, the Vue shell passes a callback prop exactly as `TwoFactorRequired.vue:25-27` does with `cancelAction`. This mirrors the "prop-callback bridge" from SQL Editor Stage 1 §5.

Registering each new React page under `frontend/src/react/pages/auth/` requires one addition in `frontend/src/react/mount.ts`: a `./pages/auth/*.tsx` glob alongside the existing `./pages/settings/*.tsx`, `./pages/project/*.tsx`, `./pages/workspace/*.tsx`, and `./components/auth/*.tsx` globs (mount.ts:3-14).

### React directory layout

```
frontend/src/react/
├── pages/
│   └── auth/
│       ├── SigninPage.tsx                      # phase 8
│       ├── SigninAdminPage.tsx                 # phase 7
│       ├── SignupPage.tsx                      # phase 9
│       ├── PasswordForgotPage.tsx              # phase 2
│       ├── PasswordResetPage.tsx               # phase 6
│       ├── MultiFactorPage.tsx                 # phase 3
│       ├── OAuth2ConsentPage.tsx               # phase 5
│       ├── OAuthCallbackPage.tsx               # phase 4
│       └── SetupPage.tsx                       # phase 10
└── components/
    └── auth/
        ├── SigninBridge.tsx                    # rewritten in phase 8
        ├── SessionExpiredSurface.tsx           # unchanged
        ├── AuthFooter.tsx                      # phase 1
        ├── PasswordSigninForm.tsx              # phase 8 (ships with Signin)
        ├── EmailCodeSigninForm.tsx             # phase 8
        └── DemoSigninForm.tsx                  # phase 8
```

`BytebaseLogo.tsx` already exists in `frontend/src/react/components/` from SQL Editor Stage 1 and is reused here; no duplication. (Traces to Design Goal 3.)

### Data boundaries

| Need | Source | React access |
|---|---|---|
| Sign-in, MFA challenge, logout | `useAuthStore()` | Direct Pinia import + `useVueState` where reactivity is needed; direct function call for actions (`authStore.login(...)`). |
| Server info (restriction flags, `isDemo`, `isSaaSMode`, `activeUserCount`) | `useActuatorV1Store()` | Direct Pinia import + `useVueState`. |
| IdP list | `useIdentityProviderStore()` | `useVueState(() => useIdentityProviderStore().identityProviderList)`. |
| Subscription (for `SplashLayout` gating) | `useSubscriptionV1Store()` | N/A during this migration — `SplashLayout` stays Vue. |
| OAuth state (popup/redirect, token) | `frontend/src/utils/sso.ts` | Imported directly from React (`retrieveOAuthState`, `clearOAuthState`, `openWindowForSSO`). |
| Workspace-scoped actuator fetch | `resolveWorkspaceName()` from `frontend/src/utils` | Imported directly. |
| Router navigation | `router` from `frontend/src/router` | Imported directly; `router.push({ name: AUTH_SIGNIN_MODULE, ... })` from React, same API as Vue. |
| Form validation + state | — | React-local `useState`/`useReducer`; no new library. |
| Permission checks for Setup | `hasWorkspacePermissionV2` | `usePermissionCheck` (`frontend/src/react/components/PermissionGuard.tsx:12`). |

No new bridge is needed. Everything above is already in use in other React surfaces.

### The `SigninBridge` transition

The existing `frontend/src/react/components/auth/SigninBridge.tsx:11-109` creates a nested Vue app around `@/views/auth/Signin.vue` to render the session-expired relogin dialog. When Phase 8 deletes `Signin.vue`, the bridge rewrites to:

- Import the new React `SigninPage` (or a splittable core form) directly.
- Render it inside the existing `<Dialog.Popup>` layout owned by `SessionExpiredSurface.tsx:18-32`.
- Drop `createApp`, `NConfigProvider`, `pinia`, `NaiveUI`, and the Vue router/i18n wiring — React already has these via `mount.ts`'s `loadCoreDeps` and the owning `SessionExpiredSurfaceMount.vue` host.
- Preserve the `{ redirect: false, redirectUrl, allowSignup: false }` contract by accepting those as props.

This coordinated change is the highest-risk moment in the migration. It happens in one PR with:

- The new `SigninPage.tsx` behind a React-native `<SigninForm>` component that both the route page and the bridge import.
- A pre-merge run of `frontend/src/react/components/auth/SessionExpiredSurface.test.tsx` plus a manual session-expiration path test.

### Phase-by-phase plan

Order minimizes blast radius by shipping shared primitives first, then the simplest leaves, and saves the `Signin` + `SigninBridge` rewrite for last before `Signup` and `Setup`.

**Phase 1 — Shared primitives + infrastructure**
- Add `./pages/auth/*.tsx` glob to `frontend/src/react/mount.ts`.
- Port `AuthFooter.vue` → `frontend/src/react/components/auth/AuthFooter.tsx`. Keep Vue version for remaining Vue pages.
- Seed React auth i18n entries under `frontend/src/react/locales/{en-US,...}.json` for the `auth.*`, `common.*`, `multi-factor.*`, `two-factor.*`, `oauth.*`, `oauth2.consent.*`, `setup.*` keys used by auth pages. Verify via existing `check-react-i18n` scripts.
- No Vue file deletions.
- Exit: mount.ts resolves a `PasswordForgotPage.tsx` path, `AuthFooter` tests green.

**Phase 2 — `PasswordForgot` (simplest form; 120 LOC)**
- New `SubmissionConfirmation`-style page: email input → `authStore.requestPasswordReset()` call path → success state.
- Route component swapped to a `PasswordForgotPage.vue` thin Vue shell that mounts the React page.
- Delete `frontend/src/views/auth/PasswordForgot.vue` after `rg` confirms no other callers.
- Exit: /auth/password-forgot renders React body, full flow works.

**Phase 3 — `MultiFactor` (OTP + recovery code; 129 LOC)**
- Replaces NInputOtp with a React OTP input. Candidate: `input-otp` (shadcn-compatible) or a simple 6-digit inputs implementation; pick the simpler one at implementation time. `mfaTempToken` query-param read via `router.currentRoute.value.query.mfaTempToken` from React.
- Delete `frontend/src/views/auth/MultiFactor.vue`.

**Phase 4 — `OAuthCallback` (205 LOC, popup + redirect paths)**
- This route does **not** use `SplashLayout`, so the migrated page is a plain centered container.
- Imports `retrieveOAuthState`, `clearOAuthState` from `@/utils/sso`. `window.opener.dispatchEvent` and `window.close()` remain native JS — no framework coupling.
- Delete `frontend/src/views/OAuthCallback.vue`.

**Phase 5 — `OAuth2Consent` (192 LOC)**
- Delete `frontend/src/views/OAuth2Consent.vue`.

**Phase 6 — `PasswordReset` (270 LOC)**
- Reuses `UserPassword` equivalent — port a local React `PasswordWithConfirmField` component or inline the two inputs (defer until implementation to avoid over-abstraction). Do not touch the Vue `UserPassword.vue` — it has non-auth callers.
- Delete `frontend/src/views/auth/PasswordReset.vue`.

**Phase 7 — `SigninAdmin` (44 LOC variant; prepares Phase 8)**
- Small; ideal testbed for the form-rendering primitives reused in Phase 8.
- Delete `frontend/src/views/auth/SigninAdmin.vue`.

**Phase 8 — `Signin` + form subcomponents + `SigninBridge` rewrite (highest-risk)**
- New `frontend/src/react/pages/auth/SigninPage.tsx` composed of:
  - `PasswordSigninForm.tsx`, `EmailCodeSigninForm.tsx`, `DemoSigninForm.tsx` (under `react/components/auth/`).
  - A React `Tabs` (`@/react/components/ui/tabs`) instead of `NTabs`/`NCard`.
  - Extracts a reusable inner `<SigninForm>` component that both `SigninPage` and the rewritten `SigninBridge` render.
- `SigninBridge.tsx` rewritten to mount the React `<SigninForm>` directly; Vue-app bridge deleted.
- `SessionExpiredSurface.test.tsx` updated to exercise the new React signin (no Vue app instantiation in tests).
- Delete `frontend/src/views/auth/Signin.vue` and `frontend/src/components/{Password,EmailCode,Demo}SigninForm.vue`.
- Verification: full session-expiration manual test + OAuth/OIDC end-to-end path + the five permission/restriction-flag states from `Signin.vue`'s `showSignInForm` logic.

**Phase 9 — `Signup` (251 LOC)**
- Delete `frontend/src/views/auth/Signup.vue`.

**Phase 10 — `/setup` (Setup + AdminSetup + WorkspaceMode; 31 + 228 + 75 LOC)**
- `SetupPage.tsx` with a React `RoutePermissionGuard` equivalent wrapping the page body. If a React `RoutePermissionGuard` does not yet exist, use `ComponentPermissionGuard` (`frontend/src/react/components/ComponentPermissionGuard.tsx`) which already implements the same shape (Playbook §Permission Guards).
- AdminSetup's initial-workspace form → React form with `useRoleStore` / `useWorkspaceV1Store` reads via `useVueState`.
- Delete `frontend/src/views/Setup/{Setup,AdminSetup,WorkspaceMode}.vue`.

### Per-phase verification checklist

Every phase runs before opening its PR:

- `pnpm --dir frontend fix` — format + lint
- `pnpm --dir frontend check` — CI-equivalent
- `pnpm --dir frontend type-check` — both `vue-tsc` and `tsconfig.react.json`
- `pnpm --dir frontend test` — includes the new `*.test.tsx` for the phase's components
- `rg` for imports of every file the phase deletes, attached to PR description (Playbook §Deletion Rule)
- Manual: log in, log out, force session expiration, trigger password reset, sign in via each enabled IdP (only for phases that touch those flows). Phase 8 requires all of them.
- `SessionExpiredSurface.test.tsx` green on every phase, not just Phase 8.

### Rejected alternatives

- **Route-level flip to a React Router in `/auth/*`**: rejected because it requires either (a) porting `frontend/src/router/index.ts:85-276` guard logic simultaneously or (b) duplicating guard behavior across two routers. Both violate Design Goal 1 and contradict outside guidance to migrate the router last. (Vue to React Migration Strategy)
- **React-island pattern (SQL Editor Stage 1 shape)**: rejected for auth because each auth page is essentially one form; there are no independently shippable sub-leaves. The Vue-shell-wraps-React-body pattern (`TwoFactorRequired.vue`) yields the same incremental-validation benefit for this surface with fewer files. The island pattern remains correct for the SQL Editor because its `EditorPanel` genuinely has many independent leaves.
- **Porting `SplashLayout.vue` to React in Phase 1**: rejected because the layout's branding-image behavior depends on `useSubscriptionV1Store()` and on the Vue-router `route.name` — both would have to cross back into Vue state for the remaining-Vue callers (the auth pages before they migrate). Leaving `SplashLayout` as Vue is strictly simpler and loses nothing; a future PR can flip it in O(minutes) once every inhabitant is React.
- **One-PR big-bang migration of all 10 pages**: rejected per Playbook §Migration Order and Strangler Fig principle — blast radius too large, no intermediate validation, and the `SigninBridge` coupling forces a single-point-of-failure release moment.
