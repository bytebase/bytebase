## Task List

This is the cumulative plan for `2026-04-21-auth-setup-react-migration`. Completed phases remain in the log; the active phase is what's currently being executed or planned.

---

## Phase 1 — Shared primitives + infrastructure (COMPLETED)

**Task Index**: T1 Register `pages/auth/` glob in mount.ts [S] — T2 Port AuthFooter.vue to React + test [M]

See `execution.md` for the completed log. Deliverables:
- `frontend/src/react/mount.ts` registers `./pages/auth/*.tsx` and lists `"./pages/auth"` in `pageDirs`.
- `frontend/src/react/components/auth/AuthFooter.tsx` + `AuthFooter.test.tsx`.

---

## Phase 2 — `/auth/password-forgot` migration (ACTIVE)

First real page migration. Validates the end-to-end shape: React page under `pages/auth/`, thin Vue shell that mounts it, i18n keys landing alongside their consuming code.

### Implementation approach — file-replacement vs. new-shell

`design.md` §Route-file pattern says "the same route record points at a new Vue shell `frontend/src/views/auth/PasswordForgotPage.vue`". However, the live precedent in the repo (`frontend/src/views/TwoFactorRequired.vue:1-34`) **overwrites the existing Vue file in place** with the mount-shell body, leaving the router import untouched. We follow the live precedent: **replace `PasswordForgot.vue` body in place**, do not create a new shell file, do not touch `frontend/src/router/auth.ts:58`. Design Goal 1 (zero router-guard changes) extends naturally to zero router-file changes.

### Task Index

T3: Add `auth.password-forget` i18n keys to React locales [S] — T4: Create `PasswordForgotPage` React component + test [M] — T5: Replace `PasswordForgot.vue` body with `ReactPageMount` shell [S]

---

### T3: Add `auth.password-forget` i18n keys to React locales [S]

**Objective**: Pre-seed the 5 translation keys the new `PasswordForgotPage.tsx` (T4) will reference, so T4's `t("auth.password-forget.<key>")` calls pass `check-react-i18n.mjs`. T4 technically blocks on T3; this ordering is just for execution clarity — T3 is committed alongside T4 in one PR.

**Files**:
- Modify: `frontend/src/react/locales/en-US.json`, `frontend/src/react/locales/zh-CN.json`, `frontend/src/react/locales/es-ES.json`, `frontend/src/react/locales/ja-JP.json`, `frontend/src/react/locales/vi-VN.json`

**Implementation**:
- In each of the 5 React locale files, add (at the top level) an `"auth"` key with a `"password-forget"` sub-object containing exactly these 5 sub-keys — values copied verbatim from `frontend/src/locales/<locale>.json`'s `auth.password-forget.*`:
  - `failed-to-send-code`, `return-to-sign-in`, `selfhost`, `send-reset-code`, `title`
- Do NOT add any other `auth.*` keys (e.g., `auth.back-to-signin`, `auth.close-window`) — future phases land those.
- Do NOT add `common.email` — already present at `common.email` in all 5 React locale files.
- After edits, run `pnpm --dir frontend fix` — `scripts/sort_i18n_keys.mjs` will re-sort alphabetically.

**Validation**:
- `pnpm --dir frontend check` — expect: `check-react-i18n.mjs` reports all checks passed. Note: this check **will fail** if run before T4 lands, because the added keys are unused. Run the full Phase 2 validation only after T4.

---

### T4: Create `PasswordForgotPage` React component + test [M]

**Objective**: Ship a React page behavior-equivalent to `frontend/src/views/auth/PasswordForgot.vue:1-120`. Consumes the i18n keys from T3.

**Size**: M — one new page file + one new test file. No edits to existing files.

**Files**:
- Create: `frontend/src/react/pages/auth/PasswordForgotPage.tsx`
- Test: `frontend/src/react/pages/auth/PasswordForgotPage.test.tsx`

**Implementation**:

