import { computed, unref } from "vue";
import { t } from "@/plugins/i18n";
import { useSubscriptionV1Store } from "@/store";
import type { MaybeRef } from "@/types";
import {
  isValidInstanceName,
  languageOfEngineV1,
  unknownInstance,
} from "@/types";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
// Using proto-es types directly, no conversions needed
import type {
  Instance,
  InstanceResource,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  type DataSource,
  DataSourceType,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";

export function instanceV1Name(instance: Instance | InstanceResource) {
  const store = useSubscriptionV1Store();
  let name = instance.title;
  // For unknown instance, we will use the name as the title.
  if (instance.title === unknownInstance().title) {
    name = extractInstanceResourceName(instance.name);
  }
  if ((instance as Instance).state === State.DELETED) {
    name += ` (${t("common.archived")})`;
  } else if (
    isValidInstanceName(instance.name) &&
    !instance.activation &&
    store.currentPlan !== PlanType.FREE
  ) {
    name += ` (${t("common.no-license")})`;
  }
  return name;
}

export const extractInstanceResourceName = (name: string) => {
  const pattern = /(?:^|\/)instances\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const readableDataSourceType = (type: DataSourceType): string => {
  if (type === DataSourceType.ADMIN) {
    return t("data-source.admin");
  } else if (type === DataSourceType.READ_ONLY) {
    return t("data-source.read-only");
  } else {
    return "Unknown";
  }
};

export const hostPortOfDataSource = (ds: DataSource | undefined): string => {
  if (!ds) {
    return "";
  }
  const parts = [ds.host];
  if (ds.port) {
    parts.push(ds.port);
  }
  return parts.join(":");
};

export const hostPortOfInstanceV1 = (instance: Instance | InstanceResource) => {
  const ds =
    instance.dataSources.find((ds) => ds.type === DataSourceType.ADMIN) ??
    instance.dataSources[0];
  return hostPortOfDataSource(ds);
};

export const supportedEngineV1List = () => {
  const engines: Engine[] = [
    Engine.MYSQL,
    Engine.POSTGRES,
    Engine.ORACLE,
    Engine.MSSQL,
    Engine.SNOWFLAKE,
    Engine.CLICKHOUSE,
    Engine.MONGODB,
    Engine.REDIS,
    Engine.TIDB,
    Engine.OCEANBASE,
    Engine.SPANNER,
    Engine.REDSHIFT,
    Engine.MARIADB,
    Engine.STARROCKS,
    Engine.HIVE,
    Engine.ELASTICSEARCH,
    Engine.BIGQUERY,
    Engine.DYNAMODB,
    Engine.DATABRICKS,
    Engine.COCKROACHDB,
    Engine.COSMOSDB,
    Engine.CASSANDRA,
    Engine.TRINO,
    Engine.DORIS,
  ];
  return engines;
};

export const instanceV1HasAlterSchema = (
  instanceOrEngine: Instance | InstanceResource | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  if (engine === Engine.REDIS) return false;
  return true;
};

export const instanceV1HasReadonlyMode = (
  _instanceOrEngine: Instance | InstanceResource | Engine
): boolean => {
  // For MongoDB and Redis, we rely on users setting up read-only data source for queries.
  return true;
};

export const enginesSupportCreateDatabase = () => {
  return [
    Engine.MYSQL,
    Engine.POSTGRES,
    Engine.MSSQL,
    Engine.SNOWFLAKE,
    Engine.CLICKHOUSE,
    Engine.MONGODB,
    Engine.TIDB,
    Engine.OCEANBASE,
    Engine.REDSHIFT,
    Engine.MARIADB,
    Engine.STARROCKS,
    Engine.HIVE,
    Engine.COCKROACHDB,
    Engine.DORIS,
  ];
};

export const instanceV1HasCreateDatabase = (
  instanceOrEngine: Instance | InstanceResource | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return enginesSupportCreateDatabase().includes(engine);
};

export const instanceV1HasSSL = (
  instanceOrEngine: Instance | InstanceResource | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [
    Engine.CLICKHOUSE,
    Engine.MYSQL,
    Engine.TIDB,
    Engine.POSTGRES,
    Engine.COCKROACHDB,
    Engine.REDIS,
    Engine.ORACLE,
    Engine.MARIADB,
    Engine.OCEANBASE,
    Engine.STARROCKS,
    Engine.DORIS,
    Engine.MONGODB,
    Engine.ELASTICSEARCH,
    Engine.MSSQL,
    Engine.CASSANDRA,
    Engine.TRINO,
  ].includes(engine);
};

export const instanceV1HasSSH = (
  instanceOrEngine: Instance | InstanceResource | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [
    Engine.MYSQL,
    Engine.TIDB,
    Engine.MARIADB,
    Engine.OCEANBASE,
    Engine.POSTGRES,
    Engine.REDIS,
  ].includes(engine);
};

export const instanceV1HasExtraParameters = (
  instanceOrEngine: Instance | InstanceResource | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [
    Engine.MYSQL,
    Engine.MARIADB,
    Engine.OCEANBASE,
    Engine.POSTGRES,
    Engine.ORACLE,
    Engine.MSSQL,
    Engine.MONGODB,
  ].includes(engine);
};

export const instanceV1HasCollationAndCharacterSet = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);

  const excludedList: Engine[] = [
    Engine.MONGODB,
    Engine.CLICKHOUSE,
    Engine.SNOWFLAKE,
    Engine.REDSHIFT,
    Engine.STARROCKS,
    Engine.DORIS,
    Engine.COSMOSDB,
  ];
  return !excludedList.includes(engine);
};

