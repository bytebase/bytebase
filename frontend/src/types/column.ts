import { ColumnId, DatabaseId, IdType } from "./id";

// TODO(steven): remove it.
// Column
export type Column = {
  id: ColumnId;

  // Related fields
  databaseId: DatabaseId;
  tableId: IdType;

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
