import {
  ColumnMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  TableMetadata,
  TablePartitionMetadata,
} from "@/types/proto/api/v1alpha/database_service";

export const ComparableTableFields: (keyof TableMetadata)[] = [
  "name",
  "comment",
  "userComment",
  "collation",
  "engine",
];
export const ComparableIndexFields: (keyof IndexMetadata)[] = [
  "name",
  "definition",
  "primary",
  "unique",
  "comment",
  "expressions",
];
export const ComparableForeignKeyFields: (keyof ForeignKeyMetadata)[] = [
  "name",
  "columns",
  "referencedSchema",
  "referencedTable",
  "referencedColumns",
];
export const ComparableTablePartitionFields: (keyof TablePartitionMetadata)[] =
  ["name", "type", "expression", "value"];
export const ComparableColumnFields: (keyof ColumnMetadata)[] = [
  "name",
  "comment",
  "userComment",
  "type",
  "hasDefault",
  "defaultExpression",
  "defaultNull",
  "defaultString",
  "onUpdate",
  "nullable",
];
