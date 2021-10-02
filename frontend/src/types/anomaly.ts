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
  | "bb.anomaly.database.connection";

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

export type AnomalyPayload =
  | AnomalyBackupPolicyViolationPayload
  | AnomalyBackupMissingPayload
  | AnomalyDatabaseConnectionPayload;

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
