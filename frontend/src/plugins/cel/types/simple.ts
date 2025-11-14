/// Define a simplified version (less nested) of CEL Expr.
/// Convenient for local editing.
import type { NumberFactor, StringFactor, TimestampFactor } from "./factor";
import {
  type CollectionOperator,
  type CompareOperator,
  type EqualityOperator,
  isCollectionOperator,
  isCompareOperator,
  isEqualityOperator,
  isStringOperator,
  type LogicalOperator,
  type StringOperator,
} from "./operator";

export enum ExprType {
  RawString = "RawString",
  Condition = "Condition",
  ConditionGroup = "ConditionGroup",
}

interface BaseConditionExpr {
  type: ExprType.Condition;
}

export interface EqualityExpr extends BaseConditionExpr {
  operator: EqualityOperator;
  args: [StringFactor | NumberFactor, string | number];
}

export interface CompareExpr extends BaseConditionExpr {
  operator: CompareOperator;
  args: [NumberFactor | TimestampFactor, number | Date];
}

export interface CollectionExpr extends BaseConditionExpr {
  operator: CollectionOperator;
  args: [StringFactor | NumberFactor, string[] | number[]];
}

export interface StringExpr extends BaseConditionExpr {
  operator: StringOperator;
  args: [StringFactor, string];
}

export type ConditionExpr =
  | EqualityExpr
  | CompareExpr
  | CollectionExpr
  | StringExpr;

// RawStringExpr is used to represent a string that is unable to be parsed as a condition.
export type RawStringExpr = {
  type: ExprType.RawString;
  content: string;
};

export type LogicalExpr = {
  type: ExprType.ConditionGroup;
  operator: LogicalOperator;
  args: (ConditionExpr | RawStringExpr | ConditionGroupExpr)[];
};

export type ConditionGroupExpr = LogicalExpr;

export type SimpleExpr = ConditionExpr | ConditionGroupExpr | RawStringExpr;

export const isConditionGroupExpr = (
  expr: SimpleExpr
): expr is ConditionGroupExpr => {
  return expr.type === ExprType.ConditionGroup;
};

export const isConditionExpr = (expr: SimpleExpr): expr is ConditionExpr => {
  return expr.type === ExprType.Condition;
};

export const isEqualityExpr = (expr: SimpleExpr): expr is EqualityExpr => {
  return isConditionExpr(expr) && isEqualityOperator(expr.operator);
};

export const isCompareExpr = (expr: SimpleExpr): expr is CompareExpr => {
  return isConditionExpr(expr) && isCompareOperator(expr.operator);
};

export const isCollectionExpr = (expr: SimpleExpr): expr is CollectionExpr => {
  return isConditionExpr(expr) && isCollectionOperator(expr.operator);
};

export const isStringExpr = (expr: SimpleExpr): expr is StringExpr => {
  return isConditionExpr(expr) && isStringOperator(expr.operator);
};

export const isRawStringExpr = (expr: SimpleExpr): expr is RawStringExpr => {
  return expr.type === ExprType.RawString;
};
