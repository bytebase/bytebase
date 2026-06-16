# AIO Plan Detail Review Section Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring the full issue-backed review workflow (approval flow, review actions, rejection recovery, activity timeline + composer, rollout-readiness footer) into the Plan Detail Review section, per `docs/superpowers/specs/2026-06-12-aio-plan-review-section-design.md`.

**Architecture:** Self-contained React components under `frontend/src/react/pages/project/plan-detail/components/review/`, driven by three unit-tested pure modules (approval-flow layout, timeline fold, footer state). Reuses the existing `issueComment` app-store slice, `approveIssue`/`rejectIssue`/`requestIssue` connect calls, and `PlanDetailContext` plumbing. No backend, proto, or store changes.

**Tech Stack:** React + TypeScript, Tailwind v4 + shadcn-style shared UI (`@/react/components/ui/`), Zustand app store, connect-es clients, vitest.

**Read the spec first** — it embeds the authoritative mockups: `docs/superpowers/specs/2026-06-12-aio-plan-review-section-design.md`.

**Conventions that apply to every task** (from `AGENTS.md` / `frontend/AGENTS.md`):
- All user-facing strings via `useTranslation()` and nested keys in `frontend/src/locales/en-US.json` + `zh-CN.json` (Task 2 adds them all up front). Never hardcode display text.
- Use shared UI components (`Button`, `Popover`, `Tooltip`, `Badge`) — no ad-hoc styled controls; `gap-x-2` for button groups; semantic color tokens (`text-control`, `bg-success`, `text-error`…), never raw colors.
- Tests run with: `pnpm --dir frontend test -- run <path>` (vitest). Type check: `pnpm --dir frontend type-check`. Autofix: `pnpm --dir frontend fix`.
- Commit after each task. Do not switch branches, reset, or cherry-pick.

**File structure (final state):**

```
frontend/src/react/lib/plan/diffPlanSpecs.ts                 (moved from issue-detail/utils/)
frontend/src/react/pages/project/plan-detail/components/review/
├── PlanReviewSection.tsx          assembly + CHECKING/SKIPPED states + comment fetching
├── ReviewSectionHeader.tsx        risk chip, issue chip, Review button + popover host
├── ReviewActionPopover.tsx        comment/approve/reject radio + markdown editor
├── ReviewApprovalFlow.tsx         horizontal adaptive row / vertical compact stepper
├── approvalFlowLayout.ts          pure layout math               + .test.ts
├── useApprovalCandidates.ts       per-role candidate computation (extracted logic)
├── ReviewRejectionBanner.tsx      rejected-state alert + re-request inline action
├── ReviewActivityTimeline.tsx     two-tier rows + torn-separator fold + comment editing
├── timelineEvents.ts              event mapping (pure)           + .test.ts
├── foldTimeline.ts                fold math (pure)               + .test.ts
├── ReviewCommentComposer.tsx      collapsed composer + localStorage draft
├── ReviewReadinessFooter.tsx      status line + Bypass and deploy
└── readinessFooterState.ts        state + gating math (pure)     + .test.ts
Modified:  ProjectPlanDetailPage.tsx (ReviewBranch), PlanDetailDeployFuture.tsx (dedup),
           locales en-US.json / zh-CN.json
Deleted:   components/PlanDetailApprovalFlow.tsx + .test.tsx
```

---

### Task 1: Move `diffPlanSpecs` to the shared plan lib

The timeline needs spec-diff rendering; the module is pure and currently lives under issue-detail.

**Files:**
- Move: `frontend/src/react/pages/project/issue-detail/utils/diffPlanSpecs.ts` → `frontend/src/react/lib/plan/diffPlanSpecs.ts`
- Modify: every importer of the old path

- [ ] **Step 1: Move the file**

```bash
cd frontend && git mv src/react/pages/project/issue-detail/utils/diffPlanSpecs.ts src/react/lib/plan/diffPlanSpecs.ts
```

- [ ] **Step 2: Update importers**

Find them: `grep -rn "utils/diffPlanSpecs\|\./diffPlanSpecs" frontend/src/react --include="*.ts*"`. Expected importers: `issue-detail/components/IssueDetailCommentList.tsx` (relative `../utils/diffPlanSpecs`) and any test file. Change each import to:

```ts
import {
  diffEntryKey,
  diffPlanSpecsForEvent,
  type SpecDiffEntry,
} from "@/react/lib/plan/diffPlanSpecs";
```

If `diffPlanSpecs.test.ts` exists next to the old location, `git mv` it to `frontend/src/react/lib/plan/diffPlanSpecs.test.ts` and fix its relative import to `./diffPlanSpecs`.

- [ ] **Step 3: Verify**

Run: `pnpm --dir frontend type-check && pnpm --dir frontend test -- run src/react/lib/plan`
Expected: type check passes; any moved tests pass.

- [ ] **Step 4: Commit**

```bash
git add -A frontend/src/react && git commit -m "refactor: move diffPlanSpecs to shared plan lib"
```

---

### Task 2: Add all `plan.review.*` locale keys

**Files:**
- Modify: `frontend/src/locales/en-US.json` (inside the existing `"plan"` object — there is no `"review"` key yet)
- Modify: `frontend/src/locales/zh-CN.json` (same position)

- [ ] **Step 1: Add the nested `review` object to `plan` in en-US.json**

```json
"review": {
  "action": "Review",
  "comment-required-to-reject": "A comment is required to reject",
  "approval-flow": {
    "n-approved": "{{n}} approved",
    "n-pending": "{{n}} pending",
    "current": "Current",
    "approved-by": "Approved by {{user}}",
    "rejected-by": "Rejected by {{user}}",
    "n-reviewers": "{{n}} reviewers"
  },
  "rejection": {
    "title": "Rejected by {{user}}",
    "guidance-prefix": "Update your changes to address the feedback, or",
    "re-request-review": "re-request review",
    "guidance-suffix": "without changes"
  },
  "activity": {
    "self": "Activity",
    "created-this-plan": "created this plan",
    "marked-ready-for-review": "marked this plan ready for review",
    "approved-review": "approved {{role}} review",
    "rejected-review": "rejected {{role}} review",
    "re-requested-review": "re-requested review",
    "n-hidden-events": "{{count}} hidden events",
    "show-all": "Show all",
    "add-a-comment": "Add a comment..."
  },
  "footer": {
    "waiting-on-review": "Waiting on review",
    "auto-rollout-after-approval": "rollout is created automatically after approval",
    "n-checks-passed": "{{n}} checks passed",
    "n-checks-failed": "{{n}} checks failed,",
    "all-gates-passed": "All gates passed",
    "creating-rollout-automatically": "creating rollout automatically...",
    "approved-but-checks-failed": "Review approved, but plan checks failed",
    "errors-passed-not-created": "{{errors}} errors, {{passed}} passed. Rollout was not created automatically.",
    "blocked-by-rejection": "Blocked by the rejected review — address the feedback above to continue",
    "bypass-and-deploy": "Bypass and deploy"
  }
}
```

- [ ] **Step 2: Add the zh-CN translations in the same position**

```json
"review": {
  "action": "审批",
  "comment-required-to-reject": "拒绝时必须填写评论",
  "approval-flow": {
    "n-approved": "{{n}} 人已批准",
    "n-pending": "{{n}} 个待审批",
    "current": "当前",
    "approved-by": "由 {{user}} 批准",
    "rejected-by": "由 {{user}} 拒绝",
    "n-reviewers": "{{n}} 位审批人"
  },
  "rejection": {
    "title": "被 {{user}} 拒绝",
    "guidance-prefix": "请根据反馈修改变更，或",
    "re-request-review": "重新发起审批",
    "guidance-suffix": "（不做修改）"
  },
  "activity": {
    "self": "动态",
    "created-this-plan": "创建了此变更单",
    "marked-ready-for-review": "将此变更单提交审批",
    "approved-review": "批准了{{role}}审批",
    "rejected-review": "拒绝了{{role}}审批",
    "re-requested-review": "重新发起了审批",
    "n-hidden-events": "{{count}} 条隐藏动态",
    "show-all": "显示全部",
    "add-a-comment": "添加评论..."
  },
  "footer": {
    "waiting-on-review": "等待审批",
    "auto-rollout-after-approval": "审批通过后将自动创建发布",
    "n-checks-passed": "{{n}} 项检查通过",
    "n-checks-failed": "{{n}} 项检查失败，",
    "all-gates-passed": "所有条件已满足",
    "creating-rollout-automatically": "正在自动创建发布...",
    "approved-but-checks-failed": "审批已通过，但变更检查失败",
    "errors-passed-not-created": "{{errors}} 项错误，{{passed}} 项通过。未自动创建发布。",
    "blocked-by-rejection": "已被拒绝的审批阻塞 — 请先处理上方反馈",
    "bypass-and-deploy": "跳过并发布"
  }
}
```

- [ ] **Step 3: Verify JSON validity**

Run: `python3 -c "import json; json.load(open('frontend/src/locales/en-US.json')); json.load(open('frontend/src/locales/zh-CN.json')); print('ok')"`
Expected: `ok`

- [ ] **Step 4: Commit**

```bash
git add frontend/src/locales && git commit -m "feat(plan): add plan.review locale keys"
```

---

### Task 3: Pure module — `approvalFlowLayout`

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/approvalFlowLayout.ts`
- Test: `frontend/src/react/pages/project/plan-detail/components/review/approvalFlowLayout.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, test } from "vitest";
import {
  computeApprovalFlowLayout,
  VERTICAL_BREAKPOINT_PX,
} from "./approvalFlowLayout";
import type { ApprovalStepStatus } from "./approvalFlowLayout";

const statuses = (s: string): ApprovalStepStatus[] =>
  s.split("") .map((c) =>
    c === "a" ? "approved" : c === "c" ? "current" : c === "r" ? "rejected" : "pending"
  );

