import { keyBy, orderBy } from "lodash-es";
import { computed, unref } from "vue";
import { t, locale } from "@/plugins/i18n";
import { useEnvironmentV1Store, useSubscriptionV1Store } from "@/store";
import type { ComposedInstance, MaybeRef } from "@/types";
import { isValidProjectName, languageOfEngineV1 } from "@/types";
import { Engine, State } from "@/types/proto/v1/common";
import type { Environment } from "@/types/proto/v1/environment_service";
import type {
  Instance,
  InstanceResource,
} from "@/types/proto/v1/instance_service";
import {
  DataSourceType,
  type DataSource,
} from "@/types/proto/v1/instance_service";
import { PlanType } from "@/types/proto/v1/subscription_service";

export function instanceV1Name(instance: Instance | InstanceResource) {
  const store = useSubscriptionV1Store();
  let name = instance.title;
  // instance cannot be deleted and activated at the same time.
  if ((instance as Instance).state === State.DELETED) {
    name += ` (${t("common.archived")})`;
  } else if (
    isValidProjectName(instance.name) &&
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

export const sortInstanceV1List = (
  instanceList: (ComposedInstance | InstanceResource)[]
) => {
  return orderBy(
    instanceList,
    [
      (instance) =>
        useEnvironmentV1Store().getEnvironmentByName(instance.environment)
          .order,
      (instance) => instance.name,
      (instance) => instance.title,
    ],
    ["desc", "asc", "asc"]
  );
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

// Sort the list to put prod items first.
export const sortInstanceV1ListByEnvironmentV1 = <T extends Instance>(
  list: T[],
  environmentList: Environment[]
): T[] => {
  const environmentMap = keyBy(environmentList, (env) => env.name);

  return list.sort((a, b) => {
    const aEnvOrder = environmentMap[a.environment]?.order ?? -1;
    const bEnvOrder = environmentMap[b.environment]?.order ?? -1;

    return -(aEnvOrder - bEnvOrder);
  });
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
    Engine.OCEANBASE_ORACLE,
    Engine.SPANNER,
    Engine.REDSHIFT,
    Engine.MARIADB,
    Engine.STARROCKS,
    Engine.RISINGWAVE,
    Engine.HIVE,
    Engine.ELASTICSEARCH,
    Engine.BIGQUERY,
    Engine.DYNAMODB,
    Engine.DATABRICKS,
    Engine.COCKROACHDB,
  ];
  if (locale.value === "zh-CN") {
    engines.push(Engine.DM);
    engines.push(Engine.DORIS);
  }
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

export const instanceV1HasCreateDatabase = (
  instanceOrEngine: Instance | InstanceResource | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  if (engine === Engine.REDIS) return false;
  if (engine === Engine.ORACLE) return false;
  if (engine === Engine.DM) return false;
  if (engine === Engine.ELASTICSEARCH) return false;
  if (engine === Engine.SPANNER) return false;
  if (engine === Engine.BIGQUERY) return false;
  if (engine === Engine.DYNAMODB) return false;
  if (engine == Engine.DATABRICKS) return false;
  return true;
};

export const instanceV1HasStructuredQueryResult = (
  instanceOrEngine: Instance | InstanceResource | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  if (engine === Engine.REDIS) return false;
  return true;
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
    Engine.DM,
    Engine.STARROCKS,
    Engine.DORIS,
    Engine.MONGODB,
    Engine.ELASTICSEARCH,
    Engine.MSSQL,
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

export const instanceV1HasCollationAndCharacterSet = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);

  const excludedList: Engine[] = [
    Engine.MONGODB,
    Engine.CLICKHOUSE,
    Engine.SNOWFLAKE,
    Engine.REDSHIFT,
    Engine.RISINGWAVE,
    Engine.STARROCKS,
    Engine.DORIS,
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

export const instanceV1AllowsReorderColumns = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const instanceV1SupportsConciseSchema = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [Engine.ORACLE, Engine.POSTGRES].includes(engine);
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
  return [Engine.ORACLE, Engine.OCEANBASE_ORACLE].includes(engine);
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

export const engineOfInstanceV1 = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  if (typeof instanceOrEngine === "string") {
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
    case Engine.OCEANBASE_ORACLE:
      return "OceanBase (Oracle)";
    case Engine.DM:
      return "DM";
    case Engine.RISINGWAVE:
      return "RisingWave";
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
  }
  return "";
};

export const hasSchemaProperty = (databaseEngine: Engine) => {
  return (
    databaseEngine === Engine.POSTGRES ||
    databaseEngine === Engine.SNOWFLAKE ||
    databaseEngine === Engine.MSSQL ||
    databaseEngine === Engine.REDSHIFT ||
    databaseEngine === Engine.RISINGWAVE ||
    databaseEngine === Engine.COCKROACHDB ||
    databaseEngine === Engine.SPANNER
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
  return ![Engine.POSTGRES, Engine.COCKROACHDB, Engine.SNOWFLAKE].includes(
    engine
  );
};
export const hasIndexSizeProperty = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return ![Engine.CLICKHOUSE, Engine.SNOWFLAKE].includes(engine);
};
export const hasCollationProperty = (
  instanceOrEngine: Instance | InstanceResource | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return ![
    Engine.POSTGRES,
    Engine.COCKROACHDB,
    Engine.CLICKHOUSE,
    Engine.SNOWFLAKE,
  ].includes(engine);
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