1. In `frontend/src/react/pages/auth/PasswordForgotPage.tsx`:
   - **Imports**: `useEffect`, `useState` from `react`; `useTranslation` from `react-i18next`; `Alert` + `AlertTitle` from `@/react/components/ui/alert`; `Button` from `@/react/components/ui/button`; `Input` from `@/react/components/ui/input`; `useVueState` from `@/react/hooks/useVueState`; `authServiceClientConnect` from `@/connect`; `router` from `@/router`; `AUTH_PASSWORD_RESET_MODULE`, `AUTH_SIGNIN_MODULE` from `@/router/auth`; `useActuatorV1Store`, `pushNotification` from `@/store`; `isValidEmail`, `resolveWorkspaceName` from `@/utils`; `logoFull` (default import) from `@/assets/logo-full.svg`.
   - **Exported function**: `export function PasswordForgotPage()`.
   - **State**: `const [email, setEmail] = useState("");`, `const [isLoading, setIsLoading] = useState(false);`
   - **Store subscriptions** (via `useVueState`):
     - `const passwordResetEnabled = useVueState(() => useActuatorV1Store().serverInfo?.restriction?.passwordResetEnabled ?? false);`
     - `const disallowPasswordSignin = useVueState(() => useActuatorV1Store().serverInfo?.restriction?.disallowPasswordSignin ?? false);`
   - **Mount-time redirect** (matches Vue `onMounted` at `PasswordForgot.vue:89-93`):
     ```
     useEffect(() => {
       if (disallowPasswordSignin) {
         router.replace({ name: AUTH_SIGNIN_MODULE, query: router.currentRoute.value.query });
       }
     }, [disallowPasswordSignin]);
     ```
   - **Submit handler** — early-return if `!isValidEmail(email) || isLoading`; set loading, call `authServiceClientConnect.requestPasswordReset({ email, workspace: resolveWorkspaceName() })`; on success `router.push({ name: AUTH_PASSWORD_RESET_MODULE, query: { ...router.currentRoute.value.query, email } })`; on error `pushNotification({ module: "bytebase", style: "CRITICAL", title: t("auth.password-forget.failed-to-send-code") })`; always clear loading.
   - **JSX** — single `<div className="mx-auto w-full max-w-sm">` matching the Vue template structure:
     - Header block: raw `<img src={logoFull} alt="Bytebase" className="h-12 w-auto" />` (parity with Vue which bypasses `BytebaseLogo` and uses the raw SVG — do NOT substitute the `BytebaseLogo` React component, which adds workspace-custom-logo behavior). Heading `<h2 className="mt-6 text-3xl leading-9 font-extrabold text-main">{t("auth.password-forget.title")}</h2>`.
     - Body (`mt-8 > mt-6 flex flex-col gap-y-4`):
       - If `!passwordResetEnabled`: `<Alert variant="warning"><AlertTitle>{t("auth.password-forget.selfhost")}</AlertTitle></Alert>`.
       - Else: email label + `Input` (`id="forgot-email"`, `type="email"`, `autoComplete="email"`, `placeholder="jim@example.com"`, `required`, `onChange={e => setEmail(e.target.value)}`, `onKeyUp={e => e.key === "Enter" && onSubmit()}`), then a primary `Button` spanning full width with `size="lg"`, `disabled={!isValidEmail(email) || isLoading}`, click → `onSubmit`. Label: `{t("auth.password-forget.send-reset-code")}`.
     - Divider + return-to-sign-in link block (matches Vue `PasswordForgot.vue:53-65`): `<a>` styled with `accent-link` whose click calls `router.push({ name: AUTH_SIGNIN_MODULE, query: router.currentRoute.value.query })`. Label: `{t("auth.password-forget.return-to-sign-in")}`.
   - No use of React Router — all navigation through Vue `router` per design.
   - No form `<form>` wrapper with `onSubmit={preventDefault}` — Vue version uses `@click` + `@keyup.enter` without a form. Mirror that.

2. In `frontend/src/react/pages/auth/PasswordForgotPage.test.tsx`:
   - Mock setup (follows `BytebaseLogo.test.tsx` / `AuthFooter.test.tsx` hoisted-mock pattern):
     - `vi.mock("@/react/hooks/useVueState", ...)` — the hook gets called per-subscription; use a single mock that tracks the arg and returns a queued value (or, simpler, mock `useActuatorV1Store` directly).
     - `vi.mock("@/store", ...)` — stub `useActuatorV1Store` returning `{ serverInfo: { restriction: { passwordResetEnabled: ..., disallowPasswordSignin: ... } } }`; stub `pushNotification` as a `vi.fn()`.
     - `vi.mock("@/router", ...)` — `router` with `push`, `replace`, `currentRoute: { value: { query: {} } }` — all as `vi.fn()`.
     - `vi.mock("@/connect", ...)` — `authServiceClientConnect: { requestPasswordReset: vi.fn() }`.
     - `vi.mock("@/assets/logo-full.svg", ...)` — return `"/assets/logo-full.svg"` (follows `BytebaseLogo.test.tsx:22`).
     - `vi.mock("@/utils", ...)` — partial mock keeping `isValidEmail` real (import actual), `resolveWorkspaceName: vi.fn(() => undefined)`.
     - Mock `react-i18next` `useTranslation` to return identity `t` (key-as-value) OR import the real configured `@/react/i18n`. Prefer the identity-t pattern since BytebaseLogo tests don't exercise i18n.
   - **Tests** (use `renderIntoContainer` pattern from `AuthFooter.test.tsx`):
     1. `"renders self-host warning when passwordResetEnabled is false"` — assert the warning text is present and email input is not.
     2. `"renders email input and disabled send button when passwordResetEnabled is true"` — assert `input#forgot-email` exists; button text matches `auth.password-forget.send-reset-code`; button has `disabled` attr.
     3. `"enables submit button for a valid email"` — fire `change` on input with `"foo@bar.com"`, assert button no longer disabled.
     4. `"submit calls requestPasswordReset then navigates to password-reset"` — prime mock to resolve; fire click; await microtask; assert `requestPasswordReset` called with `{ email: "foo@bar.com", workspace: undefined }` and `router.push` called with `{ name: AUTH_PASSWORD_RESET_MODULE, query: { email: "foo@bar.com" } }`.
     5. `"shows notification when requestPasswordReset rejects"` — prime mock to reject; fire click; assert `pushNotification` called with `style: "CRITICAL"`.
     6. `"redirects to signin when disallowPasswordSignin is true"` — stub actuator with `disallowPasswordSignin: true`; render; assert `router.replace` called with `{ name: AUTH_SIGNIN_MODULE, query: {} }`.

