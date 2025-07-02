import type { ApprovalStep, Issue_Approver } from "./proto-es/v1/issue_service_pb";
import type { ApprovalTemplate } from "./proto-es/v1/issue_service_pb";

export type ReviewFlow = {
  template: ApprovalTemplate;
  approvers: Issue_Approver[];
  currentStepIndex: number; // -1 if finished
};

export type WrappedReviewStep = {
  index: number;
  step: ApprovalStep;
  status: "APPROVED" | "REJECTED" | "CURRENT" | "PENDING";
  approver: string | undefined;
  candidates: string[];
};

export const emptyFlow = (): ReviewFlow => {
  return {
    template: ApprovalTemplate.fromPartial({}),
    approvers: [],
    currentStepIndex: -1,
  };
};
