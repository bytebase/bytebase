import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { cloneElement, type ReactElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { PositionSchema } from "@/types/proto-es/v1/common_pb";
import {
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ReviewMetadataSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  IssueSchema,
  ReviewRun_Status,
  ReviewRun_Type,
  ReviewRunSchema,
  StatementAnchorSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { Engine, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import type { PlanReviewState } from "../../hooks/usePlanReviewState";
import {
  type AutomatedReviewRow,
  emptyFindings,
  type ReviewerFindings,
} from "../../utils/planGates";

const mocks = vi.hoisted(() => ({
  comments: [] as unknown[],
  focusInStatement: vi.fn(),
  setWalkerFilter: vi.fn(),
  runChecks: vi.fn(async () => undefined),
  reload: vi.fn(async () => undefined),
  routerPush: vi.fn(),
  ruleReview: undefined as unknown,
  rules: undefined as unknown,
  runReview: vi.fn(async () => ({})),
  pushNotification: vi.fn(),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { runReview: mocks.runReview },
}));

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: () => true,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options?.count === undefined ? key : `${key}:${options.count}`,
  }),
}));

vi.mock("@/app/router", () => ({ router: { push: mocks.routerPush } }));

// A hover's content renders beside its trigger, so a test can read it.
vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({
    content,
    children,
    render,
  }: {
    content: ReactNode;
    children: ReactNode;
    render?: ReactElement;
  }) => (
    <>
      {render ? cloneElement(render, undefined, children) : children}
      {content ? <span data-testid="tooltip">{content}</span> : null}
    </>
  ),
}));

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

vi.mock("@/components/plan-check/PlanCheckSection", () => ({
  PlanCheckResultsDrawer: ({ open }: { open: boolean }) =>
    open ? <div data-testid="checks-drawer" /> : null,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      getIssueComments: () => mocks.comments,
      databasesByName: {
        "instances/i/databases/hr_test": {
          instanceResource: { engine: Engine.POSTGRES },
        },
      },
      dbGroupsByName: {},
      dbGroupViewByName: {},
    }),
}));

vi.mock("../../hooks/useEffectiveStandardRules", () => ({
  useEffectiveStandardRules: () => mocks.rules,
}));

vi.mock("../../hooks/usePlanCheckActions", () => ({
  usePlanCheckActions: () => ({
    allowRunChecks: true,
    isRunningChecks: false,
    runChecks: mocks.runChecks,
  }),
}));

vi.mock("../../hooks/useFocusThread", () => ({
  useFocusThread: () => ({
    focusInActivity: vi.fn(),
    focusInStatement: mocks.focusInStatement,
  }),
}));

vi.mock("../../hooks/useSqlReviewV2", () => ({
  useSqlReviewV2: () => ({ ruleReview: mocks.ruleReview }),
}));

vi.mock("../../shared/stores/usePlanDetailStore", () => {
  const state = {
    reviewRowRequest: undefined,
    setWalkerFilter: mocks.setWalkerFilter,
  };
  return {
    usePlanDetailStore: (selector: (s: unknown) => unknown) => selector(state),
    usePlanDetailStoreApi: () => ({ getState: () => state }),
  };
});

vi.mock("../../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => ({
    issue: create(IssueSchema, {
      name: "projects/p/issues/1",
      status: IssueStatus.OPEN,
    }),
    isEditing: false,
    readonly: false,
    rollout: undefined,
    plan: create(PlanSchema, {
      specs: [
        {
          id: "spec-1",
          config: {
            case: "changeDatabaseConfig",
            value: create(Plan_ChangeDatabaseConfigSchema, {
              targets: ["instances/i/databases/hr_test"],
            }),
          },
        },
      ],
    }),
    planCheckRuns: [],
    project: { ciSamplingSize: 0 },
    projectId: "p",
  }),
}));

import { AutomatedReviewSection } from "./AutomatedReviewSection";

