import { create } from "@bufbuild/protobuf";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { useCurrentUserV1 } from "@/store";
import { type ComposedIssue, emptyIssue } from "@/types";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";

export const createIssueFromPlan = async (project: string, plan: Plan) => {
  const me = useCurrentUserV1();
  const issue = create(IssueSchema, {
    plan: plan.name,
    creator: `users/${me.value.email}`,
    title: plan.title,
    description: plan.description,
    status: IssueStatus.OPEN,
    type: Issue_Type.DATABASE_CHANGE,
  });
  const request = create(CreateIssueRequestSchema, {
    parent: project,
    issue: issue,
  });
  const createdIssue = await issueServiceClientConnect.createIssue(request);
  const rolloutRequest = create(CreateRolloutRequestSchema, {
    parent: plan.name,
  });
  const createdRollout =
    await rolloutServiceClientConnect.createRollout(rolloutRequest);
  const composedIssue: ComposedIssue = {
    ...emptyIssue(),
    ...createdIssue,
    planEntity: plan,
    rolloutEntity: createdRollout,
  };
  return composedIssue;
};
