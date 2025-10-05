import type { ApprovalNode } from "@/types/proto-es/v1/issue_service_pb";
import { ApprovalNode_Type } from "@/types/proto-es/v1/issue_service_pb";
import { displayRoleTitle } from "@/utils";

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
