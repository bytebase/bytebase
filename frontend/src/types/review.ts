import { type User } from "@/types/proto/v1/auth_service";
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
  approver: User | undefined;
  candidates: User[];
};

export const emptyFlow = (): ReviewFlow => {
  return {
    template: ApprovalTemplate.fromJSON({}),
    approvers: [],
    currentStepIndex: -1,
  };
};
