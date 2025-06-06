import { isGroupingChangeSpec } from "@/components/Plan/logic";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import { semverCompare } from "@/utils";

export const GHOST_AVAILABLE_ENGINES = [Engine.MYSQL, Engine.MARIADB];

export const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.6.0";

export const MIN_GHOST_SUPPORT_MARIADB_VERSION = "10.6.0";

export const allowGhostForDatabase = (database: ComposedDatabase) => {
  return (
    (database.instanceResource.engine === Engine.MYSQL &&
      semverCompare(
        database.instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION,
        "gte"
      )) ||
    (database.instanceResource.engine === Engine.MARIADB &&
      semverCompare(
        database.instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MARIADB_VERSION,
        "gte"
      ))
  );
};

export const allowGhostForSpec = (spec: Plan_Spec | undefined) => {
  const config = spec?.changeDatabaseConfig;
  if (!config) return false;

  return [
    Plan_ChangeDatabaseConfig_Type.MIGRATE,
    Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST,
  ].includes(config.type);
};

export const getGhostEnabledForSpec = (
  spec: Plan_Spec
): boolean | undefined => {
  if (isGroupingChangeSpec(spec)) {
    return undefined;
  }
  const config = spec?.changeDatabaseConfig;
  if (!config) {
    return undefined;
  }
  if (config.type === Plan_ChangeDatabaseConfig_Type.MIGRATE) {
    return false;
  }
  if (config.type === Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST) {
    return true;
  }
  return undefined;
};
