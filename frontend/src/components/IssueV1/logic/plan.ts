import { head } from "lodash-es";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import { UNKNOWN_ID, isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { extractDatabaseResourceName, extractDatabaseGroupName } from "@/utils";

/**
 *
 * @returns empty string if no sheet found
 */
export const sheetNameForSpec = (spec: Plan_Spec): string => {
  return spec.changeDatabaseConfig?.sheet ?? spec.exportDataConfig?.sheet ?? "";
};

export const databaseEngineForSpec = async (
  specOrTarget?: Plan_Spec | string
) => {
  if (!specOrTarget) return Engine.ENGINE_UNSPECIFIED;

  const getTarget = (specOrTarget: Plan_Spec | string) => {
    if (typeof specOrTarget === "string") {
      return specOrTarget;
    }
    const config =
      specOrTarget.changeDatabaseConfig || specOrTarget.exportDataConfig;
    if (!config) {
      return Engine.ENGINE_UNSPECIFIED;
    }
    return config.target;
  };
  const target = getTarget(specOrTarget);

  if (extractDatabaseResourceName(target).databaseName !== String(UNKNOWN_ID)) {
    const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
      target,
      /* silent */ true
    );
    if (isValidDatabaseName(db.name)) {
      return db.instanceResource.engine;
    }
  }
  if (extractDatabaseGroupName(target)) {
    const dbGroupStore = useDBGroupStore();
    const dbGroup = await dbGroupStore.getOrFetchDBGroupByName(target);
    // Might be flaky: use the first database in the db group
    const dbName = head(dbGroup.matchedDatabases)?.name;
    if (dbName) {
      const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
        dbName,
        /* silent */ true
      );
      if (isValidDatabaseName(db.name)) {
        return db.instanceResource.engine;
      }
    }
  }
  return Engine.ENGINE_UNSPECIFIED;
};
