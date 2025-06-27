import { head } from "lodash-es";
import { mockDatabase } from "@/components/IssueV1/logic/utils";
import { useDatabaseV1Store, useDBGroupStore } from "@/store";
import {
  isValidDatabaseGroupName,
  isValidDatabaseName,
  unknownDatabase,
  type ComposedProject,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";

export const databaseForSpec = (project: ComposedProject, spec: Plan_Spec) => {
  const targets = targetsForSpec(spec);
  if (targets.length === 0) {
    return unknownDatabase();
  }
  const db = useDatabaseV1Store().getDatabaseByName(targets[0]);
  if (!isValidDatabaseName(db.name)) {
    return mockDatabase(project, targets[0]);
  }
  return db;
};

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
    const targets = targetsForSpec(specOrTarget);
    return head(targets);
  };

  const target = getTarget(specOrTarget);
  if (!target) return Engine.ENGINE_UNSPECIFIED;
  if (isValidDatabaseName(target)) {
    const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
      target,
      true /* silent */
    );
    return db.instanceResource.engine;
  }
  if (isValidDatabaseGroupName(target)) {
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

export const isDatabaseChangeSpec = (spec?: Plan_Spec) => {
  if (!spec) return false;
  const config = spec.changeDatabaseConfig || spec.exportDataConfig;
  if (config) {
    return targetsForSpec(spec).every(isValidDatabaseName);
  }
  return false;
};

export const isDBGroupChangeSpec = (spec?: Plan_Spec) => {
  if (!spec) return false;
  const config = spec.changeDatabaseConfig || spec.exportDataConfig;
  if (config) {
    return targetsForSpec(spec).every(isValidDatabaseGroupName);
  }
  return false;
};

export const targetsForSpec = (spec: Plan_Spec): string[] => {
  const config = spec.changeDatabaseConfig || spec.exportDataConfig;
  if (config) {
    return config.targets || [];
  }
  return [];
};
