import {
  ApprovalFlow,
  ApprovalNode_GroupValue,
  ApprovalNode_Type,
  ApprovalStep_Type,
  ApprovalTemplate,
} from "@/types/proto/v1/issue_service";

const validateApprovalFlow = (flow: ApprovalFlow) => {
  const SupportedStepTypes = new Set([
    ApprovalStep_Type.ALL,
    ApprovalStep_Type.ANY,
  ]);
  const SupportedGroupValues = new Set([
    ApprovalNode_GroupValue.PROJECT_MEMBER,
    ApprovalNode_GroupValue.PROJECT_OWNER,
    ApprovalNode_GroupValue.WORKSPACE_DBA,
    ApprovalNode_GroupValue.WORKSPACE_OWNER,
  ]);

  if (flow.steps.length === 0) {
    return false;
  }

  return flow.steps.every((step) => {
    const { type, nodes } = step;
    if (!SupportedStepTypes.has(type)) {
      return false;
    }
    return nodes.every((node) => {
      const { type, groupValue, role, externalNodeId } = node;
      if (type !== ApprovalNode_Type.ANY_IN_GROUP) {
        return false;
      }
      if (!groupValue) {
        if (!role && !externalNodeId) {
          return false;
        }
      } else {
        if (!SupportedGroupValues.has(groupValue)) {
          return false;
        }
      }
      return true;
    });
  });
};

export const validateApprovalTemplate = (
  template: Omit<ApprovalTemplate, "creator">
) => {
  const { title = "", description = "", flow } = template;
  if (title.trim().length === 0) return false;
  if (description.trim().length === 0) return false;
  if (!flow) return false;
  return validateApprovalFlow(flow);
};
