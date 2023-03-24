import { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { WorkspaceApprovalSetting_Rule as ApprovalRule } from "@/types/proto/store/setting";
import { Risk_Source } from "@/types/proto/v1/risk_service";

export type ParsedApprovalRule = {
  source: Risk_Source;
  level: number;
  rule: ApprovalRule;
};

export type UnrecognizedApprovalRule = {
  expr?: Expr;
  rule: ApprovalRule;
};
