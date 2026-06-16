import { describe, expect, test } from "vitest";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  getCreateIssueBlockingErrors,
  getCreateIssueConfirmErrors,
  getCreatePlanBlockingReasons,
  hasChecksWarning,
  shouldStayOnPlanDetailPage,
} from "./header";

const t = (key: string) => key;

const makePlan = (cases: string[], statusCount = {}): Plan =>
  ({
    planCheckRunStatusCount: statusCount,
    specs: cases.map((caseName) => ({
      config: { case: caseName, value: {} },
    })),
  }) as unknown as Plan;

describe("plan detail header create issue helpers", () => {
  test("keeps database change plans on plan detail after issue creation", () => {
    expect(shouldStayOnPlanDetailPage(makePlan(["changeDatabaseConfig"]))).toBe(
      true
    );
    expect(shouldStayOnPlanDetailPage(makePlan(["exportDataConfig"]))).toBe(
      false
    );
    expect(shouldStayOnPlanDetailPage(makePlan(["createDatabaseConfig"]))).toBe(
      false
    );
  });

  test("flags non-blocking check warnings without adding a confirm error", () => {
    const plan = makePlan(["changeDatabaseConfig"], { ERROR: 1 });
    const project = {
      enforceSqlReview: false,
      forceIssueLabels: false,
    } as Project;
    const blockingErrors = getCreateIssueBlockingErrors({
      emptySpecCount: 0,
      plan,
      project,
      t,
    });

    expect(blockingErrors).toEqual([]);
    expect(hasChecksWarning(plan)).toBe(true);
    expect(
      getCreateIssueConfirmErrors({
        blockingErrors,
        project,
        selectedLabelCount: 0,
        t,
      })
    ).toEqual([]);
  });

  test("keeps SQL review enforcement as a blocking error", () => {
    const plan = makePlan(["changeDatabaseConfig"], { ERROR: 1 });
    const project = {
      enforceSqlReview: true,
      forceIssueLabels: false,
    } as Project;

    expect(
      getCreateIssueBlockingErrors({
        emptySpecCount: 0,
        plan,
        project,
        t,
      })
    ).toContain(
      "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
    );
  });
});

describe("getCreatePlanBlockingReasons", () => {
  test("flags an empty title", () => {
    expect(
      getCreatePlanBlockingReasons({ title: "", emptySpecCount: 0, t })
    ).toEqual(["plan.title-required"]);
  });

  test("flags a whitespace-only title", () => {
    expect(
      getCreatePlanBlockingReasons({ title: "   ", emptySpecCount: 0, t })
    ).toEqual(["plan.title-required"]);
  });

  test("flags empty statements", () => {
    expect(
      getCreatePlanBlockingReasons({
        title: "Add column",
        emptySpecCount: 2,
        t,
      })
    ).toEqual(["plan.navigator.statement-empty"]);
  });

  test("lists both blockers, title first", () => {
    expect(
      getCreatePlanBlockingReasons({ title: "", emptySpecCount: 1, t })
    ).toEqual(["plan.title-required", "plan.navigator.statement-empty"]);
  });

  test("returns no reasons when valid", () => {
    expect(
      getCreatePlanBlockingReasons({
        title: "Add column",
        emptySpecCount: 0,
        t,
      })
    ).toEqual([]);
  });
});
