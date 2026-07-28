import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";

export type DefaultValue = Pick<ColumnMetadata, "hasDefault" | "default">;

export const getColumnDefaultValuePlaceholder = (
  column: DefaultValue
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
