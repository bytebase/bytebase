import { Column } from "./column";
import { Database } from "./database";
import { TableId } from "./id";
import { Principal } from "./principal";
import { TableIndex } from "./tableIndex";

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
  type: TableType;
  engine: TableEngineType;
  collation: string;
  rowCount: number;
  dataSize: number;
  indexList: TableIndex[];
  indexSize: number;
  dataFree: number;
  createOptions: string;
  comment: string;
  columnList: Column[];
};
