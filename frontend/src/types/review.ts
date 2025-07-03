import { create as createProto } from "@bufbuild/protobuf";
import type {
  ApprovalStep,
  Issue_Approver,
  ApprovalTemplate,
} from "./proto-es/v1/issue_service_pb";
import { ApprovalTemplateSchema } from "./proto-es/v1/issue_service_pb";

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
    template: createProto(ApprovalTemplateSchema, {}),
    approvers: [],
    currentStepIndex: -1,
  };
};
