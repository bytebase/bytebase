import type { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import type {
  StringFactor,
  Operator,
  CollectionOperator,
  CompareOperator,
  StringOperator,
  EqualityOperator,
  SimpleExpr,
  CollectionExpr,
  ConditionGroupExpr,
  CompareExpr,
  EqualityExpr,
  StringExpr,
  LogicalOperator,
  RawStringExpr,
} from "../types";
import {
  isEqualityOperator,
  isConditionGroupExpr,
  isCollectionOperator,
  isLogicalOperator,
  isCompareOperator,
  isStringOperator,
  isNumberFactor,
  isStringFactor,
  isTimestampFactor,
  isNegativeOperator,
  ExprType,
} from "../types";
import stringifyExpr from "./stringify";

// For simplify UI implementation, the "root" condition need to be a group.
export const wrapAsGroup = (
  expr: SimpleExpr,
  operator: LogicalOperator = "_&&_"
): ConditionGroupExpr => {
  if (isConditionGroupExpr(expr)) return expr;
  return {
    type: ExprType.ConditionGroup,
    operator,
    args: [expr],
  };
};

// Convert common expr to simple expr
export const resolveCELExpr = (expr: CELExpr): SimpleExpr => {
  const dfs = (expr: CELExpr, negative: boolean = false): SimpleExpr => {
    const { callExpr } = expr;
    if (!callExpr) {
      // If no callExpr, we treat it as a raw string.
      return resolveRawStringExpr(expr);
    }

    try {
      const { args } = callExpr;
      const operator = callExpr.function as Operator;
      if (isLogicalOperator(operator)) {
        const group: ConditionGroupExpr = {
          type: ExprType.ConditionGroup,
          operator,
          args: [],
        };
        const [left, right] = args;
        const sub = (subTree: CELExpr, expand: boolean) => {
          const subExpr = dfs(subTree);
          if (
            expand &&
            isConditionGroupExpr(subExpr) &&
            subExpr.operator === operator
          ) {
            group.args.push(...subExpr.args);
          } else {
            group.args.push(subExpr);
          }
        };
        sub(left, true);
        sub(right, false);
        return group;
      }
      if (isNegativeOperator(operator)) {
        return dfs(args[0], true);
      }
      if (isEqualityOperator(operator)) {
        return resolveEqualityExpr(expr);
      }
      if (isCompareOperator(operator)) {
        return resolveCompareExpr(expr);
      }
      if (isStringOperator(operator)) {
        return resolveStringExpr(expr);
      }
      if (isCollectionOperator(operator)) {
        return resolveCollectionExpr(expr, negative);
      }
      throw new Error(`unsupported expr "${JSON.stringify(expr)}"`);
    } catch {
      // Any error occurs, we treat it as a raw string.
      return resolveRawStringExpr(expr);
    }
  };
  return dfs(expr);
};

const resolveEqualityExpr = (expr: CELExpr): EqualityExpr => {
  const operator = expr.callExpr!.function as EqualityOperator;
  const [factorExpr, valueExpr] = expr.callExpr!.args;
  const factor = getFactorName(factorExpr);
  if (isNumberFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [factor, valueExpr.constExpr!.int64Value!.toNumber() ?? 0],
    };
  }
  if (isStringFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [factor, valueExpr.constExpr!.stringValue! ?? ""],
    };
  }
  throw new Error(`cannot resolve expr ${JSON.stringify(expr)}`);
};

const resolveCompareExpr = (expr: CELExpr): CompareExpr => {
  const operator = expr.callExpr!.function as CompareOperator;
  const [factorExpr, valueExpr] = expr.callExpr!.args;
  const factor = getFactorName(factorExpr);
  if (isNumberFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [factor, valueExpr.constExpr!.int64Value!.toNumber()],
    };
  }
  if (isTimestampFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [
        factor,
        new Date(valueExpr.callExpr!.args[0].constExpr!.stringValue!),
      ],
    };
  }
  throw new Error(`cannot resolve expr ${JSON.stringify(expr)}`);
};

const resolveStringExpr = (expr: CELExpr): StringExpr => {
  const operator = expr.callExpr!.function as StringOperator;
  const factor = getFactorName(expr.callExpr!.target!);
  const value = expr.callExpr!.args[0];
  return {
    type: ExprType.Condition,
    operator,
    args: [factor as StringFactor, value.constExpr!.stringValue!],
  };
};

const resolveCollectionExpr = (
  expr: CELExpr,
  negative: boolean = false
): CollectionExpr => {
  let operator = expr.callExpr!.function as CollectionOperator;
  if (negative) {
    operator = "@not_in";
  }
  const [factorExpr, valuesExpr] = expr.callExpr!.args;
  const factor = getFactorName(factorExpr);

  if (isNumberFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [
        factor,
        valuesExpr.listExpr?.elements?.map(
          (constant) => constant.constExpr?.int64Value?.toNumber() ?? 0
        ) ?? [],
      ],
    };
  }
  if (isStringFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [
        factor,
        valuesExpr.listExpr?.elements?.map(
          (constant) => constant.constExpr?.stringValue ?? ""
        ) ?? [],
      ],
    };
  }
  throw new Error(`cannot resolve expr ${JSON.stringify(expr)}`);
};

const resolveRawStringExpr = (expr: CELExpr): RawStringExpr => {
  return {
    type: ExprType.RawString,
    content: stringifyExpr(expr),
  };
};

export const emptySimpleExpr = (
  operator: LogicalOperator = "_&&_"
): ConditionGroupExpr => {
  return {
    type: ExprType.ConditionGroup,
    operator: operator,
    args: [],
  };
};

const getFactorName = (expr: CELExpr): string => {
  if (expr.identExpr !== undefined) {
    return expr.identExpr.name;
  } else if (expr.selectExpr !== undefined) {
    return `${expr.selectExpr.operand!.identExpr!.name!}.${expr.selectExpr
      .field!}`;
  }
  throw new Error(`cannot resolve factor name ${JSON.stringify(expr)}`);
};
