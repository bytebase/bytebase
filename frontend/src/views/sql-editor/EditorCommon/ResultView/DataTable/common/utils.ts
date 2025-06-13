import type { Header } from "@tanstack/vue-table";
import type { QueryRow } from "@/types/proto/v1/sql_service";

interface ColumnMeta {
  columnType: string;
}

export const getColumnType = (
  column: Header<QueryRow, unknown> | undefined
) => {
  return (
    (column?.column.columnDef.meta as ColumnMeta | undefined)?.columnType ?? ""
  );
};
