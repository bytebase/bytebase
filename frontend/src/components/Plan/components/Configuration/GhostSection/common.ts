import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { getInstanceResource, semverCompare } from "@/utils";

export const GHOST_AVAILABLE_ENGINES = [Engine.MYSQL, Engine.MARIADB];

export const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.6.0";

export const MIN_GHOST_SUPPORT_MARIADB_VERSION = "10.6.0";

export const allowGhostForDatabase = (database: Database) => {
  const instanceResource = getInstanceResource(database);
  const engine = instanceResource.engine;
  return (
    (engine === Engine.MYSQL &&
      semverCompare(
        instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION,
        "gte"
      )) ||
    (engine === Engine.MARIADB &&
      semverCompare(
        instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MARIADB_VERSION,
        "gte"
      ))
  );
};
