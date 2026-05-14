import { describe, expect, test } from "vitest";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  getCreateIssueBlockingErrors,
  getCreateIssueConfirmErrors,
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
