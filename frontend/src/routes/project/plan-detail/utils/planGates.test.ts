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
  ReviewRun_Status,
  ReviewRun_Type,
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
  aiReviewRow,
  computePlanGates,
  countFindings,
  countOpenDiscussions,
  emptyFindings,
  findingSetOf,
  inFindingGroup,
  planCheckRow,
  planCheckRunRow,
  type ReviewerFindings,
  resolvedByPeople,
  rowNeedsAttention,
  ruleReviewApplies,
  ruleReviewRow,
} from "./planGates";

const finding = (
  runType: ReviewRun_Type,
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
      runType,
      priority: options.priority ?? IssueComment_ReviewMetadata_Priority.P1,
      superseded: options.superseded ?? false,
    }),
  });

const discussion = (threadState: IssueComment_ThreadState) =>
  create(IssueCommentSchema, { name: `d${Math.random()}`, threadState });

const run = (status: ReviewRun_Status, extra: { outdated?: boolean } = {}) =>
  create(ReviewRunSchema, { status, ...extra });

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
      expect(
        findingSetOf(
          finding(ReviewRun_Type.RULE, state, { priority, superseded })
        )
      ).toBe(set);
    }
  );

  test("a discussion is no finding", () => {
    expect(findingSetOf(discussion(OPEN))).toBeUndefined();
  });
});

describe("inFindingGroup", () => {
  test("open covers every open finding, whatever its priority", () => {
    for (const priority of [P0, P1, P2]) {
      expect(
        inFindingGroup(finding(ReviewRun_Type.RULE, OPEN, { priority }), "open")
      ).toBe(true);
    }
    expect(
      inFindingGroup(finding(ReviewRun_Type.RULE, RESOLVED, {}), "open")
    ).toBe(false);
  });
});

describe("countFindings", () => {
  test("splits open findings by priority and sets resolved P0s apart", () => {
    const comments = [
      finding(ReviewRun_Type.RULE, OPEN, { priority: P0 }),
      finding(ReviewRun_Type.RULE, OPEN, { superseded: true }),
      finding(ReviewRun_Type.RULE, RESOLVED, { priority: P0 }),
      finding(ReviewRun_Type.RULE, RESOLVED, {}),
      finding(ReviewRun_Type.RULE, RESOLVED, { priority: P2 }),
      finding(ReviewRun_Type.RULE, RESOLVED, { superseded: true }),
      finding(ReviewRun_Type.AI, OPEN, { priority: P2 }),
      discussion(OPEN),
    ];
    const rule = countFindings(comments, "rule");
    expect(rule).toEqual({
      open: 2,
      p0: 1,
      p1: 1,
      p2: 0,
      overridden: 1,
      resolved: 2,
      earlier: 1,
    });
    expect(resolvedByPeople(rule)).toBe(3);
    expect(countFindings(comments, "ai")).toMatchObject({ open: 1, p2: 1 });
    expect(countOpenDiscussions(comments)).toBe(1);
  });
});

describe("rowNeedsAttention", () => {
  test("a P0 resolved without a change needs attention though it holds nothing", () => {
    const row = ruleReviewRow({
      run: run(ReviewRun_Status.DONE),
      stale: false,
      loadFailed: false,
      findings: findings({ overridden: 1 }),
    });
    expect(row.conclusion).toBe("resolved");
    expect(row.holdsRollout).toBe(false);
    expect(rowNeedsAttention(row)).toBe(true);
  });
});

describe("ruleReviewApplies", () => {
  test.each([
    ["a missing run on a plan with SQL", null, true, true],
    ["a missing run on a plan with nothing to review", null, false, false],
    [
      "a run on a plan with nothing left to review",
      run(ReviewRun_Status.DONE),
      false,
      true,
    ],
    ["a run not loaded yet", undefined, false, true],
  ] as const)("%s", (_, reviewRun, reviewsSQL, applies) => {
    expect(ruleReviewApplies(reviewRun, reviewsSQL)).toBe(applies);
  });
});

