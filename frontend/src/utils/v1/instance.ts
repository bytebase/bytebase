import slug from "slug";
import { keyBy, orderBy } from "lodash-es";

import { useI18n } from "vue-i18n";
import { DataSourceType, Instance } from "@/types/proto/v1/instance_service";
import { Engine, State } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";
import { ComposedInstance } from "@/types";
import { useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto/v1/subscription_service";

export const instanceV1Slug = (instance: Instance): string => {
  return [slug(instance.title), instance.uid].join("-");
};

export function instanceV1Name(instance: Instance) {
  const { t } = useI18n();
  const store = useSubscriptionV1Store();
  let name = instance.title;
  // instance cannot be deleted and activated at the same time.
  if (instance.state === State.DELETED) {
    name += ` (${t("common.archived")})`;
  } else if (!instance.activation && store.currentPlan !== PlanType.FREE) {
    name += ` (${t("common.no-license")})`;
  }
  return name;
}

export const extractInstanceResourceName = (name: string) => {
  const pattern = /(?:^|\/)instances\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};

export const sortInstanceV1List = (instanceList: ComposedInstance[]) => {
  return orderBy(
    instanceList,
    [
      (instance) => instance.environmentEntity.order,
      (instance) => Number(instance.uid),
      (instance) => instance.title,
    ],
    ["desc", "asc", "asc"]
  );
};

export const hostPortOfInstanceV1 = (instance: Instance) => {
  const ds =
    instance.dataSources.find((ds) => ds.type === DataSourceType.ADMIN) ??
    instance.dataSources[0];
  if (!ds) {
    return "";
  }
  const parts = [ds.host];
  if (ds.port) {
    parts.push(ds.port);
  }
  return parts.join(":");
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
  const { locale } = useI18n();
  const engines: Engine[] = [
    Engine.MYSQL,
    Engine.POSTGRES,
    Engine.TIDB,
    Engine.SNOWFLAKE,
    Engine.CLICKHOUSE,
    Engine.MONGODB,
    Engine.REDIS,
    Engine.SPANNER,
    Engine.ORACLE,
    Engine.OCEANBASE,
    Engine.MARIADB,
    Engine.MSSQL,
    Engine.REDSHIFT,
  ];
  if (locale.value === "zh-CN") {
    engines.push(Engine.DM);
  }
  return engines;
};

// export const useInstanceEditorLanguage = (
//   instance: MaybeRef<Instance | undefined>
// ) => {
//   return computed((): Language => {
//     return languageOfEngine(unref(instance)?.engine);
//   });
// };

// export const isValidSpannerHost = (host: string) => {
//   const RE =
//     /^projects\/(?<PROJECT_ID>(?:[a-z]|[-.:]|[0-9])+)\/instances\/(?<INSTANCE_ID>(?:[a-z]|[-]|[0-9])+)$/;
//   return RE.test(host);
// };

export const instanceV1HasAlterSchema = (
  instanceOrEngine: Instance | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  if (engine === Engine.REDIS) return false;
  return true;
};

export const instanceV1HasBackupRestore = (
  instanceOrEngine: Instance | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  if (engine === Engine.MONGODB) return false;
  if (engine === Engine.REDIS) return false;
  if (engine === Engine.SPANNER) return false;
  if (engine === Engine.REDSHIFT) return false;
  return true;
};

export const instanceV1HasReadonlyMode = (
  instanceOrEngine: Instance | Engine
): boolean => {
  // For MongoDB and Redis, we rely on users setting up read-only data source for queries.
  return true;
};

export const instanceV1HasCreateDatabase = (
  instanceOrEngine: Instance | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  if (engine === Engine.REDIS) return false;
  if (engine === Engine.ORACLE) return false;
  if (engine === Engine.DM) return false;
  return true;
};

export const instanceV1HasStructuredQueryResult = (
  instanceOrEngine: Instance | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  if (engine === Engine.MONGODB) return false;
  if (engine === Engine.REDIS) return false;
  return true;
};

export const instanceV1HasSSL = (
  instanceOrEngine: Instance | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  return [
    Engine.CLICKHOUSE,
    Engine.MYSQL,
    Engine.TIDB,
    Engine.POSTGRES,
    Engine.REDIS,
    Engine.ORACLE,
    Engine.MARIADB,
    Engine.OCEANBASE,
    Engine.DM,
  ].includes(engine);
};

export const instanceV1HasSSH = (
  instanceOrEngine: Instance | Engine
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
  instanceOrEngine: Instance | Engine
) => {
  const engine = engineOfInstanceV1(instanceOrEngine);

  const excludedList: Engine[] = [
    Engine.MONGODB,
    Engine.CLICKHOUSE,
    Engine.SNOWFLAKE,
    Engine.REDSHIFT,
  ];
  return !excludedList.includes(engine);
};

export const engineOfInstanceV1 = (instanceOrEngine: Instance | Engine) => {
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
      return "OceanBase";
    case Engine.DM:
      return "DM";
  }
  return "";
};

export const formatEngineV1 = (instance: Instance): string => {
  switch (instance.engine) {
    case Engine.POSTGRES:
      return "PostgreSQL";
    // Use MySQL as default engine.
    default:
      return "MySQL";
  }
};
