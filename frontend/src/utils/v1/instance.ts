import slug from "slug";
import { Instance } from "@/types/proto/v1/instance_service";
import { Engine } from "@/types/proto/v1/common";

export const instanceV1Slug = (instance: Instance): string => {
  return [slug(instance.title), instance.uid].join("-");
};

export const extractInstanceResourceName = (name: string) => {
  const pattern = /(?:^|\/)instances\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
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

// export const instanceHasAlterSchema = (
//   instanceOrEngine: Instance | EngineType
// ): boolean => {
//   const engine = engineOfInstance(instanceOrEngine);
//   if (engine === "REDIS") return false;
//   return true;
// };

// export const instanceHasBackupRestore = (
//   instanceOrEngine: Instance | EngineType
// ): boolean => {
//   const engine = engineOfInstance(instanceOrEngine);
//   if (engine === "MONGODB") return false;
//   if (engine === "REDIS") return false;
//   if (engine === "SPANNER") return false;
//   if (engine === "REDSHIFT") return false;
//   return true;
// };

// export const instanceHasReadonlyMode = (
//   instanceOrEngine: Instance | EngineType
// ): boolean => {
//   const engine = engineOfInstance(instanceOrEngine);
//   if (engine === "MONGODB") return false;
//   if (engine === "REDIS") return false;
//   return true;
// };

export const instanceV1HasCreateDatabase = (
  instanceOrEngine: Instance | Engine
): boolean => {
  const engine = engineOfInstanceV1(instanceOrEngine);
  if (engine === Engine.REDIS) return false;
  if (engine === Engine.ORACLE) return false;
  return true;
};

// export const instanceHasStructuredQueryResult = (
//   instanceOrEngine: Instance | EngineType
// ): boolean => {
//   const engine = engineOfInstance(instanceOrEngine);
//   if (engine === "MONGODB") return false;
//   if (engine === "REDIS") return false;
//   return true;
// };

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

// export const instanceHasCollationAndCharacterSet = (
//   instanceOrEngine: Instance | EngineType
// ) => {
//   const engine = engineOfInstance(instanceOrEngine);

//   const excludedList: EngineType[] = [
//     "MONGODB",
//     "CLICKHOUSE",
//     "SNOWFLAKE",
//     "REDSHIFT",
//   ];
//   return !excludedList.includes(engine);
// };

export const engineOfInstanceV1 = (instanceOrEngine: Instance | Engine) => {
  if (typeof instanceOrEngine === "number") {
    return instanceOrEngine;
  }
  return instanceOrEngine.engine;
};
