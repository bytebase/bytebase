import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import type { ColumnDefaultValue } from "@/types/v1/schemaEditor";

export type DefaultValue = Pick<ColumnMetadata, "hasDefault" | "default">;

export const getColumnDefaultValuePlaceholder = (
  column: ColumnDefaultValue
): string => {
  if (!column.hasDefault) {
    return "No default";
  }
  if (column.default === "NULL") {
    return "Null";
  }
  if (column.default !== undefined) {
    return column.default || "Empty string";
  }
  return "";
};
