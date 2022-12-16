// For now the ADMIN requires the same database privilege as RW.
// The seperation is to make it explicit which one serves as the ADMIN data source,

import {
  DatabaseId,
  DataSourceId,
  InstanceId,
  IssueId,
  PrincipalId,
} from "./id";
import { Principal } from "./principal";

// which from the ops perspective, having different meaning from the normal RW data source.
export type DataSourceType = "ADMIN" | "RW" | "RO";

// DataSourceOptions is the options for a data source.
export type DataSourceOptions = {
  srv: boolean;
  authSource: string;
};

export type DataSource = {
  id: DataSourceId;

  // Related fields
  databaseId: DatabaseId;
  instanceId: InstanceId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: string;
  type: DataSourceType;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
  sslCa?: string;
  sslCert?: string;
  sslKey?: string;

  // hostOverride and portOverride are only used for read-only data sources for user's read-replica instances.
  hostOverride: string;
  portOverride: string;

  options: DataSourceOptions;
  // UI-only fields
  updateSsl?: boolean;
};

export type DataSourceCreate = {
  // Related fields
  databaseId: DatabaseId;
  instanceId: InstanceId;

  // Domain specific fields
  name: string;
  type: DataSourceType;
  username?: string;
  password?: string;
  sslCa?: string;
  sslCert?: string;
  sslKey?: string;
  hostOverride: string;
  portOverride: string;
};

export type DataSourcePatch = {
  // Domain specific fields
  name?: string;
  username?: string;
  password?: string;
  useEmptyPassword?: boolean;
  sslCa?: string;
  sslCert?: string;
  sslKey?: string;
  hostOverride?: string;
  portOverride?: string;
  options?: DataSourceOptions;
};

export type DataSourceMember = {
  // Standard fields
  createdTs: number;

  // Domain specific fields
  principal: Principal;
  issueId?: IssueId;
};

export type DataSourceMemberCreate = {
  // Domain specific fields
  principalId: PrincipalId;
  issueId?: IssueId;
};
