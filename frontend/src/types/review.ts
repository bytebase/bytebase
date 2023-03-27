import type {
  ApprovalTemplate,
  Review_Approver,
} from "./proto/v1/review_service";

export type ReviewFlow = {
  template: ApprovalTemplate;
  approvers: Review_Approver[];
  currentStepIndex: number; // -1 if finished
};
