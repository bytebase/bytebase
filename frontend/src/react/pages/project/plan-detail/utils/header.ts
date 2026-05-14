import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

type T = (key: string, options?: Record<string, unknown>) => string;

export const shouldStayOnPlanDetailPage = (plan: Plan): boolean => {
  if (plan.specs.length === 0) {
    return true;
  }

  return !plan.specs.every(
    (spec) =>
      spec.config?.case === "createDatabaseConfig" ||
      spec.config?.case === "exportDataConfig"
  );
};

export const hasChecksWarning = (plan: Plan): boolean => {
  const statusCount = plan.planCheckRunStatusCount || {};
  const hasError =
    (statusCount.ERROR ?? 0) > 0 || (statusCount.FAILED ?? 0) > 0;
  return (
    hasError &&
    plan.specs.length > 0 &&
    plan.specs.every((spec) => spec.config?.case === "changeDatabaseConfig")
  );
};

export const getCreateIssueBlockingErrors = ({
  emptySpecCount,
  plan,
  project,
  t,
}: {
  emptySpecCount: number;
  plan: Plan;
  project: Pick<Project, "enforceSqlReview">;
  t: T;
}): string[] => {
  const errors: string[] = [];
  const statusCount = plan.planCheckRunStatusCount || {};

  if (emptySpecCount > 0) {
    errors.push(t("plan.navigator.statement-empty"));
  }
  if ((statusCount.RUNNING ?? 0) > 0) {
    errors.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-are-still-running"
      )
    );
  }
  if (
    ((statusCount.ERROR ?? 0) > 0 || (statusCount.FAILED ?? 0) > 0) &&
    project.enforceSqlReview
  ) {
    errors.push(
      t(
        "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
      )
    );
  }

  return errors;
};

export const getCreateIssueConfirmErrors = ({
  blockingErrors,
  project,
  selectedLabelCount,
  t,
}: {
  blockingErrors: string[];
  project: Pick<Project, "forceIssueLabels">;
  selectedLabelCount: number;
  t: T;
}): string[] => {
  const errors = [...blockingErrors];

  if (project.forceIssueLabels && selectedLabelCount === 0) {
    errors.push(t("plan.labels-required-for-review"));
  }

  return errors;
};
