import { computed } from "vue";
import {
  EnvironmentId,
  IssueCreate,
  IssueType,
  Pipeline,
  PipelineApprovalPolicyPayload,
  Policy,
} from "@/types";
import { useIssueLogic } from ".";
import { usePolicyByEnvironmentAndType } from "@/store";

export const useAllowProjectOwnerToApprove = () => {
  const { create, issue, activeStageOfPipeline } = useIssueLogic();

  const activeEnvironmentId = computed((): EnvironmentId => {
    if (create.value) {
      // When creating an issue, activeEnvironmentId is the first stage's environmentId
      const stage = (issue.value as IssueCreate).pipeline!.stageList[0];
      return stage.environmentId;
    }

    const stage = activeStageOfPipeline(issue.value.pipeline as Pipeline);
    return stage.environment.id;
  });

  const activeEnvironmentApprovalPolicy = usePolicyByEnvironmentAndType(
    computed(() => ({
      environmentId: activeEnvironmentId.value,
      type: "bb.policy.pipeline-approval",
    }))
  );

  const allowProjectOwnerAsAssignee = computed((): boolean => {
    const policy = activeEnvironmentApprovalPolicy.value;
    if (!policy) return false;

    return allowProjectOwnerToApprove(policy, issue.value.type);
  });

  return allowProjectOwnerAsAssignee;
};

export const allowProjectOwnerToApprove = (
  policy: Policy,
  issueType: IssueType
): boolean => {
  const payload = policy.payload as PipelineApprovalPolicyPayload;
  if (payload.value === "MANUAL_APPROVAL_NEVER") {
    return false;
  }

  const assigneeGroup = payload.assigneeGroupList.find(
    (group) => group.issueType === issueType
  );

  if (!assigneeGroup) {
    return false;
  }

  return assigneeGroup.value === "PROJECT_OWNER";
};
