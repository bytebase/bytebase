import { ColumnId, DatabaseId, TableId } from "./id";

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
  name: string;
  position: number;
  default?: string;
  nullable: boolean;
  type: string;
  characterSet: string;
  collation: string;
  comment: string;
};
