import { BackupId, BackupSettingId, DatabaseId } from "./id";
import { Principal } from "./principal";

export type BackupStatus = "PENDING_CREATE" | "DONE" | "FAILED";

export type BackupType = "MANUAL" | "AUTOMATIC";

export type BackupStorageBackend = "LOCAL";

// Backup
export type Backup = {
  id: BackupId;

  // Related fields
  databaseId: DatabaseId;

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
  databaseId: DatabaseId;

  // Domain specific fields
  name: string;
  type: BackupType;
  storageBackend: BackupStorageBackend;
};

// Backup setting.
export type BackupSetting = {
  id: BackupSettingId;

  // Related fields
  databaseId: DatabaseId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  enabled: boolean;
  hour: number;
  dayOfWeek: number;
  hookUrl: string;
};

export type BackupSettingUpsert = {
  // Related fields
  databaseId: DatabaseId;

  // Domain specific fields
  enabled: boolean;
  hour: number;
  dayOfWeek: number;
  hookUrl: string;
};
