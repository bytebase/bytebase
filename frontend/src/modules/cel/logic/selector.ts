import type { Factor } from "../types/factor";
import { getOperatorListByFactor, type Operator } from "../types/operator";
import {
  isConditionExpr,
  isConditionGroupExpr,
  type SimpleExpr,
} from "../types/simple";

export function canRenderAsSelector(
  expr: SimpleExpr,
  factors: Factor[],
  overrideMap?: Map<Factor, Operator[]>
): boolean {
  if (isConditionGroupExpr(expr)) {
    return expr.args.every((arg) =>
      canRenderAsSelector(arg, factors, overrideMap)
    );
  }
  if (!isConditionExpr(expr)) return false;
  const factor = expr.args[0];
  return (
    factors.includes(factor) &&
    (overrideMap?.get(factor) ?? getOperatorListByFactor(factor)).includes(
      expr.operator
    )
  );
}
