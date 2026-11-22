// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  IssueComment_ReviewMetadata_Priority,
  IssueComment_ReviewMetadataSchema,
  IssueComment_ThreadState,
  IssueCommentSchema,
  IssueSchema,
  ReviewRun_State,
  ReviewRunSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  PlanCheckRun_Result_Type,
  PlanCheckRun_ResultSchema,
  PlanCheckRun_Status,
  PlanCheckRunSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import {
  computePlanGates,
  countFindings,
  countOpenDiscussions,
  emptyFindings,
  findingSetOf,
  findingsRow,
  inFindingGroup,
  planCheckRow,
  planCheckRunRow,
  type ReviewerFindings,
  resolvedByPeople,
  reviewApplies,
  reviewRow,
  rowNeedsAttention,
  statementReportFails,
} from "./planGates";

const finding = (
  threadState: IssueComment_ThreadState,
  options: {
    priority?: IssueComment_ReviewMetadata_Priority;
    superseded?: boolean;
  } = {}
) =>
  create(IssueCommentSchema, {
    name: `c${Math.random()}`,
    threadState,
    reviewMetadata: create(IssueComment_ReviewMetadataSchema, {
      priority: options.priority ?? IssueComment_ReviewMetadata_Priority.P1,
      superseded: options.superseded ?? false,
    }),
  });

const discussion = (threadState: IssueComment_ThreadState) =>
  create(IssueCommentSchema, { name: `d${Math.random()}`, threadState });

const run = (
  state: ReviewRun_State,
  extra: { outdated?: boolean; error?: { message: string } } = {}
) => create(ReviewRunSchema, { state, ...extra });

const NO_FINDINGS = emptyFindings();
const findings = (extra: Partial<ReviewerFindings>): ReviewerFindings => ({
  ...emptyFindings(),
  ...extra,
});
const { P0, P1, P2 } = IssueComment_ReviewMetadata_Priority;
const { OPEN, RESOLVED } = IssueComment_ThreadState;

describe("findingSetOf", () => {
  test.each([
    [OPEN, P0, false, "p0"],
    [OPEN, P1, false, "p1"],
    [OPEN, P2, false, "p2"],
    [OPEN, P1, true, "p1"],
    [RESOLVED, P0, false, "overridden"],
    [RESOLVED, P1, false, "resolved"],
    [RESOLVED, P2, false, "resolved"],
    [RESOLVED, P0, true, "earlier"],
  ] as const)(
    "a %s %s finding (superseded %s) is %s",
    (state, priority, superseded, set) => {
      expect(findingSetOf(finding(state, { priority, superseded }))).toBe(set);
    }
  );

  test("a discussion is no finding", () => {
    expect(findingSetOf(discussion(OPEN))).toBeUndefined();
  });
});

describe("inFindingGroup", () => {
  test("open covers every open finding, whatever its priority", () => {
    for (const priority of [P0, P1, P2]) {
      expect(inFindingGroup(finding(OPEN, { priority }), "open")).toBe(true);
    }
    expect(inFindingGroup(finding(RESOLVED, {}), "open")).toBe(false);
  });
});

describe("countFindings", () => {
  test("splits open findings by priority and sets resolved P0s apart", () => {
    const comments = [
      finding(OPEN, { priority: P0 }),
      finding(OPEN, { superseded: true }),
      finding(OPEN, { priority: P2 }),
      finding(RESOLVED, { priority: P0 }),
      finding(RESOLVED, {}),
      finding(RESOLVED, { priority: P2 }),
      finding(RESOLVED, { superseded: true }),
      discussion(OPEN),
    ];
    const counts = countFindings(comments);
    expect(counts).toEqual({
      open: 3,
      p0: 1,
      p1: 1,
      p2: 1,
      overridden: 1,
      resolved: 2,
      earlier: 1,
    });
    expect(resolvedByPeople(counts)).toBe(3);
    expect(countOpenDiscussions(comments)).toBe(1);
  });
});

