import { Database } from "./database";
import { TableId } from "./id";
import { Principal } from "./principal";

// Similar to DatabaseSyncStatus, see comment
export type TableSyncStatus = "OK" | "NOT_FOUND";
export type TableType = "BASE TABLE" | "VIEW";
export type TableEngineType = "InnoDB";

// Table
export type Table = {
  id: TableId;

  // Related fields
  database: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  table: string;
  type: TableType;
  engine: TableEngineType;
  collation: string;
  syncStatus: TableSyncStatus;
  lastSuccessfulSyncTs: number;
  rowCount: number;
  dataSize: number;
  indexSize: number;
  dataFree: number;
  createOptions: string;
  comment: string;
};
