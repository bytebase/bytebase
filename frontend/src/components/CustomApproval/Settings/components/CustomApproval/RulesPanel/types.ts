import type { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import type { Risk_Source } from "@/types/proto/v1/risk_service";
import type { WorkspaceApprovalSetting_Rule as ApprovalRule } from "@/types/proto/v1/setting_service";

export type ParsedApprovalRule = {
  source: Risk_Source;
  level: number;
  rule: ApprovalRule;
};

export type UnrecognizedApprovalRule = {
  expr?: Expr;
  rule: ApprovalRule;
};
