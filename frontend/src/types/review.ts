import type { User } from "./proto/v1/auth_service";
import type {
  ApprovalStep,
  ApprovalTemplate,
  Review_Approver,
} from "./proto/v1/review_service";

export type ApprovalEvent = {
  status: "APPROVED" | "REJECTED" | "PENDING";
};

export type ReviewFlow = {
  template: ApprovalTemplate;
  approvers: Review_Approver[];
  currentStepIndex: number; // -1 if finished
};

export type WrappedReviewStep = {
  index: number;
  step: ApprovalStep;
  status: "APPROVED" | "REJECTED" | "CURRENT" | "PENDING";
  approver: User | undefined;
  candidates: User[];
};
