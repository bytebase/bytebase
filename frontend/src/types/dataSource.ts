// For now the ADMIN requires the same database privilege as RW.
// The seperation is to make it explicit which one serves as the ADMIN data source,
import { DatabaseId, DataSourceId, InstanceId } from "./id";

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
