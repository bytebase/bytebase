import { describe, expect, test } from "vitest";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { getCreatePlanBlockers, getSubmitReviewAdvance } from "./advanceState";

const t = (key: string) => key;

const makePlan = (cases: string[], statusCount = {}): Plan =>
  ({
    planCheckRunStatusCount: statusCount,
    specs: cases.map((caseName) => ({
      config: { case: caseName, value: {} },
    })),
  }) as unknown as Plan;

const project = (
  overrides: Partial<
    Pick<Project, "enforceSqlReview" | "forceIssueLabels">
  > = {}
) =>
  ({
    enforceSqlReview: false,
    forceIssueLabels: false,
    ...overrides,
  }) as Project;

const submitArgs = (
  overrides: Partial<Parameters<typeof getSubmitReviewAdvance>[0]> = {}
) => ({
  emptySpecCount: 0,
  isEditing: false,
  plan: makePlan(["changeDatabaseConfig"]),
  project: project(),
  selectedLabelCount: 1,
  t,
  ...overrides,
});

const ids = (blockers: { id: string }[]) => blockers.map((b) => b.id);

describe("getCreatePlanBlockers", () => {
  test("flags an empty title", () => {
    expect(getCreatePlanBlockers({ emptySpecCount: 0, title: "", t })).toEqual([
      { id: "title", kind: "fix", message: "plan.title-required" },
    ]);
  });

  test("flags a whitespace-only title", () => {
    expect(
      ids(getCreatePlanBlockers({ emptySpecCount: 0, title: "   ", t }))
    ).toEqual(["title"]);
  });

  test("flags empty statements", () => {
    expect(
      getCreatePlanBlockers({ emptySpecCount: 2, title: "Add column", t })
    ).toEqual([
      {
        id: "statement",
        kind: "fix",
        message: "plan.navigator.statement-empty",
      },
    ]);
  });

  test("lists every blocker, title first", () => {
    expect(
      ids(
        getCreatePlanBlockers({
          emptySpecCount: 1,
          permissionReason: "common.missing-required-permission",
          title: "",
          t,
        })
      )
    ).toEqual(["title", "statement", "permission"]);
  });

  test("reports a missing permission as a blocker rather than hiding the action", () => {
    expect(
      getCreatePlanBlockers({
        emptySpecCount: 0,
        permissionReason: "common.missing-required-permission",
        title: "Add column",
        t,
      })
    ).toEqual([
      {
        id: "permission",
        kind: "fix",
        message: "common.missing-required-permission",
      },
    ]);
  });

  test("returns no blockers when valid", () => {
    expect(
      getCreatePlanBlockers({ emptySpecCount: 0, title: "Add column", t })
    ).toEqual([]);
  });
});

