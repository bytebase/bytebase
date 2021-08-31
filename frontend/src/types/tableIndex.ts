import { DatabaseId, TableId, TableIndexId } from "./id";

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
  name: string;
  expression: string;
  position: number;
  type: string;
  unique: boolean;
  visible: boolean;
  comment: string;
};