const finding = (
  name: string,
  line: number,
  options: {
    ruleType?: ReviewRuleType;
    resolved?: boolean;
    priority?: IssueComment_ReviewMetadata_Priority;
  } = {}
) =>
  create(IssueCommentSchema, {
    name,
    threadState: options.resolved
      ? IssueComment_ThreadState.RESOLVED
      : IssueComment_ThreadState.OPEN,
    statementAnchor: create(StatementAnchorSchema, {
      spec: "spec-1",
      startPosition: create(PositionSchema, { line }),
    }),
    reviewMetadata: create(IssueComment_ReviewMetadataSchema, {
      runType: ReviewRun_Type.RULE,
      ruleType: options.ruleType ?? ReviewRuleType.DISALLOW_DROP_OBJECT,
      priority: options.priority ?? IssueComment_ReviewMetadata_Priority.P1,
    }),
  });

const findings = (extra: Partial<ReviewerFindings> = {}): ReviewerFindings => ({
  ...emptyFindings(),
  ...extra,
});

const row = (
  reviewer: AutomatedReviewRow["reviewer"],
  conclusion: AutomatedReviewRow["conclusion"],
  rowFindings = findings(),
  extra: Partial<AutomatedReviewRow> = {}
): AutomatedReviewRow => ({
  reviewer,
  conclusion,
  findings: rowFindings,
  holdsRollout: conclusion !== "passed" && conclusion !== "resolved",
  ...extra,
});

const ai = (overrides: Partial<PlanReviewState["ai"]> = {}) => ({
  allowRun: true,
  canStart: true,
  editing: false,
  run: null,
  runReview: vi.fn(async () => undefined),
  shown: true,
  starting: false,
  ...overrides,
});

const renderSection = (
  rows: AutomatedReviewRow[],
  aiState: PlanReviewState["ai"] = ai()
) =>
  render(
    <AutomatedReviewSection
      state={{ rows, gates: [], openDiscussions: 0, ai: aiState }}
    />
  );

beforeEach(() => {
  vi.clearAllMocks();
  mocks.comments = [];
  mocks.ruleReview = { loadFailed: false, reload: mocks.reload };
  mocks.rules = { rules: [1, 2, 3], source: "project" };
});

afterEach(cleanup);

