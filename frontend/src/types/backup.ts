import { BackupId, BackupSettingId, DatabaseId } from "./id";

export type BackupStatus = "PENDING_CREATE" | "DONE" | "FAILED";

export type BackupType = "MANUAL" | "AUTOMATIC" | "PITR";

export type BackupStorageBackend = "LOCAL" | "S3" | "GCS";

// Backup
export type Backup = {
  id: BackupId;

  // Related fields
  databaseId: DatabaseId;

  // Standard fields
  createdTs: number;

  name: string;
  status: BackupStatus;
  type: BackupType;
  storageBackend: BackupStorageBackend;
  migrationHistoryVersion: string;
  path: string;
  comment: string;
};

// Backup setting.
export type BackupSetting = {
  id: BackupSettingId;

  // Related fields
  databaseId: DatabaseId;

  enabled: boolean;
  hour: number;
  dayOfWeek: number;
  retentionPeriodTs: number;
  hookUrl: string;
};
