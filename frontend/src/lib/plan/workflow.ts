import { create } from "@bufbuild/protobuf";
import { IssueStatus } from "@/types/proto-es/v1/common_pb";
import type {
  CreateIssueRequest,
  Issue,
  UpdateIssueRequest,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type {
  CreatePlanRequest,
  Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import { CreatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";

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

// Submitting only flips the draft flag. Labels are owned by the plan metadata
// row, which persists them on change — the submit path must not write them back
// and silently undo an edit made while the header was open.
export async function submitDraftReview({
  issue,
  updateIssue,
}: {
  issue: Issue;
  updateIssue: (request: UpdateIssueRequest) => Promise<Issue>;
}): Promise<Issue> {
  return updateIssue(
    create(UpdateIssueRequestSchema, {
      issue: create(IssueSchema, {
        ...issue,
        draft: false,
      }),
      updateMask: { paths: ["draft"] },
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
