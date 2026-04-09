import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_ApprovalStatus } from "@/types/proto-es/v1/issue_service_pb";

export const hasNoApprovalRequired = (issue: Issue | undefined): boolean => {
  if (!issue) return false;
  if (issue.approvalStatus === Issue_ApprovalStatus.CHECKING) return false;
  const roles = issue.approvalTemplate?.flow?.roles ?? [];
  return roles.length === 0;
};

export const isApprovalCompleted = (issue: Issue | undefined): boolean => {
  if (!issue) return false;

  return (
    issue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
    issue.approvalStatus === Issue_ApprovalStatus.SKIPPED ||
    hasNoApprovalRequired(issue)
  );
};
