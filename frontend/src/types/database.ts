// We periodically sync the underlying db schema and stores those info
// in the "database" object.
// Physically, a database belongs to an instance. Logically, it belongs to a project.

import { DatabaseId, InstanceId, IssueId, PrincipalId, ProjectId } from "./id";
import { Instance } from "./instance";
import { Principal } from "./principal";
import { Project } from "./project";
import { DataSource } from "./dataSource";

// "OK" means find the exact match
// "DRIFTED" means we find the database with the same name, but the fingerprint is different,
//            this usually indicates the underlying database has been recreated (might for a entirely different purpose)
// "NOT_FOUND" means no matching database name found, this ususally means someone changes
//            the underlying db name.
export type DatabaseSyncStatus = "OK" | "DRIFTED" | "NOT_FOUND";
// Database
export type Database = {
  id: DatabaseId;

  // Related fields
  instance: Instance;
  project: Project;
  dataSourceList: DataSource[];

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  characterSet: string;
  collation: string;
  syncStatus: DatabaseSyncStatus;
  lastSuccessfulSyncTs: number;
  fingerprint: string;
};

export type DatabaseCreate = {
  // Related fields
  instanceId: InstanceId;
  projectId: ProjectId;

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  characterSet: string;
  collation: string;
  issueId?: IssueId;
};

export type DatabasePatch = {
  // Related fields
  projectId: ProjectId;
};
