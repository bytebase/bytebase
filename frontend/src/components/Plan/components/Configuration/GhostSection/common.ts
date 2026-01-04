import { isDBGroupChangeSpec } from "@/components/Plan/logic";
import type { ComposedDatabase } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { type Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import { semverCompare } from "@/utils";

export const GHOST_AVAILABLE_ENGINES = [Engine.MYSQL, Engine.MARIADB];

export const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.6.0";

export const MIN_GHOST_SUPPORT_MARIADB_VERSION = "10.6.0";

export const allowGhostForDatabase = (database: ComposedDatabase) => {
  const engine = database.instanceResource.engine;
  return (
    (engine === Engine.MYSQL &&
      semverCompare(
        database.instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION,
        "gte"
      )) ||
    (engine === Engine.MARIADB &&
      semverCompare(
        database.instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MARIADB_VERSION,
        "gte"
      ))
  );
};

export const getGhostEnabledForSpec = (
  spec: Plan_Spec
): boolean | undefined => {
  if (isDBGroupChangeSpec(spec)) {
    return undefined;
  }
  const config =
    spec?.config?.case === "changeDatabaseConfig"
      ? spec.config.value
      : undefined;
  if (!config) {
    return undefined;
  }
  // Ghost is only available for sheet-based migrations (not release-based).
  // Release-based migrations don't support ghost configuration.
  if (!config.release) {
    return config.enableGhost === true;
  }
  return undefined;
};
