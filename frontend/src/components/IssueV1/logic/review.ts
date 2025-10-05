import { computed, unref } from "vue";
import type { ComputedRef } from "vue";
import type { ReviewFlow, MaybeRef } from "@/types";
import { emptyFlow } from "@/types";
import type { ApprovalNode, Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  ApprovalNode_Type,
  Issue_Approver_Status,
  Issue_ApprovalStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { displayRoleTitle } from "@/utils";

export const extractReviewContext = (
  issue: MaybeRef<Issue>
): ComputedRef<ReviewFlow> => {
  return computed((): ReviewFlow => {
    const issueValue = unref(issue);
    const approvalStatus = issueValue.approvalStatus;
    if (
      approvalStatus === Issue_ApprovalStatus.CHECKING ||
      approvalStatus === Issue_ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED
    ) {
      return emptyFlow();
    }

    const { approvalTemplates, approvers } = issueValue;
    if (approvalTemplates.length === 0) return emptyFlow();

    const rejectedIndex = approvers.findIndex(
      (ap) => ap.status === Issue_Approver_Status.REJECTED
    );
    const currentStepIndex =
      rejectedIndex >= 0 ? rejectedIndex : approvers.length;

    return {
      template: approvalTemplates[0],
      approvers,
      currentStepIndex,
    };
  });
};

export const displayReviewRoleTitle = (node: ApprovalNode) => {
  const { type, role } = node;
  if (type !== ApprovalNode_Type.ANY_IN_GROUP) {
    return "";
  }

  if (role) {
    return displayRoleTitle(role);
  }
  return "";
};
