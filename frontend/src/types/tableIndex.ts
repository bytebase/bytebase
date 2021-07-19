import { DatabaseId, TableId, TableIndexId } from "./id";

// Similar to DatabaseSyncStatus, see comment
export type IndexSyncStatus = "OK" | "NOT_FOUND";

// Index
export type TableIndex = {
  id: TableIndexId;

  // Related fields
  databaseId: DatabaseId;
  tableId: TableId;

  // Standard fields
  creatorId: number;
  createdTs: number;
  updaterId: number;
  updatedTs: number;

  // Domain specific fields
  syncStatus: IndexSyncStatus;
  lastSuccessfulSyncTs: number;
  name: string;
  expression: string;
  position: number;
  type: string;
  unique: boolean;
  visible: boolean;
  comment: string;
};
