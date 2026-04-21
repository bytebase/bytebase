## Background & Context

Linear BYT-9167 covers the Vue-to-React migration of Bytebase's auth/setup surface: sign-in, sign-up, password reset, MFA challenge, SSO callbacks, OAuth2 consent, two-factor setup, and initial workspace setup. This rolls up under BYT-9185, the broader Vue-to-React migration tracked in `docs/plans/2026-04-08-react-migration-playbook.md`.

Prior migration work established two coexistence patterns:

1. **Vue-shell-mounts-React**: `frontend/src/views/TwoFactorRequired.vue:22-28` mounts the React `TwoFactorSetupPage` (`frontend/src/react/pages/settings/two-factor/TwoFactorSetupPage.tsx`) via `mountReactPage()` exported from `frontend/src/react/mount.ts:120`. Route stays Vue; page body is React.
2. **React-island-in-Vue**: `docs/superpowers/specs/2026-04-20-sql-editor-react-migration-stage-1-design.md` migrates interior leaves by embedding them inside the Vue orchestrator through `frontend/src/react/ReactPageMount.vue`. The Vue parent continues to own routing, shared `provide/inject` context, and Pinia state; React leaves call Pinia and the router directly via `useVueState` (`frontend/src/react/hooks/useVueState.ts:15`) and `usePermissionCheck` (`frontend/src/react/components/PermissionGuard.tsx:12`).

One auth component already runs React-in-Vue in the reverse direction: `frontend/src/react/components/auth/SigninBridge.tsx:11-109` mounts the Vue `Signin.vue` inside a React `SessionExpiredSurface` dialog by creating an ad-hoc Vue app. This bridge will be invalidated once `Signin.vue` is removed.

The two-factor setup page is the only auth/setup surface migrated so far; all other pages remain Vue.

## Issue Statement

