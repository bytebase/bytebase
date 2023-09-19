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
  authenticationDatabase: string;
  // sid and serviceName are used for Oracle database. Required one of them.
  sid: string;
  serviceName: string;
  // Connection over SSH.
  sshHost: string;
  sshPort: string;
  sshUser: string;
  sshPassword: string;
  sshPrivateKey: string;
};

export type DataSource = {
  id: DataSourceId;

  // Related fields
  databaseId: DatabaseId;
  instanceId: InstanceId;

  // Domain specific fields
  name: string;
  type: DataSourceType;
  // In mysql, username can be empty which means anonymous user
  username: string;
  password?: string;
  sslCa?: string;
  sslCert?: string;
  sslKey?: string;
  host: string;
  port: string;
  database: string;

  options: DataSourceOptions;
  // UI-only fields
  updateSsl?: boolean;
  updateSsh?: boolean;
};

export type DataSourceCreate = {
  // Related fields
  databaseId: DatabaseId;
  instanceId: InstanceId;

  // Domain specific fields
  name: string;
  type: DataSourceType;
  username: string;
  password?: string;
  sslCa?: string;
  sslCert?: string;
  sslKey?: string;
  host: string;
  port: string;
  database: string;
  options: DataSourceOptions;
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
  host?: string;
  port?: string;
  database?: string;
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
