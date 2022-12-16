import { DatabaseId, IdType, TableIndexId } from "./id";

// TODO(steven): remove it.
// Index
export type TableIndex = {
  id: TableIndexId;

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
  expression: string;
  position: number;
  type: string;
  unique: boolean;
  visible: boolean;
  comment: string;
};
