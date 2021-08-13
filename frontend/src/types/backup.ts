import { Database } from "./database";
import { BackupId, BackupSettingId, DatabaseId } from "./id";
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

export type RestoreBackup = {
  // Related fields

  // Domain specific fields
  backupId: BackupId;
};

// Backup setting.
export type BackupSetting = {
  id: BackupSettingId;

  // Related fields
  database: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  enabled: boolean;
  hour: number;
  dayOfWeek: number;
  pathTemplate: string;
};

export type BackupSettingUpsert = {
  // Related fields
  databaseId: DatabaseId;

  // Domain specific fields
  enabled: boolean;
  hour: number;
  dayOfWeek: number;
  pathTemplate: string;
};
