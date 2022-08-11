import { IssueType, PipelineApprovalPolicyPayload, Policy } from "@/types";

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
