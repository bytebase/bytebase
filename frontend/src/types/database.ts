// We periodically sync the underlying db schema and stores those info
// in the "database" object.
// Physically, a database belongs to an instance. Logically, it belongs to a project.
import type { DataSource } from "./dataSource";
import type { DatabaseId, InstanceId, ProjectId } from "./id";
import type { Instance } from "./instance";
import type { Project } from "./project";

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

  // Domain specific fields
  syncStatus: DatabaseSyncStatus;
  lastSuccessfulSyncTs: number;
  name: string;
  characterSet: string;
  collation: string;
  schemaVersion: string;
  labels: { key: string; value: string }[];
};
