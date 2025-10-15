import type { SimpleExpr } from "@/plugins/cel";
import type { RiskLevel } from "@/types/proto-es/v1/common_pb";
import type { ApprovalTemplate } from "@/types/proto-es/v1/issue_service_pb";
import type { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";

export type LocalApprovalRule = {
  // Use template.id as the identifier instead of generating ephemeral uid
  // This allows proper identification of built-in vs custom flows
  expr?: SimpleExpr;
  template: ApprovalTemplate;
};

export type ParsedApprovalRule = {
  source: Risk_Source;
  level: RiskLevel;
  rule: string; // ApprovalTemplate.id
};

export type UnrecognizedApprovalRule = {
  expr?: SimpleExpr;
  rule: string; // ApprovalTemplate.id
};

export type LocalApprovalConfig = {
  rules: LocalApprovalRule[];
  parsed: ParsedApprovalRule[];
  unrecognized: UnrecognizedApprovalRule[];
};
