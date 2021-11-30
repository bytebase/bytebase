// We periodically sync the underlying db schema and stores those info
// in the "database" object.
// Physically, a database belongs to an instance. Logically, it belongs to a project.

import { Anomaly } from ".";
import { Backup } from "./backup";
import { DataSource } from "./dataSource";
import { DatabaseID, InstanceID, IssueID, ProjectID } from "./id";
import { Instance } from "./instance";
import { Principal } from "./principal";
import { Project } from "./project";

// "OK" means we find the database with the same name.
// "NOT_FOUND" means no matching database name found, this usually means someone changes the underlying db name without Bytebase knowledge.
export type DatabaseSyncStatus = "OK" | "NOT_FOUND";
// Database
export type Database = {
  id: DatabaseID;

  // Related fields
  instance: Instance;
  project: Project;
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
};

export type DatabaseCreate = {
  // Related fields
  instanceID: InstanceID;
  projectID: ProjectID;

  // Domain specific fields
  name: string;
  characterSet: string;
  collation: string;
  issueID?: IssueID;
};

export type DatabasePatch = {
  // Related fields
  projectID: ProjectID;
};
