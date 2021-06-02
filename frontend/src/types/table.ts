// "OK" means find the exact match
// "DRIFTED" means we find the table with the same name, but the fingerprint is different,
//            this usually indicates the underlying table has been recreated (might for a entirely different purpose)
// "NOT_FOUND" means no matching table name found, this ususally means someone changes

import { Database } from "./database";
import { TableId } from "./id";
import { Principal } from "./principal";

//            the underlying table name.
export type TableSyncStatus = "OK" | "DRIFTED" | "NOT_FOUND";
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
  engine: string;
  collation: string;
  rowCount: number;
  dataSize: number;
  indexSize: number;
  syncStatus: TableSyncStatus;
  lastSuccessfulSyncTs: number;
};