**Boundaries**:
- Do NOT touch `frontend/src/views/auth/PasswordForgot.vue` in this task — that is T5.
- Do NOT modify `frontend/src/router/auth.ts` or any other router file.
- Do NOT import `@/react/components/BytebaseLogo` — Vue source uses raw SVG, not the workspace-aware logo.
- Do NOT introduce a new PasswordField/EmailField component; inline `<Input>` suffices (one field).

**Dependencies**: T3 (keys must exist before `check-react-i18n.mjs` passes); no dependency on T5.

**Expected Outcome**:
- `frontend/src/react/pages/auth/PasswordForgotPage.tsx` exists, exports `PasswordForgotPage`.
- 6 tests pass.
- Page is resolvable via `mountReactPage(container, "PasswordForgotPage")` (uses Phase 1's `./pages/auth/*.tsx` glob).

**Validation**:
- `pnpm --dir frontend test -- PasswordForgotPage` — expect: 6 tests pass.
- `pnpm --dir frontend type-check` — expect: exits 0.
- `pnpm --dir frontend fix` — expect: no diffs beyond format.
- `pnpm --dir frontend check` — expect: exits 0 (includes `check-react-i18n.mjs` now that T3 seeded keys and T4 uses them).

---

### T5: Replace `PasswordForgot.vue` body with `ReactPageMount` shell [S]

**Objective**: Flip the `/auth/password-forgot` route to render the new React page, keeping the existing `frontend/src/router/auth.ts:56-59` import unchanged.

**Files**: `frontend/src/views/auth/PasswordForgot.vue`

**Implementation**: Overwrite the file so its entire contents become:

```vue
<template>
  <ReactPageMount page="PasswordForgotPage" />
</template>

<script lang="ts" setup>
import ReactPageMount from "@/react/ReactPageMount.vue";
</script>
```

No props, no callbacks — this page is fully self-contained on the React side. The `SplashLayout` still wraps the route via `router/auth.ts:33 component: SplashLayout`. The shell is a direct parallel to `TwoFactorRequired.vue:1-10` minus the `cancelAction` prop (which MultiFactor/2FA need but PasswordForgot does not).

**Boundaries**:
- Do NOT change `frontend/src/router/auth.ts`.
- Do NOT delete `frontend/src/views/auth/PasswordForgot.vue` — the file stays; the body is the change.

**Dependencies**: T4 (React page must exist before Vue shell references it).

**Expected Outcome**:
- Visiting `/auth/password-forgot` mounts `PasswordForgotPage.tsx` via `ReactPageMount`.
- `git diff frontend/src/views/auth/PasswordForgot.vue` shows the entire body replaced.

**Validation**:
- `pnpm --dir frontend type-check` — expect: exits 0 (`vue-tsc` accepts the new shell).
- `pnpm --dir frontend test` — expect: full suite stays green.
- `pnpm --dir frontend check` — expect: exits 0.
- Verify no live Vue callers of the old PasswordForgot body: `grep -rn "PasswordForgot" frontend/src --include="*.vue" --include="*.ts" --include="*.tsx"` should show only `router/auth.ts` (importing the shell) and the shell itself.

---

---

## Phase 3 — `/auth/mfa` migration (ACTIVE)

Second real page migration. Introduces the first primitive that materially differs from the Vue source — Vue uses naive-ui's `NInputOtp`; React uses the existing shadcn-style `OtpInput` at `frontend/src/react/components/ui/otp-input.tsx`. The `MultiFactor` page also exercises two-mode switching (OTP ↔ recovery code), read of a query param (`mfaTempToken`), and the `authStore.login` call with an MFA-specific request.

### Task Index

T6: Seed `multi-factor.*` + `common.verify` i18n keys to React locales [S] — T7: Create `MultiFactorPage` React component + test [M] — T8: Replace `MultiFactor.vue` body with `ReactPageMount` shell [S]

---

### T6: Seed `multi-factor.*` + `common.verify` i18n keys [S]

**Objective**: Pre-seed the 8 translation keys the new `MultiFactorPage.tsx` (T7) will reference.

**Files**:
- Modify: `frontend/src/react/locales/en-US.json`, `zh-CN.json`, `es-ES.json`, `ja-JP.json`, `vi-VN.json`

**Implementation**:
- Add `"multi-factor"` top-level object with:
  - `auth-code`, `recovery-code`, `self`
  - nested `other-methods.self`
  - nested `other-methods.use-auth-app.{self, description}`
  - nested `other-methods.use-recovery-code.{self, description}`
  - All translations copied verbatim from `frontend/src/locales/<locale>.json` — the Vue master already has the full `multi-factor` sub-tree.
- Add `common.verify` to each React locale (merge into existing `common` object) — verbatim from Vue master.
- Run `node scripts/sort_i18n_keys.mjs` to resort.

**Validation**:
- `node scripts/sort_i18n_keys.mjs` — expect: updates 5 files, leaves 25 unchanged.
- `pnpm check` deferred until T7 consumers land.

---

### T7: Create `MultiFactorPage` React component + test [M]

**Objective**: Ship a React page behavior-equivalent to `frontend/src/views/auth/MultiFactor.vue:1-129`. Consumes keys from T6.

**Size**: M — one page + one test. No edits to existing files.

**Files**:
- Create: `frontend/src/react/pages/auth/MultiFactorPage.tsx`
- Test: `frontend/src/react/pages/auth/MultiFactorPage.test.tsx`

**Implementation**:

1. In `frontend/src/react/pages/auth/MultiFactorPage.tsx`:
   - **Imports**: `useMemo`, `useState` from `react`; `useTranslation` from `react-i18next`; `create` from `@bufbuild/protobuf`; `KeyRound`, `Smartphone` (lucide-react icon equivalents of the Vue `SmartphoneIcon` and `heroicons-outline:key`); `Button` from `@/react/components/ui/button`; `Input` from `@/react/components/ui/input`; `OtpInput` from `@/react/components/ui/otp-input`; `useVueState` from `@/react/hooks/useVueState`; `router` from `@/router`; `useAuthStore` from `@/store`; `LoginRequestSchema` from `@/types/proto-es/v1/auth_service_pb`; `resolveWorkspaceName` from `@/utils`; `logoFull` from `@/assets/logo-full.svg`.
   - **Type**: `type MFAType = "OTP" | "RECOVERY_CODE";`
   - **State**:
     - `const [mfaType, setMfaType] = useState<MFAType>("OTP");`
     - `const [otpCodes, setOtpCodes] = useState<string[]>([]);`
     - `const [recoveryCode, setRecoveryCode] = useState("");`
   - **Query read via `useVueState`**:
     - `const mfaTempToken = useVueState(() => (router.currentRoute.value.query.mfaTempToken as string | undefined) ?? "");`
   - **Challenge description** (derived, translates per `mfaType`):
     ```
     const challengeDescription = useMemo(() => {
       if (mfaType === "OTP") return t("multi-factor.other-methods.use-auth-app.description");
       if (mfaType === "RECOVERY_CODE") return t("multi-factor.other-methods.use-recovery-code.description");
       return "";
     }, [mfaType, t]);
     ```
   - **`challenge` async function**:
     - Build `LoginRequest` via `create(LoginRequestSchema, { mfaTempToken, workspace: resolveWorkspaceName(), ...(mfaType === "OTP" ? { otpCode: otpCodes.join("") } : { recoveryCode }) })`.
     - Call `await useAuthStore().login({ request, redirect: true });`
   - **`onOtpFinish(value: string[])`** — `setOtpCodes(value)` then call `challenge()` (matches Vue's `onOtpCodeFinish` at `MultiFactor.vue:109-112`).
   - **JSX** (mirrors Vue template structure):
     - Outer: `<div className="mx-auto max-w-2xl h-full py-6 flex flex-col justify-center items-center">`.
     - Card: border + shadow-sm + `w-80 p-8 py-6`. Use `div` with tailwind classes (no `NCard` equivalent needed; the Vue card is just styling).
     - Inside card: `<img src={logoFull} alt="Bytebase" className="h-12 w-auto mx-auto mb-8" />`, then a `<form onSubmit>` with flex-column layout.
     - Branch on `mfaType`:
       - `"OTP"`: `Smartphone` icon (`className="w-8 h-auto opacity-60"`), `<p className="my-2 mb-4">{t("multi-factor.auth-code")}</p>`, then `<OtpInput value={otpCodes} onChange={setOtpCodes} onFinish={onOtpFinish} />`.
       - `"RECOVERY_CODE"`: `KeyRound` icon (same class), `<p>{t("multi-factor.recovery-code")}</p>`, then `<Input value={recoveryCode} onChange={e => setRecoveryCode(e.target.value)} placeholder="XXXXXXXXXX" className="w-full" />`.
     - Verify button (primary, `w-full!` equivalent = `className="w-full"`), type="submit", label `{t("common.verify")}`.
     - Challenge description `<p className="textinfolabel mt-2">{challengeDescription}</p>`.
     - `<hr className="my-3" />` separator.
     - "Other methods" section: `<p>{t("multi-factor.other-methods.self")}:</p>` + `<ul>` with two `<li>` entries; each `<li>` is an `<button className="accent-link">` that sets `mfaType`. Show only the one(s) not matching current `mfaType` — mirror Vue `v-if="state.selectedMFAType !== 'OTP'"`.

2. In `frontend/src/react/pages/auth/MultiFactorPage.test.tsx`:
   - Mocks via `vi.hoisted`: `useVueState` (calls getter), `useAuthStore: vi.fn(() => ({ login: vi.fn() }))`, `router` with `currentRoute.value.query`, `resolveWorkspaceName: vi.fn(() => undefined)`, `LoginRequestSchema` pass-through (import actual), `react-i18next` identity-t, `@/assets/logo-full.svg` default. Actually cleanest: mock `@/store` directly so each test can control `login` behavior.
   - Mock `@bufbuild/protobuf` `create` — identity function returning the second arg — so the proto schema doesn't need to be fully mocked.
   - **Tests**:
     1. `"renders OTP mode by default with Authentication code label"` — assert label text, OtpInput container exists.
     2. `"switches to RECOVERY_CODE when the user clicks the other method link"` — click first `<button>` inside `ul`, assert recovery-code label visible, 6-digit inputs gone.
     3. `"switches back to OTP from RECOVERY_CODE"` — set initial to recovery via click, then click to OTP.
     4. `"challenge submits otpCode when all 6 digits are entered"` — simulate OtpInput finish by directly calling its `onFinish` (via imperative re-entry in OTP mode is tricky in DOM tests; instead use a stub: re-render with pre-filled OTP digits by firing `input` events on 6 inputs). Easier approach: assert on the imperative `challenge` via form-submit path — render, switch not needed, fill 6 digits via descriptor-setter helper, press Verify, assert `login` called with `otpCode: "123456"` and `mfaTempToken: "TOKEN"`.
     5. `"challenge submits recoveryCode in RECOVERY_CODE mode"` — switch mode, set input value, click Verify, assert `login` called with `recoveryCode: "AAA-BBB"` and no `otpCode`.
     6. `"reads mfaTempToken from route query"` — configure `currentRoute.value.query.mfaTempToken = "TOKEN"`, assert `login` is called with `mfaTempToken: "TOKEN"`.
   - Use the `setInputValue` helper from `PasswordForgotPage.test.tsx` (copy it inline — small, no shared-helpers barrel yet).

**Boundaries**:
- Do NOT edit `frontend/src/views/auth/MultiFactor.vue` in this task (T8).
- Do NOT change `@/store/modules/v1/auth.ts`.
- Do NOT edit `OtpInput` — reuse as-is.
- Do NOT replicate `NCard` via a new component; inline Tailwind is sufficient for this one page.
- Do NOT introduce a separate `MFAForm` subcomponent; keep logic in the page. Extract only if Phase 7/8 needs it.

**Dependencies**: T6 (i18n keys must exist for `check-react-i18n` to pass once T7 lands).

**Expected Outcome**:
- `MultiFactorPage.tsx` exports `MultiFactorPage`.
- 6 tests pass.
- Page resolvable via `mountReactPage(container, "MultiFactorPage")`.

**Validation**:
- `pnpm test -- MultiFactorPage` — expect: 6 tests pass.
- `pnpm type-check` — expect: exits 0.
- `pnpm fix` — expect: no outstanding diffs.
- `pnpm check` — expect: exits 0.

---

### T8: Replace `MultiFactor.vue` body with `ReactPageMount` shell [S]

**Objective**: Flip `/auth/mfa` to the React page.

**Files**: `frontend/src/views/auth/MultiFactor.vue`

**Implementation**: Overwrite to:

```vue
<template>
  <ReactPageMount page="MultiFactorPage" />
</template>

<script lang="ts" setup>
import ReactPageMount from "@/react/ReactPageMount.vue";
</script>
```

No props — MFA is fully self-contained on the React side (reads `mfaTempToken` from the router query directly, calls `authStore.login` which handles redirects).

**Dependencies**: T7.

**Validation**:
- `pnpm type-check` — expect: exits 0.
- `pnpm test` — expect: full suite green.
- `pnpm check` — expect: exits 0.

---

---

## Phase 4 — `/oauth/callback` + `/oidc/callback` migration (ACTIVE)

Both routes are registered at the top level in `frontend/src/router/auth.ts:74-83` and point at the same Vue file (`frontend/src/views/OAuthCallback.vue`). They bypass `SplashLayout` entirely. Phase 4 produces one React page + one Vue-shell overwrite.

### Notable parity call-outs

- The Vue page has several **hardcoded English status/error strings** (e.g., `"Authentication failed: Invalid callback state. Please try again."`, `"Processing authentication..."`). These strings are not in `frontend/src/locales/`. Per UX-parity goal, the React port preserves them as hardcoded literals. This breaches CLAUDE.md's "no hardcoded display text" rule, but the rule is already broken on the Vue side — fixing it is a separate i18n-hygiene task. The only strings routed through i18n are `auth.close-window` and `auth.back-to-signin` (both already used by the Vue page via `$t(...)`) and the native `common.close`.
- No `NSpin` equivalent is pulled in. Use a lightweight spinner — `Loader2` from `lucide-react` with `animate-spin`, which is the established pattern elsewhere in the React surface.

### Task Index

T9: Seed `auth.close-window` + `auth.back-to-signin` i18n keys [S] — T10: Create `OAuthCallbackPage` React component + test [M] — T11: Replace `OAuthCallback.vue` body with `ReactPageMount` shell [S]

---

### T9: Seed `auth.close-window` + `auth.back-to-signin` i18n keys [S]

**Objective**: Pre-seed the 2 translation keys the new `OAuthCallbackPage.tsx` (T10) will reference via `t(...)`. `common.close` already exists in React locales from a prior migration and does not need to be added.

**Files**:
- Modify: `frontend/src/react/locales/en-US.json`, `zh-CN.json`, `es-ES.json`, `ja-JP.json`, `vi-VN.json`

**Implementation**:
- Merge `auth.close-window` and `auth.back-to-signin` into each React locale's existing `"auth"` object (already has `auth.password-forget.*` from Phase 2). Values copied verbatim from `frontend/src/locales/<locale>.json`'s `auth.close-window` / `auth.back-to-signin`.
- Run `node scripts/sort_i18n_keys.mjs` to resort alphabetically.

**Validation**:
- `node scripts/sort_i18n_keys.mjs` — expect: updates 5 files.
- `pnpm check` deferred until T10 consumers land.

---

### T10: Create `OAuthCallbackPage` React component + test [M]

**Objective**: Ship a React page behavior-equivalent to `frontend/src/views/OAuthCallback.vue:1-205`. Handles both popup-mode (dispatches event to `window.opener` then closes) and redirect-mode (calls `authStore.login`) OAuth/OIDC callbacks.

**Size**: M — one page + one test. No edits to existing files.

**Files**:
- Create: `frontend/src/react/pages/auth/OAuthCallbackPage.tsx`
- Test: `frontend/src/react/pages/auth/OAuthCallbackPage.test.tsx`

**Implementation**:

1. In `frontend/src/react/pages/auth/OAuthCallbackPage.tsx`:
   - **Imports**: `useEffect`, `useState` from `react`; `useTranslation` from `react-i18next`; `create` from `@bufbuild/protobuf`; `Loader2` from `lucide-react`; `Button` from `@/react/components/ui/button`; `useVueState` from `@/react/hooks/useVueState`; `router` from `@/router`; `AUTH_SIGNIN_MODULE` from `@/router/auth`; `useAuthStore` from `@/store`; `LoginRequestSchema` from `@/types/proto-es/v1/auth_service_pb`; `IdentityProviderType` from `@/types/proto-es/v1/idp_service_pb`; `OAuthState`, `OAuthWindowEventPayload` from `@/types/oauth`; `resolveWorkspaceName` from `@/utils`; `clearOAuthState`, `retrieveOAuthState` from `@/utils/sso`.
   - **Local state** (all `useState`):
     - `message: string` — initial `""`
     - `hasError: boolean` — initial `false`
     - `oAuthState: OAuthState | undefined` — initial `undefined`
     - `showCloseButton: boolean` — initial `false`
   - **Payload ref** — build the `{ error: "", code: "" }` payload from route query once inside the mount effect (not stored in state since it doesn't affect rendering). The Vue keeps this in reactive `state.payload` but only reads it during the dispatch; a `useRef<OAuthWindowEventPayload>` suffices.
   - **Mount effect** (`useEffect(..., [])`) — single-run initialization mirroring Vue's `onMounted` at `OAuthCallback.vue:63-114`:
     1. Read `stateToken` from `router.currentRoute.value.query.state`.
     2. If invalid: `setHasError(true)`, `setMessage("Authentication failed: Invalid callback state. Please try again.")`, call `triggerAuthCallback(undefined, true)` with the bad-state branch (no `oAuthState`).
     3. Else retrieve `storedState = retrieveOAuthState(stateToken)`. If null: `setHasError`, `setMessage("Authentication failed: Session expired or invalid. Please try again.")`, trigger.
     4. Else if `storedState.token !== stateToken`: `setHasError`, `setMessage("Authentication failed: Security validation failed. Please try again.")`, `clearOAuthState(stateToken)`, trigger.
     5. Else valid: `setOAuthState(storedState)`, `setHasError(false)`, `setMessage("Successfully authorized. Redirecting back to Bytebase...")`, set `payload.code = router.currentRoute.value.query.code as string || ""`, `clearOAuthState(storedState.token)`, trigger.
   - **`triggerAuthCallback(state, isError)`** function (plain async function, not a hook):
     - If `state?.popup`:
       - Try/catch around `window.opener` check → if opener is null/closed: setHasError, `setMessage("Authentication completed, but the parent window is no longer available. Please close this window and try again.")`, setShowCloseButton(true), return.
       - Compute `eventName = isError ? "bb.oauth.unknown" : state.event`.
       - `window.opener.dispatchEvent(new CustomEvent(eventName, { detail: payloadRef.current }))`.
       - Try `window.close()`. After 500ms timeout, if `!window.closed`, setShowCloseButton(true), setMessage(ifError? existing : `"Authentication completed. You can close this window."`).
       - Catch window.close errors: setShowCloseButton(true), setMessage(similar fallback).
       - Outer catch: setHasError, setMessage(`"Authentication completed, but failed to communicate with the parent window. Please close this window and try again."`), setShowCloseButton(true).
     - Else (redirect mode):
       - If `isError || !state`: template already displays the error — return (Vue comment at line 173-175).
       - Else if `state.event.startsWith("bb.oauth.signin")`:
         - `isOidc = state.idpType === IdentityProviderType.OIDC`
         - `idpName = state.event.split(".").pop()` — if falsy, return.
         - `await useAuthStore().login({ request: create(LoginRequestSchema, { idpName, idpContext: { context: { case: isOidc ? "oidcContext" : "oauth2Context", value: { code: payloadRef.current.code } } }, workspace: resolveWorkspaceName() }), redirect: true, redirectUrl: state.redirect })`.
   - **JSX** (mirrors Vue template):
     - Outer wrapper `<div className="p-4">`.
     - If `hasError`: `<div className="mt-2">`, `<div>{message}</div>`, then branch on `oAuthState?.popup` → popup mode renders a `<Button onClick={() => window.close()}>{t("common.close")}</Button>`; non-popup mode renders an `<a>` styled as `btn-normal` (matching Vue class) that pushes to `AUTH_SIGNIN_MODULE` via `router.push`.
     - Else processing state: `<div className="mt-2">` containing `<div className="flex items-center gap-x-2">` with `<Loader2 className="size-4 animate-spin" />` + `<span>{message || "Processing authentication..."}</span>`; if `oAuthState?.popup && showCloseButton`, render close button below.
   - Do NOT wrap in `SplashLayout` — this page is registered at the top level in `router/auth.ts:74-83` without a layout.
   - Do NOT render `BytebaseLogo` or `AuthFooter` — the Vue page doesn't.

2. In `frontend/src/react/pages/auth/OAuthCallbackPage.test.tsx`:
   - Mock strategy (use the `importOriginal` partial-mock pattern established in `MultiFactorPage.test.tsx` for `@bufbuild/protobuf` and the proto modules):
     - `vi.mock("@/react/hooks/useVueState", ...)` — identity (calls the getter).
     - `vi.mock("@/store", ...)` — `useAuthStore: vi.fn(() => ({ login: vi.fn() }))`.
     - `vi.mock("@/router", ...)` — `router.push`, `router.currentRoute.value.query`.
     - `vi.mock("@/utils/sso", ...)` — `retrieveOAuthState`, `clearOAuthState`.
     - `vi.mock("@/utils", async (importOriginal) => ...)` — partial mock of `resolveWorkspaceName`.
     - `vi.mock("@/types/proto-es/v1/idp_service_pb", async (importOriginal) => ...)` — keep real `IdentityProviderType` enum.
     - `vi.mock("@bufbuild/protobuf", ...)` partial — identity `create`.
     - `vi.mock("@/types/proto-es/v1/auth_service_pb", ...)` partial — stub `LoginRequestSchema`.
     - `vi.mock("react-i18next", ...)` — identity `t`.
   - **Tests** (6):
     1. `"renders error + back-to-signin link when state token is missing"` — `currentRoute.value.query = {}`, render, assert error text visible, `a` with text `auth.back-to-signin` exists.
     2. `"renders error when stored state not found"` — `currentRoute.value.query = { state: "xyz" }`, `retrieveOAuthState.mockReturnValue(null)`, render, assert session-expired error text.
     3. `"renders error when stored token mismatch"` — stub `retrieveOAuthState` to return `{ token: "OTHER", ... }`, assert security-validation error + `clearOAuthState` called with `"xyz"`.
     4. `"redirect mode: valid signin event calls authStore.login with oauth2Context"` — stub state with `event: "bb.oauth.signin.gh"`, `idpType: IdentityProviderType.OAUTH2`, `popup: false`, `redirect: "/home"`. `query.code = "abc"`. Assert `login` called with `{ request: objectContaining({ idpName: "gh", idpContext: { context: { case: "oauth2Context", value: { code: "abc" } } } }), redirect: true, redirectUrl: "/home" }`.
     5. `"redirect mode: OIDC event uses oidcContext"` — same as #4 but `idpType: IdentityProviderType.OIDC`, assert case `"oidcContext"`.
     6. `"popup mode: dispatches CustomEvent on window.opener with payload"` — mock `window.opener = { closed: false, dispatchEvent: vi.fn() }` via `Object.defineProperty`. Stub state with `popup: true, event: "bb.oauth.signin.gh", idpType: OAUTH2`. Render. Assert `window.opener.dispatchEvent` called with a `CustomEvent` whose `.detail.code === query.code`.
   - Test helper: restore `window.opener` after each test via `afterEach`.

**Boundaries**:
- Do NOT edit `frontend/src/views/OAuthCallback.vue` (T11).
- Do NOT edit `frontend/src/utils/sso.ts` or `frontend/src/types/oauth.ts`.
- Do NOT add i18n keys for the hardcoded English status strings — parity with Vue source.
- Do NOT introduce a React Router; navigation via Vue `router.push`.

**Dependencies**: T9.

**Validation**:
- `pnpm test -- OAuthCallbackPage` — expect: 6 tests pass.
- `pnpm type-check` — exit 0.
- `pnpm fix` — no diffs.
- `pnpm check` — exit 0.

---

### T11: Replace `OAuthCallback.vue` body with `ReactPageMount` shell [S]

**Objective**: Flip both `/oauth/callback` and `/oidc/callback` routes to the React page. Router entries at `frontend/src/router/auth.ts:74-83` — both already point at `@/views/OAuthCallback.vue`; overwriting the file flips both routes simultaneously.

**Files**: `frontend/src/views/OAuthCallback.vue`

**Implementation**: Overwrite to:

```vue
<template>
  <ReactPageMount page="OAuthCallbackPage" />
</template>

<script lang="ts" setup>
import ReactPageMount from "@/react/ReactPageMount.vue";
</script>
```

**Dependencies**: T10.

**Validation**:
- `pnpm type-check` — exit 0.
- `pnpm test` — full suite green.
- `pnpm check` — exit 0.

---

## Out-of-Scope Tasks (for this `/executing-tasks` invocation)

- **Phases 5–10** (OAuth2Consent, PasswordReset, SigninAdmin, Signin + SigninBridge, Signup, Setup).
- **Consolidating the `auth.*` namespace in React locales.** Only the keys Phase 2 touches are added. Future phases append incrementally.
- **Manual `/auth/password-forgot` UX parity testing in a running dev server.** Flagged as a suggested manual verification step for the PR description, but not part of automated validation.
- **Port of `BytebaseLogo` usage for auth pages.** Signin/Signup use `BytebaseLogo`; PasswordForgot uses raw SVG. Decision deferred to Phase 7/8 when Signin-adjacent code lands.
- **Port of `common.email` from Vue to React locale files.** Already present in React locales.
