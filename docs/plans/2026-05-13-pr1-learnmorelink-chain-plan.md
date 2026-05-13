# PR-1 — LearnMoreLink Chain: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete `frontend/src/components/LearnMoreLink.vue` by inlining its `<a + ExternalLinkIcon>` markup into its two Vue callers (`bbkit/BBAttention.vue` and `MonacoEditor/utils.ts`).

**Architecture:** All three changes (two file edits + one deletion) land in a single atomic commit. Splitting would leave an intermediate state with a dangling import. Validation chain matches Phase A's: type-check, lint (with cleared ESLint cache to avoid the local-vs-CI mismatch we hit in PR #20321), tests, and the existing React→Vue CI guard.

**Tech Stack:** Vue 3, Naive UI, lucide-vue-next, vue-i18n, vitest, TypeScript, pnpm.

**Spec:** [`docs/plans/2026-05-13-pr1-learnmorelink-chain-design.md`](./2026-05-13-pr1-learnmorelink-chain-design.md)

---

## Pre-work: Baseline & branch

- [ ] **Step 0.1:** Confirm working tree clean and on a fresh main.

```bash
git status
git branch --show-current
```

Expected: clean tree, on `main`. If not on main, ask the user before proceeding.

- [ ] **Step 0.2:** Pull latest main.

```bash
git fetch origin main
git pull --ff-only origin main
```

- [ ] **Step 0.3:** Create the feature branch.

```bash
git checkout -b chore/pr1-drop-learnmorelink-vue
```

- [ ] **Step 0.4:** Baseline type-check (≈1 min).

Run: `pnpm --dir frontend type-check`
Expected: exit 0.

If type-check fails on a clean tree, stop and report — do not start edits.

---

## Task 1: Inline the link in BBAttention.vue

**Files:**
- Modify: `frontend/src/bbkit/BBAttention.vue` — line 10 (template usage) and line 37 (import)

- [ ] **Step 1.1: Replace the import on line 37**

Edit `frontend/src/bbkit/BBAttention.vue`:

```diff
- import LearnMoreLink from "@/components/LearnMoreLink.vue";
+ import { ExternalLinkIcon } from "lucide-vue-next";
```

After this step, lines 33–37 should read:

```vue
<script lang="ts" setup>
import { NAlert, NButton } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ExternalLinkIcon } from "lucide-vue-next";
```

- [ ] **Step 1.2: Replace the `<LearnMoreLink>` element on line 10**

Edit `frontend/src/bbkit/BBAttention.vue`:

```diff
-            <LearnMoreLink v-if="link" :url="link" class="ml-1 text-sm" />
+            <a
+              v-if="link"
+              :href="link"
+              target="__BLANK"
+              class="inline-flex items-center normal-link ml-1 text-sm"
+            >
+              {{ $t("common.learn-more") }}
+              <ExternalLinkIcon class="w-4 h-4 ml-1" />
+            </a>
```

After this step, the `<slot name="default">` block (lines 6–13 originally) should read:

```vue
      <slot name="default">
        <div v-if="description" class="text-sm">
          <p class="whitespace-pre-wrap">
            {{ $te(description) ? $t(description) : description }}
            <a
              v-if="link"
              :href="link"
              target="__BLANK"
              class="inline-flex items-center normal-link ml-1 text-sm"
            >
              {{ $t("common.learn-more") }}
              <ExternalLinkIcon class="w-4 h-4 ml-1" />
            </a>
          </p>
        </div>
      </slot>
```

Notes:
- `target="__BLANK"` is intentional — matches the original `LearnMoreLink.vue` template verbatim. Browsers treat any non-`_blank` non-`_self`/`_parent`/`_top` value as a new window, so behavior is unchanged. Preserving the literal avoids accidental UX drift.
- The classes `inline-flex items-center normal-link` come from the original `LearnMoreLink.vue` template (`color === "normal"` branch). `normal-link` is defined in `frontend/src/assets/css/tailwind.css:302` — verified to exist.

---

## Task 2: Inline the link in MonacoEditor/utils.ts

**Files:**
- Modify: `frontend/src/components/MonacoEditor/utils.ts` — line 7 (import) and lines 129–131 (`h(LearnMoreLink, ...)` call)

- [ ] **Step 2.1: Replace the import on line 7**

Edit `frontend/src/components/MonacoEditor/utils.ts`:

```diff
- import LearnMoreLink from "../LearnMoreLink.vue";
+ import { ExternalLinkIcon } from "lucide-vue-next";
```

After this step, lines 1–9 should read:

