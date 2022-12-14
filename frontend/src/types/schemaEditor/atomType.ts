import { DatabaseId } from "../id";
import { TableEngineType, TableType } from "../table";

type TableOrColumnStatus = "normal" | "created" | "dropped";

export interface Column {
  // Related fields
  databaseId: DatabaseId;

  // Domain specific fields
  oldName: string;
  newName: string;
  type: string;
  nullable: boolean;
  comment: string;
  default: string | null;

  status: TableOrColumnStatus;
}

export interface Table {
  // Related fields
  databaseId: DatabaseId;

  // Domain specific fields
  oldName: string;
  newName: string;
  type: TableType;
  engine: TableEngineType;
  collation: string;
  rowCount: number;
  dataSize: number;
  comment: string;
  columnList: Column[];
  originColumnList: Column[];

  status: TableOrColumnStatus;
}
