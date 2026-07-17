import { create } from "@bufbuild/protobuf";
import type {
  CreateIssueRequest,
  Issue,
  UpdateIssueRequest,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  IssueStatus,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type {
  CreatePlanRequest,
  Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import { CreatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

type T = (key: string, options?: Record<string, unknown>) => string;

export class DraftReviewIssueCreationError extends Error {
  readonly plan: Plan;
  override readonly cause: unknown;

  constructor(plan: Plan, cause: unknown) {
    super("The plan was created, but its Draft Review Issue was not created");
    this.name = "DraftReviewIssueCreationError";
    this.plan = plan;
    this.cause = cause;
  }
}

export async function createPlanWithDraftReview({
  createIssue,
  createPlan,
  creator,
  labels,
  parent,
  plan,
}: {
  createIssue: (request: CreateIssueRequest) => Promise<Issue>;
  createPlan: (request: CreatePlanRequest) => Promise<Plan>;
  creator: string;
  labels: string[];
  parent: string;
  plan: Plan;
}): Promise<{ issue: Issue; plan: Plan }> {
  const createdPlan = await createPlan(
    create(CreatePlanRequestSchema, { parent, plan })
  );

  try {
    const issue = await createIssue(
      create(CreateIssueRequestSchema, {
        parent,
        issue: create(IssueSchema, {
          creator,
          description: createdPlan.description,
          draft: true,
          labels,
          plan: createdPlan.name,
          status: IssueStatus.OPEN,
          title: createdPlan.title,
          type: Issue_Type.DATABASE_CHANGE,
        }),
      })
    );
    return { issue, plan: createdPlan };
  } catch (error) {
    throw new DraftReviewIssueCreationError(createdPlan, error);
  }
}

export async function submitDraftReview({
  issue,
  labels,
  updateIssue,
}: {
  issue: Issue;
  labels: string[];
  updateIssue: (request: UpdateIssueRequest) => Promise<Issue>;
}): Promise<Issue> {
  return updateIssue(
    create(UpdateIssueRequestSchema, {
      issue: create(IssueSchema, {
        ...issue,
        draft: false,
        labels,
      }),
      updateMask: { paths: ["draft", "labels"] },
    })
  );
}

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

export const getCreatePlanBlockingReasons = ({
  title,
  emptySpecCount,
  t,
}: {
  title: string;
  emptySpecCount: number;
  t: T;
}): string[] => {
  const reasons: string[] = [];
  if (!title.trim()) {
    reasons.push(t("plan.title-required"));
  }
  if (emptySpecCount > 0) {
    reasons.push(t("plan.navigator.statement-empty"));
  }
  return reasons;
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
  if (plan.specs.some((spec) => spec.config?.case === "exportDataConfig")) {
    errors.push(t("issue.data-export.creation-not-supported"));
  }
  if ((statusCount.AVAILABLE ?? 0) > 0 || (statusCount.RUNNING ?? 0) > 0) {
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
