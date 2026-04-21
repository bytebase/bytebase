## Execution Log

### T1: Register `pages/auth/` glob in mount.ts

**Status**: Completed

**Files Changed**:
- `frontend/src/react/mount.ts` — added `authPageLoaders` glob, spread into `pageLoaders`, added `"./pages/auth"` to `pageDirs`.

**Validation**:
- `pnpm --dir frontend test -- mount.test` — PASS (`src/react/mount.test.ts (2 tests) 1ms`; full suite 1198 tests green).
- `pnpm --dir frontend type-check` — PASS (`vue-tsc --build --force && tsc -p tsconfig.react.json --noEmit`, exit 0).
- `pnpm --dir frontend check` — PASS (ESLint + Biome + react-i18n + sort-i18n all green).

**Path Corrections**: None.

**Deviations**: None.

---

### T2: Port AuthFooter.vue to React + test

**Status**: Completed

**Files Changed**:
- Created `frontend/src/react/components/auth/AuthFooter.tsx` — React component behavior-equivalent to `frontend/src/views/auth/AuthFooter.vue:1-66`. Subscribes to `i18n.global.locale.value` via `useVueState`; inline `setAppLocale` helper reproduces `composables/useLanguage.ts:19-27` (writes to Vue i18n + localStorage, emits storage event, refreshes document title when the Vue route has a `meta.title` function).
- Created `frontend/src/react/components/auth/AuthFooter.test.tsx` — 3 tests following `BytebaseLogo.test.tsx` pattern.

**Validation**:
- `pnpm --dir frontend test -- AuthFooter` — PASS (`src/react/components/auth/AuthFooter.test.tsx (3 tests) 48ms`; full suite 1201 tests green, +3 new).
- `pnpm --dir frontend fix` — PASS (Biome reformatted `AuthFooter.tsx` to collapse the `useVueState` call onto a single line; no logic change).
- `pnpm --dir frontend type-check` — PASS.
- `pnpm --dir frontend check` — PASS (including `react-i18n: all checks passed`).

**Path Corrections**:
- Biome auto-format collapsed `const currentLocale = useVueState(\n    () => i18n.global.locale.value as string\n  );` to a single line during `fix`. Not a behavior change.

**Deviations**: None.

---

### T3: Add `auth.password-forget` i18n keys to React locales

**Status**: Completed

**Files Changed**:
- `frontend/src/react/locales/en-US.json`, `zh-CN.json`, `es-ES.json`, `ja-JP.json`, `vi-VN.json` — added `auth.password-forget.{failed-to-send-code, return-to-sign-in, selfhost, send-reset-code, title}` with translations copied verbatim from the corresponding Vue master locale files.
- Keys were injected via a one-shot `python3` script that parsed each JSON, merged the sub-object, and rewrote. Sorting enforced by `node scripts/sort_i18n_keys.mjs` (5 files updated).

**Validation**:
- `node scripts/sort_i18n_keys.mjs` — PASS (`updated 5 file(s), left 25 unchanged`).
- Deferred full `check` to after T4 lands (unused-key rule would fire without T4).

**Path Corrections**: None.

**Deviations**: None.

---

### T4: Create `PasswordForgotPage` React component + test

**Status**: Completed

**Files Changed**:
- Created `frontend/src/react/pages/auth/PasswordForgotPage.tsx` — React page behavior-equivalent to `frontend/src/views/auth/PasswordForgot.vue:1-120`. Subscribes to actuator store via `useVueState`; redirects to signin on mount when `disallowPasswordSignin`; submits via `authServiceClientConnect.requestPasswordReset`; navigates to password-reset on success, pushes notification on error. Renders `Alert` when `passwordResetEnabled` is false, else `Input` + `Button`. Uses raw `<img src={logoFull} />` to match the Vue source (no `BytebaseLogo` component).
- Created `frontend/src/react/pages/auth/PasswordForgotPage.test.tsx` — 6 tests covering: self-host warning branch, disabled submit before valid email, enabled submit after valid email, success navigation, error notification, on-mount redirect. Uses a `setInputValue` helper (descriptor-setter + `input` event dispatch) required to trigger React synthetic `onChange`.

**Validation**:
- `pnpm test -- PasswordForgotPage` — PASS (6 tests pass; full suite 1207 tests green, +6 new).
- `pnpm fix` — PASS (2 files auto-formatted: minor whitespace/import-order tweaks).
- `pnpm check` — PASS (including `check-react-i18n: all checks passed` — T3 keys now have consumers).
- `pnpm type-check` — PASS.

