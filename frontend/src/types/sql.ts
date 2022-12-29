import { EngineType } from ".";
import { InstanceId } from "./id";

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
