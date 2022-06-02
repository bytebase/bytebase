// We periodically sync the underlying db schema and stores those info
// in the "database" object.
// Physically, a database belongs to an instance. Logically, it belongs to a project.

import { Anomaly } from ".";
import { Backup } from "./backup";
import { DataSource } from "./dataSource";
import { DatabaseId, InstanceId, IssueId, ProjectId } from "./id";
import { Instance } from "./instance";
import { Principal } from "./principal";
import { Project } from "./project";
import { DatabaseLabel } from "./label";

// "OK" means we find the database with the same name.
// "NOT_FOUND" means no matching database name found, this usually means someone changes the underlying db name without Bytebase knowledge.
export type DatabaseSyncStatus = "OK" | "NOT_FOUND";
// Database
export type Database = {
  id: DatabaseId;

  // Related fields
  projectId: ProjectId;
  project: Project;
  instanceId: InstanceId;
  instance: Instance;
  dataSourceList: DataSource[];
  sourceBackup?: Backup;
  anomalyList: Anomaly[];

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  syncStatus: DatabaseSyncStatus;
  lastSuccessfulSyncTs: number;
  name: string;
  characterSet: string;
  collation: string;
  schemaVersion: string;
  labels: DatabaseLabel[];
};

export type DatabaseCreate = {
  // Related fields
  instanceId: InstanceId;
  projectId: ProjectId;

  // Domain specific fields
  name: string;
  characterSet: string;
  collation: string;
  issueId?: IssueId;
  labels?: DatabaseLabel[];
};

export type DatabaseFind = {
  // Related fields
  projectId?: ProjectId;
  instanceId?: InstanceId;

  // Domain specific fields
  name?: string;
  syncStatus?: DatabaseSyncStatus;
};

export type DatabasePatch = {
  // Related fields
  projectId?: ProjectId;
  labels?: DatabaseLabel[];
};