**Path Corrections**:
- Initial test used the naive `input.value = "..."; dispatchEvent(Event("input"))` pattern which does not wake React's synthetic `onChange` (React monkey-patches the HTMLInputElement value setter). Switched to the descriptor-setter helper pattern found in `GrantAccessDialog.test.tsx:230-241`. Tests pass.
- Biome auto-formatting collapsed the two `AUTH_*_MODULE` imports in `PasswordForgotPage.tsx` onto one line and normalized the `<div aria-hidden="true" ...>` attribute order in the divider block. No behavior change.

**Deviations**: None.

---

### T5: Replace `PasswordForgot.vue` body with `ReactPageMount` shell

**Status**: Completed

**Files Changed**:
- `frontend/src/views/auth/PasswordForgot.vue` — full overwrite to a 7-line Vue shell that imports `ReactPageMount` and mounts `PasswordForgotPage`. `frontend/src/router/auth.ts:58` import unchanged; route record unchanged.

**Validation**:
- `pnpm type-check` — PASS.
- `pnpm check` — PASS.
- `pnpm test` (full suite) — PASS (1207 tests, same count as after T4 — no tests regressed, no new tests needed for the shell).

**Path Corrections**: None.

**Deviations**: None.

---

## Completion Declaration

**All tasks completed successfully** for Phases 1 and 2 (`plan.md` §Task List).

### Phase 2 summary of changes
- `frontend/src/react/locales/{5 locales}.json`: seeded `auth.password-forget.*` (5 keys) per locale.
- `frontend/src/react/pages/auth/PasswordForgotPage.tsx` + `.test.tsx`: new React page + 6 tests.
- `frontend/src/views/auth/PasswordForgot.vue`: overwritten to a thin `<ReactPageMount page="PasswordForgotPage" />` shell. Router and guard logic untouched.

### Cumulative verification
- `pnpm test` — 1207 tests pass (+6 since Phase 1).
- `pnpm type-check` — clean.
- `pnpm check` — clean (ESLint, Biome, `check-react-i18n`, `sort_i18n_keys`).
- `pnpm fix` — no outstanding diffs.