Ten Vue SFCs under `frontend/src/views/auth/`, `frontend/src/views/Setup/`, `frontend/src/views/OAuthCallback.vue`, `frontend/src/views/OAuth2Consent.vue`, and `frontend/src/views/TwoFactorRequired.vue` (plus the `SplashLayout.vue` shell and three `components/*SigninForm.vue` subcomponents) remain the live implementation for /auth/*, /oauth2/consent, /oauth/callback, /oidc/callback, /2fa/setup, and /setup. The auth router (`frontend/src/router/auth.ts`) and setup router (`frontend/src/router/setup.ts`) register these Vue components directly, and the global navigation guard (`frontend/src/router/index.ts:85-276`) reads `useAuthStore()` state and route names to enforce sign-in, MFA setup, and forced-password-reset redirects. The `SessionExpiredSurface` React dialog reuses the Vue `Signin.vue` via a Vue-app bridge, so migration must preserve a working session-expired relogin path.

## Current State

### Route registration

- `frontend/src/router/auth.ts:1-92` — registers 9 auth/oauth2/2fa routes as children of `SplashLayout`; `/oauth/callback` and `/oidc/callback` are registered at the top level (no layout wrapper) and both point at `frontend/src/views/OAuthCallback.vue`.
- `frontend/src/router/setup.ts:1-31` — registers `/setup` under `SplashLayout` with `requiredPermissionList` metadata.
- `frontend/src/router/index.ts:85-276` — global `beforeEach` guard enforces:
  - auth-callback bypass (lines 97-115)
  - already-logged-in redirect away from auth pages (lines 143-167)
  - auth-page store reset (lines 169-179)
  - unauthenticated redirect to signin (lines 181-204)
  - `requireMfa` → `/2fa/setup` redirect (lines 206-220, reads `settingV1Store.workspaceProfile.requireMfa`)
  - `authStore.requireResetPassword` → password reset redirect (lines 222-232)

### Vue pages (10 route components, 1,795 LOC)

| Path | File | LOC |
|---|---|---|
| `/auth/` (alias `/auth/signin`) | `frontend/src/views/auth/Signin.vue` | 397 |
| `/auth/admin` | `frontend/src/views/auth/SigninAdmin.vue` | 44 |
| `/auth/signup` | `frontend/src/views/auth/Signup.vue` | 251 |
| `/auth/password-forgot` | `frontend/src/views/auth/PasswordForgot.vue` | 120 |
| `/auth/password-reset` | `frontend/src/views/auth/PasswordReset.vue` | 270 |
| `/auth/mfa` | `frontend/src/views/auth/MultiFactor.vue` | 129 |
| `/oauth2/consent` | `frontend/src/views/OAuth2Consent.vue` | 192 |
| `/oauth/callback`, `/oidc/callback` | `frontend/src/views/OAuthCallback.vue` | 205 |
| `/2fa/setup` | `frontend/src/views/TwoFactorRequired.vue` | 34 (already a thin shell over `TwoFactorSetupPage.tsx`) |
| `/setup` | `frontend/src/views/Setup/Setup.vue` | 31 (+ `AdminSetup.vue` 228, `WorkspaceMode.vue` 75) |

### Shared Vue surface on the critical path

- `frontend/src/layouts/SplashLayout.vue:1-48` — branding-image-left / router-view-right split; gated by `useSubscriptionV1Store()` plan check; route-name switches the illustration.
- `frontend/src/components/BytebaseLogo.vue` — used by `Signin`, `Signup`, `MultiFactor`, `SigninAdmin`, `PasswordForgot`, `PasswordReset`, and elsewhere (still live non-auth callers).
- `frontend/src/components/PasswordSigninForm.vue:1-150`, `EmailCodeSigninForm.vue:1-195`, `DemoSigninForm.vue:1-70` — embedded in `Signin.vue`; no non-auth callers.
- `frontend/src/components/RequiredStar.vue`, `frontend/src/components/UserPassword.vue` — used by auth forms and other Vue pages (still live non-auth callers).
- `frontend/src/views/auth/AuthFooter.vue:1-66` — footer shared by all auth pages and Setup.
- `frontend/src/views/auth/SigninModal.vue:1-43`, `InactiveRemindModal.vue:1-104` — mounted from `frontend/src/AuthContext.vue:8-9` (not route-reached; `InactiveRemindModal` shown when logged in; `SessionExpiredSurface` shown via `frontend/src/components/SessionExpiredSurfaceMount.vue` when `authStore.unauthenticatedOccurred`).

### Stores and RPC clients consumed

- `useAuthStore()` — `login()`, `logout()`, `requireResetPassword`, `isLoggedIn`, `unauthenticatedOccurred`, `currentUser` (`frontend/src/store/modules/v1/auth.ts`).
- `useActuatorV1Store()` — `serverInfo.restriction.{disallowPasswordSignin,disallowSignup,allowEmailCodeSignin}`, `isDemo`, `isSaaSMode`, `activeUserCount`, `fetchServerInfo()`.
- `useIdentityProviderStore()` — `fetchIdentityProviderList(workspaceName?)`, `identityProviderList`.
- `useSubscriptionV1Store()` — `currentPlan`, `isTrialing` (gates `SplashLayout` branding image).
- `useWorkspaceV1Store()`, `useRoleStore()` — loaded in `Setup.vue:22-29`.
- `useSettingV1Store()` — read by router guard for `requireMfa`.
- Connect RPC: `authServiceClientConnect.login` via `authStore.login()`; `LoginRequest` built with `create(LoginRequestSchema, ...)` (`@/types/proto-es/v1/auth_service_pb`).
- OAuth state storage: `frontend/src/utils/sso.ts` (`retrieveOAuthState`, `clearOAuthState`, `openWindowForSSO`) — opaque state tokens kept in `localStorage`.

### Existing React surface

- `frontend/src/react/pages/settings/two-factor/TwoFactorSetupPage.tsx` — real React page mounted by the Vue `TwoFactorRequired.vue` shell.
- `frontend/src/react/components/auth/SigninBridge.tsx:1-109` — mounts `@/views/auth/Signin.vue` inside a child Vue app to present the session-expired relogin UI.
- `frontend/src/react/components/auth/SessionExpiredSurface.tsx:1-35` — React dialog that renders `SigninBridge`; registered in `frontend/src/react/mount.ts:9-11` and mounted from Vue via `frontend/src/components/SessionExpiredSurfaceMount.vue`.
- `frontend/src/react/ReactPageMount.vue:1-59` — generic Vue-in-React mount component that forwards page props, handles locale switching, and unmounts on route change.
- `frontend/src/react/mount.ts:120` — `mountReactPage(container, page, props)` entry used by the Vue shells.

### i18n

- Active namespaces: `auth.sign-in.*`, `auth.sign-up.*`, `auth.password-forgot`, `auth.password-reset.*`, `auth.back-to-signin`, `auth.close-window`, `auth.token-expired-*`, `multi-factor.*`, `two-factor.*`, `oauth.*`, `oauth2.consent.*`, `setup.self`, `common.{sign-in,sign-up,verify,logout,password,username,email,or,close}`.
- React locale files: `frontend/src/react/locales/{en-US,es-ES,ja-JP,vi-VN,zh-CN}.json` — currently hold only the keys touched by existing React surfaces; do not yet include the auth namespaces.

## Non-Goals

- Rewriting `useAuthStore` or any Pinia store. Stores stay and are consumed from React via `useVueState`.
- Rewriting `frontend/src/utils/sso.ts` (OAuth state/token handling) or changing the opaque-state-token protocol.
- Changing `authServiceClientConnect` behavior, proto definitions, or login/challenge request shapes.
- Migrating the global router guard (`frontend/src/router/index.ts:85-276`) from Vue Router to a React router. The Vue router continues to own URL-to-page dispatch.
- Migrating `frontend/src/AuthContext.vue`, `InactiveRemindModal.vue`, or `SessionExpiredSurfaceMount.vue` shells. `SessionExpiredSurface.tsx` is already React; it is in scope only to the extent the removal of `Signin.vue` forces it to consume the new React sign-in instead of the Vue one.
- Introducing TanStack Query, zustand, or any new React state library for this surface. Playbook §State and Data Guidance applies.
- Replacing naive-ui primitives in remaining Vue callers of `BytebaseLogo`, `UserPassword`, `RequiredStar` — those shared components stay Vue until their non-auth callers migrate.
- Adding a distinct `/auth/idp-init` route. The BYT-9167 description mentions it, but no such route exists today; IdP selection is an inline tab/button in `Signin.vue`. Treated as a naming artifact, not a missing route.
- Redesigning any auth UX (layout, copy, flow). Migration is a port, not a redesign.
- Building a React Router surface under `/auth/*`. The route stays Vue; only the page body is swapped.
- Removing `SigninBridge.tsx` or rearchitecting how `SessionExpiredSurface` is mounted. The bridge is swapped to point at the new React sign-in component when `Signin.vue` is deleted; the mount path (`SessionExpiredSurfaceMount.vue`) stays.

## Open Questions

1. Should auth pages ship with a new React `SplashLayout` equivalent, or should the existing Vue `SplashLayout` keep wrapping them and each page mount its body via `ReactPageMount.vue`? (default: keep the Vue `SplashLayout` and mount React page bodies through `ReactPageMount.vue`, mirroring the `TwoFactorRequired.vue` precedent; revisit only after all auth/setup bodies are React.)
2. Should `BytebaseLogo` be moved to React in this migration, or reused via a Vue-in-React bridge like `SigninBridge`? (default: port to React at `frontend/src/react/components/BytebaseLogo.tsx` — already done for SQL Editor Stage 1 — and have remaining Vue callers keep importing the Vue version until they migrate. Two copies coexist, playbook §Shared Component Rule and §Deletion Rule explicitly allow this.)
3. Should `PasswordSigninForm`, `EmailCodeSigninForm`, `DemoSigninForm` be ported as standalone React components, or inlined into the new React `SigninPage`? (default: port as standalone components under `frontend/src/react/components/auth/` so they can be reused by `SigninBridge` / `SessionExpiredSurface` without a second Vue app.)
4. When does `SigninBridge.tsx` stop mounting `Signin.vue`? (default: the PR that deletes `frontend/src/views/auth/Signin.vue` is the same PR that rewrites `SigninBridge` to render the React signin component directly; no intermediate state where both exist.)
5. Do the two router guards that check route names (`AUTH_*_MODULE`, `SETUP_MODULE`) need to change? (default: no — route-name constants stay in `frontend/src/router/auth.ts` / `setup.ts`, and the route components get swapped from Vue SFCs to `ReactPageMount` wrappers with `page="SigninPage"` etc.)
6. Does `/setup` migrate in the same cycle, or defer? (default: migrate in the last phase of this effort — it reuses `SplashLayout` and `AuthFooter`, so it benefits from the same shared React primitives; but ship phases independently.)
7. Does `OAuth2Consent.vue` stay Vue? (default: migrate — it is on the auth critical path, uses the same `SplashLayout`, and BYT-9167 explicitly names it.)
8. Should the `SigninAdmin.vue` backdoor (`/auth/admin`) migrate? (default: yes — it's a thin 44-line variant of `Signin`; port as a `SigninAdminPage.tsx` or as a prop on the shared React signin component.)

## Scope

**L** — Ten Vue pages (~1,795 LOC of route components plus ~415 LOC of shared signin forms and ~114 LOC of layout/footer) touch the authStore-driven router guards, a Vue-mounted React dialog (`SessionExpiredSurface`) that currently re-uses the Vue sign-in component, and a partially-migrated Vue shell (`TwoFactorRequired.vue`). Multiple viable migration orderings exist (shell-flip vs. body-flip vs. island), the session-expired path creates a coupling that must be resolved mid-migration, and no existing design document covers the auth surface. This matches the L criterion: "new subsystem, novel problem, multiple viable approaches."
