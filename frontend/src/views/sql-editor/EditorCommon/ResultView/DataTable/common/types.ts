import type { QueryRow } from "@/types/proto-es/v1/sql_service_pb";

export interface ResultTableColumn {
  id: string;
  name: string;
  columnType: string;
}

export interface ResultTableRow {
  key: number;
  item: QueryRow;
}

export type SortDirection = "asc" | "desc" | false;

export interface SortState {
  columnIndex: number;
  direction: SortDirection;
}