describe("rowNeedsAttention", () => {
  test("a P0 resolved without a change needs attention though it holds nothing", () => {
    const row = reviewRow({
      run: run(ReviewRun_State.SUCCEEDED),
      stale: false,
      loadFailed: false,
      findings: findings({ overridden: 1 }),
      off: false,
    });
    expect(row.conclusion).toBe("resolved");
    expect(row.holdsRollout).toBe(false);
    expect(rowNeedsAttention(row)).toBe(true);
  });
});

describe("reviewApplies", () => {
  test.each([
    ["a missing run on a plan with SQL", null, true, true],
    ["a missing run on a plan with nothing to review", null, false, false],
    [
      "a run on a plan with nothing left to review",
      run(ReviewRun_State.SUCCEEDED),
      false,
      true,
    ],
    ["a run not loaded yet", undefined, false, true],
  ] as const)("%s", (_, reviewRun, reviewsSQL, applies) => {
    expect(reviewApplies(reviewRun, reviewsSQL)).toBe(applies);
  });
});

describe("reviewRow", () => {
  test.each([
    [run(ReviewRun_State.SUCCEEDED), NO_FINDINGS, "passed", false],
    [
      run(ReviewRun_State.SUCCEEDED),
      findings({ resolved: 3 }),
      "resolved",
      false,
    ],
    [
      run(ReviewRun_State.SUCCEEDED),
      findings({ open: 2, p1: 2, resolved: 1 }),
      "open",
      true,
    ],
    [run(ReviewRun_State.RUNNING), NO_FINDINGS, "running", true],
    [run(ReviewRun_State.FAILED), NO_FINDINGS, "failed", true],
    [
      run(ReviewRun_State.SUCCEEDED, { outdated: true }),
      NO_FINDINGS,
      "outdated",
      true,
    ],
    [null, NO_FINDINGS, "not-run", true],
    [undefined, NO_FINDINGS, "loading", true],
  ] as const)("run %# concludes %s", (input, findings, conclusion, holds) => {
    const row = reviewRow({
      run: input,
      stale: false,
      loadFailed: false,
      findings,
      off: false,
    });
    expect(row.reviewer).toBe("review");
    expect(row.conclusion).toBe(conclusion);
    expect(row.holdsRollout).toBe(holds);
  });

  test("a stale completion is not a result yet", () => {
    const row = reviewRow({
      run: run(ReviewRun_State.SUCCEEDED),
      stale: true,
      loadFailed: false,
      findings: NO_FINDINGS,
      off: false,
    });
    expect(row.conclusion).toBe("running");
  });

  test("a load failure is a failure", () => {
    const row = reviewRow({
      run: undefined,
      stale: false,
      loadFailed: true,
      findings: NO_FINDINGS,
      off: false,
    });
    expect(row.conclusion).toBe("failed");
  });

  test("a failed run carries its error message and the findings that stand", () => {
    const row = reviewRow({
      run: run(ReviewRun_State.FAILED, { error: { message: "rate limited" } }),
      stale: false,
      loadFailed: false,
      findings: findings({ open: 1, p0: 1 }),
      off: false,
    });
    expect(row.conclusion).toBe("failed");
    expect(row.error).toBe("rate limited");
    expect(row.findings.p0).toBe(1);
  });

  test("a review that is off is never a pass", () => {
    for (const input of [run(ReviewRun_State.SUCCEEDED), null]) {
      const row = reviewRow({
        run: input,
        stale: false,
        loadFailed: false,
        findings: NO_FINDINGS,
        off: true,
      });
      expect(row.conclusion).toBe("off");
      expect(row.holdsRollout).toBe(false);
      expect(rowNeedsAttention(row)).toBe(false);
    }
  });

  test("a review that is off still shows a run the server waits for", () => {
    const conclude = (input: Parameters<typeof reviewRow>[0]["run"]) =>
      reviewRow({
        run: input,
        stale: false,
        loadFailed: false,
        findings: NO_FINDINGS,
        off: true,
      });
    const failed = conclude(
      run(ReviewRun_State.FAILED, { error: { message: "rate limited" } })
    );
    expect(failed.conclusion).toBe("failed");
    expect(failed.error).toBe("rate limited");
    expect(failed.holdsRollout).toBe(true);
    expect(conclude(run(ReviewRun_State.RUNNING)).conclusion).toBe("running");
    expect(
      conclude(run(ReviewRun_State.SUCCEEDED, { outdated: true })).conclusion
    ).toBe("outdated");
    expect(conclude(undefined).conclusion).toBe("loading");
  });

  test("a review that is off still shows the findings earlier runs left", () => {
    const open = reviewRow({
      run: run(ReviewRun_State.SUCCEEDED),
      stale: false,
      loadFailed: false,
      findings: findings({ open: 1, p1: 1 }),
      off: true,
    });
    expect(open.conclusion).toBe("open");
    expect(open.holdsRollout).toBe(true);
    const resolved = reviewRow({
      run: run(ReviewRun_State.SUCCEEDED),
      stale: false,
      loadFailed: false,
      findings: findings({ resolved: 1 }),
      off: true,
    });
    expect(resolved.conclusion).toBe("resolved");
  });
});