describe("computeApprovalFlowLayout", () => {
  test("narrow container switches to vertical", () => {
    const layout = computeApprovalFlowLayout(statuses("aacpp"), 400);
    expect(layout.kind).toBe("vertical");
  });

  test("everything fits: nothing folds", () => {
    const layout = computeApprovalFlowLayout(statuses("aaacppp"), 2000);
    expect(layout).toEqual({
      kind: "horizontal",
      foldedApproved: 0,
      namedPending: 3,
    });
  });

  test("approved folds first; nearest pending stays named", () => {
    // 3 approved + current + 3 pending at ~900px: approved chip + current +
    // 1 named pending + pending chip (per the mockup's middle row).
    const layout = computeApprovalFlowLayout(statuses("aaacppp"), 900);
    expect(layout.kind).toBe("horizontal");
    if (layout.kind !== "horizontal") return;
    expect(layout.foldedApproved).toBe(3);
    expect(layout.namedPending).toBeGreaterThanOrEqual(1);
    expect(layout.namedPending).toBeLessThan(3);
  });

  test("minimum form: chip + current + chip", () => {
    const layout = computeApprovalFlowLayout(statuses("aaacppp"), 640);
    expect(layout).toEqual({
      kind: "horizontal",
      foldedApproved: 3,
      namedPending: 0,
    });
  });

  test("rejected step is the anchor like current", () => {
    const layout = computeApprovalFlowLayout(statuses("aarpp"), 640);
    expect(layout).toEqual({
      kind: "horizontal",
      foldedApproved: 2,
      namedPending: 0,
    });
  });

  test("no approved steps: no approved chip cost, pending folds only as needed", () => {
    const layout = computeApprovalFlowLayout(statuses("cpppp"), 2000);
    expect(layout).toEqual({
      kind: "horizontal",
      foldedApproved: 0,
      namedPending: 4,
    });
  });

  test("exports the vertical breakpoint", () => {
    expect(VERTICAL_BREAKPOINT_PX).toBeGreaterThan(0);
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `pnpm --dir frontend test -- run src/react/pages/project/plan-detail/components/review/approvalFlowLayout.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

```ts
// Pure layout math for the horizontal approval flow (spec: "Approval flow
// renderer"). The current (or rejected) step is the anchor and never folds;
// approved steps fold first into one leading chip; trailing pending steps
// fold into one dashed chip, farthest-from-current first.
export type ApprovalStepStatus = "approved" | "rejected" | "current" | "pending";

export type ApprovalFlowLayout =
  | { kind: "vertical" }
  | {
      kind: "horizontal";
      // Leading approved steps folded into the "N approved" chip (0 = all named).
      foldedApproved: number;
      // Pending steps after the anchor rendered as full named nodes,
      // counted from the one nearest the anchor.
      namedPending: number;
    };

export const VERTICAL_BREAKPOINT_PX = 560;

// Width budget per element, connector included. Estimates — the row also has
// min-w-0 truncation, so being a little conservative is fine.
const NAMED_NODE_PX = 190;
const ANCHOR_NODE_PX = 240;
const CHIP_PX = 150;

export function computeApprovalFlowLayout(
  statuses: ApprovalStepStatus[],
  containerWidth: number
): ApprovalFlowLayout {
  if (containerWidth < VERTICAL_BREAKPOINT_PX) {
    return { kind: "vertical" };
  }

  const anchorIndex = statuses.findIndex(
    (s) => s === "current" || s === "rejected"
  );
  // Fully-approved (or skipped) flows have no anchor; render all named if they
  // fit, otherwise fold approved into the chip.
  const approvedCount = anchorIndex === -1 ? statuses.length : anchorIndex;
  const pendingCount =
    anchorIndex === -1 ? 0 : statuses.length - anchorIndex - 1;
  const anchorCost = anchorIndex === -1 ? 0 : ANCHOR_NODE_PX;

  const fits = (foldedApproved: number, namedPending: number): boolean => {
    const namedApproved = approvedCount - foldedApproved;
    const foldedPending = pendingCount - namedPending;
    const width =
      (foldedApproved > 0 ? CHIP_PX : 0) +
      namedApproved * NAMED_NODE_PX +
      anchorCost +
      namedPending * NAMED_NODE_PX +
      (foldedPending > 0 ? CHIP_PX : 0);
    return width <= containerWidth;
  };

  // 1. Everything named.
  if (fits(0, pendingCount)) {
    return { kind: "horizontal", foldedApproved: 0, namedPending: pendingCount };
  }
  // 2. Approved fold first (all-or-nothing chip), keep pending named, then
  //    fold pending from the far end until it fits.
  for (let named = pendingCount; named >= 0; named--) {
    if (fits(approvedCount, named)) {
      return {
        kind: "horizontal",
        foldedApproved: approvedCount,
        namedPending: named,
      };
    }
  }
  // 3. Minimum form regardless: chip + anchor + chip.
  return { kind: "horizontal", foldedApproved: approvedCount, namedPending: 0 };
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `pnpm --dir frontend test -- run src/react/pages/project/plan-detail/components/review/approvalFlowLayout.test.ts`
Expected: PASS (7 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): approval flow layout math"
```

---

### Task 4: Pure modules — `timelineEvents` + `foldTimeline`

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/timelineEvents.ts`
- Create: `frontend/src/react/pages/project/plan-detail/components/review/foldTimeline.ts`
- Test: `.../review/timelineEvents.test.ts`, `.../review/foldTimeline.test.ts`

- [ ] **Step 1: Write the failing tests**

`timelineEvents.test.ts`:

```ts
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  IssueComment_Approval_Status,
  IssueCommentSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { buildTimelineEntries } from "./timelineEvents";

const userComment = (name: string, comment: string) =>
  create(IssueCommentSchema, { name, comment, creator: "users/a@x.com" });

const approvalComment = (
  name: string,
  status: IssueComment_Approval_Status,
  comment = ""
) =>
  create(IssueCommentSchema, {
    name,
    comment,
    creator: "users/r@x.com",
    event: { case: "approval", value: { status } },
  });

describe("buildTimelineEntries", () => {
  test("synthetic head rows come first, oldest-first", () => {
    const entries = buildTimelineEntries({
      planCreator: "users/a@x.com",
      planCreateTime: { seconds: 1n, nanos: 0 },
      issueCreator: "users/a@x.com",
      issueCreateTime: { seconds: 2n, nanos: 0 },
      comments: [userComment("c1", "hello")],
    });
    expect(entries.map((e) => e.source.type)).toEqual([
      "plan-created",
      "ready-for-review",
      "comment",
    ]);
  });

  test("user comments are island cards", () => {
    const [entry] = buildTimelineEntries({
      comments: [userComment("c1", "hi")],
    });
    expect(entry.tier).toBe("card");
    expect(entry.island).toBe(true);
  });

  test("rejection with comment is an island card; bare approval is a system island", () => {
    const entries = buildTimelineEntries({
      comments: [
        approvalComment("c1", IssueComment_Approval_Status.REJECTED, "no"),
        approvalComment("c2", IssueComment_Approval_Status.APPROVED),
      ],
    });
    expect(entries[0]).toMatchObject({ tier: "card", island: true });
    expect(entries[1]).toMatchObject({ tier: "system", island: true });
  });

  test("issue/plan update events are foldable system rows", () => {
    const update = create(IssueCommentSchema, {
      name: "c1",
      creator: "users/a@x.com",
      event: {
        case: "issueUpdate",
        value: { fromTitle: "a", toTitle: "b", fromLabels: [], toLabels: [] },
      },
    });
    const [entry] = buildTimelineEntries({ comments: [update] });
    expect(entry).toMatchObject({ tier: "system", island: false });
  });
});
```

`foldTimeline.test.ts`:

```ts
import { describe, expect, test } from "vitest";
import { FOLD_MIN_HIDDEN, foldTimeline } from "./foldTimeline";

interface E {
  id: string;
  island: boolean;
}
const sys = (id: string): E => ({ id, island: false });
const island = (id: string): E => ({ id, island: true });

describe("foldTimeline", () => {
  test("short timelines never fold", () => {
    const entries = Array.from({ length: 14 }, (_, i) => sys(`e${i}`));
    const items = foldTimeline(entries, false);
    expect(items).toHaveLength(14);
    expect(items.every((i) => i.type === "entry")).toBe(true);
  });

  test("folds when the middle has >= FOLD_MIN_HIDDEN hidable events", () => {
    const entries = Array.from({ length: 15 }, (_, i) => sys(`e${i}`));
    const items = foldTimeline(entries, false);
    const fold = items.find((i) => i.type === "fold");
    expect(fold).toMatchObject({ type: "fold", count: FOLD_MIN_HIDDEN });
    // first 5 + fold + last 5
    expect(items).toHaveLength(11);
  });

  test("islands in the folded range stay visible and don't count", () => {
    const entries = [
      ...Array.from({ length: 5 }, (_, i) => sys(`head${i}`)),
      sys("h0"),
      island("c0"),
      ...Array.from({ length: 5 }, (_, i) => sys(`h${i + 1}`)),
      ...Array.from({ length: 5 }, (_, i) => sys(`tail${i}`)),
    ];
    const items = foldTimeline(entries, false);
    const ids = items.map((i) => (i.type === "entry" ? i.entry.id : "FOLD"));
    expect(ids).toContain("c0");
    expect(ids).not.toContain("h0");
    const fold = items.find((i) => i.type === "fold");
    expect(fold).toMatchObject({ count: 6 });
    // fold marker sits at the position of the first hidden event (before c0)
    expect(ids.indexOf("FOLD")).toBeLessThan(ids.indexOf("c0"));
  });

  test("middle with too few hidable events does not fold", () => {
    const entries = [
      ...Array.from({ length: 5 }, (_, i) => sys(`head${i}`)),
      sys("h0"),
      ...Array.from({ length: 4 }, (_, i) => island(`c${i}`)),
      ...Array.from({ length: 5 }, (_, i) => sys(`tail${i}`)),
    ];
    expect(foldTimeline(entries, false)).toHaveLength(15);
  });

  test("expanded shows everything", () => {
    const entries = Array.from({ length: 30 }, (_, i) => sys(`e${i}`));
    expect(foldTimeline(entries, true)).toHaveLength(30);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `pnpm --dir frontend test -- run src/react/pages/project/plan-detail/components/review`
Expected: FAIL — modules not found.

- [ ] **Step 3: Implement `timelineEvents.ts`**

```ts
// Maps the review timeline's event sources into a flat, oldest-first entry
// list with the two-tier weight model (spec: "Activity timeline").
// Plan check results are never timeline entries.
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { IssueComment_Approval_Status } from "@/types/proto-es/v1/issue_service_pb";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/react/stores/app/issueComment";

export type TimelineTier = "card" | "system";

export type TimelineSource =
  | { type: "plan-created"; creator: string; time?: Timestamp }
  | { type: "ready-for-review"; creator: string; time?: Timestamp }
  | { type: "comment"; comment: IssueComment };

export interface TimelineEntry {
  id: string;
  tier: TimelineTier;
  // Islands never fold: comments and review decisions stay visible inside
  // the folded range.
  island: boolean;
  source: TimelineSource;
}

export function buildTimelineEntries(input: {
  planCreator?: string;
  planCreateTime?: Timestamp;
  issueCreator?: string;
  issueCreateTime?: Timestamp;
  comments: IssueComment[];
}): TimelineEntry[] {
  const entries: TimelineEntry[] = [];
  if (input.planCreator) {
    entries.push({
      id: "plan-created",
      tier: "system",
      island: false,
      source: {
        type: "plan-created",
        creator: input.planCreator,
        time: input.planCreateTime,
      },
    });
  }
  if (input.issueCreator) {
    entries.push({
      id: "ready-for-review",
      tier: "system",
      island: false,
      source: {
        type: "ready-for-review",
        creator: input.issueCreator,
        time: input.issueCreateTime,
      },
    });
  }
  for (const comment of input.comments) {
    const type = getIssueCommentType(comment);
    const isApproval = type === IssueCommentType.APPROVAL;
    const isCard =
      type === IssueCommentType.USER_COMMENT ||
      (isApproval &&
        comment.event.case === "approval" &&
        comment.event.value.status === IssueComment_Approval_Status.REJECTED &&
        comment.comment !== "");
    entries.push({
      id: comment.name,
      tier: isCard ? "card" : "system",
      island: isCard || isApproval,
      source: { type: "comment", comment },
    });
  }
  return entries;
}
```

- [ ] **Step 4: Implement `foldTimeline.ts`**

```ts
// Torn-separator fold (spec: "Long-history fold"). First FOLD_HEAD and last
// FOLD_TAIL entries always render; in between, only non-island entries hide,
// replaced by a single fold marker carrying the exact hidden count at the
// position of the first hidden entry. One click ("Show all") expands all.
export interface FoldableEntry {
  id: string;
  island: boolean;
}

export type FoldedItem<T extends FoldableEntry> =
  | { type: "entry"; entry: T }
  | { type: "fold"; count: number };

export const FOLD_HEAD = 5;
export const FOLD_TAIL = 5;
export const FOLD_MIN_HIDDEN = 5;

export function foldTimeline<T extends FoldableEntry>(
  entries: T[],
  expanded: boolean
): FoldedItem<T>[] {
  const all = entries.map((entry) => ({ type: "entry" as const, entry }));
  if (expanded || entries.length < FOLD_HEAD + FOLD_TAIL + FOLD_MIN_HIDDEN) {
    return all;
  }
  const middle = entries.slice(FOLD_HEAD, entries.length - FOLD_TAIL);
  const hiddenCount = middle.filter((e) => !e.island).length;
  if (hiddenCount < FOLD_MIN_HIDDEN) {
    return all;
  }
  const items: FoldedItem<T>[] = entries
    .slice(0, FOLD_HEAD)
    .map((entry) => ({ type: "entry" as const, entry }));
  let foldEmitted = false;
  for (const entry of middle) {
    if (entry.island) {
      items.push({ type: "entry", entry });
    } else if (!foldEmitted) {
      items.push({ type: "fold", count: hiddenCount });
      foldEmitted = true;
    }
  }
  for (const entry of entries.slice(entries.length - FOLD_TAIL)) {
    items.push({ type: "entry", entry });
  }
  return items;
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `pnpm --dir frontend test -- run src/react/pages/project/plan-detail/components/review`
Expected: PASS (all tests in both files plus Task 3's).

- [ ] **Step 6: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): timeline event mapping and fold math"
```

---

### Task 5: Pure module — `readinessFooterState`

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/readinessFooterState.ts`
- Test: `.../review/readinessFooterState.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, expect, test } from "vitest";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  computeBypassActionWeight,
  computeReadinessFooterState,
} from "./readinessFooterState";

const checks = (error = 0, success = 8) => ({
  error,
  running: 0,
  success,
  total: error + success,
  warning: 0,
});

const base = {
  hasRollout: false,
  issueStatus: IssueStatus.OPEN,
  checks: checks(),
};

describe("computeReadinessFooterState", () => {
  test("hidden once the rollout exists", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        hasRollout: true,
        approvalStatus: ApprovalStatus.APPROVED,
      }).kind
    ).toBe("hidden");
  });

  test("hidden when the issue is not open", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        issueStatus: IssueStatus.CANCELED,
        approvalStatus: ApprovalStatus.PENDING,
      }).kind
    ).toBe("hidden");
  });

  test("pending review -> waiting-review (checks failed or not)", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        approvalStatus: ApprovalStatus.PENDING,
      }).kind
    ).toBe("waiting-review");
    expect(
      computeReadinessFooterState({
        ...base,
        approvalStatus: ApprovalStatus.PENDING,
        checks: checks(2),
      }).kind
    ).toBe("waiting-review");
  });

  test("approved or skipped with passing checks -> all-gates-passed", () => {
    for (const approvalStatus of [
      ApprovalStatus.APPROVED,
      ApprovalStatus.SKIPPED,
    ]) {
      expect(
        computeReadinessFooterState({ ...base, approvalStatus }).kind
      ).toBe("all-gates-passed");
    }
  });

  test("approved with failed checks -> approved-checks-failed", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        approvalStatus: ApprovalStatus.APPROVED,
        checks: checks(2),
      }).kind
    ).toBe("approved-checks-failed");
  });

  test("rejected -> rejected", () => {
    expect(
      computeReadinessFooterState({
        ...base,
        approvalStatus: ApprovalStatus.REJECTED,
      }).kind
    ).toBe("rejected");
  });
});

describe("computeBypassActionWeight", () => {
  const allowed = {
    canCreateRollout: true,
    requireIssueApproval: false,
    requirePlanCheckNoError: false,
  };

  test("primary button only when approved and checks failed", () => {
    expect(
      computeBypassActionWeight({
        ...allowed,
        state: "approved-checks-failed",
        checksFailed: true,
      })
    ).toBe("button");
  });

  test("muted link while review in progress or while waiting on automation", () => {
    expect(
      computeBypassActionWeight({
        ...allowed,
        state: "waiting-review",
        checksFailed: false,
      })
    ).toBe("link");
    expect(
      computeBypassActionWeight({
        ...allowed,
        state: "waiting-review",
        checksFailed: true,
      })
    ).toBe("link");
    expect(
      computeBypassActionWeight({
        ...allowed,
        state: "all-gates-passed",
        checksFailed: false,
      })
    ).toBe("link");
  });

  test("never shown when rejected or hidden", () => {
    expect(
      computeBypassActionWeight({
        ...allowed,
        state: "rejected",
        checksFailed: false,
      })
    ).toBe("none");
    expect(
      computeBypassActionWeight({
        ...allowed,
        state: "hidden",
        checksFailed: false,
      })
    ).toBe("none");
  });

  test("hidden without bb.rollouts.create", () => {
    expect(
      computeBypassActionWeight({
        ...allowed,
        canCreateRollout: false,
        state: "approved-checks-failed",
        checksFailed: true,
      })
    ).toBe("none");
  });

  test("project enforcement hides the action it would violate", () => {
    expect(
      computeBypassActionWeight({
        ...allowed,
        requireIssueApproval: true,
        state: "waiting-review",
        checksFailed: false,
      })
    ).toBe("none");
    expect(
      computeBypassActionWeight({
        ...allowed,
        requirePlanCheckNoError: true,
        state: "approved-checks-failed",
        checksFailed: true,
      })
    ).toBe("none");
  });
});
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `pnpm --dir frontend test -- run src/react/pages/project/plan-detail/components/review/readinessFooterState.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

```ts
// Footer state machine + bypass gating (spec: "Rollout readiness footer").
// The backend only enforces bb.rollouts.create on CreateRollout; the project
// "require issue approval" / "require no failed checks" settings are
// client-side gates, so the action hides whenever clicking it would violate
// one of them in the current state. The status line renders for everyone.
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { PlanCheckSummary } from "../../utils/phaseSummary";

export type ReadinessFooterStateKind =
  | "hidden"
  | "waiting-review"
  | "all-gates-passed"
  | "approved-checks-failed"
  | "rejected";

export interface ReadinessFooterState {
  kind: ReadinessFooterStateKind;
  checks: PlanCheckSummary;
}

export function computeReadinessFooterState(input: {
  hasRollout: boolean;
  issueStatus: IssueStatus | undefined;
  approvalStatus: ApprovalStatus;
  checks: PlanCheckSummary;
}): ReadinessFooterState {
  const { checks } = input;
  if (input.hasRollout || input.issueStatus !== IssueStatus.OPEN) {
    return { kind: "hidden", checks };
  }
  if (input.approvalStatus === ApprovalStatus.REJECTED) {
    return { kind: "rejected", checks };
  }
  if (
    input.approvalStatus === ApprovalStatus.APPROVED ||
    input.approvalStatus === ApprovalStatus.SKIPPED
  ) {
    return {
      kind: checks.error > 0 ? "approved-checks-failed" : "all-gates-passed",
      checks,
    };
  }
  return { kind: "waiting-review", checks };
}

export type BypassActionWeight = "none" | "link" | "button";

export function computeBypassActionWeight(input: {
  state: ReadinessFooterStateKind;
  canCreateRollout: boolean;
  requireIssueApproval: boolean;
  requirePlanCheckNoError: boolean;
  checksFailed: boolean;
}): BypassActionWeight {
  if (input.state === "hidden" || input.state === "rejected") {
    return "none";
  }
  if (!input.canCreateRollout) {
    return "none";
  }
  if (input.requireIssueApproval && input.state === "waiting-review") {
    return "none";
  }
  if (input.requirePlanCheckNoError && input.checksFailed) {
    return "none";
  }
  return input.state === "approved-checks-failed" ? "button" : "link";
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `pnpm --dir frontend test -- run src/react/pages/project/plan-detail/components/review/readinessFooterState.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): readiness footer state and bypass gating math"
```

---

### Task 6: `useApprovalCandidates` hook

Extract the per-role candidate computation out of `PlanDetailApprovalFlow.tsx`'s `useApprovalStep` (lines ~653–838) so both the approval-flow renderer (avatar groups) and the header Review button (visibility) can use it. This is a move-and-trim, not new logic.

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/useApprovalCandidates.ts`
- Reference (do not modify yet): `frontend/src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.tsx:653-838`

- [ ] **Step 1: Implement the hook**

Copy the candidate-resolution pipeline from `useApprovalStep` verbatim where possible (IAM bindings → group expansion → user fetch → active-USER filter → self-approval filter). The hook is per-role, not per-step-index:

```ts
import { useCallback, useEffect, useMemo, useState } from "react";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { useAppStore } from "@/react/stores/app";
import { ensureGroupIdentifier } from "@/react/stores/app/group";
import { projectNamePrefix, userNamePrefix } from "@/store/modules/v1/common";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  AccountType,
  getAccountTypeByEmail,
  groupBindingPrefix,
} from "@/types/v1/user";
import {
  ensureUserFullName,
  isBindingPolicyExpired,
  memberMapToRolesInProjectIAM,
} from "@/utils";

export interface ApprovalCandidates {
  // Active human users who can act on this role's approval step, with the
  // self-approval restriction already applied.
  candidates: User[];
  // True when the current user may approve/reject this step.
  isCurrentUserCandidate: boolean;
  // True when the current user holds the role but is blocked by the
  // project's self-approval rule.
  selfApprovalBlocked: boolean;
}

export function useApprovalCandidates(
  issue: Issue,
  projectId: string,
  role: string
): ApprovalCandidates {
  const currentUser = useCurrentUser();
  const currentUserEmail = currentUser?.email ?? "";
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useProjectByName(projectName);
  const projectIamPolicy = useAppStore(
    (state) => state.projectPoliciesByName[projectName]
  );
  const batchGetOrFetchUsers = useAppStore(
    (state) => state.batchGetOrFetchUsers
  );
  const batchGetOrFetchGroups = useAppStore(
    (state) => state.batchGetOrFetchGroups
  );
  const groupsByName = useAppStore((state) => state.groupsByName);
  const getGroupByIdentifier = useCallback(
    (identifier: string) => groupsByName[ensureGroupIdentifier(identifier)],
    [groupsByName]
  );
  const [candidates, setCandidates] = useState<User[]>([]);

  // Prefetch any groups bound to this role so candidate expansion can see
  // their members (same dance as the old useApprovalStep).
  const groupNamesKey = useMemo(() => {
    if (!projectIamPolicy) return "";
    const names: string[] = [];
    for (const binding of projectIamPolicy.bindings) {
      if (binding.role !== role || isBindingPolicyExpired(binding)) continue;
      for (const member of binding.members) {
        if (member.startsWith(groupBindingPrefix)) {
          names.push(member);
        }
      }
    }
    return [...new Set(names)].sort().join(" ");
  }, [projectIamPolicy, role]);

  useEffect(() => {
    const names = groupNamesKey ? groupNamesKey.split(" ") : [];
    void batchGetOrFetchGroups(names).catch(() => undefined);
  }, [groupNamesKey, batchGetOrFetchGroups]);

  const candidateEmailsKey = useMemo(() => {
    if (!projectIamPolicy) return "";
    const memberMap = memberMapToRolesInProjectIAM(
      projectIamPolicy,
      role,
      getGroupByIdentifier
    );
    const names: string[] = [];
    for (const fullname of memberMap.keys()) {
      if (fullname.startsWith(userNamePrefix)) {
        names.push(fullname);
      }
    }
    return [...new Set(names)].sort().join(" ");
  }, [projectIamPolicy, role, getGroupByIdentifier]);

  const isCreator = issue.creator === `${userNamePrefix}${currentUserEmail}`;
  const allowSelfApproval = project.allowSelfApproval;

  const filteredEmailsKey = useMemo(() => {
    const emails = candidateEmailsKey ? candidateEmailsKey.split(" ") : [];
    if (!allowSelfApproval && isCreator) {
      return emails
        .filter((e) => e !== `${userNamePrefix}${currentUserEmail}`)
        .join(" ");
    }
    return emails.join(" ");
  }, [allowSelfApproval, candidateEmailsKey, currentUserEmail, isCreator]);

  useEffect(() => {
    let canceled = false;
    const load = async () => {
      const emails = filteredEmailsKey ? filteredEmailsKey.split(" ") : [];
      if (emails.length === 0) {
        setCandidates([]);
        return;
      }
      const users = await batchGetOrFetchUsers(emails.map(ensureUserFullName));
      if (canceled) return;
      setCandidates(
        users
          .filter(
            (user) =>
              user &&
              user.state === State.ACTIVE &&
              getAccountTypeByEmail(user.email) === AccountType.USER
          )
          .sort((left, right) => {
            if (left.email === currentUserEmail) return -1;
            if (right.email === currentUserEmail) return 1;
            return left.title.localeCompare(right.title);
          })
      );
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [batchGetOrFetchUsers, currentUserEmail, filteredEmailsKey]);

  const isCurrentUserCandidate = useMemo(
    () => candidates.some((user) => user.email === currentUserEmail),
    [candidates, currentUserEmail]
  );
  const selfApprovalBlocked = useMemo(() => {
    if (allowSelfApproval || !isCreator) return false;
    const raw = candidateEmailsKey ? candidateEmailsKey.split(" ") : [];
    return raw.includes(`${userNamePrefix}${currentUserEmail}`);
  }, [allowSelfApproval, candidateEmailsKey, currentUserEmail, isCreator]);

  return { candidates, isCurrentUserCandidate, selfApprovalBlocked };
}
```

- [ ] **Step 2: Verify it compiles**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): extract approval candidate computation hook"
```

---

### Task 7: `ReviewApprovalFlow` component

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/ReviewApprovalFlow.tsx`

Renders the adaptive flow per the spec's "Approval flow renderer" section and `assets/2026-06-12-aio-plan-review/approval-flow-compaction.png`. Anatomy:

- **Horizontal** (container ≥ `VERTICAL_BREAKPOINT_PX`): `[N approved chip | named approved nodes] → … → anchor node → named pending nodes → [N pending chip]`, joined by short arrow connectors.
- **Vertical** (< breakpoint): every step as a compact stacked row; only the anchor row carries the candidate avatar group.
- Chips reveal their folded nodes in a hover `Tooltip` (read-only: role · ✓ · approver · time for approved; role · candidate count for pending).
- Step status derives from `issue.approvers[index]` exactly like the old component (`approved` / `rejected` / `current` / `pending`).

- [ ] **Step 1: Implement**

```tsx
import { Check, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getAvatarColor, getInitials } from "@/react/components/UserAvatar";
import { Badge } from "@/react/components/ui/badge";
import { Tooltip } from "@/react/components/ui/tooltip";
import { displayRoleTitleFromList } from "@/react/lib/role";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import { unknownUser } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Approver_Status } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import {
  type ApprovalStepStatus,
  computeApprovalFlowLayout,
} from "./approvalFlowLayout";
import { useApprovalCandidates } from "./useApprovalCandidates";

interface FlowStep {
  index: number;
  role: string;
  status: ApprovalStepStatus;
  approver?: string; // principal of the recorded decision
}

function deriveSteps(issue: Issue): FlowStep[] {
  const roles = issue.approvalTemplate?.flow?.roles ?? [];
  return roles.map((role, index) => {
    const approver = issue.approvers[index];
    let status: ApprovalStepStatus = "pending";
    if (approver?.status === Issue_Approver_Status.APPROVED) {
      status = "approved";
    } else if (approver?.status === Issue_Approver_Status.REJECTED) {
      status = "rejected";
    } else {
      const blocked = roles.slice(0, index).some((_, i) => {
        const prev = issue.approvers[i];
        return prev?.status !== Issue_Approver_Status.APPROVED;
      });
      status = blocked ? "pending" : "current";
    }
    return { index, role, status, approver: approver?.principal };
  });
}

export function ReviewApprovalFlow({ issue }: { issue: Issue }) {
  const hostRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const observer = new ResizeObserver((entries) => {
      setWidth(entries[0]?.contentRect.width ?? 0);
    });
    observer.observe(host);
    return () => observer.disconnect();
  }, []);

  const steps = useMemo(() => deriveSteps(issue), [issue]);
  const layout = useMemo(
    () => computeApprovalFlowLayout(steps.map((s) => s.status), width || 9999),
    [steps, width]
  );

  return (
    <div ref={hostRef} className="min-w-0 px-4 py-3">
      {layout.kind === "vertical" ? (
        <VerticalFlow issue={issue} steps={steps} />
      ) : (
        <HorizontalFlow
          foldedApproved={layout.foldedApproved}
          issue={issue}
          namedPending={layout.namedPending}
          steps={steps}
        />
      )}
    </div>
  );
}

function HorizontalFlow({
  foldedApproved,
  issue,
  namedPending,
  steps,
}: {
  foldedApproved: number;
  issue: Issue;
  namedPending: number;
  steps: FlowStep[];
}) {
  const anchorIndex = steps.findIndex(
    (s) => s.status === "current" || s.status === "rejected"
  );
  const approved = steps.filter((s) => s.status === "approved");
  const foldedApprovedSteps = approved.slice(0, foldedApproved);
  const namedSteps = steps.filter(
    (s) =>
      !foldedApprovedSteps.includes(s) &&
      (anchorIndex === -1 || s.index <= anchorIndex + namedPending)
  );
  const foldedPendingSteps =
    anchorIndex === -1 ? [] : steps.slice(anchorIndex + 1 + namedPending);

  const items: React.ReactNode[] = [];
  if (foldedApprovedSteps.length > 0) {
    items.push(
      <ApprovedChip key="approved-chip" issue={issue} steps={foldedApprovedSteps} />
    );
  }
  for (const step of namedSteps) {
    items.push(<FlowNode key={step.index} issue={issue} step={step} />);
  }
  if (foldedPendingSteps.length > 0) {
    items.push(<PendingChip key="pending-chip" steps={foldedPendingSteps} />);
  }

  return (
    <div className="flex min-w-0 items-center gap-x-2 overflow-hidden">
      {items.map((item, i) => (
        <div key={i} className="flex min-w-0 items-center gap-x-2">
          {i > 0 && (
            <div className="h-px w-8 shrink-0 bg-control-border" aria-hidden />
          )}
          {item}
        </div>
      ))}
    </div>
  );
}

function VerticalFlow({ issue, steps }: { issue: Issue; steps: FlowStep[] }) {
  return (
    <div className="flex flex-col gap-y-3">
      {steps.map((step) => (
        <FlowNode key={step.index} issue={issue} step={step} vertical />
      ))}
    </div>
  );
}

function StatusDot({ status }: { status: ApprovalStepStatus }) {
  return (
    <div
      className={cn(
        "flex size-6 shrink-0 items-center justify-center rounded-full",
        status === "approved" && "bg-success text-white",
        status === "rejected" && "bg-error text-white",
        status === "current" && "bg-accent text-white",
        status === "pending" && "border-2 border-dashed border-control-border"
      )}
    >
      {status === "approved" && <Check className="size-3.5" />}
      {status === "rejected" && <X className="size-3.5" />}
      {status === "current" && <div className="size-2 rounded-full bg-white" />}
    </div>
  );
}

function FlowNode({
  issue,
  step,
  vertical = false,
}: {
  issue: Issue;
  step: FlowStep;
  vertical?: boolean;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const roleList = useAppStore((state) => state.roleList);
  const approverUser = useAppStore((state) =>
    step.approver ? state.getUserByIdentifier(step.approver) : undefined
  );
  const roleName = displayRoleTitleFromList(step.role, roleList);
  const isAnchor = step.status === "current" || step.status === "rejected";

  const subtitle =
    step.status === "approved"
      ? t("plan.review.approval-flow.approved-by", {
          user: (approverUser ?? unknownUser(step.approver ?? "")).title,
        })
      : step.status === "rejected"
        ? t("plan.review.approval-flow.rejected-by", {
            user: (approverUser ?? unknownUser(step.approver ?? "")).title,
          })
        : undefined;

  return (
    <div className={cn("flex min-w-0 items-start gap-x-2", !vertical && "shrink-0")}>
      <StatusDot status={step.status} />
      <div className="min-w-0">
        <div className="flex items-center gap-x-1.5">
          <span className="truncate text-sm font-medium text-main">
            {roleName}
          </span>
          {step.status === "current" && (
            <Badge variant="secondary">
              {t("plan.review.approval-flow.current")}
            </Badge>
          )}
        </div>
        {subtitle ? (
          <div className="truncate text-xs text-control-light">{subtitle}</div>
        ) : isAnchor && step.status === "current" ? (
          <CandidateAvatars issue={issue} projectId={page.projectId} role={step.role} />
        ) : step.status === "pending" ? (
          <CandidateCount issue={issue} projectId={page.projectId} role={step.role} />
        ) : null}
      </div>
    </div>
  );
}

function CandidateAvatars({
  issue,
  projectId,
  role,
}: {
  issue: Issue;
  projectId: string;
  role: string;
}) {
  const { candidates } = useApprovalCandidates(issue, projectId, role);
  const visible = candidates.slice(0, 4);
  const overflow = candidates.length - visible.length;
  if (candidates.length === 0) return null;
  return (
    <div className="mt-1 flex items-center -space-x-1">
      {visible.map((user) => {
        const name = user.title || user.email.split("@")[0];
        return (
          <Tooltip content={user.email} key={user.name}>
            <span
              className="flex size-5 items-center justify-center rounded-full text-[10px] font-medium text-white ring-2 ring-white"
              style={{ backgroundColor: getAvatarColor(name) }}
            >
              {getInitials(name)}
            </span>
          </Tooltip>
        );
      })}
      {overflow > 0 && (
        <span className="flex size-5 items-center justify-center rounded-full bg-control-bg text-[10px] text-control ring-2 ring-white">
          +{overflow}
        </span>
      )}
    </div>
  );
}

function CandidateCount({
  issue,
  projectId,
  role,
}: {
  issue: Issue;
  projectId: string;
  role: string;
}) {
  const { t } = useTranslation();
  const { candidates } = useApprovalCandidates(issue, projectId, role);
  if (candidates.length === 0) return null;
  return (
    <div className="text-xs text-control-light">
      {t("plan.review.approval-flow.n-reviewers", { n: candidates.length })}
    </div>
  );
}

function ApprovedChip({ issue, steps }: { issue: Issue; steps: FlowStep[] }) {
  const { t } = useTranslation();
  const roleList = useAppStore((state) => state.roleList);
  const usersByIdentifier = useAppStore((state) => state.getUserByIdentifier);
  return (
    <Tooltip
      content={
        <div className="flex flex-col gap-y-1">
          {steps.map((step) => (
            <div key={step.index} className="flex items-center gap-x-1 text-xs">
              <span>{displayRoleTitleFromList(step.role, roleList)}</span>
              <Check className="size-3 text-success" />
              <span>
                {(usersByIdentifier(step.approver ?? "") ??
                  unknownUser(step.approver ?? "")).title}
              </span>
            </div>
          ))}
        </div>
      }
    >
      <div className="flex shrink-0 cursor-default items-center gap-x-1.5 rounded-full border px-2.5 py-1">
        <div className="flex size-4 items-center justify-center rounded-full bg-success text-white">
          <Check className="size-3" />
        </div>
        <ChipAvatars principals={steps.map((s) => s.approver ?? "")} />
        <span className="text-xs font-medium text-control">
          {t("plan.review.approval-flow.n-approved", { n: steps.length })}
        </span>
      </div>
    </Tooltip>
  );
}

function ChipAvatars({ principals }: { principals: string[] }) {
  const getUserByIdentifier = useAppStore((state) => state.getUserByIdentifier);
  return (
    <span className="flex items-center -space-x-1">
      {principals.slice(0, 3).map((principal, i) => {
        const user = getUserByIdentifier(principal) ?? unknownUser(principal);
        const name = user.title || user.email.split("@")[0];
        return (
          <span
            className="flex size-4 items-center justify-center rounded-full text-[9px] font-medium text-white ring-1 ring-white"
            key={`${principal}-${i}`}
            style={{ backgroundColor: getAvatarColor(name) }}
          >
            {getInitials(name)}
          </span>
        );
      })}
    </span>
  );
}

function PendingChip({ steps }: { steps: FlowStep[] }) {
  const { t } = useTranslation();
  const roleList = useAppStore((state) => state.roleList);
  return (
    <Tooltip
      content={
        <div className="flex flex-col gap-y-1">
          {steps.map((step) => (
            <div key={step.index} className="text-xs">
              {displayRoleTitleFromList(step.role, roleList)}
            </div>
          ))}
        </div>
      }
    >
      <div className="flex shrink-0 cursor-default items-center gap-x-1.5 rounded-full border border-dashed px-2.5 py-1">
        <span className="text-xs text-control-light">
          {t("plan.review.approval-flow.n-pending", { n: steps.length })}
        </span>
      </div>
    </Tooltip>
  );
}
```

Export `deriveSteps` as well (the rejection banner and header need step status): add `export` to its declaration and export the `FlowStep` interface.

- [ ] **Step 2: Verify it compiles**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): adaptive approval flow renderer"
```

---

### Task 8: `ReviewRejectionBanner` component

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/ReviewRejectionBanner.tsx`

Per spec "Rejection banner" + `assets/2026-06-12-aio-plan-review/rejected-review.png`: red-tinted panel, "Rejected by {user} · {time}", full comment as markdown, one guidance line with the inline re-request action (creator only).

- [ ] **Step 1: Implement**

```tsx
import { create } from "@bufbuild/protobuf";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/react/stores/app/issueComment";
import { pushNotification } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueComment_Approval_Status,
  RequestIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";

export function findLastRejection(
  comments: IssueComment[]
): IssueComment | undefined {
  for (let i = comments.length - 1; i >= 0; i--) {
    const comment = comments[i];
    if (
      getIssueCommentType(comment) === IssueCommentType.APPROVAL &&
      comment.event.case === "approval" &&
      comment.event.value.status === IssueComment_Approval_Status.REJECTED
    ) {
      return comment;
    }
  }
  return undefined;
}

export function ReviewRejectionBanner({
  comments,
  issue,
}: {
  comments: IssueComment[];
  issue: Issue;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const currentUser = useCurrentUser();
  const [reRequesting, setReRequesting] = useState(false);
  const rejection = useMemo(() => findLastRejection(comments), [comments]);
  const rejectorUser = useAppStore((state) =>
    rejection ? state.getUserByIdentifier(rejection.creator) : undefined
  );

  if (issue.approvalStatus !== ApprovalStatus.REJECTED || !rejection) {
    return null;
  }

  const rejector = rejectorUser ?? unknownUser(rejection.creator);
  const isCreator =
    issue.creator === `${userNamePrefix}${currentUser?.email ?? ""}`;

  const handleReRequest = async () => {
    if (reRequesting) return;
    try {
      setReRequesting(true);
      const response = await issueServiceClientConnect.requestIssue(
        create(RequestIssueRequestSchema, { name: issue.name })
      );
      page.patchState({ issue: response });
      await page.refreshState();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setReRequesting(false);
    }
  };

  return (
    <div className="mx-4 mt-3 rounded-md border border-error/30 bg-error/5 px-3 py-2.5">
      <div className="flex items-center gap-x-2 text-sm font-medium text-error">
        <span>
          {t("plan.review.rejection.title", { user: rejector.title })}
        </span>
        {rejection.createTime && (
          <HumanizeTs
            className="text-xs font-normal text-error/70"
            ts={getTimeForPbTimestampProtoEs(rejection.createTime, 0) / 1000}
          />
        )}
      </div>
      {rejection.comment && (
        <div className="mt-1 text-sm text-control">
          <MarkdownEditor content={rejection.comment} mode="preview" />
        </div>
      )}
      <div className="mt-1.5 text-xs text-control-light">
        {t("plan.review.rejection.guidance-prefix")}{" "}
        {isCreator && !page.readonly ? (
          <button
            className="underline hover:text-control disabled:opacity-60"
            disabled={reRequesting}
            onClick={() => void handleReRequest()}
            type="button"
          >
            {t("plan.review.rejection.re-request-review")}
          </button>
        ) : (
          <span>{t("plan.review.rejection.re-request-review")}</span>
        )}{" "}
        {t("plan.review.rejection.guidance-suffix")}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify it compiles**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): rejection banner with inline re-request"
```

---

### Task 9: `ReviewCommentComposer` + `ReviewActivityTimeline`

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/ReviewCommentComposer.tsx`
- Create: `frontend/src/react/pages/project/plan-detail/components/review/ReviewActivityTimeline.tsx`

- [ ] **Step 1: Implement the composer**

Collapsed one-liner that expands into the markdown editor; draft persisted to localStorage keyed by issue name (spec: "Comment composer").

```tsx
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";

const draftKey = (issueName: string) => `bb.plan-review.draft.${issueName}`;

export function ReviewCommentComposer({ issueName }: { issueName: string }) {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const [expanded, setExpanded] = useState(false);
  const [content, setContent] = useState("");
  const [posting, setPosting] = useState(false);

  // Restore the unsent draft on mount / issue change.
  useEffect(() => {
    const draft = localStorage.getItem(draftKey(issueName)) ?? "";
    setContent(draft);
    setExpanded(false);
  }, [issueName]);

  const persistDraft = useCallback(
    (value: string) => {
      setContent(value);
      if (value) {
        localStorage.setItem(draftKey(issueName), value);
      } else {
        localStorage.removeItem(draftKey(issueName));
      }
    },
    [issueName]
  );

  const post = async () => {
    if (!content.trim() || posting) return;
    try {
      setPosting(true);
      await useAppStore.getState().createIssueComment({
        issueName,
        comment: content,
      });
      persistDraft("");
      setExpanded(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setPosting(false);
    }
  };

  if (!expanded) {
    return (
      <button
        className="flex w-full items-center gap-x-3 rounded-md border px-3 py-2 text-left text-sm text-control-placeholder hover:border-control-border"
        onClick={() => setExpanded(true)}
        type="button"
      >
        <UserAvatar size="sm" title={currentUser?.title || currentUser?.email} />
        <span>{t("plan.review.activity.add-a-comment")}</span>
      </button>
    );
  }

  return (
    <div className="flex items-start gap-x-3">
      <div className="shrink-0 pt-1">
        <UserAvatar size="sm" title={currentUser?.title || currentUser?.email} />
      </div>
      <div className="min-w-0 flex-1">
        <MarkdownEditor
          content={content}
          onChange={persistDraft}
          onSubmit={() => void post()}
          placeholder={t("plan.review.activity.add-a-comment")}
        />
        <div className="mt-2 flex items-center justify-end gap-x-2">
          <Button
            onClick={() => setExpanded(false)}
            size="sm"
            variant="ghost"
          >
            {t("common.cancel")}
          </Button>
          <Button
            disabled={posting || content.trim().length === 0}
            onClick={() => void post()}
            size="sm"
          >
            {posting && <Loader2 className="size-4 animate-spin" />}
            {t("common.comment")}
          </Button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Implement the timeline**

Two-tier rows per `timelineEvents.ts`, fold per `foldTimeline.ts` (torn separator per `assets/2026-06-12-aio-plan-review/activity-timeline-fold.png`), inline editing of own comments, composer as last entry.

```tsx
import { Loader2, Pencil } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { UserAvatar } from "@/react/components/UserAvatar";
import { Button } from "@/react/components/ui/button";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { useProjectByName } from "@/react/hooks/useProjectByName";
import { diffEntryKey, diffPlanSpecsForEvent } from "@/react/lib/plan/diffPlanSpecs";
import { displayRoleTitleFromList } from "@/react/lib/role";
import { cn } from "@/react/lib/utils";
import { useAppStore } from "@/react/stores/app";
import {
  getIssueCommentType,
  IssueCommentType,
} from "@/react/stores/app/issueComment";
import { extractUserEmail, pushNotification } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs, unknownUser } from "@/types";
import type { Issue, IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueComment_Approval_Status,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { foldTimeline } from "./foldTimeline";
import { ReviewCommentComposer } from "./ReviewCommentComposer";
import {
  buildTimelineEntries,
  type TimelineEntry,
  type TimelineSource,
} from "./timelineEvents";

export function ReviewActivityTimeline({
  comments,
  issue,
  plan,
}: {
  comments: IssueComment[];
  issue: Issue;
  plan: Plan;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const project = useProjectByName(`${projectNamePrefix}${page.projectId}`);
  const [expanded, setExpanded] = useState(false);

  const entries = useMemo(
    () =>
      buildTimelineEntries({
        planCreator: plan.creator,
        planCreateTime: plan.createTime,
        issueCreator: issue.creator,
        issueCreateTime: issue.createTime,
        comments,
      }),
    [comments, issue.createTime, issue.creator, plan.createTime, plan.creator]
  );
  const items = useMemo(() => foldTimeline(entries, expanded), [entries, expanded]);
  const allowComment = Boolean(
    project &&
      issue.status === IssueStatus.OPEN &&
      hasProjectPermissionV2(project, "bb.issueComments.create")
  );

  return (
    <div className="flex flex-col gap-y-2 px-4 pb-3">
      <h4 className="text-sm font-medium text-main">
        {t("plan.review.activity.self")}
      </h4>
      <div className="flex flex-col gap-y-2">
        {items.map((item, i) =>
          item.type === "fold" ? (
            <TornSeparator
              count={item.count}
              key={`fold-${i}`}
              onShowAll={() => setExpanded(true)}
            />
          ) : (
            <TimelineRow entry={item.entry} key={item.entry.id} />
          )
        )}
        {allowComment && <ReviewCommentComposer issueName={issue.name} />}
      </div>
    </div>
  );
}

function TornSeparator({
  count,
  onShowAll,
}: {
  count: number;
  onShowAll: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center gap-x-3 py-1">
      <div className="h-px flex-1 border-t border-dashed border-control-border" />
      <div className="flex flex-col items-center rounded-sm border bg-control-bg/50 px-3 py-1">
        <span className="text-xs text-control">
          {t("plan.review.activity.n-hidden-events", { count })}
        </span>
        <button
          className="text-xs text-accent hover:underline"
          onClick={onShowAll}
          type="button"
        >
          {t("plan.review.activity.show-all")}
        </button>
      </div>
      <div className="h-px flex-1 border-t border-dashed border-control-border" />
    </div>
  );
}

function TimelineRow({ entry }: { entry: TimelineEntry }) {
  if (entry.tier === "card") {
    // Cards are always comments (user comments or rejections with a body).
    if (entry.source.type !== "comment") return null;
    return <CommentCard comment={entry.source.comment} />;
  }
  return <SystemRow source={entry.source} />;
}

function ActorName({ principal }: { principal: string }) {
  const user = useAppStore((state) => state.getUserByIdentifier(principal));
  const resolved = user ?? unknownUser(principal);
  return (
    <span className="font-medium text-main">
      {resolved.title || resolved.email}
    </span>
  );
}

function SystemRow({ source }: { source: TimelineSource }) {
  const { t } = useTranslation();

  if (source.type === "plan-created" || source.type === "ready-for-review") {
    return (
      <div className="flex min-w-0 items-center gap-x-2 text-sm text-control-light">
        <ActorName principal={source.creator} />
        <span className="min-w-0 truncate">
          {source.type === "plan-created"
            ? t("plan.review.activity.created-this-plan")
            : t("plan.review.activity.marked-ready-for-review")}
        </span>
        {source.time && (
          <HumanizeTs
            className="shrink-0 text-xs text-control-placeholder"
            ts={getTimeForPbTimestampProtoEs(source.time, 0) / 1000}
          />
        )}
      </div>
    );
  }

  const comment = source.comment;
  return (
    <div className="flex min-w-0 items-center gap-x-2 text-sm text-control-light">
      <ActorName principal={comment.creator} />
      <span className="min-w-0 truncate">
        <SystemSentence comment={comment} />
      </span>
      {comment.createTime && (
        <HumanizeTs
          className="shrink-0 text-xs text-control-placeholder"
          ts={getTimeForPbTimestampProtoEs(comment.createTime, 0) / 1000}
        />
      )}
    </div>
  );
}

function SystemSentence({ comment }: { comment: IssueComment }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const roleList = useAppStore((state) => state.roleList);
  const type = getIssueCommentType(comment);

  if (type === IssueCommentType.APPROVAL && comment.event.case === "approval") {
    const { status } = comment.event.value;
    // The role of the decided step isn't on the event; use the generic
    // sentence from the issue-detail vocabulary when it can't be derived.
    const stepIndex = page.issue?.approvers.findIndex(
      (a) => a.principal === comment.creator
    );
    const role =
      stepIndex !== undefined && stepIndex >= 0
        ? page.issue?.approvalTemplate?.flow?.roles[stepIndex]
        : undefined;
    const roleTitle = role ? displayRoleTitleFromList(role, roleList) : "";
    if (status === IssueComment_Approval_Status.APPROVED) {
      return roleTitle
        ? t("plan.review.activity.approved-review", { role: roleTitle })
        : t("custom-approval.issue-review.approved-issue");
    }
    if (status === IssueComment_Approval_Status.REJECTED) {
      return roleTitle
        ? t("plan.review.activity.rejected-review", { role: roleTitle })
        : t("custom-approval.issue-review.rejected-issue");
    }
    return t("plan.review.activity.re-requested-review");
  }

  if (type === IssueCommentType.ISSUE_UPDATE && comment.event.case === "issueUpdate") {
    const e = comment.event.value;
    if (e.fromTitle !== undefined && e.toTitle !== undefined) {
      return t("activity.sentence.changed-from-to", {
        name: t("issue.issue-name").toLowerCase(),
        newValue: e.toTitle,
        oldValue: e.fromTitle,
      });
    }
    if (e.fromDescription !== undefined && e.toDescription !== undefined) {
      return t("activity.sentence.changed-description");
    }
    if (e.fromStatus !== undefined && e.toStatus !== undefined) {
      if (e.toStatus === IssueStatus.DONE)
        return t("activity.sentence.resolved-issue");
      if (e.toStatus === IssueStatus.CANCELED)
        return t("activity.sentence.canceled-issue");
      return t("activity.sentence.reopened-issue");
    }
    if (e.fromLabels.length !== 0 || e.toLabels.length !== 0) {
      return t("activity.sentence.changed-labels");
    }
    return t("common.updated");
  }

  if (type === IssueCommentType.PLAN_UPDATE && comment.event.case === "planUpdate") {
    const entries = diffPlanSpecsForEvent(comment.event.value);
    if (entries.length === 0) return t("common.updated");
    const labels = entries.map((entry) =>
      entry.kind === "added"
        ? t("activity.sentence.added-spec")
        : entry.kind === "removed"
          ? t("activity.sentence.removed-spec")
          : entry.sheetChanged
            ? t("activity.sentence.modified-sql-of")
            : entry.targetsChanged
              ? t("activity.sentence.changed-targets-of")
              : t("common.updated")
    );
    return (
      <>
        {entries.map((entry, i) => (
          <span key={diffEntryKey(entry)}>
            {i > 0 && ", "}
            {labels[i]} {t("plan.spec.change")}
          </span>
        ))}
      </>
    );
  }

  return t("common.updated");
}

function CommentCard({ comment }: { comment: IssueComment }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const currentUser = useCurrentUser();
  const project = useProjectByName(`${projectNamePrefix}${page.projectId}`);
  const creatorUser = useAppStore((state) =>
    state.getUserByIdentifier(comment.creator)
  );
  const creator = creatorUser ?? unknownUser(comment.creator);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState(comment.comment);
  const [saving, setSaving] = useState(false);

  const allowEdit = Boolean(
    project &&
      (currentUser?.email === extractUserEmail(comment.creator) ||
        hasProjectPermissionV2(project, "bb.issueComments.update"))
  );
  const createdTs = getTimeForPbTimestampProtoEs(comment.createTime, 0);
  const updatedTs = getTimeForPbTimestampProtoEs(comment.updateTime, 0);
  const isEdited =
    createdTs !== updatedTs &&
    getIssueCommentType(comment) === IssueCommentType.USER_COMMENT;
  const isRejection =
    getIssueCommentType(comment) === IssueCommentType.APPROVAL;

  const save = async () => {
    if (!editContent || editContent === comment.comment) {
      setIsEditing(false);
      return;
    }
    try {
      setSaving(true);
      await useAppStore.getState().updateIssueComment({
        issueCommentName: comment.name,
        comment: editContent,
      });
      setIsEditing(false);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex items-start gap-x-3">
      <div className="shrink-0 pt-1.5">
        <UserAvatar size="sm" title={creator.title || creator.email} />
      </div>
      <div
        className={cn(
          "min-w-0 flex-1 overflow-hidden rounded-md border",
          isRejection && "border-error/30"
        )}
      >
        <div
          className={cn(
            "flex items-center justify-between px-3 py-1.5",
            isRejection ? "bg-error/5" : "bg-control-bg/50"
          )}
        >
          <div className="flex min-w-0 flex-wrap items-center gap-x-2 text-sm">
            <span className="font-medium text-main">
              {creator.title || creator.email}
            </span>
            {isRejection && (
              <span className="text-control-light">
                <SystemSentence comment={comment} />
              </span>
            )}
            {comment.createTime && (
              <HumanizeTs
                className="text-xs text-control-placeholder"
                ts={createdTs / 1000}
              />
            )}
            {isEdited && (
              <span className="text-xs text-control-placeholder">
                ({t("common.edited")})
              </span>
            )}
          </div>
          {allowEdit && !isEditing && !isRejection && (
            <Button
              onClick={() => {
                setEditContent(comment.comment);
                setIsEditing(true);
              }}
              size="xs"
              variant="ghost"
            >
              <Pencil className="size-3.5" />
            </Button>
          )}
        </div>
        <div className="border-t px-3 py-2 text-sm text-control">
          {isEditing ? (
            <div>
              <MarkdownEditor
                content={editContent}
                onChange={setEditContent}
                onSubmit={() => void save()}
              />
              <div className="mt-2 flex items-center justify-end gap-x-2">
                <Button
                  onClick={() => setIsEditing(false)}
                  size="xs"
                  variant="ghost"
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  disabled={saving || editContent === comment.comment}
                  onClick={() => void save()}
                  size="xs"
                >
                  {saving && <Loader2 className="size-3.5 animate-spin" />}
                  {t("common.save")}
                </Button>
              </div>
            </div>
          ) : (
            <MarkdownEditor content={comment.comment} mode="preview" />
          )}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Verify it compiles**

Run: `pnpm --dir frontend type-check`
Expected: PASS. (If `plan.createTime` doesn't exist on the Plan proto, use `plan.createTime` per `plan_service_pb` — verify with `grep -n "createTime" frontend/src/types/proto-es/v1/plan_service_pb.d.ts`; it exists on Plan.)

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): activity timeline with fold and draft-preserving composer"
```

---

### Task 10: `ReviewActionPopover` + `ReviewSectionHeader`

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/ReviewActionPopover.tsx`
- Create: `frontend/src/react/pages/project/plan-detail/components/review/ReviewSectionHeader.tsx`

- [ ] **Step 1: Implement the popover**

Comment / Approve / Reject radio + markdown, like Issue Detail's review popover but: reject requires a non-empty comment, and submission never navigates away (spec: "Review action").

```tsx
import { create } from "@bufbuild/protobuf";
import { Check, Loader2, MessageCircle, X } from "lucide-react";
import { type ReactNode, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { MarkdownEditor } from "@/react/components/MarkdownEditor";
import { Button } from "@/react/components/ui/button";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  ApproveIssueRequestSchema,
  RejectIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";

type ReviewAction = "COMMENT" | "APPROVE" | "REJECT";

export function ReviewActionPopover({
  issue,
  onClose,
}: {
  issue: Issue;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [action, setAction] = useState<ReviewAction>("COMMENT");
  const [comment, setComment] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setAction("COMMENT");
    setComment("");
  }, [issue.name]);

  const commentMissing = comment.trim().length === 0;
  const submitDisabled =
    loading ||
    (action === "COMMENT" && commentMissing) ||
    (action === "REJECT" && commentMissing);

  const submit = async () => {
    if (submitDisabled) return;
    try {
      setLoading(true);
      if (action === "APPROVE") {
        const response = await issueServiceClientConnect.approveIssue(
          create(ApproveIssueRequestSchema, { comment, name: issue.name })
        );
        page.patchState({ issue: response });
      } else if (action === "REJECT") {
        const response = await issueServiceClientConnect.rejectIssue(
          create(RejectIssueRequestSchema, { comment, name: issue.name })
        );
        page.patchState({ issue: response });
      } else {
        await useAppStore.getState().createIssueComment({
          issueName: issue.name,
          comment,
        });
      }
      await page.refreshState();
      onClose();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex w-[min(34rem,calc(100vw-2rem))] flex-col gap-y-3">
      <MarkdownEditor
        content={comment}
        onChange={setComment}
        onSubmit={() => void submit()}
      />
      <div className="flex flex-col gap-y-2.5">
        <ReviewOption
          description={t("issue.review.comment-description")}
          icon={<MessageCircle className="size-4 text-control" />}
          label={t("common.comment")}
          onSelect={() => setAction("COMMENT")}
          selected={action === "COMMENT"}
        />
        <ReviewOption
          description={t("issue.review.approve-description")}
          icon={<Check className="size-4 text-success" />}
          label={t("common.approve")}
          onSelect={() => setAction("APPROVE")}
          selected={action === "APPROVE"}
        />
        <ReviewOption
          description={t("issue.review.reject-description")}
          icon={<X className="size-4 text-error" />}
          label={t("common.reject")}
          onSelect={() => setAction("REJECT")}
          selected={action === "REJECT"}
        />
      </div>
      <div className="flex items-center justify-start gap-x-2 pt-1">
        <Tooltip
          content={
            action === "REJECT" && commentMissing
              ? t("plan.review.comment-required-to-reject")
              : undefined
          }
        >
          <span className="inline-flex">
            <Button
              disabled={submitDisabled}
              onClick={() => void submit()}
              size="sm"
            >
              {loading && <Loader2 className="size-4 animate-spin" />}
              {t("common.submit")}
            </Button>
          </span>
        </Tooltip>
        <Button onClick={onClose} size="sm" variant="ghost">
          {t("common.cancel")}
        </Button>
      </div>
    </div>
  );
}

function ReviewOption({
  description,
  icon,
  label,
  onSelect,
  selected,
}: {
  description?: string;
  icon?: ReactNode;
  label: string;
  onSelect: () => void;
  selected: boolean;
}) {
  return (
    <label
      className={cnSelected(selected)}
    >
      <input
        checked={selected}
        className="mt-1 size-4 accent-accent"
        onChange={onSelect}
        type="radio"
      />
      {icon && <span className="mt-1 shrink-0">{icon}</span>}
      <span className="flex flex-col">
        <span className="text-sm font-medium leading-6">{label}</span>
        {description && (
          <span className="text-xs text-control-light">{description}</span>
        )}
      </span>
    </label>
  );
}

function cnSelected(selected: boolean) {
  return [
    "flex cursor-pointer items-start gap-3 text-left transition-colors",
    selected ? "text-main" : "text-control",
  ].join(" ");
}
```

- [ ] **Step 2: Implement the section header**

Risk chip + issue chip + the only header action (spec: "Review action"; mockup `review-in-progress.png`).

```tsx
import { ChevronDown, ShieldAlert, ShieldCheck } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { RouterLink } from "@/react/components/RouterLink";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/react/router/handles";
import { ApprovalStatus, RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { extractIssueUID } from "@/utils/v1/issue/issue";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { deriveSteps } from "./ReviewApprovalFlow";
import { ReviewActionPopover } from "./ReviewActionPopover";
import { useApprovalCandidates } from "./useApprovalCandidates";

export function PlanReviewSectionHeader({ issue }: { issue: Issue }) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [open, setOpen] = useState(false);
  const issueUID = extractIssueUID(issue.name);

  const currentRole = useMemo(() => {
    const steps = deriveSteps(issue);
    return steps.find((s) => s.status === "current")?.role ?? "";
  }, [issue]);
  const { isCurrentUserCandidate } = useApprovalCandidates(
    issue,
    page.projectId,
    currentRole
  );

  const showReview =
    !page.readonly &&
    issue.status === IssueStatus.OPEN &&
    issue.approvalStatus === ApprovalStatus.PENDING &&
    currentRole !== "" &&
    isCurrentUserCandidate;

  return (
    <div className="flex items-center gap-x-2 px-4 pt-3">
      <h3 className="text-base font-medium text-main">
        {t("issue.approval-flow.self")}
      </h3>
      <div className="flex min-w-0 flex-1 items-center gap-x-1.5 pl-1">
        <RiskChip riskLevel={issue.riskLevel} />
        <RouterLink
          className="inline-flex shrink-0 items-center rounded-full border px-2 py-0.5 text-xs text-control hover:border-control-border"
          to={{
            name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
            params: { issueId: issueUID, projectId: page.projectId },
          }}
        >
          {t("common.issue")} #{issueUID}
        </RouterLink>
      </div>
      {showReview && (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger render={<Button className="gap-x-1.5" />}>
            {t("plan.review.action")}
            <ChevronDown className="size-4" />
          </PopoverTrigger>
          <PopoverContent align="end" className="px-4 py-4">
            <ReviewActionPopover issue={issue} onClose={() => setOpen(false)} />
          </PopoverContent>
        </Popover>
      )}
    </div>
  );
}

function RiskChip({ riskLevel }: { riskLevel: RiskLevel }) {
  const { t } = useTranslation();
  if (riskLevel === RiskLevel.RISK_LEVEL_UNSPECIFIED) return null;
  const label =
    riskLevel === RiskLevel.LOW
      ? t("issue.risk-level.low")
      : riskLevel === RiskLevel.MODERATE
        ? t("issue.risk-level.moderate")
        : t("issue.risk-level.high");
  const Icon = riskLevel === RiskLevel.LOW ? ShieldCheck : ShieldAlert;
  return (
    <span className="inline-flex shrink-0 items-center gap-x-1 rounded-full border px-2 py-0.5 text-xs text-control">
      <Icon
        className={
          riskLevel === RiskLevel.LOW
            ? "size-3.5 text-success"
            : riskLevel === RiskLevel.MODERATE
              ? "size-3.5 text-warning"
              : "size-3.5 text-error"
        }
      />
      {label}
    </span>
  );
}
```

Note: the header keeps the risk-level *tooltip* semantics of the old component as a visible chip per the mockup ("High risk" pill).

- [ ] **Step 3: Verify it compiles**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): review header action and review popover"
```

---

### Task 11: `ReviewReadinessFooter`

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/ReviewReadinessFooter.tsx`

Per spec table and `assets/2026-06-12-aio-plan-review/readiness-footer-states.png`.

- [ ] **Step 1: Implement**

```tsx
import { create } from "@bufbuild/protobuf";
import { Ban, CircleCheck, CircleX, Clock3, Loader2 } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { pushNotification } from "@/store";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { getPlanCheckSummary } from "../../utils/phaseSummary";
import {
  computeBypassActionWeight,
  computeReadinessFooterState,
} from "./readinessFooterState";

export function ReviewReadinessFooter({
  issue,
  plan,
}: {
  issue: Issue;
  plan: Plan;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [creating, setCreating] = useState(false);

  const checks = useMemo(() => getPlanCheckSummary(plan), [plan]);
  const state = useMemo(
    () =>
      computeReadinessFooterState({
        hasRollout: plan.hasRollout,
        issueStatus: issue.status,
        approvalStatus: issue.approvalStatus,
        checks,
      }),
    [checks, issue.approvalStatus, issue.status, plan.hasRollout]
  );
  const weight = computeBypassActionWeight({
    state: state.kind,
    canCreateRollout: page.projectCanCreateRollout && !page.readonly,
    requireIssueApproval: page.projectRequireIssueApproval,
    requirePlanCheckNoError: page.projectRequirePlanCheckNoError,
    checksFailed: checks.error > 0,
  });

  if (state.kind === "hidden") return null;

  const bypass = async () => {
    if (creating) return;
    try {
      setCreating(true);
      await rolloutServiceClientConnect.createRollout(
        create(CreateRolloutRequestSchema, { parent: plan.name })
      );
      await page.refreshState();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setCreating(false);
    }
  };

  const checkCounts = (
    <>
      {checks.error > 0 && (
        <span className="text-error">
          {t("plan.review.footer.n-checks-failed", { n: checks.error })}
        </span>
      )}
      <span>
        {t("plan.review.footer.n-checks-passed", { n: checks.success })}
      </span>
    </>
  );

  return (
    <div
      className={
        state.kind === "approved-checks-failed"
          ? "flex items-center gap-x-2 border-t px-4 py-2.5"
          : "flex items-center gap-x-2 border-t px-4 py-2.5 text-sm text-control-placeholder"
      }
    >
      {state.kind === "waiting-review" && (
        <>
          <Clock3 className="size-4 shrink-0" />
          <span className="font-medium text-control">
            {t("plan.review.footer.waiting-on-review")}
          </span>
          <span>·</span>
          <span className="min-w-0 truncate">
            {t("plan.review.footer.auto-rollout-after-approval")}
          </span>
          <span>·</span>
          {checkCounts}
        </>
      )}
      {state.kind === "all-gates-passed" && (
        <>
          <CircleCheck className="size-4 shrink-0 text-success" />
          <span className="font-medium text-control">
            {t("plan.review.footer.all-gates-passed")}
          </span>
          <span>·</span>
          <span className="min-w-0 truncate">
            {t("plan.review.footer.creating-rollout-automatically")}
          </span>
          <span>·</span>
          {checkCounts}
        </>
      )}
      {state.kind === "approved-checks-failed" && (
        <>
          <CircleX className="size-4 shrink-0 text-error" />
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold text-main">
              {t("plan.review.footer.approved-but-checks-failed")}
            </div>
            <div className="text-xs text-control-placeholder">
              {t("plan.review.footer.errors-passed-not-created", {
                errors: checks.error,
                passed: checks.success,
              })}
            </div>
          </div>
        </>
      )}
      {state.kind === "rejected" && (
        <>
          <Ban className="size-4 shrink-0" />
          <span className="min-w-0 truncate">
            {t("plan.review.footer.blocked-by-rejection")}
          </span>
        </>
      )}

      {state.kind !== "approved-checks-failed" && <div className="flex-1" />}

      {weight === "link" && (
        <button
          className="shrink-0 text-xs text-control-placeholder underline hover:text-control disabled:opacity-60"
          disabled={creating}
          onClick={() => void bypass()}
          type="button"
        >
          {t("plan.review.footer.bypass-and-deploy")}
        </button>
      )}
      {weight === "button" && (
        <Button
          className="shrink-0"
          disabled={creating}
          onClick={() => void bypass()}
        >
          {creating && <Loader2 className="size-4 animate-spin" />}
          {t("plan.review.footer.bypass-and-deploy")}
        </Button>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify it compiles**

Run: `pnpm --dir frontend type-check`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "feat(plan): rollout readiness footer with bypass action"
```

---

### Task 12: `PlanReviewSection` assembly + page integration

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/PlanReviewSection.tsx`
- Modify: `frontend/src/react/pages/project/plan-detail/ProjectPlanDetailPage.tsx` (the `ReviewBranch` function, ~line 523, and the import at line 26)

- [ ] **Step 1: Implement the section assembly**

```tsx
import { create } from "@bufbuild/protobuf";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { ListIssueCommentsRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { PlanReviewSectionHeader } from "./ReviewSectionHeader";
import { ReviewActivityTimeline } from "./ReviewActivityTimeline";
import { ReviewApprovalFlow } from "./ReviewApprovalFlow";
import { ReviewReadinessFooter } from "./ReviewReadinessFooter";
import { ReviewRejectionBanner } from "./ReviewRejectionBanner";

const EMPTY_COMMENTS: IssueComment[] = [];

export function PlanReviewSection() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const issue = page.issue;
  const issueName = issue?.name ?? "";
  const issueUpdateKey = `${issue?.updateTime?.seconds ?? ""}:${issue?.updateTime?.nanos ?? ""}`;
  const loadProjectIamPolicy = useAppStore(
    (state) => state.loadProjectIamPolicy
  );

  // Candidate computation needs project + IAM in the cache.
  useEffect(() => {
    const projectName = `${projectNamePrefix}${page.projectId}`;
    void useAppStore
      .getState()
      .getOrFetchProjectByName(projectName)
      .catch(() => undefined);
    void loadProjectIamPolicy(projectName).catch(() => undefined);
  }, [loadProjectIamPolicy, page.projectId]);

  // Refetch comments whenever the issue changes server-side (polling bumps
  // updateTime) or after local actions refresh the issue.
  useEffect(() => {
    if (!issueName) return;
    void useAppStore
      .getState()
      .listIssueComments(
        create(ListIssueCommentsRequestSchema, {
          parent: issueName,
          pageSize: 1000,
        })
      )
      .catch(() => undefined);
  }, [issueName, issueUpdateKey]);

  const comments = useAppStore((state) =>
    issueName ? state.getIssueComments(issueName) : EMPTY_COMMENTS
  );

  if (!issue) return null;

  if (issue.approvalStatus === ApprovalStatus.CHECKING) {
    return (
      <div className="flex items-center gap-x-2 p-4 text-sm text-control-placeholder">
        <div className="size-4 animate-spin rounded-full border-2 border-control-border border-t-accent" />
        <span>
          {t("custom-approval.issue-review.generating-approval-flow")}
        </span>
      </div>
    );
  }

  const skipped =
    issue.approvalStatus === ApprovalStatus.SKIPPED ||
    (issue.approvalTemplate?.flow?.roles ?? []).length === 0;

  return (
    <div className="flex flex-col">
      <PlanReviewSectionHeader issue={issue} />
      {skipped ? (
        <div className="px-4 py-3 text-sm text-control-placeholder">
          {t("custom-approval.approval-flow.skip")}
        </div>
      ) : (
        <ReviewApprovalFlow issue={issue} />
      )}
      <ReviewRejectionBanner comments={comments} issue={issue} />
      <ReviewActivityTimeline
        comments={comments}
        issue={issue}
        plan={page.plan}
      />
      <ReviewReadinessFooter issue={issue} plan={page.plan} />
    </div>
  );
}
```

Naming note: the header component is `PlanReviewSectionHeader` (defined in Task 10's `ReviewSectionHeader.tsx`).

- [ ] **Step 2: Wire into the page**

In `ProjectPlanDetailPage.tsx`, replace the import (line 26) and `ReviewBranch`:

```tsx
import { PlanReviewSection } from "./components/review/PlanReviewSection";
```

```tsx
function ReviewBranch() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();

  if (!page.issue) {
    return (
      <div className="p-4 text-sm text-control-light">
        {t("common.no-data")}
      </div>
    );
  }

  return <PlanReviewSection />;
}
```

- [ ] **Step 3: Verify**

Run: `pnpm --dir frontend type-check && pnpm --dir frontend test -- run src/react/pages/project/plan-detail`
Expected: type check passes. The old `PlanDetailApprovalFlow.test.tsx` still passes (component still exists until Task 14).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail && git commit -m "feat(plan): assemble review section into plan detail"
```

---

### Task 13: Deploy-future dedup

**Files:**
- Modify: `frontend/src/react/pages/project/plan-detail/components/PlanDetailDeployFuture.tsx`

Per spec "Cleanups folded in": for issue-backed plans the footer is now the single gate-status source and the single manual path. GitOps plans (no Review section) keep everything.

- [ ] **Step 1: Trim the issue-backed branch**

In `PlanDetailDeployFuture.tsx`:
- Delete the `requirementItems` array and the `<ul>` that renders it (the `page.issue && (...)` block).
- Change `planReadyForManualRollout` / button rendering so the manual-create hint, button, and confirm `Sheet` render **only when `isGitOpsPlan`**. The non-GitOps branch keeps only the description paragraph (`plan.phase.deploy-description`).
- Delete now-unused locals (`errorMessages`, `warningMessages`, `bypassWarnings`, `rolloutConfirmOpen`, `manualCreateRolloutDescription`, the Sheet and its imports) **for the non-GitOps path** — since GitOps skips the confirm sheet already (it calls `createRollout()` directly), the whole `Sheet`, `requirementItems`, warning/error message machinery, and `PlanCheckStatusSummary` become dead code; delete them. Final shape:

```tsx
export function PlanDetailDeployFuture() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [creatingRollout, setCreatingRollout] = useState(false);

  const isGitOpsPlan = useMemo(
    () => isReleaseBackedPlan(page.plan.specs),
    [page.plan.specs]
  );
  const canCreateRollout = Boolean(
    isGitOpsPlan &&
      !page.plan.hasRollout &&
      page.plan.state === State.ACTIVE &&
      page.projectCanCreateRollout
  );

  const createRollout = async () => {
    if (creatingRollout) return;
    try {
      setCreatingRollout(true);
      const createdRollout = await rolloutServiceClientConnect.createRollout(
        create(CreateRolloutRequestSchema, { parent: page.plan.name })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });
      await page.refreshState();
      void router.push(
        buildPlanDeployRouteFromRolloutName(createdRollout.name)
      );
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setCreatingRollout(false);
    }
  };

  return (
    <div className="mt-1.5">
      <p className="text-sm text-control-placeholder">
        {isGitOpsPlan
          ? t("plan.phase.deploy-description-gitops")
          : t("plan.phase.deploy-description")}
      </p>
      {canCreateRollout && (
        <div className="mt-3">
          <Button
            disabled={creatingRollout}
            onClick={() => void createRollout()}
            size="sm"
            variant="outline"
          >
            {t("plan.phase.create-rollout-action")}
          </Button>
        </div>
      )}
    </div>
  );
}
```

Clean the import list down to what remains.

- [ ] **Step 2: Verify**

Run: `pnpm --dir frontend type-check && pnpm --dir frontend check`
Expected: PASS (the `check` run also flags unused imports).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail && git commit -m "feat(plan): dedup deploy-future manual rollout into review footer"
```

---

### Task 14: Delete the superseded approval-flow component

**Files:**
- Delete: `frontend/src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.tsx`
- Delete: `frontend/src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.test.tsx`
- Possibly modify: `frontend/src/locales/*.json` (remove `plan.view-discussion` if dead)

- [ ] **Step 1: Confirm no remaining callers**

Run: `grep -rn "PlanDetailApprovalFlow\|PlanDetailSidebarApprovalFlow\|PlanDetailReviewApprovalFlow" frontend/src --include="*.ts*" | grep -v "components/PlanDetailApprovalFlow"`
Expected: no output (Task 12 removed the page import).

- [ ] **Step 2: Delete**

```bash
git rm frontend/src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.tsx frontend/src/react/pages/project/plan-detail/components/PlanDetailApprovalFlow.test.tsx
```

- [ ] **Step 3: Remove dead locale keys**

Run: `grep -rn "plan.view-discussion" frontend/src --include="*.ts*" --include="*.vue"`
If no hits remain, delete the `"view-discussion"` key from `plan` in every locale file that has it (`grep -l "view-discussion" frontend/src/locales/*.json`).

- [ ] **Step 4: Verify**

Run: `pnpm --dir frontend type-check && pnpm --dir frontend test -- run src/react/pages/project/plan-detail`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A frontend/src && git commit -m "refactor(plan): delete superseded PlanDetailApprovalFlow"
```

---

### Task 15: Component tests — five review-section states

**Files:**
- Create: `frontend/src/react/pages/project/plan-detail/components/review/PlanReviewSection.test.tsx`

Follow the mock pattern of the deleted `PlanDetailApprovalFlow.test.tsx` (it's in git history at `HEAD~1`, and Task 14's step 2 shows the path): `IS_REACT_ACT_ENVIRONMENT`, `vi.hoisted` mocks for `react-i18next` (t returns the key with interpolations appended), `@/connect`, `@/react/stores/app` (`useAppStore` selector backed by a mock state object), `@/react/components/MarkdownEditor` (render content in a div), `@/react/components/FeatureBadge`, and `@/react/router`. Render with `createRoot` inside `act`, wrapped in `PlanDetailProvider` with a `PlanDetailPageState` stub.

- [ ] **Step 1: Write the test**

```tsx
import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_Approver_Status,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { PlanDetailPageState } from "../../shell/hooks/types";
import { PlanDetailProvider } from "../../shell/PlanDetailContext";
import { PlanReviewSection } from "./PlanReviewSection";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mocks = vi.hoisted(() => ({
  comments: [] as unknown[],
  state: {
    roleList: [],
    projectsByName: {},
    projectPoliciesByName: {},
    groupsByName: {},
    getIssueComments: () => mocks.comments,
    getUserByIdentifier: () => undefined,
    listIssueComments: vi.fn(async () => ({ issueComments: [] })),
    getOrFetchProjectByName: vi.fn(async () => ({})),
    loadProjectIamPolicy: vi.fn(async () => ({})),
    batchGetOrFetchUsers: vi.fn(async () => []),
    batchGetOrFetchGroups: vi.fn(async () => []),
    createIssueComment: vi.fn(async () => undefined),
    updateIssueComment: vi.fn(async () => undefined),
  } as Record<string, unknown>,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, opts?: Record<string, unknown>) =>
      opts ? `${key}:${JSON.stringify(opts)}` : key,
  }),
}));

vi.mock("@/react/stores/app", () => {
  const useAppStore = Object.assign(
    (selector: (s: unknown) => unknown) => selector(mocks.state),
    { getState: () => mocks.state }
  );
  return { useAppStore };
});

vi.mock("@/react/components/MarkdownEditor", () => ({
  MarkdownEditor: ({ content }: { content: string }) => (
    <div data-testid="markdown">{content}</div>
  ),
}));

vi.mock("@/connect", () => ({
  issueServiceClientConnect: {
    approveIssue: vi.fn(async () => ({})),
    rejectIssue: vi.fn(async () => ({})),
    requestIssue: vi.fn(async () => ({})),
  },
  rolloutServiceClientConnect: {
    createRollout: vi.fn(async () => ({ name: "r", stages: [] })),
  },
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: { push: vi.fn(), resolve: () => ({ fullPath: "/", href: "/" }) },
}));

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: () => ({
    name: "projects/p1",
    allowSelfApproval: true,
  }),
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({
    name: "users/me",
    email: "me@x.com",
    title: "Me",
  }),
}));

function makeIssue(overrides: Partial<Issue>): Issue {
  return {
    name: "projects/p1/issues/1",
    creator: "users/creator@x.com",
    status: IssueStatus.OPEN,
    approvalStatus: ApprovalStatus.PENDING,
    approvalTemplate: {
      flow: { roles: ["roles/projectOwner", "roles/workspaceDBA"] },
    },
    approvers: [],
    riskLevel: 0,
    labels: [],
    ...overrides,
  } as unknown as Issue;
}

function makePage(issue: Issue, planOverrides = {}): PlanDetailPageState {
  return {
    projectId: "p1",
    planId: "1",
    pageKey: "k",
    projectTitle: "P1",
    projectRequireIssueApproval: false,
    projectRequirePlanCheckNoError: false,
    projectCanCreateRollout: true,
    currentUser: { name: "users/me", email: "me@x.com", title: "Me" },
    project: { name: "projects/p1" },
    isCreating: false,
    isInitializing: false,
    ready: true,
    readonly: false,
    plan: {
      name: "projects/p1/plans/1",
      creator: "users/creator@x.com",
      specs: [],
      hasRollout: false,
      planCheckRunStatusCount: { SUCCESS: 8 },
      ...planOverrides,
    },
    issue,
    rollout: undefined,
    planCheckRuns: [],
    taskRuns: [],
    isEditing: false,
    isRefreshing: false,
    isRunningChecks: false,
    setIsRunningChecks: () => {},
    activePhases: new Set(["review"]),
    pendingLeaveConfirm: false,
    layoutMode: "DESKTOP",
    containerWidth: 1400,
    patchState: vi.fn(),
    refreshState: vi.fn(async () => {}),
    bypassLeaveGuardOnce: () => {},
    setEditing: () => {},
    togglePhase: () => {},
    expandPhase: () => {},
    closeTaskPanel: () => {},
    resolveLeaveConfirm: () => {},
  } as unknown as PlanDetailPageState;
}

async function render(element: ReactElement): Promise<HTMLDivElement> {
  const host = document.createElement("div");
  document.body.appendChild(host);
  const root = createRoot(host);
  await act(async () => {
    root.render(element);
  });
  return host;
}

const renderSection = (page: PlanDetailPageState) =>
  render(
    <PlanDetailProvider value={page}>
      <PlanReviewSection />
    </PlanDetailProvider>
  );

beforeEach(() => {
  document.body.innerHTML = "";
  mocks.comments = [];
});

describe("PlanReviewSection states", () => {
  test("in progress: waiting footer with muted bypass link", async () => {
    const host = await renderSection(makePage(makeIssue({})));
    expect(host.textContent).toContain("plan.review.footer.waiting-on-review");
    expect(host.textContent).toContain("plan.review.footer.bypass-and-deploy");
    expect(host.querySelector("button.underline")).not.toBeNull();
  });

  test("rejected: banner + blocked footer, no bypass action", async () => {
    mocks.comments = [
      {
        name: "projects/p1/issues/1/issueComments/c1",
        comment: "fix it",
        creator: "users/r@x.com",
        event: { case: "approval", value: { status: 3 } }, // REJECTED
      },
    ];
    const issue = makeIssue({
      approvalStatus: ApprovalStatus.REJECTED,
      approvers: [
        { principal: "users/r@x.com", status: Issue_Approver_Status.REJECTED },
      ],
    } as Partial<Issue>);
    const host = await renderSection(makePage(issue));
    expect(host.textContent).toContain("plan.review.rejection.title");
    expect(host.textContent).toContain(
      "plan.review.footer.blocked-by-rejection"
    );
    expect(host.textContent).not.toContain(
      "plan.review.footer.bypass-and-deploy"
    );
  });

  test("approved + checks failed: primary bypass button", async () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.APPROVED });
    const host = await renderSection(
      makePage(issue, { planCheckRunStatusCount: { SUCCESS: 8, ERROR: 2 } })
    );
    expect(host.textContent).toContain(
      "plan.review.footer.approved-but-checks-failed"
    );
    // primary action is a real Button, not the muted underline link
    expect(host.querySelector("button.underline")).toBeNull();
    expect(host.textContent).toContain("plan.review.footer.bypass-and-deploy");
  });

  test("approved + checks passed: all gates passed, muted link", async () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.APPROVED });
    const host = await renderSection(makePage(issue));
    expect(host.textContent).toContain("plan.review.footer.all-gates-passed");
    expect(host.querySelector("button.underline")).not.toBeNull();
  });

  test("skipped approval: skip line, footer still present", async () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.SKIPPED });
    const host = await renderSection(makePage(issue));
    expect(host.textContent).toContain("custom-approval.approval-flow.skip");
    expect(host.textContent).toContain("plan.review.footer.all-gates-passed");
  });

  test("footer disappears once the rollout exists", async () => {
    const issue = makeIssue({ approvalStatus: ApprovalStatus.APPROVED });
    const host = await renderSection(makePage(issue, { hasRollout: true }));
    expect(host.textContent).not.toContain(
      "plan.review.footer.bypass-and-deploy"
    );
  });
});
```

Adjust mock shapes to whatever the components actually read — the test must compile against the real component imports; if a mocked store method is missing, add it to `mocks.state`.

- [ ] **Step 2: Run the test**

Run: `pnpm --dir frontend test -- run src/react/pages/project/plan-detail/components/review/PlanReviewSection.test.tsx`
Expected: PASS (6 tests). Iterate on mock completeness until green — do not weaken assertions.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/react/pages/project/plan-detail/components/review && git commit -m "test(plan): review section state coverage"
```

