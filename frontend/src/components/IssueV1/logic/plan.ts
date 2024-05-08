import { head } from "lodash-es";
import {
  useDatabaseV1Store,
  useDBGroupStore,
  useDeploymentConfigV1Store,
} from "@/store";
import { type ComposedProject, UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { Plan_Spec } from "@/types/proto/v1/rollout_service";
import {
  extractDatabaseResourceName,
  extractDatabaseGroupName,
  extractDeploymentConfigName,
  getPipelineFromDeploymentScheduleV1,
} from "@/utils";

/**
 *
 * @returns empty string if no sheet found
 */
export const sheetNameForSpec = (spec: Plan_Spec): string => {
  return spec.changeDatabaseConfig?.sheet ?? spec.exportDataConfig?.sheet ?? "";
};

export const databaseEngineForSpec = async (
  project: ComposedProject,
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
    if (db && db.uid !== String(UNKNOWN_ID)) {
      return db.instanceEntity.engine;
    }
  }
  if (extractDatabaseGroupName(target)) {
    const dbGroup = await useDBGroupStore().getOrFetchDBGroupByName(target);
    // Might be flaky: use the first database in the db group
    const dbName = head(dbGroup.matchedDatabases)?.name;
    if (dbName) {
      const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
        dbName,
        /* silent */ true
      );
      if (db && db.uid !== String(UNKNOWN_ID)) {
        return db.instanceEntity.engine;
      }
    }
  }
  if (extractDeploymentConfigName(target)) {
    const deploymentConfig =
      await useDeploymentConfigV1Store().fetchDeploymentConfigByProjectName(
        project.name
      );
    if (deploymentConfig) {
      const databaseList = useDatabaseV1Store().databaseListByProject(
        project.name
      );
      const pipeline = getPipelineFromDeploymentScheduleV1(
        databaseList,
        deploymentConfig.schedule
      );
      const db = head(head(pipeline));
      if (db && db.uid !== String(UNKNOWN_ID)) {
        return db.instanceEntity.engine;
      }
    }
  }
  return Engine.ENGINE_UNSPECIFIED;
};
