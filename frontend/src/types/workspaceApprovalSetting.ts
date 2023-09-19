import { SimpleExpr } from "@/plugins/cel";
import { ApprovalTemplate } from "@/types/proto/v1/issue_service";
import type { Risk_Source } from "@/types/proto/v1/risk_service";

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