describe("ruleReviewRow", () => {
  test.each([
    [run(ReviewRun_Status.DONE), NO_FINDINGS, "passed", false],
    [run(ReviewRun_Status.DONE), findings({ resolved: 3 }), "resolved", false],
    [
      run(ReviewRun_Status.DONE),
      findings({ open: 2, p1: 2, resolved: 1 }),
      "open",
      true,
    ],
    [run(ReviewRun_Status.RUNNING), NO_FINDINGS, "running", true],
    [run(ReviewRun_Status.AVAILABLE), NO_FINDINGS, "running", true],
    [run(ReviewRun_Status.FAILED), NO_FINDINGS, "failed", true],
    [
      run(ReviewRun_Status.DONE, { outdated: true }),
      NO_FINDINGS,
      "outdated",
      true,
    ],
    [null, NO_FINDINGS, "not-run", true],
    [undefined, NO_FINDINGS, "loading", true],
  ] as const)("run %# concludes %s", (input, findings, conclusion, holds) => {
    const row = ruleReviewRow({
      run: input,
      stale: false,
      loadFailed: false,
      findings,
    });
    expect(row.conclusion).toBe(conclusion);
    expect(row.holdsRollout).toBe(holds);
  });

  test("a stale completion is not a result yet", () => {
    const row = ruleReviewRow({
      run: run(ReviewRun_Status.DONE),
      stale: true,
      loadFailed: false,
      findings: NO_FINDINGS,
    });
    expect(row.conclusion).toBe("running");
  });

  test("a load failure is a failure", () => {
    const row = ruleReviewRow({
      run: undefined,
      stale: false,
      loadFailed: true,
      findings: NO_FINDINGS,
    });
    expect(row.conclusion).toBe("failed");
  });
});

describe("aiReviewRow", () => {
  test("holds the rollout only through open findings", () => {
    expect(
      aiReviewRow({ run: run(ReviewRun_Status.RUNNING), findings: NO_FINDINGS })
        .holdsRollout
    ).toBe(false);
    expect(aiReviewRow({ run: null, findings: NO_FINDINGS }).conclusion).toBe(
      "not-run"
    );
    const open = aiReviewRow({
      run: run(ReviewRun_Status.DONE),
      findings: findings({ open: 1, p1: 1 }),
    });
    expect(open.conclusion).toBe("open");
    expect(open.holdsRollout).toBe(true);
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

describe("statement report and run rows", () => {
  test("a report that fails holds the rollout; one that passes has no row", () => {
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
    const failing = planCheckRow("report", [report(Advice_Level.ERROR)]);
    expect(failing?.conclusion).toBe("open");
    expect(failing?.holdsRollout).toBe(true);
    expect(failing?.findings.p0).toBe(1);
    expect(
      planCheckRow("report", [report(Advice_Level.SUCCESS)])
    ).toBeUndefined();
    expect(
      planCheckRow("report", [report(Advice_Level.WARNING)])?.holdsRollout
    ).toBe(false);
  });

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
  const openRule = ruleReviewRow({
    run: run(ReviewRun_Status.DONE),
    stale: false,
    loadFailed: false,
    findings: findings({ open: 2, p1: 2 }),
  });
  const pendingAI = aiReviewRow({
    run: run(ReviewRun_Status.RUNNING),
    findings: NO_FINDINGS,
  });

  test("a draft waits on submission and its open findings", () => {
    const gates = computePlanGates({
      issue: issue({ draft: true }),
      rows: [openRule, pendingAI],
      openDiscussions: 0,
    });
    expect(gates.map((gate) => gate.kind)).toEqual(["submit", "review"]);
  });

  test("lists approval, held reviewers, and discussion in order", () => {
    const gates = computePlanGates({
      issue: issue({ approvalStatus: ApprovalStatus.PENDING }),
      currentApprovalRole: "roles/workspaceDBA",
      rows: [openRule, pendingAI],
      openDiscussions: 1,
    });
    expect(gates).toEqual([
      { kind: "approval", rejected: false, role: "roles/workspaceDBA" },
      { kind: "review", row: openRule },
      { kind: "discussion", open: 1 },
    ]);
  });

  test("is empty once every gate passes, or when the issue is closed", () => {
    expect(
      computePlanGates({
        issue: issue({ approvalStatus: ApprovalStatus.SKIPPED }),
        rows: [pendingAI],
        openDiscussions: 0,
      })
    ).toEqual([]);
    expect(
      computePlanGates({
        issue: issue({ status: IssueStatus.DONE }),
        rows: [openRule],
        openDiscussions: 3,
      })
    ).toEqual([]);
  });
});
