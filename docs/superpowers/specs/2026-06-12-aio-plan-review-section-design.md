# AIO Plan Detail Review Section — Design

Date: 2026-06-12
Source: "AIO Plan Page UI/UX Design" doc (2026-05-28, updated 2026-06-09; reviewers Peter Zhu, Xzavier Zane, Danny Xu). This spec translates the approved UX design into a frontend implementation design.

## Summary

Make Plan Detail the primary review surface for CI/CD plans. The Review phase
section gains the full issue-backed review workflow — approval flow, review
actions (comment / approve / reject), rejection recovery, a unified activity
timeline with composer, and a rollout-readiness footer — so users complete the
review decision beside SQL, targets, checks, risk, and settings without
leaving the page.

## Scope

- **In scope (3.19.1 release-plan stage):** the Review section redesign on
  Plan Detail described below. Issue Detail keeps its existing review
  capability unchanged.
- **Out of scope:** the 3.20.0 redirect behavior for issue-based review entry
  points (separate spec); backend workflow changes; approval policy or
  permission model changes; release-based (GitOps) database changes; full
  visual redesign of Plan Detail; deploy/rollout redesign; plan/issue
  title-description-label unification.

## Approach decision

Self-contained implementation (approved): new components live in
`frontend/src/react/pages/project/plan-detail/components/review/`. They reuse
the app-store `issueComment` slice, the `approveIssue` / `rejectIssue` /
`requestIssue` / `createIssueComment` connect calls, and leaf components
(`MarkdownEditor`, `UserAvatar`, `HumanizeTs`, `Tooltip`, `Popover`), plus
pure utils extracted from Issue Detail where the import is clean
(`diffPlanSpecs`, approval-candidate computation). No Issue Detail page
components are imported or modified — zero regression risk on the page that
remains the fallback in 3.19.1.

## Component structure

`ReviewBranch` in `ProjectPlanDetailPage.tsx` renders a new
`PlanReviewSection`, stacked top to bottom:

```
PlanReviewSection
├── ReviewSectionHeader      — "Approval Flow" + risk chip + issue #chip · Review button (far right)
├── ReviewApprovalFlow       — horizontal adaptive row / vertical stepper (narrow)
├── ReviewRejectionBanner    — only when rejected, pinned above the timeline
├── ReviewActivityTimeline   — unified oldest-first events + comments
│   └── ReviewCommentComposer — collapsed "Add a comment…", last timeline entry
└── ReviewReadinessFooter    — quiet status line + "Bypass and deploy"
```

Canonical anatomy (① Review action ② approval flow ③ activity timeline
④ composer ⑤ readiness footer; the footer link label in this older mockup
reads "Bypass review and deploy" — superseded by "Bypass and deploy",
decided 2026-06-10):

![Review section, review in progress](assets/2026-06-12-aio-plan-review/review-in-progress.png)

- Replaces the `review`-mode rendering of `PlanDetailApprovalFlow.tsx`. The
  `sidebar` mode of that component has no callers; the whole component is
  deleted.
- The `PhaseSection` shell (status dot, badge, expand/collapse) is unchanged.
- The CHECKING ("generating approval flow…") and SKIPPED one-line renderings
  carry over into the new section unchanged.
- The footer sits below the composer deliberately: the composer reads as the
  final timeline entry; the footer is the terminal summary of the section.

## Design guidelines (from the UX doc, binding)

- Keep review inline with plan context; no navigation away for any review act.
- Each review state has exactly one primary action with one home: Review
  (header) while actionable; re-request (rejection banner) when rejected;
  Bypass and deploy (footer) only when it is the only path forward.
- Rejection is the only alert-styled state; approved/pending/skipped stay
  neutral. Color is reserved for approval outcomes and the live blocker.
- Exception actions use secondary styling while a normal path remains.
- Plan check results are never timeline rows.

## Review action (header button)

