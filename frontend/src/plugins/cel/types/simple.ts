/// Define a simplified version (less nested) of CEL Expr.
/// Convenient for local editing.
import { NumberFactor, StringFactor, TimestampFactor } from "./factor";
import {
  type LogicalOperator,
  type EqualityOperator,
  type CompareOperator,
  type StringOperator,
  type CollectionOperator,
  isStringOperator,
  isCollectionOperator,
  isLogicalOperator,
  isCompareOperator,
  isEqualityOperator,
} from "./operator";

export type EqualityExpr = {
  operator: EqualityOperator;
  args: [StringFactor | NumberFactor, string | number];
};

export type CompareExpr = {
  operator: CompareOperator;
  args: [NumberFactor | TimestampFactor, number | Date];
};

export type CollectionExpr = {
  operator: CollectionOperator;
  args: [StringFactor | NumberFactor, string[] | number[]];
};

export type StringExpr = {
  operator: StringOperator;
  args: [StringFactor, string];
};

export type ConditionExpr =
  | EqualityExpr
  | CompareExpr
  | CollectionExpr
  | StringExpr;

export type LogicalExpr = {
  operator: LogicalOperator;
  args: (ConditionExpr | ConditionGroupExpr)[];
};

export type ConditionGroupExpr = LogicalExpr;

export type SimpleExpr = ConditionExpr | ConditionGroupExpr;

export const isConditionGroupExpr = (
  expr: SimpleExpr
): expr is ConditionGroupExpr => {
  return isLogicalOperator(expr.operator);
};

export const isEqualityExpr = (expr: SimpleExpr): expr is EqualityExpr => {
  return isEqualityOperator(expr.operator);
};

export const isCompareExpr = (expr: SimpleExpr): expr is CompareExpr => {
  return isCompareOperator(expr.operator);
};

export const isCollectionExpr = (expr: SimpleExpr): expr is CollectionExpr => {
  return isCollectionOperator(expr.operator);
};

export const isStringExpr = (expr: SimpleExpr): expr is StringExpr => {
  return isStringOperator(expr.operator);
};

export const isConditionExpr = (expr: SimpleExpr): expr is ConditionExpr => {
  return (
    isEqualityExpr(expr) ||
    isCompareExpr(expr) ||
    isCollectionExpr(expr) ||
    isStringExpr(expr)
  );
};
