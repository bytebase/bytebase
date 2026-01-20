import { create } from "@bufbuild/protobuf";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { CreateIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { CreatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";

export interface CreateIssueByPlanOptions {
  skipRollout?: boolean;
}

export const experimentalCreateIssueByPlan = async (
  project: Project,
  issueCreate: Issue,
  planCreate: Plan,
  options?: CreateIssueByPlanOptions
) => {
  const newPlan = planCreate;
  const request = create(CreatePlanRequestSchema, {
    parent: project.name,
    plan: newPlan,
  });
  const response = await planServiceClientConnect.createPlan(request);
  const createdPlan = response;
  issueCreate.plan = createdPlan.name;

  const issueRequest = create(CreateIssueRequestSchema, {
    parent: project.name,
    issue: issueCreate,
  });
  const newCreatedIssue =
    await issueServiceClientConnect.createIssue(issueRequest);
  const createdIssue = newCreatedIssue;

  // Skip rollout creation for plans that create rollout on-demand (e.g., database creation)
  if (options?.skipRollout) {
    return { createdPlan, createdIssue, createdRollout: undefined };
  }

  const rolloutRequest = create(CreateRolloutRequestSchema, {
    parent: createdPlan.name,
  });
  const rolloutResponse =
    await rolloutServiceClientConnect.createRollout(rolloutRequest);
  const createdRollout = rolloutResponse;

  return { createdPlan, createdIssue, createdRollout };
};