- **Visibility:** issue OPEN, `approvalStatus` PENDING, and the current user
  is a candidate of the current approval step (reuse the candidate logic in
  `useApprovalStep`, excluding users blocked by the self-approval rule).
  Non-candidates see no button; they comment via the composer.
- **Popover:** Comment / Approve / Reject radio options + markdown editor,
  same pattern as Issue Detail's review popover, rebuilt in the review folder.
  Approve/Reject options render only for candidates (the popover is only
  reachable by candidates, so in practice all three options show).
- **Reject requires a non-empty comment** — submit disabled otherwise. The
  rejection banner depends on the reason always existing. (Issue Detail's
  popover is not changed.)
- **After submit:** patch issue state, `refreshState()`, refetch comments.
  Stay on Plan Detail — no redirect (unlike Issue Detail's post-approve
  navigation). The footer / Deploy section communicate what happens next.

## Approval flow renderer

![Approval flow adaptive compaction at three container widths](assets/2026-06-12-aio-plan-review/approval-flow-compaction.png)

- **Wide containers — horizontal row, adaptive compaction.** Three anchors:
  - Approved steps fold into one leading chip: stacked avatars + "N approved".
  - The current (under-review) step never folds: two rows — role +
    `Current` badge, then candidate avatar group with "+N" overflow. No
    "Waiting for…" text.
  - Trailing pending steps that don't fit fold into one dashed "N pending"
    chip.
  - Fill rule: reserve the three anchors, spend leftover width expanding
    folded nodes nearest the current step, next-first; on shrink they refold,
    current last to go.
- **Narrow / mobile — vertical stepper, always compact.** Same anatomy
  stacked; chips never unfold by width; only the current node is expanded.
- **Hover to reveal:** each chip opens a read-only popover listing its folded
  nodes (approved: role · ✓ · approver · time; pending: role · reviewers).
- Step status (approved / rejected / current / pending) derives from
  `issue.approvers` exactly as today; a rejected step renders a red X node
  inline.
- Layout math is a pure function `computeApprovalFlowLayout(steps, width)`
  driven by a `ResizeObserver` on the section card (the page-level
  `containerWidth` tracks the page, not this card). Unit-tested without
  rendering.

## Rejection banner

![Rejected review state — banner pinned above the timeline](assets/2026-06-12-aio-plan-review/rejected-review.png)

