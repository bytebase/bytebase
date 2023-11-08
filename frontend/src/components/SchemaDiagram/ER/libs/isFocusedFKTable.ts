import type { TableMetadata } from "@/types/proto/v1/database_service";
import type { ForeignKey } from "../../types";

export const isFocusedForeignTable = (
  table: TableMetadata,
  focusedTables: Set<TableMetadata>,
  foreignKeys: ForeignKey[]
): boolean => {
  const fks = foreignKeys.filter(
    (fk) => fk.from.table === table || fk.to.table === table
  );
  return fks.some((fk) => {
    if (fk.from.table === table) {
      return focusedTables.has(fk.to.table);
    }
    if (fk.to.table === table) {
      return focusedTables.has(fk.from.table);
    }
    return false;
  });
};
