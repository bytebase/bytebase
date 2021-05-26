import { RowStatus } from "./common";
import { Environment } from "./environment";
import { EnvironmentId, InstanceId, PrincipalId } from "./id";
import { Principal } from "./principal";

export type EngineType = "MYSQL";

export type Instance = {
  id: InstanceId;

  // Related fields
  environment: Environment;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;
  rowStatus: RowStatus;

  // Domain specific fields
  name: string;
  engine: EngineType;
  externalLink?: string;
  host: string;
  port?: string;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
};

export type InstanceCreate = {
  // Related fields
  environmentId: EnvironmentId;

  // Standard fields
  creatorId: PrincipalId;

  // Domain specific fields
  name: string;
  externalLink?: string;
  host: string;
  port?: string;
  // In mysql, username can be empty which means anonymous user
  username?: string;
  password?: string;
};

export type InstancePatch = {
  // Standard fields
  updaterId: PrincipalId;
  rowStatus?: RowStatus;

  // Domain specific fields
  name?: string;
  externalLink?: string;
  host?: string;
  port?: string;
  username?: string;
  password?: string;
};
