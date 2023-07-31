import { computed, unref } from "vue";
import { keyBy } from "lodash-es";

import {
  EngineType,
  Environment,
  Instance,
  Language,
  languageOfEngine,
  languageOfEngineV1,
  MaybeRef,
} from "../types";
import { Environment as EnvironmentV1 } from "@/types/proto/v1/environment_service";
import { Instance as InstanceV1 } from "@/types/proto/v1/instance_service";

export const supportedEngineList = () => {
  const engines: EngineType[] = [
    "MYSQL",
    "POSTGRES",
    "TIDB",
    "SNOWFLAKE",
    "CLICKHOUSE",
    "MONGODB",
    "REDIS",
    "SPANNER",
    "ORACLE",
    "OCEANBASE",
    "MARIADB",
    "MSSQL",
    "REDSHIFT",
    "DM",
  ];
  return engines;
};

export function instanceName(instance: Instance) {
  let name = instance.name;
  if (instance.rowStatus == "ARCHIVED") {
    name += " (Archived)";
  }
  return name;
}

// Sort the list to put prod items first.
export function sortInstanceList(
  list: Instance[],
  environmentList: Environment[]
): Instance[] {
  return list.sort((a: Instance, b: Instance) => {
    let aEnvIndex = -1;
    let bEnvIndex = -1;

    for (let i = 0; i < environmentList.length; i++) {
      if (environmentList[i].id == a.environment.id) {
        aEnvIndex = i;
      }
      if (environmentList[i].id == b.environment.id) {
        bEnvIndex = i;
      }

      if (aEnvIndex != -1 && bEnvIndex != -1) {
        break;
      }
    }
    return bEnvIndex - aEnvIndex;
  });
}

// Sort the list to put prod items first.
export function sortInstanceListByEnvironmentV1(
  list: Instance[],
  environmentList: EnvironmentV1[]
): Instance[] {
  const environmentMap = keyBy(environmentList, (env) => env.uid);

  return list.sort((a, b) => {
    const aEnvOrder = environmentMap[String(a.environment.id)]?.order ?? -1;
    const bEnvOrder = environmentMap[String(b.environment.id)]?.order ?? -1;

    return bEnvOrder - aEnvOrder;
  });
}

export const useInstanceEditorLanguage = (
  instance: MaybeRef<Instance | undefined>
) => {
  return computed((): Language => {
    return languageOfEngine(unref(instance)?.engine);
  });
};

export const useInstanceV1EditorLanguage = (
  instance: MaybeRef<InstanceV1 | undefined>
) => {
  return computed((): Language => {
    return languageOfEngineV1(unref(instance)?.engine);
  });
};

export const isValidSpannerHost = (host: string) => {
  const RE =
    /^projects\/(?<PROJECT_ID>(?:[a-z]|[-.:]|[0-9])+)\/instances\/(?<INSTANCE_ID>(?:[a-z]|[-]|[0-9])+)$/;
  return RE.test(host);
};

export const instanceHasAlterSchema = (
  instanceOrEngine: Instance | EngineType
): boolean => {
  const engine = engineOfInstance(instanceOrEngine);
  if (engine === "REDIS") return false;
  return true;
};

export const instanceHasBackupRestore = (
  instanceOrEngine: Instance | EngineType
): boolean => {
  const engine = engineOfInstance(instanceOrEngine);
  if (engine === "MONGODB") return false;
  if (engine === "REDIS") return false;
  if (engine === "SPANNER") return false;
  if (engine === "REDSHIFT") return false;
  return true;
};

export const instanceHasReadonlyMode = (
  instanceOrEngine: Instance | EngineType
): boolean => {
  const engine = engineOfInstance(instanceOrEngine);
  if (engine === "MONGODB") return false;
  if (engine === "REDIS") return false;
  return true;
};

export const instanceHasCreateDatabase = (
  instanceOrEngine: Instance | EngineType
): boolean => {
  const engine = engineOfInstance(instanceOrEngine);
  if (engine === "REDIS") return false;
  if (engine === "ORACLE") return false;
  if (engine === "DM") return false;
  return true;
};

export const instanceHasStructuredQueryResult = (
  instanceOrEngine: Instance | EngineType
): boolean => {
  const engine = engineOfInstance(instanceOrEngine);
  if (engine === "MONGODB") return false;
  if (engine === "REDIS") return false;
  return true;
};

export const instanceHasSSL = (
  instanceOrEngine: Instance | EngineType
): boolean => {
  const engine = engineOfInstance(instanceOrEngine);
  return [
    "CLICKHOUSE",
    "MYSQL",
    "TIDB",
    "POSTGRES",
    "REDIS",
    "ORACLE",
    "MARIADB",
    "OCEANBASE",
    "DM",
  ].includes(engine);
};

export const instanceHasSSH = (
  instanceOrEngine: Instance | EngineType
): boolean => {
  const engine = engineOfInstance(instanceOrEngine);
  return [
    "MYSQL",
    "TIDB",
    "MARIADB",
    "OCEANBASE",
    "POSTGRES",
    "REDIS",
  ].includes(engine);
};

export const instanceHasCollationAndCharacterSet = (
  instanceOrEngine: Instance | EngineType
) => {
  const engine = engineOfInstance(instanceOrEngine);

  const excludedList: EngineType[] = [
    "MONGODB",
    "CLICKHOUSE",
    "SNOWFLAKE",
    "REDSHIFT",
  ];
  return !excludedList.includes(engine);
};

export const engineOfInstance = (instanceOrEngine: Instance | EngineType) => {
  if (typeof instanceOrEngine === "string") {
    return instanceOrEngine;
  }
  return instanceOrEngine.engine;
};
