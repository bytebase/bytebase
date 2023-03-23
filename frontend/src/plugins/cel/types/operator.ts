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
export const OperatorList: Record<Factor, Operator[]> = {
  insert_rows: uniq([...EqualityOperatorList, ...CompareOperatorList]),
  update_delete_rows: uniq([...EqualityOperatorList, ...CompareOperatorList]),

  risk: uniq([...EqualityOperatorList, ...CollectionOperatorList]),
  source: uniq([...EqualityOperatorList, ...CollectionOperatorList]),

  environment: uniq([
    ...EqualityOperatorList, // using `environment_id`
    ...CollectionOperatorList, // using `environment_id`
    ...StringOperatorList, // using `environment_name`
  ]),
  project: uniq([
    ...EqualityOperatorList, // using `project_id`
    ...CollectionOperatorList, // using `project_id`
    ...StringOperatorList, // using `project_name`
  ]),
  database_name: uniq([
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
};

export const getOperatorListByFactor = (factor: Factor) => {
  const list = OperatorList[factor];
  if (!list) {
    console.warn(`unsupported factor '${factor}'`);
    return [];
  }
  return list;
};
