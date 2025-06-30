import type { Expr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { Risk_Source } from "@/types/proto-es/v1/risk_service_pb";
import type { WorkspaceApprovalSetting_Rule as ApprovalRule } from "@/types/proto-es/v1/setting_service_pb";

export type ParsedApprovalRule = {
  source: Risk_Source;
  level: number;
  rule: ApprovalRule;
};

export type UnrecognizedApprovalRule = {
  expr?: Expr;
  rule: ApprovalRule;
};
