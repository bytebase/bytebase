import { Engine } from "@/types/proto/v1/common";
import { ColumnMetadata } from "@/types/proto/v1/database_service";

export const engineList = [Engine.MYSQL, Engine.POSTGRES];

export const getDefaultValue = (column: ColumnMetadata | undefined) => {
  if (column?.default) {
    return column.default;
  }
  if (column?.nullable) {
    return "NULL";
  }
  return "EMPTY";
};
