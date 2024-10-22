import { issueServiceClient, rolloutServiceClient } from "@/grpcweb";
import { useCurrentUserV1 } from "@/store";
import { emptyIssue, type ComposedIssue } from "@/types";
import { Issue, Issue_Type, IssueStatus } from "@/types/proto/v1/issue_service";
import type { Plan } from "@/types/proto/v1/plan_service";

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
  const createdRollout = await rolloutServiceClient.createRollout({
    parent: project,
    rollout: {
      plan: plan.name,
    },
  });
  const composedIssue: ComposedIssue = {
    ...emptyIssue(),
    ...createdIssue,
    planEntity: plan,
    rollout: createdRollout.name,
    rolloutEntity: createdRollout,
  };
  return composedIssue;
};
