import { uniq } from "lodash-es";
import { Factor } from "./factor";

export const LogicalOperatorList = ["_&&_", "_||_"] as const;
export type LogicalOperator = typeof LogicalOperatorList[number];

export const EqualityOperatorList = ["_==_", "_!=_"] as const;
export type EqualityOperator = typeof EqualityOperatorList[number];

export const CompareOperatorList = ["_<_", "_<=_", "_>=_", "_>_"] as const;
export type CompareOperator = typeof CompareOperatorList[number];

export const CollectionOperatorList = ["@in"] as const;
export type CollectionOperator = typeof CollectionOperatorList[number];

export const StringOperatorList = [
  "contains",
  "matches",
  "startsWith",
  "endsWith",
] as const;
export type StringOperator = typeof StringOperatorList[number];

export type ConditionOperator =
  | EqualityOperator
  | CompareOperator
  | CollectionOperator
  | StringOperator;
export type Operator = LogicalOperator | ConditionOperator;

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
  affected_rows: uniq([...EqualityOperatorList, ...CompareOperatorList]),

  level: uniq([...EqualityOperatorList, ...CollectionOperatorList]),
  source: uniq([...EqualityOperatorList, ...CollectionOperatorList]),

  environment_id: uniq([...EqualityOperatorList, ...CollectionOperatorList]),
  instance_id: uniq([...EqualityOperatorList, ...CollectionOperatorList]),
  project_id: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  database_name: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  table_name: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  db_engine: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),
  sql_type: uniq([
    ...EqualityOperatorList,
    ...CollectionOperatorList,
    ...StringOperatorList,
  ]),

  // Database group related fields.
  "resource.environment_name": uniq(["_==_"]),
  "resource.instance_id": uniq([
    "_==_",
    "_!=_",
    "startsWith",
    "endsWith",
    "contains",
    "matches",
  ]),
  "resource.database_name": uniq([
    "_==_",
    "_!=_",
    "startsWith",
    "endsWith",
    "contains",
    "matches",
  ]),
  "resource.table_name": uniq([
    "_==_",
    "_!=_",
    "startsWith",
    "endsWith",
    "contains",
    "matches",
  ]),

  // Request query/export factors
  expiration_days: uniq([...EqualityOperatorList, ...CompareOperatorList]),
  export_rows: uniq([...EqualityOperatorList, ...CompareOperatorList]),
};

export const getOperatorListByFactor = (factor: Factor) => {
  const list = OperatorList[factor];
  if (!list) {
    console.warn(`unsupported factor '${factor}'`);
    return [];
  }
  return list;
};
