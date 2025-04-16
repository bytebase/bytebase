import type {
  ApprovalFlow,
  ApprovalTemplate,
} from "@/types/proto/v1/issue_service";
import {
  ApprovalNode_Type,
  ApprovalStep_Type,
} from "@/types/proto/v1/issue_service";

const validateApprovalFlow = (flow: ApprovalFlow) => {
  const SupportedStepTypes = new Set([
    ApprovalStep_Type.ALL,
    ApprovalStep_Type.ANY,
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
      const { type, role } = node;
      if (type !== ApprovalNode_Type.ANY_IN_GROUP) {
        return false;
      }
      return !!role;
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