export const instanceV1AllowsCrossDatabaseQuery = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [
    Engine.MYSQL,
    Engine.TIDB,
    Engine.CLICKHOUSE,
    Engine.MARIADB,
    Engine.OCEANBASE,
    Engine.STARROCKS,
    Engine.DORIS,
  ].includes(engine);
};

export const instanceV1AllowsExplain = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [
    Engine.MYSQL,
    Engine.TIDB,
    Engine.CLICKHOUSE,
    Engine.COCKROACHDB,
    Engine.HIVE,
    Engine.MSSQL,
    Engine.ORACLE,
    Engine.POSTGRES,
    Engine.REDSHIFT,
    Engine.SNOWFLAKE,
    Engine.STARROCKS,
    Engine.SPANNER,
  ].includes(engine);
};

export const instanceV1AllowsReorderColumns = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const instanceV1SupportsTablePartition = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const instanceV1SupportsExternalTable = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.POSTGRES, Engine.HIVE].includes(engine);
};

export const instanceV1SupportsPackage = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.ORACLE].includes(engine);
};

export const instanceV1SupportsSequence = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.POSTGRES].includes(engine);
};

export const instanceV1SupportsTrigger = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.MYSQL].includes(engine);
};

export const instanceV1SupportsColumn = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return ![Engine.MONGODB, Engine.COSMOSDB].includes(engine);
};

export const instanceV1SupportsIndex = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return ![Engine.SNOWFLAKE].includes(engine);
};

export const instanceV1MaskingForNoSQL = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.MONGODB, Engine.COSMOSDB].includes(engine);
};

export const engineOfInstanceV1 = (
  instanceOrEngine: Instance | InstanceResource | Engine
): Engine => {
  if (typeof instanceOrEngine === "number") {
    return instanceOrEngine;
  }
  return instanceOrEngine.engine;
};

