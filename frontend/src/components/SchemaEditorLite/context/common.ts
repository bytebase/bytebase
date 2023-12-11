import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";

export const keyForResource = (
  database: ComposedDatabase,
  metadata: {
    schema?: SchemaMetadata;
    table?: TableMetadata;
    column?: ColumnMetadata;
  } = {}
) => {
  const { schema, table, column } = metadata;
  return keyForResourceName(
    database.name,
    schema?.name,
    table?.name,
    column?.name
  );
};

export const keyForResourceName = (
  database: string,
  schema?: string,
  table?: string,
  column?: string
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
  return parts.join("/");
};
