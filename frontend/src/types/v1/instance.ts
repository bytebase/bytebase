import { create } from "@bufbuild/protobuf";
import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { Engine, State } from "../proto-es/v1/common_pb";
import type {
  Instance,
  InstanceResource,
} from "../proto-es/v1/instance_service_pb";
import {
  InstanceResourceSchema,
  InstanceSchema,
} from "../proto-es/v1/instance_service_pb";

export const EMPTY_INSTANCE_NAME = `instances/${EMPTY_ID}`;
export const UNKNOWN_INSTANCE_NAME = `instances/${UNKNOWN_ID}`;

export const unknownInstance = (): Instance => {
  const instance = create(InstanceSchema, {
    name: UNKNOWN_INSTANCE_NAME,
    state: State.ACTIVE,
    title: "<<Unknown instance>>",
    engine: Engine.MYSQL,
  });
  return instance;
};

export const unknownInstanceResource = (): InstanceResource => {
  const instance = unknownInstance();
  return create(InstanceResourceSchema, {
    name: UNKNOWN_INSTANCE_NAME,
    engine: instance.engine,
    title: "<<Unknown instance>>",
    activation: true,
    dataSources: [],
  });
};

export const isValidInstanceName = (name: unknown): name is string => {
  return (
    typeof name === "string" &&
    name.startsWith("instances/") &&
    name !== EMPTY_INSTANCE_NAME &&
    name !== UNKNOWN_INSTANCE_NAME
  );
};

export const defaultCharsetOfEngineV1 = (engine: Engine): string => {
  switch (engine) {
    case Engine.CLICKHOUSE:
    case Engine.SNOWFLAKE:
    case Engine.STARROCKS:
      return "";
    case Engine.MYSQL:
    case Engine.TIDB:
    case Engine.MARIADB:
    case Engine.OCEANBASE:
      return "utf8mb4";
    case Engine.POSTGRES:
      return "UTF8";
    case Engine.COCKROACHDB:
      return "UTF8";
    case Engine.MONGODB:
      return "";
    case Engine.SPANNER:
      return "";
    case Engine.REDIS:
      return "";
    case Engine.ORACLE:
      return "UTF8";
    case Engine.MSSQL:
      return "";
    case Engine.REDSHIFT:
      return "UNICODE";
    case Engine.HIVE:
      return "default";
    case Engine.BIGQUERY:
      return "";
    case Engine.DYNAMODB:
      return "";
    case Engine.DATABRICKS:
      return "";
  }
  return "";
};

export const defaultCollationOfEngineV1 = (engine: Engine): string => {
  switch (engine) {
    case Engine.CLICKHOUSE:
    case Engine.SNOWFLAKE:
    case Engine.STARROCKS:
      return "";
    case Engine.MYSQL:
    case Engine.TIDB:
    case Engine.MARIADB:
    case Engine.OCEANBASE:
      return "utf8mb4_general_ci";
    // For postgres, we don't explicitly specify a default since the default might be UNSET (denoted by "C").
    // If that's the case, setting an explicit default such as "en_US.UTF-8" might fail if the instance doesn't
    // install it.
    case Engine.POSTGRES:
      return "";
    case Engine.COCKROACHDB:
      return "";
    case Engine.MONGODB:
      return "";
    case Engine.SPANNER:
      return "";
    case Engine.REDIS:
      return "";
    case Engine.ORACLE:
      return "BINARY_CI";
    case Engine.MSSQL:
      return "";
    case Engine.REDSHIFT:
      return "";
    case Engine.HIVE:
      return "default";
    case Engine.BIGQUERY:
      return "";
    case Engine.DYNAMODB:
      return "";
    case Engine.DATABRICKS:
      return "";
  }
  return "";
};

export function isPostgresFamily(type: Engine): boolean {
  return (
    type == Engine.POSTGRES ||
    type == Engine.REDSHIFT ||
    type == Engine.COCKROACHDB
  );
}
