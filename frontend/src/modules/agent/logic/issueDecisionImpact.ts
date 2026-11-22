import {
  getNameParentTokens,
  issueNamePrefix,
  planNamePrefix,
  projectNamePrefix,
} from "@/stores/modules/v1/common";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  type Issue,
  Issue_Approver_Status,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";

export type IssueDecisionImpact = {
  kind:
    | "reject"
    | "intermediate-approval"
    | "final-approval"
    | "final-approval-with-rollout"
    | "final-approval-with-access-grant"
    | "final-approval-with-role-grant"
    | "unknown";
};

const hasValidResourceNames = (issue: Issue): boolean => {
  const [projectId, issueId] = getNameParentTokens(issue.name, [
    projectNamePrefix,
    issueNamePrefix,
  ]);
  if (!projectId || !issueId) return false;
  if (!issue.plan) {
    return true;
  }
  const [planProjectId, planId] = getNameParentTokens(issue.plan, [
    projectNamePrefix,
    planNamePrefix,
  ]);
  return planProjectId === projectId && Boolean(planId);
};

export const getIssueDecisionImpact = (
  issue: Issue,
  action: "approve" | "reject"
): IssueDecisionImpact => {
  if (
    issue.status !== IssueStatus.OPEN ||
    issue.draft ||
    issue.approvalStatus !== ApprovalStatus.PENDING ||
    !hasValidResourceNames(issue) ||
    ![
      Issue_Type.DATABASE_CHANGE,
      Issue_Type.ROLE_GRANT,
      Issue_Type.ACCESS_GRANT,
    ].includes(issue.type)
  ) {
    return { kind: "unknown" };
  }

  const roles = issue.approvalTemplate?.flow?.roles;
  if (
    !roles?.length ||
    roles.some((role) => !role.trim()) ||
    issue.approvers.length >= roles.length ||
    issue.approvers.some(
      (approver) => approver.status !== Issue_Approver_Status.APPROVED
    )
  ) {
    return { kind: "unknown" };
  }

  if (action === "reject") {
    return { kind: "reject" };
  }

  if (issue.approvers.length < roles.length - 1) {
    return { kind: "intermediate-approval" };
  }

  if (issue.type === Issue_Type.DATABASE_CHANGE && issue.plan) {
    return { kind: "final-approval-with-rollout" };
  }

  if (issue.type === Issue_Type.ACCESS_GRANT) {
    return { kind: "final-approval-with-access-grant" };
  }

  if (issue.type === Issue_Type.ROLE_GRANT) {
    return { kind: "final-approval-with-role-grant" };
  }

  return { kind: "final-approval" };
};