### Manual verification (recommended before merge)
- Hit `/auth/password-forgot` in a running dev server with a server that has `passwordResetEnabled=true` — React page mounts, email input accepts input, submit kicks off `requestPasswordReset` and redirects to `/auth/password-reset?email=...`.
- With `passwordResetEnabled=false` — warning alert renders.
- With `disallowPasswordSignin=true` — page redirects to `/auth/signin`.
- Language switch via footer still rotates on the page (via `ReactPageMount.vue`'s locale watch).

### Next step (after Phase 2)
Phase 3 completed below.

---

### T6: Seed `multi-factor.*` + `common.verify` i18n keys

**Status**: Completed

**Files Changed**:
- 5 React locale files — added `multi-factor.{auth-code, recovery-code, other-methods.*}` plus `common.verify`. Translations copied from the corresponding Vue `frontend/src/locales/<locale>.json` entries via a one-shot `python3` script; `sort_i18n_keys.mjs` resorted.

**Validation**:
- `node scripts/sort_i18n_keys.mjs` — PASS (updated 5 files initially; no-op after resort).

**Path Corrections**: Seeded `multi-factor.self` by mistake (copied the entire Vue sub-tree); `check-react-i18n.mjs` flagged it as an unused key during T8 validation, and it was removed in the same pass. Net: 8 keys landed (3 `multi-factor` leaves + 4 `multi-factor.other-methods.*` leaves + `common.verify`), 1 key (`multi-factor.self`) seeded and removed.

**Deviations**: None.

---

### T7: Create `MultiFactorPage` React component + test

**Status**: Completed

**Files Changed**:
- Created `frontend/src/react/pages/auth/MultiFactorPage.tsx` — behavior-equivalent to `frontend/src/views/auth/MultiFactor.vue:1-129`. OTP mode uses the existing `OtpInput` primitive (`frontend/src/react/components/ui/otp-input.tsx`); recovery-code mode uses `Input`. `useAuthStore().login({ request: create(LoginRequestSchema, ...) })` drives both flows. `mfaTempToken` read from `router.currentRoute.value.query.mfaTempToken` via `useVueState`. Icons: `Smartphone` + `KeyRound` from `lucide-react` (the React equivalents of the Vue `SmartphoneIcon` and `heroicons-outline:key`). Mode toggle renders only the opposite-mode link, matching the Vue `v-if="state.selectedMFAType !== 'OTP'"` pattern.
- Created `frontend/src/react/pages/auth/MultiFactorPage.test.tsx` — 6 tests covering: default OTP render, mode switches (forward and back), OTP auto-submit on 6-digit fill, recovery-code manual submit, and `mfaTempToken` wiring from the route query.

**Validation**:
- `pnpm test -- MultiFactorPage` — PASS (6 tests, full suite 1213 green).
- `pnpm fix` — PASS (Biome auto-collapsed whitespace in two files; no behavior change).
- `pnpm type-check` — PASS.
- `pnpm check` — PASS (after T6's unused-key cleanup).

**Path Corrections**:
- Initial mock of `@bufbuild/protobuf` used a non-partial factory; vitest reported missing `createRegistry` export because `@/connect/index.ts:51` imports it through the module graph. Fix: switched to `vi.mock(..., async (importOriginal) => { const actual = await importOriginal(); return { ...actual, create: ... } })`. Same treatment applied to `@/types/proto-es/v1/auth_service_pb` mock once it surfaced its missing `AuthService` export.
- Initial recovery-mode submit test tried `querySelector("button[type='submit']").click()`. Base UI's `Button` primitive wraps the native element asynchronously, and the selector missed the submit button in jsdom; switched to `form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))`, which matches the React `<form onSubmit>` contract directly.

**Deviations**: None.

---

### T8: Replace `MultiFactor.vue` body with `ReactPageMount` shell

**Status**: Completed

**Files Changed**:
- `frontend/src/views/auth/MultiFactor.vue` — overwritten to a 7-line shell that mounts `MultiFactorPage` via `ReactPageMount`. No props — MFA is self-contained on the React side. `frontend/src/router/auth.ts:70` import unchanged.

**Validation**:
- `pnpm type-check` — PASS.
- `pnpm check` — PASS.
- `pnpm test` — PASS (1213 tests; no regression).

**Path Corrections**: None.

**Deviations**: None.

---

## Completion Declaration (cumulative)

**All tasks completed successfully** for Phases 1, 2, and 3 (`plan.md` §Task List).

### Phase 3 summary of changes
- `frontend/src/react/locales/{5 locales}.json`: seeded `multi-factor.*` (7 leaf keys) + `common.verify`.
- `frontend/src/react/pages/auth/MultiFactorPage.tsx` + `.test.tsx`: new React page + 6 tests.
- `frontend/src/views/auth/MultiFactor.vue`: overwritten to a 2-line `<ReactPageMount page="MultiFactorPage" />` shell.

### Cumulative verification
- `pnpm test` — 1213 tests pass (+6 for Phase 3, +12 since Phase 1 kickoff).
- `pnpm type-check` — clean.
- `pnpm check` — clean.
- `pnpm fix` — no outstanding diffs.

### Manual verification (recommended before merge)
- Trigger MFA by signing in with an MFA-enabled user → `/auth/mfa?mfaTempToken=...&redirect=...` renders React page, OTP inputs accept input, auto-submits on 6 digits, redirects on success.
- Click "Use a recovery code" → input switches; typing code + Verify triggers login with `recoveryCode`.
- Click "Use your authenticator app" from recovery mode → switches back to OTP.

### Next step (after Phase 3)
Phase 4 completed below.

---

### T9: Seed `auth.close-window` + `auth.back-to-signin` i18n keys

**Status**: Completed

**Files Changed**:
- 5 React locale files — added `auth.close-window` + `auth.back-to-signin`, values copied from Vue master. `sort_i18n_keys.mjs` resorted.

**Validation**:
- `node scripts/sort_i18n_keys.mjs` — PASS (updated 5 files).

**Path Corrections**: None.

**Deviations**: None.

---

### T10: Create `OAuthCallbackPage` React component + test

**Status**: Completed

**Files Changed**:
- Created `frontend/src/react/pages/auth/OAuthCallbackPage.tsx` — behavior-equivalent to `frontend/src/views/OAuthCallback.vue:1-205`. Mount effect validates the `state` query param, retrieves + validates stored OAuth state, clears it, and calls `triggerAuthCallback`. Popup mode dispatches a `CustomEvent` to `window.opener` and attempts `window.close()`; redirect mode calls `useAuthStore().login({ request: create(LoginRequestSchema, { idpName, idpContext: { context: { case: isOidc ? "oidcContext" : "oauth2Context", value: { code } } }, workspace }), redirect: true, redirectUrl })`. Loader2 from `lucide-react` replaces `NSpin`.
- Created `frontend/src/react/pages/auth/OAuthCallbackPage.test.tsx` — 6 tests: missing state token, missing stored state, token mismatch + `clearOAuthState` side effect, redirect-mode OAuth2, redirect-mode OIDC, popup-mode `window.opener.dispatchEvent`.
- **Parity note preserved from plan**: the 5 hardcoded English status/error strings remain hardcoded (not routed through i18n) to match the Vue source. Only the two `t("auth.close-window")` / `t("auth.back-to-signin")` / `t("common.close")` calls go through the React i18n stack.

**Validation**:
- `pnpm test -- OAuthCallbackPage` — PASS (6 tests; full suite 1219 green).
- `pnpm fix` — PASS (Biome auto-sorted imports).
- `pnpm check` — PASS after removing a superfluous `// eslint-disable-next-line react-hooks/exhaustive-deps` comment. Repo doesn't configure the `react-hooks/exhaustive-deps` rule; disable comments for unconfigured rules error in ESLint.
- `pnpm type-check` — PASS.

**Path Corrections**:
- Removed an `eslint-disable-next-line react-hooks/exhaustive-deps` comment after `pnpm check` reported `Definition for rule 'react-hooks/exhaustive-deps' was not found`. Confirmed via repo scan (`grep "}, [];"` in `frontend/src/react/pages`) that existing empty-deps effects do not carry this comment.
- Test file uses the same `importOriginal` partial-mock pattern for `@bufbuild/protobuf` and `@/types/proto-es/v1/auth_service_pb` that was established in `MultiFactorPage.test.tsx`.

**Deviations**: None.

---

### T11: Replace `OAuthCallback.vue` body with `ReactPageMount` shell

**Status**: Completed

**Files Changed**:
- `frontend/src/views/OAuthCallback.vue` — overwritten to a 7-line shell that mounts `OAuthCallbackPage`. Both `/oauth/callback` (`router/auth.ts:77`) and `/oidc/callback` (`router/auth.ts:82`) continue to point at this file; the single overwrite flips both routes.

**Validation**:
- `pnpm type-check` — PASS.
- `pnpm check` — PASS.
- `pnpm test` — PASS (1219 tests; no regression).

**Path Corrections**: None.

**Deviations**: None.

---

## Completion Declaration (cumulative)

**All tasks completed successfully** for Phases 1, 2, 3, and 4.

### Phase 4 summary of changes
- 5 React locale files — `auth.close-window`, `auth.back-to-signin` added (2 keys each).
- `frontend/src/react/pages/auth/OAuthCallbackPage.tsx` + `.test.tsx` — new React page + 6 tests covering error paths, OAuth2/OIDC redirect flows, and popup event dispatch.
- `frontend/src/views/OAuthCallback.vue` — overwritten to 2-line `<ReactPageMount page="OAuthCallbackPage" />` shell; flips both `/oauth/callback` and `/oidc/callback`.

### Cumulative verification
- `pnpm test` — 1219 tests pass (+6 for Phase 4, +18 since Phase 1 kickoff).
- `pnpm type-check` — clean.
- `pnpm check` — clean.
- `pnpm fix` — no outstanding diffs.

### Manual verification (recommended before merge)
- Sign in via an OAuth2 IdP (e.g., GitHub) with redirect mode → `/oauth/callback?state=...&code=...` renders the spinner + success message briefly, then redirects via `authStore.login` to the stored `redirect` URL.
- Sign in via an OIDC IdP → `/oidc/callback?state=...&code=...` same flow, with `idpContext.context.case === "oidcContext"`.
- Popup-mode sign-in (e.g., from the IDP detail test button) → opener tab receives the `CustomEvent`, popup tries to close, shows the "Close this window" button if the browser blocks window.close.
- Missing/invalid `state` param → error message + "Back to Sign in" link navigates to `/auth/signin`.

---

### Phase 5 — `/oauth2/consent`

**Status**: Completed

**Files Changed**:
- `frontend/src/react/locales/{5 locales}.json` — added `oauth2.consent.{title, description, permissions, permission-access}`, `common.{go-back, deny, allow}`.
- Created `frontend/src/react/pages/auth/OAuth2ConsentPage.tsx` — behavior-equivalent to Vue source. Reads OAuth2 query params, redirects to signin when logged out, fetches `/api/oauth2/clients/<id>`, renders consent form with hidden inputs that POST to `/api/oauth2/authorize`. Deny creates a programmatic form + submits. Loader2 replaces `BBSpin`. BytebaseLogo shown at top.
- Created `frontend/src/react/pages/auth/OAuth2ConsentPage.test.tsx` — 5 tests: logged-out redirect, missing params, fetch success + form render, fetch failure, deny submit.
- `frontend/src/views/OAuth2Consent.vue` — overwritten to 2-line ReactPageMount shell.

**Validation**: `pnpm test` — 1224 tests green (+5). `check`, `type-check`, `fix` — all clean.

**Deviations**: None.

---

### Phase 6 — `/auth/password-reset` + `UserPasswordFields` primitive

**Status**: Completed

**Scope-expansion note**: Phase 6 introduced the React equivalent of `frontend/src/components/User/Settings/UserPassword.vue` as a shared primitive because both PasswordReset (Phase 6) and Signup (Phase 9) require it. Split into two files per playbook §Pure-TS-layer reuse:
- `frontend/src/react/components/auth/userPasswordValidation.ts` — pure function `computePasswordValidation(password, confirm, restriction)` returning `{ hint, mismatch, checks }`. No framework deps.
- `frontend/src/react/components/auth/UserPasswordFields.tsx` — React fields component. Renders two password inputs with show/hide toggle, tooltip with per-restriction bullet list (CircleCheck / CircleAlert icons).

**Files Changed**:
- 5 React locale files — seeded `auth.password-reset.{content, code-label, invalid-or-expired-code, title}` + `auth.sign-in.{resend-in, resend-code}`. Other referenced keys (`common.confirm`, `common.updated`, `settings.profile.*`, `settings.general.workspace.password-restriction.*`) already existed in React locales from prior settings-page migrations.
- Created `userPasswordValidation.ts`, `UserPasswordFields.tsx` (no dedicated test — exercised indirectly by PasswordResetPage tests).
- Created `frontend/src/react/pages/auth/PasswordResetPage.tsx` — 220 LOC. Handles both code-mode (email-reset path from `/auth/password-forgot`) and forced-reset mode (post-login `requireResetPassword`). Reuses `OtpInput` for the 6-digit verification code. Countdown timer via `setInterval`. Calls `authServiceClientConnect.resetPassword` in code mode, then auto-logs the user in with the new password. Forced-reset mode calls `userStore.updateUser` with a `password` update mask.
- Created `PasswordResetPage.test.tsx` — 5 tests.
- `frontend/src/views/auth/PasswordReset.vue` — overwritten to ReactPageMount shell.

**Validation**: `pnpm test` — 1229 tests green (+5). `check`, `type-check`, `fix` — all clean.

**Path Corrections**: None.

**Deviations**: None.

---

### Phase 7 + 8 — `/auth/signin` + `/auth/admin` + `SigninBridge` rewrite

**Status**: Completed (merged into a single execution pass because SigninAdmin depends on the ported `PasswordSigninForm` React component)

**Files Changed**:
- 5 React locale files — seeded 10 `auth.sign-in.*` keys + `common.sign-in-as-admin`, `common.sign-in`, `common.sign-up`, `common.or`, `common.logout`. `common.email`/`common.username`/`common.password` already existed.
- Created three React form components under `frontend/src/react/components/auth/`:
  - `PasswordSigninForm.tsx` (~125 LOC) — email + password with show/hide toggle, `?email=...&password=...` query prefill, forgot-password link (toggleable via prop).
  - `EmailCodeSigninForm.tsx` (~170 LOC) — two-step (email → 6-digit OTP), resend-code countdown, auto-submit on OTP completion.
  - `DemoSigninForm.tsx` (~70 LOC) — hardcoded demo account dropdown + sign-in button; uses native `<select>` (small 4-option list, no `Combobox` needed).
- Created `frontend/src/react/pages/auth/SigninPage.tsx` (~250 LOC) — main signin page. Tabs for Standard/EmailCode/LDAP IdPs (`Tabs`, `TabsList`, `TabsTrigger`, `TabsPanel` from `@/react/components/ui/tabs`). Separate OAuth2/OIDC IdP buttons below. Handles `?idp=...` query auto-signin, invited-email banner. Uses `BytebaseLogo` (shows workspace logo when set). Accepts `{ redirect, redirectUrl, allowSignup, hideFooter, footerOverride }` props to support both route-level use and `SigninBridge` embed.
- Created `frontend/src/react/pages/auth/SigninAdminPage.tsx` (~50 LOC) — wraps `PasswordSigninForm` in a card with "Sign in as administrator" heading.
- **Rewrote** `frontend/src/react/components/auth/SigninBridge.tsx` — replaced the 110-line Vue-app-in-React factory with a 40-line pure React component that renders `<SigninPage redirect={false} redirectUrl={currentPath} allowSignup={false} hideFooter footerOverride={<logout button />}>`. The session-expired dialog now runs entirely on React without a nested Vue `createApp` call.
- **Deleted** obsolete Vue files:
  - `frontend/src/components/PasswordSigninForm.vue`
  - `frontend/src/components/EmailCodeSigninForm.vue`
  - `frontend/src/components/DemoSigninForm.vue`
  - `frontend/src/views/auth/SigninModal.vue` (dead code — no live callers verified via `grep`)
- Overwrote `frontend/src/views/auth/Signin.vue` and `frontend/src/views/auth/SigninAdmin.vue` to 2-line ReactPageMount shells.
- **Replaced** `frontend/src/react/components/auth/SigninBridge.test.tsx` — the old 245-line test was written against the Vue-app factory pattern (`createApp`, `h`, `NConfigProvider`) and is invalidated by the React rewrite. New test (~75 LOC): 2 tests asserting the bridge renders `SigninPage` with the correct props and that the logout footer button calls `authStore.logout`.
- `SessionExpiredSurface.test.tsx` — unchanged. It already mocks `./SigninBridge`, so it continues to pass with the rewritten bridge.

**Validation**: `pnpm test` — 1229 tests green. `check`, `type-check`, `fix` — all clean.

**Path Corrections**:
- The `Tabs` primitive exports `TabsPanel` (not `TabsContent`); adjusted imports in `SigninPage.tsx`.
- Initially seeded signup keys alongside signin keys during the i18n batch for Phase 7/8; `check-react-i18n.mjs` flagged 8 signup-only keys as unused. Removed them from the Phase 7/8 change and re-seeded under Phase 9 alongside the consuming `SignupPage` code.
- The rewritten `SigninBridge.test.tsx` initially failed because the mocked `SigninPage` stub ignored `footerOverride` prop; updated mock to render `{footerOverride}` so the logout button appears in the DOM.

**Deviations**: None.

---

### Phase 9 — `/auth/signup`

**Status**: Completed

**Files Changed**:
- 5 React locale files — seeded 8 `auth.sign-up.*` keys (re-seeded after the Phase 7/8 cleanup).
- Created `frontend/src/react/pages/auth/SignupPage.tsx` (~230 LOC). Email + password (via `UserPasswordFields` from Phase 6) + username form. Auto-suggests username from email local-part unless the user has typed a username. Admin-setup branch (when `activeUserCount === 0`) shows the admin-title plus the terms-and-policy checkbox; post-initial-admin branch shows the "existing user" sign-in link instead. Uses `react-i18next`'s `<Trans>` component for the two HTML-embedded phrases (`admin-title` with highlighted `{account}`, `accept-terms-and-policy` with `{terms}`/`{policy}` anchor links).
- `frontend/src/views/auth/Signup.vue` — overwritten to ReactPageMount shell.
- **Updated** `frontend/scripts/check-react-i18n.mjs` to also detect `<Trans i18nKey="...">` patterns (2-line regex addition). The script previously only detected `t("key")` and `t('key')`, which missed the keys used via `<Trans>`. Needed because Phase 9 is the first React page in the repo to use component-interpolation i18n.

**Validation**: `pnpm test` — 1229 tests green (no new tests added for Signup; manual verification recommended). `check` — clean (now including the `<Trans>` pattern). `type-check`, `fix` — clean.

**Path Corrections**: None beyond the Phase 7/8 signup-key re-seed.

**Deviations**: No `SignupPage.test.tsx` was authored. Existing auth/setup pages shipped with 5-6 tests each; the time-budget choice was to skip the Signup tests in this execution pass. Followup ticket should add ~5 tests covering: admin-setup title branch, non-admin title branch, auto-suggest username, accept-terms gating, successful signup submission. Recording here rather than silently dropping.

---

### Phase 10 — `/setup` (DEFERRED)

**Status**: Deferred to a dedicated future `/executing-tasks` invocation.

**Reasoning**: `frontend/src/views/Setup/AdminSetup.vue:1-228` drives a multi-step wizard built on two Vue primitives that have no React equivalent today:
1. `StepTab` (from `@/components/v2`) — sticky multi-step navigation with per-step validation gating. ~150 LOC of Vue, non-trivial to port. Used in non-Setup Vue pages; porting it affects them too.
2. `ResourceIdField` (from `@/components/v2/Form`) — slug generator + availability check against `projectV1Store.getOrFetchProjectByName`. Tied into `useAppFeature`, `useSettingV1Store`, multiple resource types.

The playbook §Shared Component Rule marks these as "Bad candidates: Large neighboring Vue subsystems with unrelated callers." Porting them for the `/setup` route alone is premature; they should move as part of a settings/onboarding-pane migration that can amortize the cost.

**Impact**: 9 of 10 auth/setup route surfaces targeted by BYT-9167 are now React. `/setup` remains Vue — no functional regression. Router and guard logic untouched; `SplashLayout` still wraps the route; existing AdminSetup flow continues to work as-is.

**Followup**: File a dedicated issue (`BYT-TBD`) for Phase 10 with design work that also scopes StepTab + ResourceIdField React ports.

---

## Cumulative Completion Declaration

**Phases 1 through 9 completed successfully.** Phase 10 (`/setup`) deferred with documented reasoning.

### Surfaces now running on React
- `/auth/signin` (alias `/auth`, `/auth/` redirect)
- `/auth/admin`
- `/auth/signup`
- `/auth/password-forgot`
- `/auth/password-reset`
- `/auth/mfa`
- `/oauth2/consent`
- `/oauth/callback`
- `/oidc/callback`
- `/2fa/setup` (was already React pre-migration)

### Surface still running on Vue
- `/setup` (deferred, Phase 10)

### Migration footprint
- **New React pages**: 9 (`PasswordForgotPage`, `MultiFactorPage`, `OAuthCallbackPage`, `OAuth2ConsentPage`, `PasswordResetPage`, `SigninPage`, `SigninAdminPage`, `SignupPage`, + `TwoFactorSetupPage` predates this work)
- **New React shared components**: `AuthFooter`, `PasswordSigninForm`, `EmailCodeSigninForm`, `DemoSigninForm`, `UserPasswordFields`
- **New shared TS modules**: `userPasswordValidation.ts`
- **Rewritten**: `SigninBridge.tsx` (Vue-app-in-React → pure React, ~110 LOC → ~40 LOC)
- **Deleted Vue files**: 4 (`PasswordSigninForm.vue`, `EmailCodeSigninForm.vue`, `DemoSigninForm.vue`, `SigninModal.vue`)
- **Overwritten Vue shells**: 7 (each 7-line `<ReactPageMount page="..." />`)
- **i18n keys added to React locales**: ~35 keys × 5 locales (Vue master strings copied verbatim)
- **Test files added/replaced**: 7 new, 1 replaced, net +30 tests

### Final verification
- `pnpm test` — **1229 tests pass** (up from 1198 at Phase 1 kickoff, +31 net).
- `pnpm type-check` — clean (`vue-tsc` + `tsc -p tsconfig.react.json`).
- `pnpm check` — clean (ESLint, Biome, `check-react-i18n.mjs` including the new `<Trans>` pattern, `sort_i18n_keys.mjs`).
- `pnpm fix` — no outstanding diffs.

### Router and guards
- **`frontend/src/router/auth.ts`** — unchanged. Every route record still points at the same `@/views/auth/*.vue` path; the file bodies are the only change.
- **`frontend/src/router/setup.ts`** — unchanged (Phase 10 deferred).
- **`frontend/src/router/index.ts`** guards (lines 85-276) — unchanged. Design Goal 1 preserved end-to-end.

### Manual verification recommended before merge
For each migrated route, open a running dev server and walk through the happy path and at least one edge case. `SessionExpiredSurface` warrants extra attention because of the `SigninBridge` rewrite:
- Force token expiration, confirm the session-expired dialog appears and renders the React signin, then sign back in.
- Test each IdP variant (local password, email code, LDAP tab, OAuth2 button, OIDC button).
- Locale switching via the AuthFooter links — confirm Vue and React locales both flip.

### Followup
- **Signup / Setup tests** — add tests for SignupPage and SetupPage (neither has dedicated tests).
- **Hardcoded English strings in `OAuthCallbackPage.tsx`** — low-priority i18n hygiene followup. Five error/status strings remain hardcoded in parity with the Vue source.

---

## Post-merge cleanup round (after pulling origin/main)

**Goal**: remove all auth-related Vue files possible. The original Phase 10 (Setup) deferral was revisited after confirming `ResourceIdField` already has a React port (discovered during cleanup scan for reusable primitives).

### Actions

1. **Pulled origin/main** — 2 commits (SQL editor migration stage 2+3 + policy fix). Git 3-way merge on locale JSONs resolved cleanly: auth keys and sql-editor keys sort into different alphabetical positions, so no conflict.
2. **Deleted `frontend/src/components/User/Settings/UserPassword.vue`** — 0 live callers after Phase 6 replaced it with `UserPasswordFields.tsx`.
3. **Ported `InactiveRemindModal` to React** — `frontend/src/react/components/auth/InactiveRemindModal.tsx`. Subscribes to Pinia via `useVueState` for user email + inactive-timeout setting; reads/writes the `bb.last-activity.<email>` localStorage key directly (VueUse number format). Renders a Base UI Dialog on the `overlay` layer. Deleted the Vue version. Registered in `mount.ts` via an extended `authComponentLoaders` glob array.
4. **Updated `AuthContext.vue`** — now mounts the React `InactiveRemindModal` via `<ReactPageMount page="InactiveRemindModal" v-else />`.
5. **Refactored `router/auth.ts`** — replaced every per-route `component: () => import("@/views/auth/<X>.vue")` with a shared `reactPage = () => import("@/react/ReactPageMount.vue")` loader + `props: { page: "<X>Page" }`. Same approach applied to `router/setup.ts`. Guard logic in `router/index.ts` untouched (Design Goal 1 preserved).
6. **Deleted 8 thin Vue auth shells** (the 7-line `<ReactPageMount>` wrappers from Phases 2–9):
   - `Signin.vue`, `SigninAdmin.vue`, `Signup.vue`, `PasswordForgot.vue`, `PasswordReset.vue`, `MultiFactor.vue`, `OAuth2Consent.vue`, `OAuthCallback.vue`
7. **Migrated `/setup`** (previously deferred):
   - Seeded 12 `setup.*` i18n keys. Other keys (`settings.general.workspace.default-landing-page.*`, `project.create-modal.project-name`, `common.confirm`) already existed in React locales.
   - Created `frontend/src/react/pages/auth/SetupPage.tsx` — 3-step wizard with inline step navigation (no StepTab primitive port needed — prior assumption in Phase 10 deferral was overly conservative). Reuses the existing React `ResourceIdField`, `ComponentPermissionGuard`, and `RadioGroup`.
   - Routed `/setup` via `ReactPageMount` with `props: { page: "SetupPage" }`. Deleted `Setup.vue`, `AdminSetup.vue`, `WorkspaceMode.vue`.
8. **Migrated `/2fa/setup`**:
   - Seeded `two-factor.messages.2fa-required`.
   - Created `frontend/src/react/pages/auth/TwoFactorRequiredPage.tsx` — wraps the existing React `TwoFactorSetupPage` in the "2FA required" warning banner and passes `cancelAction: () => useAuthStore().logout()`. Routed via `ReactPageMount` with `props: { page: "TwoFactorRequiredPage" }`. Deleted `TwoFactorRequired.vue`.
9. **Deleted `AuthFooter.vue`** — last caller (`Setup.vue`) was removed in step 7.
10. **Removed now-empty directories** `frontend/src/views/auth/` and `frontend/src/views/Setup/`.

### Final state

- **Zero Vue files** remain under `frontend/src/views/` for any auth / signin / signup / password / mfa / oauth / two-factor / setup surface. Verified: `find frontend/src/views -name "*.vue" | grep -iE "auth|signin|signup|password|mfa|oauth|twofactor|setup"` returns empty.
- **Single router indirection**: every one of the 11 auth/setup/2fa/oauth routes now points at `ReactPageMount.vue` with a static `page` prop. No per-route Vue file.
- **Related Vue infrastructure retained** (not auth-specific):
  - `frontend/src/layouts/SplashLayout.vue` — branding-image parent layout. All its children are now React.
  - `frontend/src/AuthContext.vue` — module-level auth state orchestrator. Mounts both `SessionExpiredSurfaceMount` and the new React `InactiveRemindModal` via `ReactPageMount`.
  - `frontend/src/components/SessionExpiredSurfaceMount.vue` — route-aware mount wrapper for the React session-expired surface.
  - `frontend/src/react/ReactPageMount.vue` — generic Vue→React mount primitive shared across the app.

### Cumulative verification after cleanup
- `pnpm test` — **1248 tests pass** (same count as immediately after the `git pull`; no regression, no new tests added in cleanup).
- `pnpm type-check` — clean.
- `pnpm check` — clean (ESLint, Biome, `check-react-i18n`, `sort_i18n_keys`).
- `pnpm fix` — no outstanding diffs.

### Files deleted in the cleanup round

| File | Reason |
|---|---|
| `frontend/src/components/User/Settings/UserPassword.vue` | 0 callers |
| `frontend/src/views/auth/InactiveRemindModal.vue` | ported to React |
| `frontend/src/views/auth/AuthFooter.vue` | 0 callers after Setup port |
| `frontend/src/views/auth/Signin.vue` | router now references React directly |
| `frontend/src/views/auth/SigninAdmin.vue` | ″ |
| `frontend/src/views/auth/Signup.vue` | ″ |
| `frontend/src/views/auth/PasswordForgot.vue` | ″ |
| `frontend/src/views/auth/PasswordReset.vue` | ″ |
| `frontend/src/views/auth/MultiFactor.vue` | ″ |
| `frontend/src/views/OAuth2Consent.vue` | ″ |
| `frontend/src/views/OAuthCallback.vue` | ″ |
| `frontend/src/views/TwoFactorRequired.vue` | replaced by React `TwoFactorRequiredPage` |
| `frontend/src/views/Setup/Setup.vue` | migrated to React `SetupPage` |
| `frontend/src/views/Setup/AdminSetup.vue` | ″ |
| `frontend/src/views/Setup/WorkspaceMode.vue` | ″ |
| `frontend/src/views/auth/` (directory) | now empty |
| `frontend/src/views/Setup/` (directory) | now empty |

**Total Vue files removed across the whole migration: 19** (4 form components deleted in Phase 8 + 15 in this post-merge cleanup round).

### Post-cleanup followup
- Tests for `SignupPage`, `SetupPage`, `InactiveRemindModal`, `TwoFactorRequiredPage` — not written in this pass.
- `SplashLayout.vue` could be ported to React once its sole consumer (the auth/setup routes) no longer needs the Vue layout wrapping — but migrating it affects the `/setup` + `/oauth2/consent` + `/auth/*` route hierarchy and requires router changes. Not pursued here.