```ts
import { Range } from "monaco-editor";
import { h, isRef, unref, watch } from "vue";
import { ExternalLinkIcon } from "lucide-vue-next";
import { t } from "@/plugins/i18n";
import { pushNotification } from "@/store";
import type { Language, MaybeRef, SQLDialect } from "@/types";
import { minmax } from "@/utils";
import sqlFormatter from "./sqlFormatter";
import type { IStandaloneCodeEditor, Selection } from "./types";
```

(Note the import order — `lucide-vue-next` is a third-party package, so it sits with the other third-party imports near the top, before the `@/...` aliases. Biome's organize-imports may rearrange; if so, accept the rearrangement.)

- [ ] **Step 2.2: Replace the `h(LearnMoreLink, ...)` call on lines 129–131**

Edit `frontend/src/components/MonacoEditor/utils.ts`:

```diff
-        h(LearnMoreLink, {
-          url: "https://docs.bytebase.com/administration/production-setup/#enable-https-and-websocket",
-        }),
+        h(
+          "a",
+          {
+            href: "https://docs.bytebase.com/administration/production-setup/#enable-https-and-websocket",
+            target: "__BLANK",
+            class: "inline-flex items-center normal-link",
+          },
+          [t("common.learn-more"), h(ExternalLinkIcon, { class: "w-4 h-4 ml-1" })]
+        ),
```

After this step, the `errorNotification` body (lines 119–135 originally) should read:

```ts
export const errorNotification = (err: unknown) => {
  pushNotification({
    module: "bytebase",
    style: "CRITICAL",
    title: messages.title(),
    description: () => {
      const message = extractErrorMessage(err);
      return [
        h("p", {}, messages.description()),
        message ? h("p", {}, message) : null,
        h(
          "a",
          {
            href: "https://docs.bytebase.com/administration/production-setup/#enable-https-and-websocket",
            target: "__BLANK",
            class: "inline-flex items-center normal-link",
          },
          [t("common.learn-more"), h(ExternalLinkIcon, { class: "w-4 h-4 ml-1" })]
        ),
      ];
    },
  });
};
```

---

## Task 3: Delete LearnMoreLink.vue

**Files:**
- Delete: `frontend/src/components/LearnMoreLink.vue`

- [ ] **Step 3.1: Confirm no remaining importers**

```bash
rg -n --type-add 'tsx:*.tsx' --type-add 'vue:*.vue' -tts -tvue -ttsx 'LearnMoreLink' frontend/src/ 2>/dev/null \
  | grep -v '^frontend/src/components/LearnMoreLink\.vue:' \
  | grep -v '^frontend/src/react/components/LearnMoreLink\.tsx:'
```

Expected: zero hits. The only remaining mentions should be in the file we're about to delete and in the React `LearnMoreLink.tsx` (which we're not touching).

If any hit appears, stop — a caller was missed. Investigate before deleting.

- [ ] **Step 3.2: Delete the file**

```bash
git rm frontend/src/components/LearnMoreLink.vue
```

---

## Task 4: Validate

- [ ] **Step 4.1: Clear the ESLint cache**

```bash
rm -f frontend/.eslintcache
```

This avoids the cache-staleness issue we hit in PR #20321 where `pnpm check` passed locally but failed in CI on an unused-locale-key violation.

- [ ] **Step 4.2: Type-check**

```bash
pnpm --dir frontend type-check
```

Expected: exit 0.

If failure: likely caused by an unmigrated reference. Inspect the type error message; it will point at the line. Fix and re-run.

- [ ] **Step 4.3: Full lint chain**

```bash
pnpm --dir frontend check
```

Expected: exit 0. This runs `eslint --cache src && biome ci . && check-react-i18n && check-react-layering && sort_i18n_keys --check`. Importantly, ESLint's `@intlify/vue-i18n/no-unused-keys` rule sees both new inlines call `$t("common.learn-more")` / `t("common.learn-more")`, so the `common.learn-more` key stays "used."

If failure on `common.learn-more`: re-verify both inlines reference it. The Vue template uses `{{ $t("common.learn-more") }}`; the TS call uses `t("common.learn-more")`. The ESLint rule scans `.vue`, `.ts`, `.tsx` and matches both.

- [ ] **Step 4.4: Tests**

```bash
pnpm --dir frontend test
```