- Renders only while `approvalStatus` is REJECTED, pinned between the
  approval flow and the timeline; red-tinted panel — the single alert-styled
  element on the page (replaces today's warning-styled banner).
- Content: "Rejected by {user} · {relative time}", the full rejection comment
  (markdown), and one guidance line naming both recovery paths: "Update your
  changes to address the feedback, or re-request review without changes" —
  with re-request review as an inline text action, visible only to the issue
  creator (existing `canReRequest` + `requestIssue`).
- Derivation: the latest APPROVAL-type issue comment whose approval status is
  REJECTED (tightens today's `lastRejection`, which takes any APPROVAL
  comment). The banner derives from live approval state and disappears the
  moment review restarts by either path.
- The rejection decision also lands in-stream as a permanent timeline entry
  (card weight, since it carries the comment).
- Editing the plan restarts review on the backend (existing behavior, doc
  CUJ 3); the page's polling/refresh picks up the regenerated flow.
  **Implementation must verify this end-to-end before relying on it; it is an
  assumption about existing backend behavior, not a change.**

## Activity timeline

One unified oldest-first stream. No tabs, no filters.

**Two visual tiers:**

- Full cards (avatar + header + markdown body): user comments; rejection
  decisions with their comment.
- Compact gray one-liners (actor + action + relative time): approvals,
  re-requests, issue status changes (resolved / canceled / reopened), title /
  description / label changes, plan spec updates (statement / targets / prior
  backup — reusing `diffPlanSpecs`, extracted from `issue-detail/utils/` to a
  shared pure module).

**Event sources:**

- Synthetic head rows from metadata: "{creator} created this plan"
  (plan.creator/createTime) and "{creator} marked this plan ready for review"
  (issue.creator/createTime).
- All issue comments from the existing `issueComment` app-store slice.
- Plan check results are never rendered as timeline rows.
- The user's own comments are editable inline (pencil → markdown editor),
  same permission rules as Issue Detail (`bb.issueComments.update` or own
  comment).

**Long-history fold (torn separator):**

![Activity timeline torn-separator fold, first 5 + last 5](assets/2026-06-12-aio-plan-review/activity-timeline-fold.png)

- The first 5 and last 5 entries always render. The middle folds behind a
  full-width torn separator with the exact count and a single action:
  "N hidden events · Show all". One click expands everything in place,
  scroll-anchored; no incremental "load more".
- Fold activates only when the middle has ≥5 hidable events
  (first 5 + last 5 + at least 5 hidden; threshold 15+).
- Comments and decision events inside the folded range stay visible as
  islands — only routine system events fold; the count counts hidden rows
  only.
- `foldTimeline(entries)` is a pure function with unit tests.

## Comment composer

- Last entry of the timeline: collapsed single line (current-user avatar +
  "Add a comment…") for anyone with `bb.issueComments.create`.
- Expands on click/focus into the markdown editor with a Comment submit
  button; posting appends the comment above and re-collapses.
- Unsent draft is preserved in `localStorage`, keyed by issue name, and
  restored on the next expand — no text is ever lost to collapse or
  navigation.

## Rollout readiness footer

A quiet single line under a hairline divider at the bottom of the section.
Exists only while `!plan.hasRollout` and the issue is OPEN; once the rollout
exists the footer disappears and Deploy answers "what happens next".

One action label in every state: **"Bypass and deploy"**. The status line
names what is blocking or running; only the action's weight changes.

![Readiness footer — one label, every gate combination](assets/2026-06-12-aio-plan-review/readiness-footer-states.png)

| State | Status line | Action |
|---|---|---|
| Review in progress · checks passed | clock icon · "Waiting on review · rollout is created automatically after approval · 8 checks passed" | muted underlined link, far right |
| Review in progress · checks failed | same, but "2 checks failed," rendered red; failed count is the only signal | muted link (review is still the normal path) |
| Approved or skipped · checks passed · rollout pending | green check · "All gates passed · creating rollout automatically… · 10 checks passed" | muted link (bypasses the wait on automatic creation) |
| Approved or skipped · checks failed | red check-failure icon · bold "Review approved, but plan checks failed" · "2 errors, 8 passed. Rollout was not created automatically." | primary button — the page's one primary action |
| Rejected | blocked icon · "Blocked by the rejected review — address the feedback above to continue" · checks count | none |

- Check counts come from `getPlanCheckSummary(plan)`.
- **Gating:** the backend enforces only `bb.rollouts.create` on
  `CreateRollout`; the project settings are client-side gates. The action is
  hidden when the user lacks `bb.rollouts.create`, or when a project
  enforcement setting forbids the bypass in the current state
  (`projectRequireIssueApproval` while review is not approved/skipped;
  `projectRequirePlanCheckNoError` while checks fail) — the same rules the
  Deploy section applies today. The status line renders for everyone.
- **Click:** direct `CreateRollout` call — no confirm sheet. The control
  disables while in flight; creation is idempotent, so a click racing the
  automatic creation is safe (first creator wins). On success: refresh state;
  the footer disappears; Deploy becomes the active section. No navigation.

## Critical user journeys (acceptance)

1. **Reviewer approves.** Open Plan Detail → inspect Changes → click Review
   (header) → approve. Their node turns green, the header action disappears,
   the footer shows "All gates passed · creating rollout automatically…".
   Once the rollout is created the footer disappears and Deploy is active.
2. **Reviewer rejects with feedback.** Review → reject with a comment. The
   rejection banner pins above the timeline; the decision also lands
   in-stream as a permanent row; the header action disappears.
3. **Creator recovers from rejection.** The banner pins the reviewer's
   comment with one guidance line naming both paths. Editing the plan inline
   and saving restarts review automatically (banner gone, checks re-run,
   approval flow regenerated). The inline re-request action restarts review
   without changes, keeping earlier approvals.
4. **Participant discusses.** Open the collapsed composer → write markdown →
   post. The comment appears above the composer; the composer re-collapses;
   an unsent draft survives collapse and is restored.
5. **Releaser bypasses the last gate.** Review approved/skipped but checks
   failed → the footer is the page's one primary action → Bypass and deploy →
   rollout created, footer gone, Deploy takes over. In every other
   non-rejected state the same action is a muted link.

## Full-page mockups

Desktop and mobile, full Plan Detail lifecycle with the Review section in
context (mobile uses the vertical, always-compact approval stepper):

![Full plan detail page, desktop and mobile](assets/2026-06-12-aio-plan-review/full-page-desktop-mobile.png)

Approved state — review completed, Deploy active:

![Full plan detail page, approved](assets/2026-06-12-aio-plan-review/full-page-approved.png)

## Data flow

Existing plumbing only: `usePlanDetailPage` polling for plan / issue /
rollout; the `issueComment` slice for the timeline (fetched on mount,
re-fetched when `issue.updateTime` changes and after any action);
`patchState` / `refreshState` after approve / reject / comment / re-request /
bypass. No new stores, no proto or API changes, no backend changes.

## Cleanups folded in

- Delete `PlanDetailApprovalFlow.tsx` (both modes; `sidebar` is dead code).
- **Deploy-future dedup (approved decision):** for issue-backed plans, remove
  the requirements checklist, the manual "Create rollout" button, and its
  confirm sheet from `PlanDetailDeployFuture` — keep only the one-line
  description. The readiness footer is the single source of gate status and
  the single manual path. GitOps (release-backed) plans render no Review
  section and keep their existing Deploy-future creation flow unchanged.

## Error handling

- All connect calls follow the page's existing pattern: `pushNotification`
  CRITICAL on failure, controls disabled while submitting, state refreshed on
  success.
- A bypass click that loses the race to automatic creation surfaces no error
  path the user must handle: refresh shows the created rollout either way.
- Comment list fetch failures degrade to an empty timeline with the existing
  loading/blank states; the approval flow and footer derive from the issue
  object and stay functional.

## i18n

All new strings under nested `plan.review.*` keys in
`frontend/src/locales/`, translated in all five locale files (en-US, zh-CN,
es-ES, ja-JP, vi-VN) — `check-react-i18n.mjs` enforces key parity with en-US,
so fallback-only is not an option. No hardcoded display strings.

## Testing

- Unit tests (vitest, colocated): `computeApprovalFlowLayout` (anchor
  reservation, expand/refold order), `foldTimeline` (threshold, islands,
  counts), footer state mapping (five states × gating), timeline event
  mapping (comment types → tier/sentence).
- Component tests: `PlanReviewSection` in the five states (in progress,
  rejected, approved-checks-failed, all-gates-passed, skipped); review
  popover reject-requires-comment; composer draft persistence.
- Gates: `pnpm --dir frontend fix`, `check`, `type-check`, `test`.

## References

- UX source: "AIO Plan Page UI/UX Design" PDF (repo root, untracked), Google
  Doc 1goYvRASKwAbHt96x3mS8eQOIwaXmonldm37CTzBUrMU. Mockups extracted from the
  PDF into `assets/2026-06-12-aio-plan-review/` and embedded above.
- Prior-art research: 6 comparators, 54 audited claims (Reference section of
  the UX doc) — adopted patterns: single unified timeline, two-tier weight,
  collapsed composer, color discipline; avoided anti-patterns: silent/middle
  truncation, newest-first pagination, decision auto-comments, server-side
  filters, always-expanded composer.
- Current code: `frontend/src/react/pages/project/plan-detail/` (page, shell
  hooks, deploy components), `frontend/src/react/pages/project/issue-detail/`
  (review popover pattern, comment list, diffPlanSpecs),
  `frontend/src/react/stores/app/issueComment.ts`.
