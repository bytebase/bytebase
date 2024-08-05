import type { ApprovalStep, Issue_Approver } from "./proto/v1/issue_service";
import { ApprovalTemplate } from "./proto/v1/issue_service";
import type { ComposedUser } from "./v1/user";

export type ReviewFlow = {
  template: ApprovalTemplate;
  approvers: Issue_Approver[];
  currentStepIndex: number; // -1 if finished
};

export type WrappedReviewStep = {
  index: number;
  step: ApprovalStep;
  status: "APPROVED" | "REJECTED" | "CURRENT" | "PENDING";
  approver: ComposedUser | undefined;
  candidates: ComposedUser[];
};

export const emptyFlow = (): ReviewFlow => {
  return {
    template: ApprovalTemplate.fromJSON({}),
    approvers: [],
    currentStepIndex: -1,
  };
};
