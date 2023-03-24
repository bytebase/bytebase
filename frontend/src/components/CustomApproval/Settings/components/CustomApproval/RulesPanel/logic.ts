import { PresetRiskLevelList, SupportedSourceList } from "@/types";
import type { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import type {
  WorkspaceApprovalSetting,
  WorkspaceApprovalSetting_Rule as ApprovalRule,
} from "@/types/proto/store/setting";
import type { ParsedApprovalRule, UnrecognizedApprovalRule } from "./types";
import { Risk_Source } from "@/types/proto/v1/risk_service";

/*
  WorkspaceApprovalSetting is a list of ApprovalRule = {
    expr, 
    approval_template
  }
  Currently, every ApprovalRule.expr is a list of "OR" expr.
  And every "OR" expr has a fixed "source==xx && level=yy" form.

  So we walk through the list of ApprovalRule and find all combinations of
  ParsedApprovalRule = {
    source,
    level,
    rule,
  }
*/

const resolveIdentExpr = (expr: Expr): string => {
  return expr.identExpr?.name ?? "";
};

const resolveNumberExpr = (expr: Expr): number | undefined => {
  return expr.constExpr?.int64Value;
};

const resolveSourceExpr = (expr: Expr): Risk_Source => {
  const { function: operator, args } = expr.callExpr ?? {};
  if (operator !== "_==_") return Risk_Source.UNRECOGNIZED;
  if (!args || args.length !== 2) return Risk_Source.UNRECOGNIZED;
  const factor = resolveIdentExpr(args[0]);
  if (factor !== "source") return Risk_Source.UNRECOGNIZED;
  const source = resolveNumberExpr(args[1]);
  if (typeof source === "undefined") return Risk_Source.UNRECOGNIZED;
  if (!SupportedSourceList.includes(source)) return Risk_Source.UNRECOGNIZED;
  return source;
};

const resolveLevelExpr = (expr: Expr): number => {
  const { function: operator, args } = expr.callExpr ?? {};
  if (operator !== "_==_") return Number.NaN;
  if (!args || args.length !== 2) return Number.NaN;
  const factor = resolveIdentExpr(args[0]);
  if (factor !== "level") return Number.NaN;
  const level = resolveNumberExpr(args[1]);
  if (typeof level === "undefined") return Number.NaN;
  if (!PresetRiskLevelList.find((item) => item.level === level))
    return Number.NaN;
  return level;
};

export const resolveApprovalConfigRules = (
  config: WorkspaceApprovalSetting
) => {
  const parsed: ParsedApprovalRule[] = [];
  const unrecognized: UnrecognizedApprovalRule[] = [];

  const fail = (expr: Expr | undefined, rule: ApprovalRule) => {
    unrecognized.push({ expr, rule });
  };

  const resolveLogicAndExpr = (expr: Expr, rule: ApprovalRule) => {
    const { function: operator, args } = expr.callExpr ?? {};
    if (operator !== "_&&_") return fail(expr, rule);
    if (!args || args.length !== 2) return fail(expr, rule);
    const source = resolveSourceExpr(args[0]);
    if (source === Risk_Source.UNRECOGNIZED) return fail(expr, rule);
    const level = resolveLevelExpr(args[1]);
    if (Number.isNaN(level)) return fail(expr, rule);

    // Found a correct (source, level) combination
    parsed.push({
      source,
      level,
      rule,
    });
  };

  const resolveLogicOrExpr = (expr: Expr, rule: ApprovalRule) => {
    const { function: operator, args } = expr.callExpr ?? {};
    if (operator !== "_||_") return fail(expr, rule);
    if (!args || args.length === 0) return fail(expr, rule);

    for (let i = 0; i < args.length; i++) {
      resolveLogicAndExpr(args[i], rule);
    }
  };

  for (let i = 0; i < config.rules.length; i++) {
    const rule = config.rules[i];
    const expr = rule.expression?.expr;
    if (!expr) {
      fail(expr, rule);
      continue;
    }
    resolveLogicOrExpr(expr, rule);
  }

  return { parsed, unrecognized };
};