describe("getSubmitReviewAdvance", () => {
  test("returns no blockers for a clean draft", () => {
    expect(getSubmitReviewAdvance(submitArgs()).blockers).toEqual([]);
  });

  test("flags empty statements", () => {
    expect(
      ids(getSubmitReviewAdvance(submitArgs({ emptySpecCount: 1 })).blockers)
    ).toEqual(["statement"]);
  });

  test("marks queued and running checks as a wait, not a chore", () => {
    for (const statusCount of [{ AVAILABLE: 1 }, { RUNNING: 1 }]) {
      expect(
        getSubmitReviewAdvance(
          submitArgs({ plan: makePlan(["changeDatabaseConfig"], statusCount) })
        ).blockers
      ).toEqual([
        {
          id: "checks-running",
          kind: "wait",
          message:
            "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running",
        },
      ]);
    }
  });

  test("blocks failed checks only where SQL review is enforced", () => {
    const plan = makePlan(["changeDatabaseConfig"], { ERROR: 1 });

    expect(ids(getSubmitReviewAdvance(submitArgs({ plan })).blockers)).toEqual(
      []
    );
    expect(
      getSubmitReviewAdvance(
        submitArgs({ plan, project: project({ enforceSqlReview: true }) })
      ).blockers
    ).toContainEqual(
      expect.objectContaining({
        id: "checks-failed",
        message:
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass",
      })
    );
  });

  test("treats a failed run the same as an errored one", () => {
    expect(
      ids(
        getSubmitReviewAdvance(
          submitArgs({
            plan: makePlan(["changeDatabaseConfig"], { FAILED: 1 }),
            project: project({ enforceSqlReview: true }),
          })
        ).blockers
      )
    ).toEqual(["checks-failed"]);
  });

  test("requires labels only where the project forces them", () => {
    expect(
      ids(
        getSubmitReviewAdvance(submitArgs({ selectedLabelCount: 0 })).blockers
      )
    ).toEqual([]);
    expect(
      getSubmitReviewAdvance(
        submitArgs({
          project: project({ forceIssueLabels: true }),
          selectedLabelCount: 0,
        })
      ).blockers
    ).toEqual([
      {
        id: "labels",
        kind: "fix",
        message: "plan.labels-required-for-review",
      },
    ]);
  });

  test("accepts an existing label when the project forces them", () => {
    expect(
      getSubmitReviewAdvance(
        submitArgs({
          project: project({ forceIssueLabels: true }),
          selectedLabelCount: 1,
        })
      ).blockers
    ).toEqual([]);
  });

  test("flags unsaved statement edits", () => {
    expect(
      getSubmitReviewAdvance(submitArgs({ isEditing: true })).blockers
    ).toEqual([
      {
        id: "editing",
        kind: "fix",
        message: "plan.editor.save-changes-before-continuing",
      },
    ]);
  });

  test("reports a missing update permission as a blocker", () => {
    expect(
      getSubmitReviewAdvance(
        submitArgs({
          permissionReason: "plan.draft-update-permission-required",
        })
      ).blockers
    ).toEqual([
      {
        id: "permission",
        kind: "fix",
        message: "plan.draft-update-permission-required",
      },
    ]);
  });

  test("lists every blocker at once", () => {
    expect(
      ids(
        getSubmitReviewAdvance(
          submitArgs({
            emptySpecCount: 1,
            isEditing: true,
            permissionReason: "plan.draft-update-permission-required",
            plan: makePlan(["changeDatabaseConfig"], { RUNNING: 1, ERROR: 1 }),
            project: project({
              enforceSqlReview: true,
              forceIssueLabels: true,
            }),
            selectedLabelCount: 0,
          })
        ).blockers
      )
    ).toEqual([
      "statement",
      "checks-running",
      "checks-failed",
      "labels",
      "editing",
      "permission",
    ]);
  });
});

describe("getSubmitReviewAdvance override", () => {
  const decisionFor = (
    overrides: Partial<Parameters<typeof getSubmitReviewAdvance>[0]>
  ) => getSubmitReviewAdvance(submitArgs(overrides)).decision;

  test("offers the override when checks failed and SQL review is not enforced", () => {
    expect(
      decisionFor({ plan: makePlan(["changeDatabaseConfig"], { ERROR: 1 }) })
    ).toEqual({
      body: "issue.checks-warning-hint",
      headline: "plan.lifecycle.gate-checks-failed",
      verb: "plan.submit-review-anyway",
    });
  });

  test("withholds the override where SQL review is enforced, and blocks instead", () => {
    const advance = getSubmitReviewAdvance(
      submitArgs({
        plan: makePlan(["changeDatabaseConfig"], { ERROR: 1 }),
        project: project({ enforceSqlReview: true }),
      })
    );

    expect(advance.decision).toBeUndefined();
    expect(ids(advance.blockers)).toEqual(["checks-failed"]);
  });

  test("offers nothing when checks pass", () => {
    expect(
      decisionFor({ plan: makePlan(["changeDatabaseConfig"], { SUCCESS: 2 }) })
    ).toBeUndefined();
  });

  test("offers nothing for a plan that is not purely a database change", () => {
    expect(
      decisionFor({
        plan: makePlan(["changeDatabaseConfig", "createDatabaseConfig"], {
          ERROR: 1,
        }),
      })
    ).toBeUndefined();
  });
});
