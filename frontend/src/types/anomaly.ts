import {
  AnomalyID,
  BackupPlanPolicySchedule,
  Database,
  DatabaseID,
  EnvironmentID,
  Instance,
  InstanceID,
  Principal,
} from ".";

export type AnomalyType =
  | "bb.anomaly.instance.connection"
  | "bb.anomaly.instance.migration-schema"
  | "bb.anomaly.database.backup.policy-violation"
  | "bb.anomaly.database.backup.missing"
  | "bb.anomaly.database.connection"
  | "bb.anomaly.database.schema.drift";

export type AnomalyInstanceConnectionPayload = {
  detail: string;
};

export type AnomalyDatabaseBackupPolicyViolationPayload = {
  environmentID: EnvironmentID;
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

export type AnomalySeverity = "MEDIUM" | "HIGH" | "CRITICAL";

export type Anomaly = {
  id: AnomalyID;

  // Related fields
  instanceID: InstanceID;
  instance: Instance;
  databaseID?: DatabaseID;
  database?: Database;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  type: AnomalyType;
  severity: AnomalySeverity;
  payload: AnomalyPayload;
};
