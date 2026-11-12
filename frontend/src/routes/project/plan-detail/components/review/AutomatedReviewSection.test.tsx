import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
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
  ReviewRun_State,
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
import { shownTimestampPrecisions } from "@/test-utils/humanizeTs";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
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
  review: undefined as unknown,
  rules: undefined as unknown,
  runReview: vi.fn(async () => ({})),
  pushNotification: vi.fn(),
  launchDiagnostic: vi.fn(),
  planName: "projects/p/plans/42",
  issuePlan: "projects/p/plans/42",
  issueName: "projects/p/issues/1",
  planCheckRuns: [] as PlanCheckRun[],
  serverInfo: undefined as unknown,
  editing: false,
  ciSamplingSize: 0,
  targets: [] as string[],
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
      serverInfo: mocks.serverInfo,
      databasesByName: {
        "instances/i/databases/hr_test": {
          instanceResource: { engine: Engine.POSTGRES },
        },
        "instances/i/databases/hr_docs": {
          instanceResource: { engine: Engine.MONGODB },
        },
      },
      dbGroupsByName: {},
      dbGroupViewByName: {},
      workspaceResourceName: () => "workspaces/example",
      currentUser: { name: "users/member@example.com" },
    });
  return { useAppStore: Object.assign((selector: (state: unknown) => unknown) => selector(getState()), { getState }) };
});

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
  useSqlReviewV2: () => ({
    review: mocks.review,
    standardRules: mocks.rules,
  }),
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
    isEditing: mocks.editing,
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
              targets: mocks.targets,
            }),
          },
        },
      ],
    }),
    planCheckRuns: mocks.planCheckRuns,
    project: { ciSamplingSize: mocks.ciSamplingSize },
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

const renderSection = (rows: AutomatedReviewRow[]) =>
  render(
    <AutomatedReviewSection state={{ rows, gates: [], openDiscussions: 0 }} />
  );

