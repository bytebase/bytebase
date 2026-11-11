import { create } from "@bufbuild/protobuf";
import { act, cleanup, fireEvent, render, screen, within } from "@testing-library/react";
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
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
  PlanCheckRun_Status,
  PlanCheckRunSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
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
  launchDiagnostic: vi.fn(),
  planName: "projects/p/plans/42",
  issuePlan: "projects/p/plans/42",
  issueName: "projects/p/issues/1",
  planCheckRuns: [] as PlanCheckRun[],
}));

vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({
  openAgentWithPrompt: mocks.launchDiagnostic,
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

vi.mock("@/components/plan-check/PlanCheckSection", async () => {
  const { getFilteredResultGroups } = await import("@/lib/plan/check");
  return {
    PlanCheckResultsDrawer: ({ open, planCheckRuns, renderResultAction }: {
      open: boolean;
      planCheckRuns: PlanCheckRun[];
      renderResultAction?: (result: PlanCheckRun_Result) => ReactNode;
    }) => open ? (
      <div data-testid="checks-drawer">
        {getFilteredResultGroups({ includeRunFailure: true, planCheckRuns,
          runFailureTitle: "common.failed", runFailureContent: "common.failed",
          runCanceledTitle: "common.canceled", runCanceledContent: "common.canceled",
        }).flatMap((group) => group.results).map((result, index) => (
          <div key={index}>{renderResultAction?.(result)}</div>
        ))}
      </div>
    ) : null,
  };
});

vi.mock("@/stores/app", () => {
  const getState = () => ({
      getIssueComments: () => mocks.comments,
      databasesByName: {
        "instances/i/databases/hr_test": {
          instanceResource: { engine: Engine.POSTGRES },
        },
      },
      dbGroupsByName: {},
      dbGroupViewByName: {},
      workspaceResourceName: () => "workspaces/example",
      currentUser: { name: "users/member@example.com" },
    });
  return { useAppStore: Object.assign((selector: (state: unknown) => unknown) => selector(getState()), { getState }) };
});

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
      name: mocks.issueName,
      plan: mocks.issuePlan,
      status: IssueStatus.OPEN,
    }),
    isEditing: false,
    readonly: false,
    rollout: undefined,
    plan: create(PlanSchema, {
      name: mocks.planName,
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
    planCheckRuns: mocks.planCheckRuns,
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

const failedReview = (reviewer: "rule" | "ai") =>
  create(ReviewRunSchema, {
    name: `projects/p/issues/1/reviewRuns/${reviewer}`,
    type: reviewer === "rule" ? ReviewRun_Type.RULE : ReviewRun_Type.AI,
    status: ReviewRun_Status.FAILED,
    error: "review service unavailable",
    createTime: { seconds: 100n, nanos: 123 },
    endTime: { seconds: 110n, nanos: 456 },
  });

const diagnosticSnapshot = () => {
  const [prompt, references, , options] = mocks.launchDiagnostic.mock.calls[0];
  expect(references).toEqual([{
    kind: "plan", name: "projects/p/plans/42", project: "projects/p",
    label: "projects/p/plans/42",
  }]);
  expect(options.capability).toEqual({ kind: "diagnostic", resource: references[0] });
  return JSON.parse(prompt.slice(prompt.indexOf("\n") + 1));
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.comments = [];
  mocks.ruleReview = { loadFailed: false, reload: mocks.reload };
  mocks.rules = { rules: [1, 2, 3], source: "project" };
  mocks.planName = "projects/p/plans/42";
  mocks.issuePlan = mocks.planName;
  mocks.issueName = "projects/p/issues/1";
  mocks.planCheckRuns = [];
});

afterEach(cleanup);

describe("AutomatedReviewSection", () => {
  test.each(["rule", "ai"] as const)("keeps failed %s reviewer help with the error metadata, separate from Retry", (reviewer) => {
    const run = failedReview(reviewer);
    if (reviewer === "rule") mocks.ruleReview = { run, loadFailed: false, reload: mocks.reload };
    renderSection([row(reviewer, "failed", undefined, { error: run.error, time: run.endTime })], ai({ run: reviewer === "ai" ? run : null }));

    const errorSummary = screen.getByText(run.error).parentElement!;
    const help = within(errorSummary).getByRole("button", { name: "agent.diagnostic-help.ask-agent" });
    const metadata = help.parentElement!;
    expect(within(metadata).getByTestId("humanize-ts").textContent).toBe("110000.000456");
    expect(within(metadata).getByRole("button", { name: "plan.automated-review.scope-label" })).toBeDefined();
    expect(within(metadata).queryByText(run.error)).toBeNull();
    const retryName = reviewer === "ai" ? "plan.ai-review.run" : "plan.automated-review.retry";
    expect(within(errorSummary).queryByRole("button", { name: retryName })).toBeNull();
    expect(screen.getByRole("button", { name: retryName }).textContent).toContain("plan.automated-review.retry");
  });

  test.each(["rule", "ai"] as const)("diagnoses the exact failed %s reviewer execution on click", (reviewer) => {
    const run = failedReview(reviewer);
    if (reviewer === "rule") mocks.ruleReview = { run, loadFailed: false, reload: mocks.reload };
    renderSection([row(reviewer, "failed", undefined, { error: run.error })], ai({ run: reviewer === "ai" ? run : null }));
    expect(mocks.launchDiagnostic).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "agent.diagnostic-help.ask-agent" }));
    expect(mocks.launchDiagnostic).toHaveBeenCalledOnce();
    const snapshot = diagnosticSnapshot();
    expect(snapshot.kind).toBe("review-run");
    expect(snapshot.sourceId).toBe(run.name);
    expect(snapshot.summary).toMatch(/execution failed/i);
    expect(snapshot.summary).not.toMatch(/finding/i);
    expect(snapshot.detail).toBe(run.error);
    expect(JSON.stringify(snapshot.facts)).toContain(reviewer.toUpperCase());
    expect(snapshot.facts).toEqual(expect.arrayContaining([
      { label: "Status", value: "FAILED" },
      { label: "Issue", value: "projects/p/issues/1" },
      { label: "Plan", value: "projects/p/plans/42" },
      { label: "Started", value: "100.000000123" },
      { label: "Ended", value: "110.000000456" },
    ]));
    expect(snapshot.sql).toBeUndefined();
  });

  test.each(["passed", "resolved", "not-run", "outdated", "running", "loading", "open"] as const)("does not turn %s reviewer state into another diagnostic action", (conclusion) => {
    mocks.ruleReview = { run: failedReview("rule"), loadFailed: false, reload: mocks.reload };
    renderSection([row("rule", conclusion, findings({ open: 1, p1: 1 }))]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
  });

  test.each([ReviewRun_Status.DONE, ReviewRun_Status.RUNNING, ReviewRun_Status.AVAILABLE, ReviewRun_Status.STATUS_UNSPECIFIED])("requires a failed underlying reviewer execution, not only the row conclusion (%s)", (status) => {
    mocks.ruleReview = { run: { ...failedReview("rule"), status }, loadFailed: false, reload: mocks.reload };
    renderSection([row("rule", "failed")]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
  });

  test.each(["invalid-plan", "different-project", "linked-plan", "run-issue", "run-type", "empty-error", "load-failed"])("hides failed reviewer help for %s evidence", (failure) => {
    const run = failedReview("rule");
    if (failure === "invalid-plan") mocks.planName = "projects/p/plans/create";
    if (failure === "different-project") mocks.planName = "projects/other/plans/42";
    if (failure === "linked-plan") mocks.issuePlan = "projects/p/plans/43";
    if (failure === "run-issue") run.name = "projects/p/issues/2/reviewRuns/rule";
    if (failure === "run-type") run.type = ReviewRun_Type.AI;
    if (failure === "empty-error") run.error = "";
    mocks.ruleReview = { run, loadFailed: failure === "load-failed", reload: mocks.reload };
    renderSection([row("rule", "failed")]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
    expect(screen.getByRole("button", { name: /plan.automated-review.retry/ })).toBeDefined();
  });

  test("keeps a generic rule-review load failure on its existing Retry path", () => {
    mocks.ruleReview = { loadFailed: true, reload: mocks.reload };
    renderSection([row("rule", "failed")]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /plan.automated-review.retry/ }));
    expect(mocks.reload).toHaveBeenCalledOnce();
    expect(mocks.launchDiagnostic).not.toHaveBeenCalled();
  });

  test.each(["success", "running", "invalid-plan", "mismatched-run", "linked-plan"])("does not diagnose %s check evidence", (scenario) => {
    mocks.planCheckRuns = [create(PlanCheckRunSchema, {
      name: scenario === "mismatched-run" ? "projects/p/plans/43/planCheckRun" : "projects/p/plans/42/planCheckRun",
      status: scenario === "running" ? PlanCheckRun_Status.RUNNING : PlanCheckRun_Status.DONE,
      results: scenario === "running" ? [] : [{ status: scenario === "success" ? Advice_Level.SUCCESS : Advice_Level.ERROR, title: "Check evidence" }],
    })];
    if (scenario === "invalid-plan") mocks.planName = "projects/p/plans/create";
    if (scenario === "linked-plan") mocks.issuePlan = "projects/p/plans/43";
    renderSection([row("checks", "running")]);
    fireEvent.click(screen.getByText("plan.automated-review.details"));
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
  });

  test.each([Advice_Level.ERROR, Advice_Level.WARNING])("wires check result diagnostics only in the details drawer (%s)", (status) => {
    mocks.planCheckRuns = [create(PlanCheckRunSchema, {
      name: "projects/p/plans/42/planCheckRun", status: PlanCheckRun_Status.DONE,
      results: [{ status, title: "Check evidence", content: "Unsafe statement", type: PlanCheckRun_Result_Type.STATEMENT_ADVISE, target: "instances/i/databases/hr_test" }],
    })];
    renderSection([row("policy", "open")]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
    fireEvent.click(screen.getByText("plan.automated-review.details"));
    fireEvent.click(screen.getByRole("button", { name: "agent.diagnostic-help.ask-agent" }));
    const snapshot = diagnosticSnapshot();
    expect(snapshot.kind).toBe("sql-check");
    expect(snapshot.sourceId).toBe("projects/p/plans/42/planCheckRun");
    expect(JSON.stringify(snapshot)).toContain("Unsafe statement");
  });

  test.each([PlanCheckRun_Status.FAILED, PlanCheckRun_Status.CANCELED])("keeps exactly one diagnostic beside synthetic stopped check evidence (%s)", (status) => {
    mocks.planCheckRuns = [create(PlanCheckRunSchema, {
      name: "projects/p/plans/42/planCheckRun", status, error: "Unable to resolve targets",
      createTime: { seconds: 120n, nanos: 0 },
    })];
    renderSection([row("checks", "failed", undefined, { error: "Unable to resolve targets" })]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
    fireEvent.click(screen.getByText("plan.automated-review.details"));
    fireEvent.click(screen.getByRole("button", { name: "agent.diagnostic-help.ask-agent" }));
    expect(mocks.launchDiagnostic).toHaveBeenCalledOnce();
    const snapshot = diagnosticSnapshot();
    expect(snapshot.sourceId).toBe("projects/p/plans/42/planCheckRun");
    expect(JSON.stringify(snapshot)).toContain("Unable to resolve targets");
    expect(JSON.stringify(snapshot)).toContain(PlanCheckRun_Status[status]);
  });

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

  test("a rerun keeps resolved findings, so it does not ask", async () => {
    renderSection([
      row("rule", "open", findings({ open: 1, p1: 1, resolved: 1, overridden: 1 })),
    ]);
    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: /plan.automated-review.rerun/ })
      );
    });
    expect(mocks.runReview).toHaveBeenCalledOnce();
    expect(screen.queryByRole("dialog")).toBeNull();
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

  test("Re-run all reruns the checks and the rule review without asking", async () => {
    renderSection([row("rule", "resolved", findings({ resolved: 3 }))]);
    fireEvent.click(screen.getByRole("button", { name: "common.more" }));
    await act(async () => {
      fireEvent.click(
        await screen.findByText("plan.automated-review.rerun-all")
      );
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