---

### Task 16: Full gates + verification

**Files:** none new.

- [ ] **Step 1: Autofix and validate everything**

```bash
pnpm --dir frontend fix
pnpm --dir frontend check
pnpm --dir frontend type-check
pnpm --dir frontend test -- run
```
Expected: all pass. `fix` may reorder imports — re-run `check` after.

- [ ] **Step 2: Verify the backend re-review assumption (spec requirement)**

The spec relies on existing backend behavior: editing the plan while rejected restarts review. Verify in code (not by changing anything): `grep -rn "ApprovalFindingDone\|approval_finding" backend/api/v1/plan_service.go | head` — `UpdatePlan` should reset the issue's approval finding. If it does NOT, stop and report: the rejection-banner guidance line overpromises and the spec needs an update — do not silently ship.

- [ ] **Step 3: Manual smoke (optional but recommended)**

Start the stack (`PG_URL=postgresql://bbdev@localhost/bbdev go run ./backend/bin/server/main.go --port 8080 --data . --debug` + `pnpm --dir frontend dev`), create a plan with an approval flow, and walk CUJ 1 (approve → footer → auto rollout), CUJ 2/3 (reject with comment → banner → re-request), CUJ 4 (comment draft survives collapse), CUJ 5 (approve with a failing check → primary Bypass and deploy). The `verify` or `bytebase-team-skills:bytebase-e2e-testing` skill can drive this.

