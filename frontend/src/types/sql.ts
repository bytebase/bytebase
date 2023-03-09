import { EngineType } from "./instance";
import { InstanceId } from "./id";
import { Advice } from "./sqlAdvice";

export type ConnectionInfo = {
  engine: EngineType;
  host: string;
  port?: string;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
  useEmptyPassword: boolean;
  database?: string;
  // Instance detail page has a Test Connection button, if user doesn't input new password, we
  // want the connection to use the existing password to test the connection, however, we do
  // not transfer the password back to client, thus we here pass the instanceId so the server
  // can fetch the corresponding password.
  instanceId?: InstanceId;
  sslCa?: string;
  sslCert?: string;
  sslKey?: string;
  srv: boolean;
  authenticationDatabase: string;
};

export type QueryInfo = {
  instanceId: InstanceId;
  databaseName?: string;
  statement: string;
  limit?: number;
};

// TODO(Jim): not used yet
export type SingleSQLResult = {
  // [columnNames: string[], types: string[], data: any[][], sensitive?: boolean[]]
  data: [string[], string[], any[][], boolean[]];
  error: string;
};

export type SQLResultSet = {
  error: string;
  resultList: SingleSQLResult[];
  adviceList: Advice[];
};
