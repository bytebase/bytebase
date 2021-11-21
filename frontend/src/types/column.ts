import { ColumnID, DatabaseID, TableID } from "./id";

// Column
export type Column = {
  id: ColumnID;

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
  position: number;
  default: string;
  nullable: boolean;
  type: string;
  characterSet: string;
  collation: string;
  comment: string;
};
