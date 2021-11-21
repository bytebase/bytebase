import { BackupID, BackupSettingID, DatabaseID } from "./id";
import { Principal } from "./principal";

export type BackupStatus = "PENDING_CREATE" | "DONE" | "FAILED";

export type BackupType = "MANUAL" | "AUTOMATIC";

export type BackupStorageBackend = "LOCAL";

// Backup
export type Backup = {
  id: BackupID;

  // Related fields
  databaseID: DatabaseID;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  name: string;
  status: BackupStatus;
  type: BackupType;
  storageBackend: BackupStorageBackend;
  migrationHistoryVersion: string;
  path: string;
  comment: string;
};

export type BackupCreate = {
  // Related fields
  databaseID: DatabaseID;

  // Domain specific fields
  name: string;
  status: BackupStatus;
  type: BackupType;
  storageBackend: BackupStorageBackend;
};

// Backup setting.
export type BackupSetting = {
  id: BackupSettingID;

  // Related fields
  databaseID: DatabaseID;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  enabled: boolean;
  hour: number;
  dayOfWeek: number;
  hookURL: string;
};

export type BackupSettingUpsert = {
  // Related fields
  databaseID: DatabaseID;

  // Domain specific fields
  enabled: boolean;
  hour: number;
  dayOfWeek: number;
  hookURL: string;
};
