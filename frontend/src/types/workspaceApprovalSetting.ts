import type {
  Expr,
  ParsedExpr,
} from "@/types/proto/google/api/expr/v1alpha1/syntax";
import type { Risk_Source } from "@/types/proto/v1/risk_service";
import type { ApprovalTemplate } from "./proto/store/approval";

export type LocalApprovalRule = {
  uid: string;
  expression?: ParsedExpr;
  template: ApprovalTemplate;
};

export type ParsedApprovalRule = {
  source: Risk_Source;
  level: number;
  rule: string; // LocalApprovalRule.uid
};

export type UnrecognizedApprovalRule = {
  expr?: Expr;
  rule: string; // LocalApprovalRule.uid
};

export type LocalApprovalConfig = {
  rules: LocalApprovalRule[];
  parsed: ParsedApprovalRule[];
  unrecognized: UnrecognizedApprovalRule[];
};