Expected: 1873 tests pass (same count as PR #20321 baseline). The React→Vue CI guard from PR #20321 has nothing new to flag (no React file imports `.vue` here).

- [ ] **Step 4.5: Visual smoke (manual)**

```bash
PG_URL=postgresql://bbdev@localhost/bbdev pnpm --dir frontend dev
```

Verify two paths:

1. **BBAttention with `link` prop.** Find one:
   ```bash
   rg ':link=' frontend/src/ -tvue -l | head -5
   ```
   Visit a page that renders one of those `BBAttention` callers (typical candidates: settings pages with subscription-gating banners) and confirm the inline learn-more anchor:
   - Renders next to the description text
   - Has the external-link icon
   - Opens in a new window when clicked
   - Has the same visual styling as before this PR

2. **Monaco connection error notification.** Trigger by running Bytebase against a misconfigured WebSocket endpoint, or temporarily mock the failure to exercise `errorNotification` in `MonacoEditor/utils.ts`. The notification should render the learn-more link as a styled anchor with the icon.

If visual parity is broken: review the `normal-link` class behavior (it's defined in `frontend/src/assets/css/tailwind.css:302`). The pre-PR `LearnMoreLink.vue` template applied `inline-flex items-center normal-link` for the "normal" color branch — we replicate it.

---

## Task 5: Commit, push, and open the PR

- [ ] **Step 5.1: Single atomic commit**

```bash
git add frontend/src/bbkit/BBAttention.vue frontend/src/components/MonacoEditor/utils.ts
git commit -m "$(cat <<'EOF'
chore(frontend): drop LearnMoreLink.vue, inline anchor markup

LearnMoreLink.vue was VUE-ONLY after Phase A — only two Vue callers
remained (bbkit/BBAttention.vue and components/MonacoEditor/utils.ts).
Its template is a thin `<a + ExternalLinkIcon>` wrapper, so inlining
is shorter than the indirection.

- BBAttention.vue: replace `<LearnMoreLink :url=link>` with a plain
  `<a target="__BLANK">` carrying the same classes; swap import for
  lucide-vue-next ExternalLinkIcon.
- MonacoEditor/utils.ts: replace `h(LearnMoreLink, {...})` with the
  equivalent `h("a", {...}, [t("common.learn-more"), h(ExternalLinkIcon, ...)])`.
- Delete frontend/src/components/LearnMoreLink.vue.

`common.learn-more` locale key stays — both inlines still call it.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

The `git rm` from Step 3.2 has already staged the deletion, so a single `git commit` captures both edits + the deletion.

- [ ] **Step 5.2: Push**

```bash
git push -u origin chore/pr1-drop-learnmorelink-vue
```

- [ ] **Step 5.3: Open PR**

```bash
gh pr create --title "chore(frontend): drop LearnMoreLink.vue, inline anchor markup" --body "$(cat <<'EOF'
## Summary

First of three planned chain-cutting PRs after Phase A
([status](docs/plans/2026-05-12-react-migration-status-and-plan.md),
[design](docs/plans/2026-05-13-pr1-learnmorelink-chain-design.md)).

`LearnMoreLink.vue` was VUE-ONLY after Phase A's cross-framework guard
landed — only two Vue callers remained:

- `bbkit/BBAttention.vue` — template usage.
- `components/MonacoEditor/utils.ts` — programmatic `h()` call.

Its template is a thin `<a + ExternalLinkIcon>` wrapper. Inlining the
markup into both callers is shorter than the indirection. The Vue file
is deleted in the same change.

The React `LearnMoreLink.tsx` is untouched — it continues serving the
React layer.

## Test plan
- [x] `pnpm --dir frontend type-check` passes
- [x] `pnpm --dir frontend check` passes (eslint, biome, react-i18n, react-layering, locale sorter)
- [x] `pnpm --dir frontend test` passes (1873 tests, including the React→Vue import guard)
- [ ] Manual smoke: an attention banner with a `link` prop renders the inline learn-more anchor with correct styling and the external-link icon
- [ ] Manual smoke: the Monaco WebSocket-error notification renders the learn-more anchor as expected

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Done when

- `frontend/src/components/LearnMoreLink.vue` no longer exists.
- `rg -n 'LearnMoreLink' frontend/src/ -tvue -tts` returns no hits inside Vue or TS files (the file is gone; the React `LearnMoreLink.tsx` remains untouched).
- All automated checks pass on CI.
- The two manual smoke paths render UI identical to pre-PR.

## Self-review notes

- **Spec coverage:** Spec's Section 1 = Task 1; Section 2 = Task 2; Section 3 = Task 3; Section 4 (Locale) is verified inline in Task 4.3; Validation list maps 1:1 to Task 4 steps.
- **Atomic commit rationale:** The spec calls for a single atomic change because the two edits + one deletion are interdependent (deleting first leaves broken imports; editing first leaves an unused file). The plan honors that with a single commit at Step 5.1.
- **No new CI guard entries needed:** Phase A's React→Vue guard allowlist (`SessionExpiredSurfaceMount`, `AgentWindowMount`) is unaffected. Adding `LearnMoreLink.vue` would have required an entry only if we were temporarily keeping it; we delete it instead.
