import type {
  ColumnMetadata,
  ForeignKeyMetadata,
  IndexMetadata,
  TableMetadata,
  TablePartitionMetadata,
} from "@/types/proto-es/v1/database_service_pb";

export const ComparableTableFields: (keyof TableMetadata)[] = [
  "name",
  "comment",
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
  "type",
  "hasDefault",
  "default",
  "onUpdate",
  "nullable",
];
