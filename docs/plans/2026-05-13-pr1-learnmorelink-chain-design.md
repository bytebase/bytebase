# PR-1 — LearnMoreLink Chain: Design

**Date:** 2026-05-13
**Parent doc:** [2026-05-12-react-migration-status-and-plan.md](./2026-05-12-react-migration-status-and-plan.md)
**Sequence:** First of three planned chain-cutting PRs. Follow-ups: PR-2 (MonacoEditor suite migration), PR-3 (BBAttention.vue migration).

---

## Goal

Delete `frontend/src/components/LearnMoreLink.vue` by inlining its short `<a + icon>` markup into the two Vue files that still call it.

`LearnMoreLink.vue` is **VUE-ONLY** (React callers already use the React `LearnMoreLink.tsx`). The two Vue callers are:

1. `frontend/src/bbkit/BBAttention.vue` — uses `<LearnMoreLink>` as a template element.
2. `frontend/src/components/MonacoEditor/utils.ts` — calls `h(LearnMoreLink, ...)` programmatically.

## Strategy: inline, don't abstract

The Vue component's template is 8 meaningful lines (an `<a>` element + an `ExternalLinkIcon`). The two callers each touch it exactly once. A shared helper would replace one import with another and add a layer for no benefit. Inline the markup directly.

## Scope: one PR

### 1. Edit `frontend/src/bbkit/BBAttention.vue`

**Replace the import:**

```diff
- import LearnMoreLink from "@/components/LearnMoreLink.vue";
+ import { ExternalLinkIcon } from "lucide-vue-next";
```

**Replace the element:**

```diff
- <LearnMoreLink v-if="link" :url="link" class="ml-1 text-sm" />
+ <a
+   v-if="link"
+   :href="link"
+   target="__BLANK"
+   class="inline-flex items-center normal-link ml-1 text-sm"
+ >
+   {{ $t("common.learn-more") }}
+   <ExternalLinkIcon class="w-4 h-4 ml-1" />
+ </a>
```

Notes:
- Keep classes `inline-flex items-center normal-link` — these come from the original `LearnMoreLink.vue` template (`color === "normal"` branch); preserving them keeps visual parity.
- `target="__BLANK"` matches the original template verbatim (note: the original uses `__BLANK`, not the conventional `_blank`; we preserve to avoid behavior drift).
- Drop the `external` conditional — `BBAttention`'s `link` prop is always an http(s) docs URL in current usage. If a future caller passes a non-external link, the icon is the only cosmetic difference.

### 2. Edit `frontend/src/components/MonacoEditor/utils.ts`

**Replace the import:**

```diff
- import LearnMoreLink from "../LearnMoreLink.vue";
+ import { ExternalLinkIcon } from "lucide-vue-next";
```

**Replace the `h()` call:**

```diff
- h(LearnMoreLink, {
-   url: "https://docs.bytebase.com/administration/production-setup/#enable-https-and-websocket",
- }),
+ h(
+   "a",
+   {
+     href: "https://docs.bytebase.com/administration/production-setup/#enable-https-and-websocket",
+     target: "__BLANK",
+     class: "inline-flex items-center normal-link",
+   },
+   [t("common.learn-more"), h(ExternalLinkIcon, { class: "w-4 h-4 ml-1" })]
+ ),
```

### 3. Delete the Vue file

```bash
git rm frontend/src/components/LearnMoreLink.vue
```

### 4. Locale

`common.learn-more` stays — both new inlines call `$t("common.learn-more")` / `t("common.learn-more")`. ESLint `@intlify/vue-i18n/no-unused-keys` will not flag it.

## Validation

1. `rm -f frontend/.eslintcache` — clear ESLint cache (avoids the local-vs-CI mismatch from PR #20321 where stale cache hid an unused-key violation).
2. `pnpm --dir frontend type-check`
3. `pnpm --dir frontend check` — full lint chain.
4. `pnpm --dir frontend test` — existing tests, including the React→Vue import guard from PR #20321. The guard should still pass (Phase A's allowlist is untouched).
5. Manual smoke:
   - **BBAttention** — find one usage that passes a `link` prop and verify the learn-more anchor renders correctly. Quick search: `rg ':link=' frontend/src/ -tvue -l | head`.
   - **Monaco connection error** — simulate the WebSocket setup failure path. The error notification (`pushNotification` with `style: "CRITICAL"`) should render the learn-more link inside the Monaco-related error toast.

## Done when

- `frontend/src/components/LearnMoreLink.vue` does not exist.
- `rg 'LearnMoreLink' frontend/src/ -tvue -tts` returns only the matches inside the React `LearnMoreLink.tsx` and the React `lucide-react` import — no Vue imports of `LearnMoreLink.vue` anywhere.
- All automated checks pass.
- Both manual smoke paths render identical UI to pre-PR.

## Risks

Low. ~10 lines of inline replacement, 1 file deleted. The CSS classes are preserved so visual parity holds. Two failure modes:

- **`__BLANK` vs `_blank`**: the original used the non-standard `__BLANK` literal. Most browsers treat any unknown `target` as a new window, so behavior is unchanged. We preserve verbatim to avoid an accidental UX change.
- **`normal-link` class**: defined elsewhere (likely a global CSS class). Verified to exist in the codebase before deletion; if a future restyle removes it, the inlines fall back to default link styling — minor cosmetic only.

## Out of scope

- React-side `LearnMoreLink.tsx` — untouched. Continues serving React callers.
- BBAttention.vue's own migration to React — that's PR-3.
- MonacoEditor's overall migration — that's PR-2; after it lands, the `utils.ts` edit in this PR gets deleted entirely (with the rest of the Vue Monaco suite).
- Other VUE-ONLY chains (`UserAvatar`, `OverlayStackManager`, etc.) — separate future PRs.

## Estimated cost

~30 minutes including manual smoke.
