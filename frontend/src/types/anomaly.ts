import {
  AnomalyId,
  BackupPlanPolicySchedule,
  Database,
  DatabaseId,
  EnvironmentId,
  Instance,
  InstanceId,
  Principal,
} from ".";

export type AnomalyType =
  | "bb.anomaly.database.backup.policy-violation"
  | "bb.anomaly.database.backup.missing"
  | "bb.anomaly.database.connection"
  | "bb.anomaly.database.schema.drift";

export type AnomalyDatabaseBackupPolicyViolationPayload = {
  environmentId: EnvironmentId;
  expectedSchedule: BackupPlanPolicySchedule;
  actualSchedule: BackupPlanPolicySchedule;
};

export type AnomalyDatabaseBackupMissingPayload = {
  expectedSchedule: BackupPlanPolicySchedule;
  lastBackupTs: number;
};

export type AnomalyDatabaseConnectionPayload = {
  detail: string;
};

export type AnomalyDatabaseSchemaDriftPayload = {
  version: string;
  expect: string;
  actual: string;
};

export type AnomalyPayload =
  | AnomalyDatabaseBackupPolicyViolationPayload
  | AnomalyDatabaseBackupMissingPayload
  | AnomalyDatabaseConnectionPayload
  | AnomalyDatabaseSchemaDriftPayload;

export type Anomaly = {
  id: AnomalyId;

  // Related fields
  instanceId: InstanceId;
  instance: Instance;
  databaseId: DatabaseId;
  database: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  type: AnomalyType;
  payload: AnomalyPayload;
};
