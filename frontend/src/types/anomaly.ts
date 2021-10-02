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
  | "bb.anomaly.backup.policy-violation"
  | "bb.anomaly.backup.missing"
  | "bb.anomaly.database.connection"
  | "bb.anomaly.database.schema.drift";

export type AnomalyBackupPolicyViolationPayload = {
  environmentId: EnvironmentId;
  expectedSchedule: BackupPlanPolicySchedule;
  actualSchedule: BackupPlanPolicySchedule;
};

export type AnomalyBackupMissingPayload = {
  expectedSchedule: BackupPlanPolicySchedule;
  lastBackupTs: number;
};

export type AnomalyDatabaseConnectionPayload = {
  detail: string;
};

export type AnomalyDatabaseSchemaDriftPayload = {
  detail: string;
};

export type AnomalyPayload =
  | AnomalyBackupPolicyViolationPayload
  | AnomalyBackupMissingPayload
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