describe("findingsRow", () => {
  test("holds the rollout only through open findings", () => {
    const open = findingsRow(findings({ open: 1, p1: 1 }));
    expect(open.conclusion).toBe("open");
    expect(open.holdsRollout).toBe(true);
    const resolved = findingsRow(findings({ resolved: 2, overridden: 1 }));
    expect(resolved.conclusion).toBe("resolved");
    expect(resolved.holdsRollout).toBe(false);
    expect(rowNeedsAttention(resolved)).toBe(true);
  });
});

describe("planCheckRow", () => {
  const checkRun = (
    status: PlanCheckRun_Status,
    results: [PlanCheckRun_Result_Type, Advice_Level, string][]
  ) =>
    create(PlanCheckRunSchema, {
      status,
      results: results.map(([type, level, target]) =>
        create(PlanCheckRun_ResultSchema, { type, status: level, target })
      ),
    });

  test("is absent without a result of its type", () => {
    const runs = [
      checkRun(PlanCheckRun_Status.DONE, [
        [
          PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
          Advice_Level.SUCCESS,
          "db1",
        ],
      ]),
    ];
    expect(planCheckRow("policy", runs)).toBeUndefined();
    expect(planCheckRow("ghost", runs)).toBeUndefined();
  });

  test("errors are open findings, warnings pass", () => {
    const runs = [
      checkRun(PlanCheckRun_Status.DONE, [
        [PlanCheckRun_Result_Type.STATEMENT_ADVISE, Advice_Level.ERROR, "db1"],
        [
          PlanCheckRun_Result_Type.STATEMENT_ADVISE,
          Advice_Level.WARNING,
          "db2",
        ],
        [PlanCheckRun_Result_Type.GHOST_SYNC, Advice_Level.SUCCESS, "db1"],
      ]),
    ];
    const policy = planCheckRow("policy", runs);
    expect(policy?.conclusion).toBe("open");
    expect(policy?.findings.open).toBe(1);
    expect(policy?.warnings).toBe(1);
    expect(policy?.targets).toBe(2);
    expect(planCheckRow("ghost", runs)?.conclusion).toBe("passed");
  });
});

describe("statement report", () => {
  const report = (level: Advice_Level) =>
    create(PlanCheckRunSchema, {
      status: PlanCheckRun_Status.DONE,
      results: [
        create(PlanCheckRun_ResultSchema, {
          type: PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT,
          status: level,
          target: "db1",
        }),
      ],
    });

  test("fails only on an ERROR, which still holds the rollout without a row", () => {
    expect(statementReportFails([report(Advice_Level.ERROR)])).toBe(true);
    expect(statementReportFails([report(Advice_Level.WARNING)])).toBe(false);
    expect(
      computePlanGates({
        issue: create(IssueSchema, {
          status: IssueStatus.OPEN,
          approvalStatus: ApprovalStatus.SKIPPED,
        }),
        rows: [],
        statementError: true,
        openDiscussions: 0,
      })
    ).toEqual([{ kind: "statement-error" }]);
  });
});

