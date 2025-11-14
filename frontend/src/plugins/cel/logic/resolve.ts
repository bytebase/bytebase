import type { Expr as CELExpr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type {
  CollectionExpr,
  CollectionOperator,
  CompareExpr,
  CompareOperator,
  ConditionGroupExpr,
  EqualityExpr,
  EqualityOperator,
  LogicalOperator,
  Operator,
  RawStringExpr,
  SimpleExpr,
  StringExpr,
  StringFactor,
  StringOperator,
} from "../types";
import {
  ExprType,
  isCollectionOperator,
  isCompareOperator,
  isConditionGroupExpr,
  isEqualityOperator,
  isLogicalOperator,
  isNegativeOperator,
  isNumberFactor,
  isStringFactor,
  isStringOperator,
  isTimestampFactor,
} from "../types";
import stringifyExpr from "./stringify";

// Helper functions to extract constant values from proto-es oneof patterns
const getConstantInt64Value = (expr: CELExpr): number => {
  if (
    expr.exprKind?.case === "constExpr" &&
    expr.exprKind.value.constantKind?.case === "int64Value"
  ) {
    return Number(expr.exprKind.value.constantKind.value);
  }
  return 0;
};

const getConstantStringValue = (expr: CELExpr): string => {
  if (
    expr.exprKind?.case === "constExpr" &&
    expr.exprKind.value.constantKind?.case === "stringValue"
  ) {
    return expr.exprKind.value.constantKind.value;
  }
  return "";
};

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
    if (expr.exprKind?.case !== "callExpr") {
      // If no callExpr, we treat it as a raw string.
      return resolveRawStringExpr(expr);
    }
    const callExpr = expr.exprKind.value;

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
        return resolveStringExpr(expr, negative);
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
  const callExpr =
    expr.exprKind?.case === "callExpr" ? expr.exprKind.value : null;
  if (!callExpr)
    throw new Error(`Expected callExpr but got ${expr.exprKind?.case}`);
  const operator = callExpr.function as EqualityOperator;
  const [factorExpr, valueExpr] = callExpr.args;
  const factor = getFactorName(factorExpr);
  if (isNumberFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [factor, getConstantInt64Value(valueExpr)],
    };
  }
  if (isStringFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [factor, getConstantStringValue(valueExpr)],
    };
  }
  throw new Error(`cannot resolve expr ${JSON.stringify(expr)}`);
};

const resolveCompareExpr = (expr: CELExpr): CompareExpr => {
  const callExpr =
    expr.exprKind?.case === "callExpr" ? expr.exprKind.value : null;
  if (!callExpr)
    throw new Error(`Expected callExpr but got ${expr.exprKind?.case}`);
  const operator = callExpr.function as CompareOperator;
  const [factorExpr, valueExpr] = callExpr.args;
  const factor = getFactorName(factorExpr);
  if (isNumberFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [factor, getConstantInt64Value(valueExpr)],
    };
  }
  if (isTimestampFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [
        factor,
        valueExpr.exprKind?.case === "callExpr" &&
        valueExpr.exprKind.value.args[0]?.exprKind?.case === "constExpr"
          ? new Date(getConstantStringValue(valueExpr.exprKind.value.args[0]))
          : new Date(),
      ],
    };
  }
  throw new Error(`cannot resolve expr ${JSON.stringify(expr)}`);
};

const resolveStringExpr = (
  expr: CELExpr,
  negative: boolean = false
): StringExpr => {
  const callExpr =
    expr.exprKind?.case === "callExpr" ? expr.exprKind.value : null;
  if (!callExpr)
    throw new Error(`Expected callExpr but got ${expr.exprKind?.case}`);
  let operator = callExpr.function as StringOperator;
  if (negative && operator == "contains") {
    operator = "@not_contains";
  }
  const factor = getFactorName(callExpr.target!);
  const value = callExpr.args[0];
  return {
    type: ExprType.Condition,
    operator,
    args: [factor as StringFactor, getConstantStringValue(value)],
  };
};

const resolveCollectionExpr = (
  expr: CELExpr,
  negative: boolean = false
): CollectionExpr => {
  const callExpr =
    expr.exprKind?.case === "callExpr" ? expr.exprKind.value : null;
  if (!callExpr)
    throw new Error(`Expected callExpr but got ${expr.exprKind?.case}`);
  let operator = callExpr.function as CollectionOperator;
  if (negative && operator == "@in") {
    operator = "@not_in";
  }
  const [factorExpr, valuesExpr] = callExpr.args;
  const factor = getFactorName(factorExpr);

  if (isNumberFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [
        factor,
        valuesExpr.exprKind?.case === "listExpr"
          ? (valuesExpr.exprKind.value.elements?.map(getConstantInt64Value) ??
            [])
          : [],
      ],
    };
  }
  if (isStringFactor(factor)) {
    return {
      type: ExprType.Condition,
      operator,
      args: [
        factor,
        valuesExpr.exprKind?.case === "listExpr"
          ? (valuesExpr.exprKind.value.elements?.map(getConstantStringValue) ??
            [])
          : [],
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
  if (expr.exprKind?.case === "identExpr") {
    return expr.exprKind.value.name;
  } else if (expr.exprKind?.case === "selectExpr") {
    const selectExpr = expr.exprKind.value;
    const operandName =
      selectExpr.operand?.exprKind?.case === "identExpr"
        ? selectExpr.operand.exprKind.value.name
        : "";
    return `${operandName}.${selectExpr.field}`;
  }
  throw new Error(`cannot resolve factor name ${JSON.stringify(expr)}`);
};
