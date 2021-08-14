import { Database } from "./database";
import { BackupId, BackupSettingId, DatabaseId } from "./id";
import { Principal } from "./principal";

export type BackupStatus = "PENDING_CREATE" | "DONE" | "FAILED";

export type BackupType = "MANUAL" | "AUTOMATIC";

export type BackupStorageBackend = "LOCAL";

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
  status: BackupStatus;
  type: BackupType;
  storageBackend: BackupStorageBackend;
  path: string;
  comment: string;
};

export type BackupCreate = {
  // Related fields
  databaseId: DatabaseId;

  // Domain specific fields
  name: string;
  status: BackupStatus;
  type: BackupType;
  storageBackend: BackupStorageBackend;
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
};

export type BackupSettingUpsert = {
  // Related fields
  databaseId: DatabaseId;

  // Domain specific fields
  enabled: boolean;
  hour: number;
  dayOfWeek: number;
};