describe("run rows", () => {
  test.each([
    [PlanCheckRun_Status.FAILED, "failed"],
    [PlanCheckRun_Status.CANCELED, "failed"],
    [PlanCheckRun_Status.RUNNING, "running"],
    [PlanCheckRun_Status.STATUS_UNSPECIFIED, "running"],
  ] as const)(
    "a run %s without results holds the rollout as %s",
    (status, conclusion) => {
      const row = planCheckRunRow([
        create(PlanCheckRunSchema, { status, error: "cannot derive targets" }),
      ]);
      expect(row?.conclusion).toBe(conclusion);
      expect(row?.holdsRollout).toBe(true);
      expect(row?.error).toBe("cannot derive targets");
    }
  );

  test("a finished run has no run row", () => {
    expect(
      planCheckRunRow([
        create(PlanCheckRunSchema, { status: PlanCheckRun_Status.DONE }),
      ])
    ).toBeUndefined();
  });
});

describe("computePlanGates", () => {
  const issue = (extra: Parameters<typeof create<typeof IssueSchema>>[1]) =>
    create(IssueSchema, { status: IssueStatus.OPEN, ...extra });
  const openReview = reviewRow({
    run: run(ReviewRun_State.SUCCEEDED),
    stale: false,
    loadFailed: false,
    findings: findings({ open: 2, p1: 2 }),
    off: false,
  });
  const passedChecks = planCheckRow("ghost", [
    create(PlanCheckRunSchema, {
      status: PlanCheckRun_Status.DONE,
      results: [
        create(PlanCheckRun_ResultSchema, {
          type: PlanCheckRun_Result_Type.GHOST_SYNC,
          status: Advice_Level.SUCCESS,
          target: "db1",
        }),
      ],
    }),
  ]);
  if (!passedChecks) throw new Error("expected a gh-ost row");

  test("a draft waits on submission and its open findings", () => {
    const gates = computePlanGates({
      issue: issue({ draft: true }),
      rows: [openReview, passedChecks],
      openDiscussions: 0,
    });
    expect(gates.map((gate) => gate.kind)).toEqual(["submit", "review"]);
  });

  test("lists approval, held reviewers, and discussion in order", () => {
    const gates = computePlanGates({
      issue: issue({ approvalStatus: ApprovalStatus.PENDING }),
      currentApprovalRole: "roles/workspaceDBA",
      rows: [openReview, passedChecks],
      openDiscussions: 1,
    });
    expect(gates).toEqual([
      { kind: "approval", rejected: false, role: "roles/workspaceDBA" },
      { kind: "review", row: openReview },
      { kind: "discussion", open: 1 },
    ]);
  });

  test("is empty once every gate passes, or when the issue is closed", () => {
    expect(
      computePlanGates({
        issue: issue({ approvalStatus: ApprovalStatus.SKIPPED }),
        rows: [passedChecks],
        openDiscussions: 0,
      })
    ).toEqual([]);
    expect(
      computePlanGates({
        issue: issue({ status: IssueStatus.DONE }),
        rows: [openReview],
        openDiscussions: 3,
      })
    ).toEqual([]);
  });

  test("a review that is off holds nothing", () => {
    const off = reviewRow({
      run: null,
      stale: false,
      loadFailed: false,
      findings: NO_FINDINGS,
      off: true,
    });
    expect(
      computePlanGates({
        issue: issue({ approvalStatus: ApprovalStatus.SKIPPED }),
        rows: [off],
        openDiscussions: 0,
      })
    ).toEqual([]);
  });
});
