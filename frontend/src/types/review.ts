import type { ApprovalStep, Issue_Approver } from "./proto/v1/issue_service";
import { ApprovalTemplate } from "./proto/v1/issue_service";

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
