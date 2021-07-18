import { ColumnId, DatabaseId, TableId } from "./id";

// Similar to DatabaseSyncStatus, see comment
export type ColumnSyncStatus = "OK" | "NOT_FOUND";

// Column
export type Column = {
  id: ColumnId;

  // Related fields
  databaseId: DatabaseId;
  tableId: TableId;

  // Standard fields
  creatorId: number;
  createdTs: number;
  updaterId: number;
  updatedTs: number;

  // Domain specific fields
  syncStatus: ColumnSyncStatus;
  lastSuccessfulSyncTs: number;
  name: string;
  position: number;
  default: string;
  nullable: boolean;
  type: string;
  characterSet: string;
  collation: string;
  comment: string;
};
