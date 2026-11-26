// @vitest-environment node
import { describe, expect, test } from "vitest";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  getCreatePlanBlockers,
  getSubmitReviewAdvance,
  hasChangeWithoutTargets,
} from "./advanceState";

const t = (key: string) => key;

const makePlan = (
  cases: string[],
  statusCount = {},
  targets = ["instances/test/databases/db"]
): Plan =>
  ({
    planCheckRunStatusCount: statusCount,
    specs: cases.map((caseName) => ({
      config: { case: caseName, value: { targets } },
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
  const createArgs = (
    overrides: Partial<Parameters<typeof getCreatePlanBlockers>[0]> = {}
  ) => ({
    emptySpecCount: 0,
    missingTargets: false,
    title: "Add column",
    t,
    ...overrides,
  });

  test("flags an empty title", () => {
    expect(getCreatePlanBlockers(createArgs({ title: "" }))).toEqual([
      { id: "title", kind: "fix", message: "plan.title-required" },
    ]);
  });

  test("flags a whitespace-only title", () => {
    expect(ids(getCreatePlanBlockers(createArgs({ title: "   " })))).toEqual([
      "title",
    ]);
  });

  test("flags a database change with no targets", () => {
    expect(getCreatePlanBlockers(createArgs({ missingTargets: true }))).toEqual(
      [{ id: "targets", kind: "fix", message: "plan.targets.required" }]
    );
  });

  test("flags empty statements", () => {
    expect(getCreatePlanBlockers(createArgs({ emptySpecCount: 2 }))).toEqual([
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
        getCreatePlanBlockers(
          createArgs({
            emptySpecCount: 1,
            missingTargets: true,
            permissionReason: "common.missing-required-permission",
            title: "",
          })
        )
      )
    ).toEqual(["title", "targets", "statement", "permission"]);
  });

  test("reports a missing permission as a blocker rather than hiding the action", () => {
    expect(
      getCreatePlanBlockers(
        createArgs({ permissionReason: "common.missing-required-permission" })
      )
    ).toEqual([
      {
        id: "permission",
        kind: "fix",
        message: "common.missing-required-permission",
      },
    ]);
  });

  test("returns no blockers when valid", () => {
    expect(getCreatePlanBlockers(createArgs())).toEqual([]);
  });
});

describe("hasChangeWithoutTargets", () => {
  test("is true when any database change has no targets", () => {
    const plan = makePlan(["changeDatabaseConfig", "changeDatabaseConfig"]);
    const second = plan.specs[1].config.value as { targets: string[] };
    second.targets = [];

    expect(hasChangeWithoutTargets(plan.specs)).toBe(true);
  });

  test("ignores specs that are not database changes", () => {
    expect(
      hasChangeWithoutTargets(makePlan(["createDatabaseConfig"], {}, []).specs)
    ).toBe(false);
  });

  test("is false when every database change has a target", () => {
    expect(
      hasChangeWithoutTargets(makePlan(["changeDatabaseConfig"]).specs)
    ).toBe(false);
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

  test("flags a database change with no targets", () => {
    expect(
      getSubmitReviewAdvance(
        submitArgs({ plan: makePlan(["changeDatabaseConfig"], {}, []) })
      ).blockers
    ).toEqual([
      { id: "targets", kind: "fix", message: "plan.targets.required" },
    ]);
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

  test("blocks failed and canceled runs when SQL review is enforced", () => {
    for (const status of ["FAILED", "CANCELED"]) {
      expect(
        ids(
          getSubmitReviewAdvance(
            submitArgs({
              plan: makePlan(["changeDatabaseConfig"], { [status]: 1 }),
              project: project({ enforceSqlReview: true }),
            })
          ).blockers
        )
      ).toEqual(["checks-failed"]);
    }
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

  test("offers the override when a run was canceled", () => {
    expect(
      decisionFor({ plan: makePlan(["changeDatabaseConfig"], { CANCELED: 1 }) })
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
