// which from the ops perspective, having different meaning from the normal RW data source.
export type DataSourceType = "ADMIN" | "RW" | "RO";

// DataSourceOptions is the options for a data source.
export interface DataSourceOptions {
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
  authenticationPrivateKey: string;
  // Extra connection parameters for the database connection string
  // For PostgreSQL HA, this can be used to set target_session_attrs=read-write
  extraConnectionParameters?: { [key: string]: string };
}
