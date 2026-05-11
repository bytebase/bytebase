import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";

const hasNoApprovalRequired = (issue: Issue | undefined): boolean => {
  if (!issue) return false;
  if (issue.approvalStatus === ApprovalStatus.CHECKING) return false;
  const roles = issue.approvalTemplate?.flow?.roles ?? [];
  return roles.length === 0;
};

export const isApprovalCompleted = (issue: Issue | undefined): boolean => {
  if (!issue) return false;

  return (
    issue.approvalStatus === ApprovalStatus.APPROVED ||
    issue.approvalStatus === ApprovalStatus.SKIPPED ||
    hasNoApprovalRequired(issue)
  );
};
