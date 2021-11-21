import { DatabaseID, TableID, TableIndexID } from "./id";

// Index
export type TableIndex = {
  id: TableIndexID;

  // Related fields
  databaseID: DatabaseID;
  tableID: TableID;

  // Standard fields
  creatorID: number;
  createdTs: number;
  updaterID: number;
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
