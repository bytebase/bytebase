import { create } from "@bufbuild/protobuf";
import { issueServiceClientConnect, rolloutServiceClientConnect } from "@/grpcweb";
import { CreateIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { convertNewIssueToOld, convertOldIssueToNew } from "@/utils/v1/issue-conversions";
import { useCurrentUserV1 } from "@/store";
import { emptyIssue, type ComposedIssue } from "@/types";
import { Issue, Issue_Type, IssueStatus } from "@/types/proto/v1/issue_service";
import type { Plan } from "@/types/proto/v1/plan_service";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { convertNewRolloutToOld } from "@/utils/v1/rollout-conversions";

export const createIssueFromPlan = async (project: string, plan: Plan) => {
  const me = useCurrentUserV1();
  const issuePartial = Issue.fromPartial({
    plan: plan.name,
    creator: `users/${me.value.email}`,
    title: plan.title,
    description: plan.description,
    status: IssueStatus.OPEN,
    type: Issue_Type.DATABASE_CHANGE,
  });
  const newIssue = convertOldIssueToNew(issuePartial);
  const request = create(CreateIssueRequestSchema, {
    parent: project,
    issue: newIssue,
  });
  const newCreatedIssue = await issueServiceClientConnect.createIssue(request);
  const createdIssue = convertNewIssueToOld(newCreatedIssue);
  const rolloutRequest = create(CreateRolloutRequestSchema, {
    parent: project,
    rollout: {
      plan: plan.name,
    },
  });
  const createdRolloutNew = await rolloutServiceClientConnect.createRollout(rolloutRequest);
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
