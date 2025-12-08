import { uniq } from "lodash-es";
import {
  CEL_ATTRIBUTE_LEVEL,
  CEL_ATTRIBUTE_REQUEST_EXPIRATION_DAYS,
  CEL_ATTRIBUTE_REQUEST_ROLE,
  CEL_ATTRIBUTE_REQUEST_TIME,
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_DB_ENGINE,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
  CEL_ATTRIBUTE_RISK_LEVEL,
  CEL_ATTRIBUTE_SOURCE,
  CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS,
  CEL_ATTRIBUTE_STATEMENT_SQL_TYPE,
  CEL_ATTRIBUTE_STATEMENT_TABLE_ROWS,
  CEL_ATTRIBUTE_STATEMENT_TEXT,
} from "@/utils/cel-attributes";
import type { Factor } from "./factor";

export const NegativeOperatorList = ["!_"] as const;
export type NegativeOperator = (typeof NegativeOperatorList)[number];

export const LogicalOperatorList = ["_&&_", "_||_"] as const;
export type LogicalOperator = (typeof LogicalOperatorList)[number];

export const EqualityOperatorList = ["_==_", "_!=_"] as const;
export type EqualityOperator = (typeof EqualityOperatorList)[number];

export const CompareOperatorList = ["_<_", "_<=_", "_>=_", "_>_"] as const;
export type CompareOperator = (typeof CompareOperatorList)[number];

export const CollectionOperatorList = ["@in", "@not_in"] as const;
export type CollectionOperator = (typeof CollectionOperatorList)[number];

export const StringOperatorList = [
  "contains",
  "@not_contains",
  "matches",
  "matches",
  "startsWith",
  "endsWith",
] as const;
export type StringOperator = (typeof StringOperatorList)[number];

export type ConditionOperator =
  | EqualityOperator
  | CompareOperator
  | CollectionOperator
  | StringOperator;
export type Operator = LogicalOperator | NegativeOperator | ConditionOperator;

export const isNegativeOperator = (op: Operator): op is NegativeOperator => {
  return NegativeOperatorList.includes(op as NegativeOperator);
};
export const isLogicalOperator = (op: Operator): op is LogicalOperator => {
  return LogicalOperatorList.includes(op as LogicalOperator);
};
export const isEqualityOperator = (op: Operator): op is EqualityOperator => {
  return EqualityOperatorList.includes(op as EqualityOperator);
};
export const isCompareOperator = (op: Operator): op is CompareOperator => {
  return CompareOperatorList.includes(op as CompareOperator);
};
export const isCollectionOperator = (
  op: Operator
): op is CollectionOperator => {
  return CollectionOperatorList.includes(op as CollectionOperator);
};
export const isStringOperator = (op: Operator): op is StringOperator => {
  return StringOperatorList.includes(op as StringOperator);
};

/// Define supported operators for each factor
const OperatorList: Record<Factor, Operator[]> = {
  [CEL_ATTRIBUTE_STATEMENT_AFFECTED_ROWS]: uniq([
    ...EqualityOperatorList,
    ...CompareOperatorList,
  ]),
  [CEL_ATTRIBUTE_STATEMENT_TABLE_ROWS]: uniq([
    ...EqualityOperatorList,
    ...CompareOperatorList,
  ]),

  [CEL_ATTRIBUTE_LEVEL]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
  ]),
  [CEL_ATTRIBUTE_SOURCE]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
  ]),
  [CEL_ATTRIBUTE_RISK_LEVEL]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
  ]),

  [CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
  ]),
  [CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  [CEL_ATTRIBUTE_RESOURCE_PROJECT_ID]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  [CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  [CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  [CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  [CEL_ATTRIBUTE_RESOURCE_DB_ENGINE]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  [CEL_ATTRIBUTE_STATEMENT_SQL_TYPE]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  [CEL_ATTRIBUTE_STATEMENT_TEXT]: uniq([...StringOperatorList]),
  [CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL]: uniq([
    ...CollectionOperatorList,
  ]),

  // Request query/export factors
  [CEL_ATTRIBUTE_REQUEST_EXPIRATION_DAYS]: uniq([
    ...EqualityOperatorList,
    ...CompareOperatorList,
  ]),
  [CEL_ATTRIBUTE_REQUEST_ROLE]: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),

  // These factors don't have operator candidates for user selection.
  [CEL_ATTRIBUTE_RESOURCE_DATABASE]: [],
  [CEL_ATTRIBUTE_REQUEST_TIME]: [],
};

export const getOperatorListByFactor = (factor: Factor) => {
  const list = OperatorList[factor];
  if (!list) {
    console.warn(`unsupported factor '${factor}'`);
    return [];
  }
  return list;
};
