import type { SimpleExpr } from "@/plugins/cel";
import type { ApprovalTemplate } from "@/types/proto/v1/issue_service";
import type { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";

export type LocalApprovalRule = {
  uid: string;
  expr?: SimpleExpr;
  template: ApprovalTemplate;
};

export type ParsedApprovalRule = {
  source: Risk_Source;
  level: number;
  rule: string; // LocalApprovalRule.uid
};

export type UnrecognizedApprovalRule = {
  expr?: SimpleExpr;
  rule: string; // LocalApprovalRule.uid
};

export type LocalApprovalConfig = {
  rules: LocalApprovalRule[];
  parsed: ParsedApprovalRule[];
  unrecognized: UnrecognizedApprovalRule[];
};
