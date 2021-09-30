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
  | "bb.anomaly.backup.missing";

export type AnomalyBackupPolicyViolationPayload = {
  environmentId: EnvironmentId;
  expectedSchedule: BackupPlanPolicySchedule;
  actualSchedule: BackupPlanPolicySchedule;
};

export type AnomalyPayload = AnomalyBackupPolicyViolationPayload;

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