export const engineNameV1 = (type: Engine): string => {
  switch (type) {
    case Engine.CLICKHOUSE:
      return "ClickHouse";
    case Engine.MYSQL:
      return "MySQL";
    case Engine.POSTGRES:
      return "PostgreSQL";
    case Engine.COCKROACHDB:
      return "CockroachDB";
    case Engine.SNOWFLAKE:
      return "Snowflake";
    case Engine.TIDB:
      return "TiDB";
    case Engine.MONGODB:
      return "MongoDB";
    case Engine.SPANNER:
      return "Spanner";
    case Engine.REDIS:
      return "Redis";
    case Engine.ORACLE:
      return "Oracle";
    case Engine.MSSQL:
      return "MSSQL";
    case Engine.REDSHIFT:
      return "Redshift";
    case Engine.MARIADB:
      return "MariaDB";
    case Engine.OCEANBASE:
      return "OceanBase (MySQL)";
    case Engine.STARROCKS:
      return "StarRocks";
    case Engine.DORIS:
      return "Doris";
    case Engine.HIVE:
      return "Hive";
    case Engine.ELASTICSEARCH:
      return "Elasticsearch";
    case Engine.BIGQUERY:
      return "BigQuery";
    case Engine.DYNAMODB:
      return "DynamoDB";
    case Engine.DATABRICKS:
      return "Databricks";
    case Engine.COSMOSDB:
      return "CosmosDB";
    case Engine.CASSANDRA:
      return "Cassandra";
    case Engine.TRINO:
      return "Trino";
  }
  return "";
};

export const hasSchemaProperty = (databaseEngine: Engine) => {
  return (
    databaseEngine === Engine.POSTGRES ||
    databaseEngine === Engine.SNOWFLAKE ||
    databaseEngine === Engine.MSSQL ||
    databaseEngine === Engine.REDSHIFT ||
    databaseEngine === Engine.COCKROACHDB ||
    databaseEngine === Engine.SPANNER ||
    databaseEngine === Engine.TRINO ||
    databaseEngine === Engine.DATABRICKS
  );
};

export const instanceAllowsSchemaScopedQuery = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return engine !== Engine.MSSQL && hasSchemaProperty(engine);
};

export const hasTableEngineProperty = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return ![
    Engine.POSTGRES,
    Engine.COCKROACHDB,
    Engine.SNOWFLAKE,
    Engine.MONGODB,
    Engine.COSMOSDB,
  ].includes(engine);
};

export const hasIndexSizeProperty = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return ![Engine.CLICKHOUSE, Engine.SNOWFLAKE].includes(engine);
};

export const useInstanceV1EditorLanguage = (
  instance: MaybeRef<Instance | InstanceResource | undefined>
) => {
  return computed(() => {
    return languageOfEngineV1(unref(instance)?.engine);
  });
};

export const isValidSpannerHost = (host: string) => {
  const RE =
    /^projects\/(?<PROJECT_ID>(?:[a-z]|[-.:]|[0-9])+)\/instances\/(?<INSTANCE_ID>(?:[a-z]|[-]|[0-9])+)$/;
  return RE.test(host);
};

export const getFixedPrimaryKey = (engine: Engine) => {
  // For MySQL and TiDB, the name of a primary key is always PRIMARY.
  if ([Engine.MYSQL, Engine.TIDB].includes(engine)) {
    return "PRIMARY";
  }
  return undefined;
};

// supportGetStringSchema returns true if the engine supports getting schema string.
export const supportGetStringSchema = (engine: Engine) => {
  return [
    Engine.MYSQL,
    Engine.OCEANBASE,
    Engine.POSTGRES,
    Engine.TIDB,
    Engine.CLICKHOUSE,
    Engine.REDSHIFT,
    Engine.ORACLE,
    Engine.MSSQL,
  ].includes(engine);
};

// instanceV1SupportsTransactionMode returns true if the engine supports configurable transaction mode.
export const instanceV1SupportsTransactionMode = (engine: Engine): boolean => {
  // Exclude NoSQL databases that don't have traditional SQL transactions
  const noSQLEngines = [
    Engine.MONGODB,
    Engine.REDIS,
    Engine.DYNAMODB,
    Engine.COSMOSDB,
    Engine.CASSANDRA,
    Engine.ELASTICSEARCH,
  ];

  return !noSQLEngines.includes(engine);
};

// getDefaultTransactionMode returns the default transaction mode.
// This mirrors the backend logic in backend/common/engine.go
// All engines default to "on" (transactional) for safety and backward compatibility.
// Users can explicitly set "-- txn-mode = off" when needed for engines with limited transactional DDL support.
export const getDefaultTransactionMode = (): boolean => {
  // All engines default to "on" for safety and backward compatibility
  return true;
};
