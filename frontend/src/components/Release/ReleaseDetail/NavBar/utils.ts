import { create } from "@bufbuild/protobuf";
import { issueServiceClient, rolloutServiceClientConnect } from "@/grpcweb";
import { useCurrentUserV1 } from "@/store";
import { emptyIssue, type ComposedIssue } from "@/types";
import { Issue, Issue_Type, IssueStatus } from "@/types/proto/v1/issue_service";
import type { Plan } from "@/types/proto/v1/plan_service";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { convertNewRolloutToOld } from "@/utils/v1/rollout-conversions";

export const createIssueFromPlan = async (project: string, plan: Plan) => {
  const me = useCurrentUserV1();
  const createdIssue = await issueServiceClient.createIssue({
    parent: project,
    issue: Issue.fromPartial({
      plan: plan.name,
      creator: `users/${me.value.email}`,
      title: plan.title,
      description: plan.description,
      status: IssueStatus.OPEN,
      type: Issue_Type.DATABASE_CHANGE,
    }),
  });
  const request = create(CreateRolloutRequestSchema, {
    parent: project,
    rollout: {
      plan: plan.name,
    },
  });
  const createdRolloutNew = await rolloutServiceClientConnect.createRollout(request);
  const createdRollout = convertNewRolloutToOld(createdRolloutNew);
  const composedIssue: ComposedIssue = {
    ...emptyIssue(),
    ...createdIssue,
    planEntity: plan,
    rollout: createdRollout.name,
    rolloutEntity: createdRollout,
  };
  return composedIssue;
};
