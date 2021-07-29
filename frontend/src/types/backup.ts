import { Database } from "./database";
import { BackupId, DatabaseId } from "./id";
import { Principal } from "./principal";

// Backup
export type Backup = {
  id: BackupId;

  // Related fields
  database: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  name: string;
  status: string;
  type: string;
  storageBackend: string;
  path: string;
  comment: string;
};

export type BackupCreate = {
  // Related fields
  databaseId: DatabaseId;

  // Domain specific fields
  name: string;
  status: string;
  type: string;
  storageBackend: string;
  path: string;
  comment: string;
};
