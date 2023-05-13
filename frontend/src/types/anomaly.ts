import {
  AnomalyId,
  Database,
  DatabaseId,
  EnvironmentId,
  Instance,
  InstanceId,
  Principal,
  RowStatus,
} from ".";
import { BackupPlanSchedule } from "@/types/proto/v1/org_policy_service";

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
  environmentId: EnvironmentId;
  expectedSchedule: BackupPlanSchedule;
  actualSchedule: BackupPlanSchedule;
};

export type AnomalyDatabaseBackupMissingPayload = {
  expectedSchedule: BackupPlanSchedule;
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
  id: AnomalyId;

  // Related fields
  instanceId: InstanceId;
  instance: Instance;
  databaseId?: DatabaseId;
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

export type AnomalyFind = {
  instanceId?: InstanceId;
  databaseId?: DatabaseId;
  rowStatus?: RowStatus;
  type?: AnomalyType;
};
