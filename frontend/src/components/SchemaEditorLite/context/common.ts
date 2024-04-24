import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
} from "@/types/proto/v1/database_service";

export const keyForResource = (
  database: ComposedDatabase,
  metadata: {
    schema?: SchemaMetadata;
    table?: TableMetadata;
    column?: ColumnMetadata;
    partition?: TablePartitionMetadata;
  } = {}
) => {
  const { schema, table, column, partition } = metadata;
  return keyForResourceName(
    database.name,
    schema?.name,
    table?.name,
    column?.name,
    partition?.name
  );
};

export const keyForResourceName = (
  database: string,
  schema?: string,
  table?: string,
  column?: string,
  partition?: string
) => {
  const parts = [database];
  if (schema !== undefined) {
    parts.push(`schemas/${schema}`);
  }
  if (table !== undefined) {
    parts.push(`tables/${table}`);
  }
  if (column !== undefined) {
    parts.push(`columns/${column}`);
  }
  if (partition !== undefined) {
    parts.push(`partitions/${partition}`);
  }
  return parts.join("/");
};