- [ ] **Step 4: Commit any fix-ups**

```bash
git add -A && git commit -m "chore(plan): review section lint/test fix-ups"
```

---

## Self-review notes (already applied)

- Spec coverage: header action (T10), approval flow + layout (T3/T7), rejection banner + re-request (T8), timeline two-tier + fold + islands (T4/T9), composer + draft (T9), footer five states + gating + direct CreateRollout (T5/T11), CHECKING/SKIPPED carry-over (T12), deploy dedup GitOps-preserving (T13), dead-code deletion (T14), i18n en+zh (T2), unit + component tests (T3/T4/T5/T15), gates (T16). The 3.20.0 redirects are intentionally absent (out of scope).
- Type consistency: `deriveSteps`/`FlowStep` exported from `ReviewApprovalFlow` and consumed by the header; `PlanReviewSectionHeader` is the single name across T10/T12; `foldTimeline` takes `(entries, expanded)` everywhere; footer math types match `getPlanCheckSummary`'s `PlanCheckSummary`.
- Known judgment calls an executor may adjust without breaking the spec: exact px constants in `approvalFlowLayout`, Tailwind classes, and Tooltip vs Popover for chip reveal (spec says popover-on-hover; shared `Tooltip` renders content on hover and is the closest existing primitive — do not build a new floating surface).
