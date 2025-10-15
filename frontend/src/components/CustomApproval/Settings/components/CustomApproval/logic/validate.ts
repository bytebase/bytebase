import type {
  ApprovalFlow,
  ApprovalTemplate,
} from "@/types/proto-es/v1/issue_service_pb";

const validateApprovalFlow = (flow: ApprovalFlow) => {
  if (flow.roles.length === 0) {
    return false;
  }

  return flow.roles.every((role) => !!role);
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