describe("AutomatedReviewSection", () => {
  test("puts each reviewer on one line with its open findings by priority", () => {
    renderSection([
      row("rule", "open", findings({ open: 3, p0: 1, p1: 2 })),
      row("ai", "not-run"),
      row("policy", "passed", undefined, { targets: 2 }),
    ]);
    expect(
      screen.getByText("plan.automated-review.needs-attention:2")
    ).toBeDefined();
    const rule = screen.getByTestId("automated-review-rule");
    expect(
      screen.getByRole("button", { name: "plan.findings.n-priority:1" })
        .textContent
    ).toBe("P01");
    expect(
      screen.getByRole("button", { name: "plan.findings.n-priority:2" })
        .textContent
    ).toBe("P12");
    expect(rule.textContent).not.toContain("P2");
    expect(screen.getByText("plan.automated-review.not-run")).toBeDefined();
    expect(screen.getByText("plan.automated-review.passed-on:2")).toBeDefined();
  });

  test("a count opens the first finding of its priority and narrows the walker", () => {
    mocks.comments = [
      finding("line-4", 4),
      finding("line-2", 2),
      finding("p0", 3, { priority: IssueComment_ReviewMetadata_Priority.P0 }),
    ];
    renderSection([row("rule", "open", findings({ open: 3, p0: 1, p1: 2 }))]);
    fireEvent.click(
      screen.getByRole("button", { name: "plan.findings.n-priority:2" })
    );
    expect(mocks.setWalkerFilter).toHaveBeenCalledWith({
      runType: ReviewRun_Type.RULE,
      group: "p1",
    });
    expect(mocks.focusInStatement).toHaveBeenCalledWith(
      expect.objectContaining({ name: "line-2" })
    );
    fireEvent.click(
      screen.getByRole("button", { name: "plan.findings.n-priority:1" })
    );
    expect(mocks.focusInStatement).toHaveBeenLastCalledWith(
      expect.objectContaining({ name: "p0" })
    );
  });

  test("counts resolved findings and names them by rule on hover", () => {
    mocks.comments = [
      finding("drop-1", 1, { resolved: true }),
      finding("drop-2", 2, { resolved: true }),
      finding("where-3", 3, {
        ruleType: ReviewRuleType.REQUIRE_WHERE,
        resolved: true,
      }),
    ];
    renderSection([row("rule", "resolved", findings({ resolved: 3 }))]);
    const resolved = screen.getByRole("button", {
      name: "plan.findings.n-resolved:3",
    });
    expect(resolved.nextElementSibling?.textContent).toContain(
      "plan.findings.name-times:2"
    );
    fireEvent.click(resolved);
    expect(mocks.setWalkerFilter).toHaveBeenCalledWith({
      runType: ReviewRun_Type.RULE,
      group: "resolved",
    });
    expect(mocks.focusInStatement).toHaveBeenCalledWith(
      expect.objectContaining({ name: "drop-1" })
    );
    expect(screen.queryByText(/plan.findings.n-overridden/)).toBeNull();
  });

  test("calls out a P0 resolved without a change and counts it as needing attention", () => {
    mocks.comments = [
      finding("null-3", 3, {
        ruleType: ReviewRuleType.REQUIRE_IS_NULL,
        priority: IssueComment_ReviewMetadata_Priority.P0,
        resolved: true,
      }),
    ];
    renderSection([row("rule", "resolved", findings({ overridden: 1 }))]);
    expect(
      screen.getByText("plan.automated-review.needs-attention:1")
    ).toBeDefined();
    const overridden = screen.getByRole("button", {
      name: "plan.findings.n-overridden:1",
    });
    expect(overridden.nextElementSibling?.textContent).toContain(
      "plan.findings.name-at-line"
    );
    fireEvent.click(overridden);
    expect(mocks.setWalkerFilter).toHaveBeenCalledWith({
      runType: ReviewRun_Type.RULE,
      group: "overridden",
    });
    expect(mocks.focusInStatement).toHaveBeenCalledWith(
      expect.objectContaining({ name: "null-3" })
    );
  });

  test("says on hover what the rule review covered", () => {
    renderSection([row("rule", "passed")]);
    expect(
      screen.getByRole("button", { name: "plan.automated-review.scope-label" })
    ).toBeDefined();
    const scope = screen.getByTestId("review-scope");
    expect(scope.textContent).toContain(
      "plan.automated-review.scope-with-engines"
    );
    expect(scope.textContent).toContain("plan.automated-review.project-rules");
  });

  test("claims no coverage for a reviewer that has not run", () => {
    renderSection([row("ai", "not-run"), row("rule", "running")]);
    expect(screen.queryByTestId("review-scope")).toBeNull();
  });

  test("Run, Retry, and Rerun start the rule review alone", async () => {
    const { unmount } = renderSection([row("rule", "outdated")]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /common.run/ }));
    });
    expect(mocks.runReview).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ name: "projects/p/issues/1/reviewRuns/rule" }),
      expect.anything()
    );
    expect(mocks.reload).toHaveBeenCalledOnce();
    expect(mocks.runChecks).not.toHaveBeenCalled();
    unmount();

    renderSection([row("rule", "failed", undefined, { error: "hr_test: metadata not synced" })]);
    expect(
      screen.getByRole("button", { name: /plan.automated-review.retry/ })
    ).toBeDefined();
  });

  test("a rule review with a result offers Rerun, since settings and schema change results", async () => {
    renderSection([row("rule", "open", findings({ open: 1, p1: 1 }))]);
    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: /plan.automated-review.rerun/ })
      );
    });
    expect(mocks.runReview).toHaveBeenCalledOnce();
  });

  test("a rerun that would reopen resolved findings asks first", async () => {
    renderSection([
      row("rule", "open", findings({ open: 1, p1: 1, resolved: 1, overridden: 1 })),
    ]);
    fireEvent.click(
      screen.getByRole("button", { name: /plan.automated-review.rerun/ })
    );
    expect(mocks.runReview).not.toHaveBeenCalled();
    expect(
      screen.getByText("plan.automated-review.rerun-rule-title")
    ).toBeDefined();
    expect(
      screen.getByText("plan.automated-review.rerun-reopens:2")
    ).toBeDefined();
    const confirm = screen
      .getAllByRole("button", { name: "plan.automated-review.rerun" })
      .at(-1);
    await act(async () => {
      fireEvent.click(confirm!);
    });
    expect(mocks.runReview).toHaveBeenCalledOnce();
  });

  test("shows a failed run's error in full", () => {
    renderSection([
      row("rule", "failed", undefined, {
        error: "hr_test: metadata not synced",
      }),
    ]);
    expect(screen.getByText("hr_test: metadata not synced")).toBeDefined();
  });

  test("an AI review waits for unsaved edits and says so on hover", () => {
    renderSection([row("ai", "not-run")], ai({ canStart: false, editing: true }));
    const run = screen.getByRole("button", { name: "plan.ai-review.run" });
    expect((run as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByTestId("tooltip").textContent).toBe(
      "plan.ai-review.save-first"
    );
  });

  test("a finished AI review offers Rerun behind a confirmation", async () => {
    const aiState = ai({
      run: create(ReviewRunSchema, { status: ReviewRun_Status.DONE }),
    });
    renderSection([row("ai", "passed")], aiState);
    fireEvent.click(
      screen.getByRole("button", { name: /plan.automated-review.rerun/ })
    );
    expect(aiState.runReview).not.toHaveBeenCalled();
    expect(screen.getByText("plan.ai-review.run-again-title")).toBeDefined();
    const confirm = screen
      .getAllByRole("button", { name: "plan.automated-review.rerun" })
      .at(-1);
    await act(async () => {
      fireEvent.click(confirm!);
    });
    expect(aiState.runReview).toHaveBeenCalledOnce();
  });

  test("Re-run all asks first when it would reopen resolved rule findings", async () => {
    renderSection([row("rule", "resolved", findings({ resolved: 3 }))]);
    fireEvent.click(screen.getByRole("button", { name: "common.more" }));
    fireEvent.click(await screen.findByText("plan.automated-review.rerun-all"));
    expect(mocks.runChecks).not.toHaveBeenCalled();
    expect(
      screen.getByText("plan.automated-review.rerun-all-description:3")
    ).toBeDefined();
    const confirm = screen
      .getAllByRole("button", { name: "plan.automated-review.rerun-all" })
      .at(-1);
    await act(async () => {
      fireEvent.click(confirm!);
    });
    expect(mocks.runChecks).toHaveBeenCalledOnce();
    expect(mocks.reload).toHaveBeenCalledOnce();
  });

  test("a plan check row reruns the checks", async () => {
    renderSection([row("policy", "passed", undefined, { targets: 1 })]);
    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: /plan.automated-review.rerun/ })
      );
    });
    expect(mocks.runChecks).toHaveBeenCalledOnce();
  });

  test("a plan check row opens the check details", () => {
    renderSection([
      row("policy", "open", findings({ open: 1, p0: 1 }), { targets: 1 }),
    ]);
    expect(screen.getByText("plan.automated-review.n-errors:1")).toBeDefined();
    fireEvent.click(screen.getByText("plan.automated-review.details"));
    expect(screen.getByTestId("checks-drawer")).toBeDefined();
  });
});