const failedReview = () =>
  create(ReviewRunSchema, {
    name: "projects/p/issues/1/reviewRun",
    state: ReviewRun_State.FAILED,
    error: { code: 14, message: "review service unavailable" },
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
const AI_ON = { aiEnabled: true, aiSqlReviewEnabled: true };
const reviewRun = (state: ReviewRun_State, outdated = false) =>
  create(ReviewRunSchema, { state, outdated });

beforeEach(() => {
  vi.clearAllMocks();
  mocks.comments = [];
  mocks.review = { loadFailed: false, reload: mocks.reload, run: null };
  mocks.rules = { rules: [1, 2, 3], source: "project" };
  mocks.planName = "projects/p/plans/42";
  mocks.issuePlan = mocks.planName;
  mocks.issueName = "projects/p/issues/1";
  mocks.planCheckRuns = [];
  mocks.serverInfo = undefined;
  mocks.editing = false;
  mocks.ciSamplingSize = 0;
  mocks.targets = ["instances/i/databases/hr_test"];
});

afterEach(cleanup);

describe("AutomatedReviewSection", () => {
  test("keeps failed review help with the error metadata and earlier findings, separate from Retry", () => {
    const run = failedReview();
    mocks.review = { run, loadFailed: false, reload: mocks.reload };
    renderSection([row("review", "failed", findings({ open: 1, p0: 1, resolved: 2, overridden: 1 }), { error: run.error!.message, time: run.endTime })]);

    const errorSummary = screen.getByText(run.error!.message).parentElement!;
    const help = within(errorSummary).getByRole("button", { name: "agent.diagnostic-help.ask-agent" });
    const metadata = help.parentElement!;
    expect(within(metadata).getByTestId("humanize-ts").textContent).toBe("110000.000456");
    expect(within(metadata).getByRole("button", { name: "plan.automated-review.scope-label" })).toBeDefined();
    expect(within(metadata).getByRole("button", { name: "plan.findings.n-priority:1" }).textContent).toBe("P01");
    expect(within(metadata).getByRole("button", { name: "plan.findings.n-resolved:2" })).toBeDefined();
    expect(within(metadata).getByRole("button", { name: "plan.findings.n-overridden:1" })).toBeDefined();
    expect(within(metadata).queryByText(run.error!.message)).toBeNull();
    const retryName = "plan.automated-review.retry";
    expect(within(errorSummary).queryByRole("button", { name: retryName })).toBeNull();
    expect(screen.getByRole("button", { name: retryName }).textContent).toContain("plan.automated-review.retry");
  });

  test("diagnoses the exact failed unified review execution on click", () => {
    const run = failedReview();
    mocks.review = { run, loadFailed: false, reload: mocks.reload };
    renderSection([row("review", "failed", undefined, { error: run.error!.message })]);
    expect(mocks.launchDiagnostic).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "agent.diagnostic-help.ask-agent" }));
    expect(mocks.launchDiagnostic).toHaveBeenCalledOnce();
    const snapshot = diagnosticSnapshot();
    expect(snapshot.kind).toBe("review-run");
    expect(snapshot.sourceId).toBe(run.name);
    expect(snapshot.summary).toMatch(/execution failed/i);
    expect(snapshot.summary).not.toMatch(/finding/i);
    expect(snapshot.detail).toBe(run.error!.message);
    expect(snapshot.facts).toEqual(expect.arrayContaining([
      { label: "Status", value: "FAILED" },
      { label: "Error code", value: "14" },
      { label: "Issue", value: "projects/p/issues/1" },
      { label: "Plan", value: "projects/p/plans/42" },
      { label: "Started", value: "100.000000123" },
      { label: "Ended", value: "110.000000456" },
    ]));
    expect(snapshot.sql).toBeUndefined();
  });

  test("diagnoses a failed review with a status code but no error message", () => {
    const run = failedReview();
    run.error = { ...run.error!, code: 8, message: "" };
    mocks.review = { run, loadFailed: false, reload: mocks.reload };
    renderSection([row("review", "failed")]);
    fireEvent.click(screen.getByRole("button", { name: "agent.diagnostic-help.ask-agent" }));
    const snapshot = diagnosticSnapshot();
    expect(snapshot.sourceId).toBe(run.name);
    expect(snapshot.detail).toBe("Review execution failed (code 8)");
    expect(snapshot.facts).toContainEqual({ label: "Error code", value: "8" });
  });

  test.each(["passed", "resolved", "not-run", "outdated", "running", "loading", "open", "off"] as const)("does not turn %s review state into another diagnostic action", (conclusion) => {
    mocks.review = { run: failedReview(), loadFailed: false, reload: mocks.reload };
    renderSection([row("review", conclusion, findings({ open: 1, p1: 1 }))]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
  });

  test.each([ReviewRun_State.SUCCEEDED, ReviewRun_State.RUNNING, ReviewRun_State.STATE_UNSPECIFIED])("requires a failed underlying review execution, not only the row conclusion (%s)", (state) => {
    mocks.review = { run: { ...failedReview(), state }, loadFailed: false, reload: mocks.reload };
    renderSection([row("review", "failed")]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
  });

  test.each(["invalid-plan", "different-project", "linked-plan", "invalid-issue", "run-issue", "legacy-run", "missing-run", "missing-error", "empty-error", "whitespace-error", "load-failed"])("hides failed review help for %s evidence", (failure) => {
    const run = failedReview();
    if (failure === "invalid-plan") mocks.planName = "projects/p/plans/create";
    if (failure === "different-project") mocks.planName = "projects/other/plans/42";
    if (failure === "linked-plan") mocks.issuePlan = "projects/p/plans/43";
    if (failure === "invalid-issue") {
      mocks.issueName = "projects/p/issues/create";
      run.name = `${mocks.issueName}/reviewRun`;
    }
    if (failure === "run-issue") run.name = "projects/p/issues/2/reviewRun";
    if (failure === "legacy-run") run.name = "projects/p/issues/1/reviewRuns/rule";
    if (failure === "missing-error") run.error = undefined;
    if (failure === "empty-error") run.error = { ...run.error!, code: 0, message: "" };
    if (failure === "whitespace-error") run.error = { ...run.error!, code: 0, message: " \n " };
    mocks.review = { run: failure === "missing-run" ? undefined : run, loadFailed: failure === "load-failed", reload: mocks.reload };
    renderSection([row("review", "failed")]);
    expect(screen.queryByRole("button", { name: "agent.diagnostic-help.ask-agent" })).toBeNull();
    expect(screen.getByRole("button", { name: /plan.automated-review.retry/ })).toBeDefined();
  });

  test("keeps a generic review load failure on its existing Retry path", () => {
    mocks.review = { loadFailed: true, reload: mocks.reload };
    renderSection([row("review", "failed")]);
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

  test("puts the review on one line with its open findings by priority", () => {
    renderSection([
      row("review", "open", findings({ open: 3, p0: 1, p1: 2 })),
      row("policy", "passed", undefined, { targets: 2 }),
    ]);
    expect(
      screen.getByText("plan.automated-review.needs-attention:1")
    ).toBeDefined();
    const review = screen.getByTestId("automated-review-review");
    expect(review.textContent).toContain("sql-review.reviewer");
    expect(
      screen.getByRole("button", { name: "plan.findings.n-priority:1" })
        .textContent
    ).toBe("P01");
    expect(
      screen.getByRole("button", { name: "plan.findings.n-priority:2" })
        .textContent
    ).toBe("P12");
    expect(review.textContent).not.toContain("P2");
    expect(screen.getByText("plan.automated-review.passed-on:2")).toBeDefined();
  });

  test("a review that never ran says so, and a running one reads as reviewing", () => {
    const { unmount } = renderSection([row("review", "not-run")]);
    expect(screen.getByText("plan.automated-review.not-run")).toBeDefined();
    unmount();

    renderSection([row("review", "running"), row("checks", "running")]);
    expect(
      screen.getByTestId("automated-review-review").textContent
    ).toContain("plan.automated-review.reviewing");
    expect(
      screen.getByTestId("automated-review-checks").textContent
    ).toContain("plan.automated-review.running");
  });

  test("reads reviewer finish times to the minute", () => {
    const { container } = renderSection([
      row("review", "open", findings({ open: 1, p1: 1 }), {
        time: timestampFromMs(1_000_000),
      }),
      row("policy", "passed", undefined, {
        targets: 1,
        time: timestampFromMs(1_008_000),
      }),
    ]);
    expect(shownTimestampPrecisions(container)).toEqual(["minute", "minute"]);
  });

  test("a count opens the first finding of its priority and narrows the walker", () => {
    mocks.comments = [
      finding("line-4", 4),
      finding("line-2", 2),
      finding("p0", 3, { priority: IssueComment_ReviewMetadata_Priority.P0 }),
    ];
    renderSection([row("review", "open", findings({ open: 3, p0: 1, p1: 2 }))]);
    fireEvent.click(
      screen.getByRole("button", { name: "plan.findings.n-priority:2" })
    );
    expect(mocks.setWalkerFilter).toHaveBeenCalledWith({ group: "p1" });
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
    renderSection([row("review", "resolved", findings({ resolved: 3 }))]);
    const resolved = screen.getByRole("button", {
      name: "plan.findings.n-resolved:3",
    });
    expect(resolved.nextElementSibling?.textContent).toContain(
      "plan.findings.name-times:2"
    );
    fireEvent.click(resolved);
    expect(mocks.setWalkerFilter).toHaveBeenCalledWith({ group: "resolved" });
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
    renderSection([row("review", "resolved", findings({ overridden: 1 }))]);
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
    expect(mocks.setWalkerFilter).toHaveBeenCalledWith({ group: "overridden" });
    expect(mocks.focusInStatement).toHaveBeenCalledWith(
      expect.objectContaining({ name: "null-3" })
    );
  });

  test("says on hover what the review covered", () => {
    renderSection([row("review", "passed")]);
    expect(
      screen.getByRole("button", { name: "plan.automated-review.scope-label" })
    ).toBeDefined();
    const scope = screen.getByTestId("review-scope");
    expect(scope.textContent).toContain(
      "plan.automated-review.scope-with-engines"
    );
    expect(scope.textContent).toContain("plan.automated-review.project-rules");
    expect(scope.textContent).not.toContain("plan.automated-review.sampled");
  });

  test("says when the review covered only a sample, whichever reviewer is on", () => {
    mocks.ciSamplingSize = 1;
    mocks.targets = [
      "instances/i/databases/hr_test",
      "instances/i/databases/hr_prod",
    ];
    const { unmount } = renderSection([row("review", "passed")]);
    expect(screen.getByTestId("review-scope").textContent).toContain(
      "plan.automated-review.sampled:1"
    );
    unmount();

    mocks.serverInfo = AI_ON;
    renderSection([row("review", "passed")]);
    expect(screen.getByTestId("review-scope").textContent).toContain(
      "plan.automated-review.sampled:1"
    );
  });

  test("covers a database only the AI review reads", () => {
    mocks.targets = ["instances/i/databases/hr_docs"];
    const { unmount } = renderSection([row("review", "passed")]);
    expect(screen.queryByTestId("review-scope")).toBeNull();
    unmount();

    mocks.serverInfo = AI_ON;
    renderSection([row("review", "passed")]);
    expect(screen.getByTestId("review-scope").textContent).toContain(
      "plan.automated-review.scope-with-engines"
    );
  });

  test("claims no coverage for a review that has not run", () => {
    const { unmount } = renderSection([row("review", "not-run")]);
    expect(screen.queryByTestId("review-scope")).toBeNull();
    unmount();
    renderSection([row("review", "running")]);
    expect(screen.queryByTestId("review-scope")).toBeNull();
  });

  test("a review that is off says so, shows no pass, and offers no run", () => {
    mocks.rules = { rules: [], source: "project" };
    renderSection([row("review", "off", undefined, { holdsRollout: false })]);
    const review = screen.getByTestId("automated-review-review");
    expect(review.textContent).toContain(
      "plan.automated-review.off-for-project"
    );
    expect(review.textContent).not.toContain(
      "plan.automated-review.no-findings"
    );
    expect(screen.queryByTestId("review-scope")).toBeNull();
    expect(
      screen.queryByRole("button", { name: /plan.automated-review.rerun/ })
    ).toBeNull();
    expect(screen.queryByRole("button", { name: /common.run/ })).toBeNull();
    expect(screen.queryByTestId("automated-review-summary")).toBeNull();
  });

  test("an off review beside passing checks is not summed up as needing nothing", () => {
    const { unmount } = renderSection([
      row("review", "off", undefined, { holdsRollout: false }),
      row("policy", "passed", undefined, { targets: 1 }),
    ]);
    expect(screen.queryByTestId("automated-review-summary")).toBeNull();
    unmount();

    renderSection([
      row("review", "off", undefined, { holdsRollout: false }),
      row("policy", "open", findings({ open: 1, p0: 1 }), { targets: 1 }),
    ]);
    expect(screen.getByTestId("automated-review-summary").textContent).toBe(
      "plan.automated-review.needs-attention:1"
    );
  });

  test("Run, Retry, and Rerun start the review with one call", async () => {
    const { unmount } = renderSection([row("review", "outdated")]);
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: /common.run/ }));
    });
    expect(mocks.runReview).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ name: "projects/p/issues/1/reviewRun" }),
      expect.anything()
    );
    expect(mocks.reload).toHaveBeenCalledOnce();
    expect(mocks.runChecks).not.toHaveBeenCalled();
    unmount();

    renderSection([
      row("review", "failed", undefined, {
        error: "hr_test: metadata not synced",
      }),
    ]);
    expect(
      screen.getByRole("button", { name: /plan.automated-review.retry/ })
    ).toBeDefined();
  });

  test("a review with a result offers Rerun, since settings and schema change results", async () => {
    renderSection([row("review", "open", findings({ open: 1, p1: 1 }))]);
    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: /plan.automated-review.rerun/ })
      );
    });
    expect(mocks.runReview).toHaveBeenCalledOnce();
  });

  test("a rerun without the AI review costs nothing, so it does not ask", async () => {
    mocks.review = {
      loadFailed: false,
      reload: mocks.reload,
      run: reviewRun(ReviewRun_State.SUCCEEDED),
    };
    renderSection([
      row(
        "review",
        "open",
        findings({ open: 1, p1: 1, resolved: 1, overridden: 1 })
      ),
    ]);
    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: /plan.automated-review.rerun/ })
      );
    });
    expect(mocks.runReview).toHaveBeenCalledOnce();
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  test("shows a failed run's error in full, beside the findings that still stand", () => {
    renderSection([
      row("review", "failed", findings({ open: 1, p0: 1 }), {
        error: "hr_test: metadata not synced",
      }),
    ]);
    expect(screen.getByText("hr_test: metadata not synced")).toBeDefined();
    expect(
      screen.getByRole("button", { name: "plan.findings.n-priority:1" })
        .textContent
    ).toBe("P01");
  });

  test("the review waits for unsaved edits and says so on hover", () => {
    mocks.editing = true;
    renderSection([row("review", "not-run")]);
    const run = screen.getByRole("button", { name: /common.run/ });
    expect((run as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByTestId("tooltip").textContent).toBe(
      "plan.automated-review.save-first"
    );
  });

  test("the review cannot start again while a run is queued or running", () => {
    mocks.review = {
      loadFailed: false,
      reload: mocks.reload,
      run: reviewRun(ReviewRun_State.RUNNING),
    };
    renderSection([row("review", "outdated")]);
    const run = screen.getByRole("button", { name: /common.run/ });
    expect((run as HTMLButtonElement).disabled).toBe(true);
  });

  test("a finished review that includes the AI review offers Rerun behind a confirmation", async () => {
    mocks.serverInfo = AI_ON;
    mocks.review = {
      loadFailed: false,
      reload: mocks.reload,
      run: reviewRun(ReviewRun_State.SUCCEEDED),
    };
    renderSection([row("review", "passed")]);
    fireEvent.click(
      screen.getByRole("button", { name: /plan.automated-review.rerun/ })
    );
    expect(mocks.runReview).not.toHaveBeenCalled();
    expect(
      screen.getByText("plan.automated-review.run-again-title")
    ).toBeDefined();
    const confirm = screen
      .getAllByRole("button", { name: "plan.automated-review.rerun" })
      .at(-1);
    await act(async () => {
      fireEvent.click(confirm!);
    });
    expect(mocks.runReview).toHaveBeenCalledOnce();
  });

  test("offers Retry when the run could not be read", async () => {
    mocks.review = { loadFailed: true, reload: mocks.reload, run: undefined };
    renderSection([row("review", "failed")]);
    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: /plan.automated-review.retry/ })
      );
    });
    expect(mocks.reload).toHaveBeenCalledOnce();
    expect(mocks.runReview).not.toHaveBeenCalled();
  });

  const rerunAll = async () => {
    fireEvent.click(screen.getByRole("button", { name: "common.more" }));
    await act(async () => {
      fireEvent.click(
        await screen.findByText("plan.automated-review.rerun-all")
      );
    });
  };

  test("Re-run all reruns the checks and the review", async () => {
    mocks.review = {
      loadFailed: false,
      reload: mocks.reload,
      run: reviewRun(ReviewRun_State.SUCCEEDED),
    };
    renderSection([row("review", "resolved", findings({ resolved: 3 }))]);
    await rerunAll();
    expect(mocks.runChecks).toHaveBeenCalledOnce();
    expect(mocks.runReview).toHaveBeenCalledOnce();
    // The review is reset before the checks can finish.
    expect(mocks.runReview.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.runChecks.mock.invocationCallOrder[0]
    );
  });

  test("Re-run all asks first when the review makes model calls", async () => {
    mocks.serverInfo = AI_ON;
    mocks.review = {
      loadFailed: false,
      reload: mocks.reload,
      run: reviewRun(ReviewRun_State.SUCCEEDED),
    };
    renderSection([row("review", "passed")]);
    await rerunAll();
    expect(mocks.runChecks).not.toHaveBeenCalled();
    expect(mocks.runReview).not.toHaveBeenCalled();
    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: "plan.automated-review.rerun-all" })
      );
    });
    expect(mocks.runChecks).toHaveBeenCalledOnce();
    expect(mocks.runReview).toHaveBeenCalledOnce();
  });

  test("Re-run all leaves alone a review that is running or off", async () => {
    mocks.review = {
      loadFailed: false,
      reload: mocks.reload,
      run: reviewRun(ReviewRun_State.RUNNING),
    };
    const { unmount } = renderSection([row("review", "running")]);
    await rerunAll();
    expect(mocks.runChecks).toHaveBeenCalledOnce();
    expect(mocks.runReview).not.toHaveBeenCalled();
    unmount();

    mocks.review = {
      loadFailed: false,
      reload: mocks.reload,
      run: reviewRun(ReviewRun_State.SUCCEEDED),
    };
    renderSection([
      row("review", "off", undefined, { holdsRollout: false }),
      row("policy", "passed", undefined, { targets: 1 }),
    ]);
    await rerunAll();
    expect(mocks.runChecks).toHaveBeenCalledTimes(2);
    expect(mocks.runReview).not.toHaveBeenCalled();
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
