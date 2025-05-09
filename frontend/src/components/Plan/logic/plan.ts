import { mockDatabase } from "@/components/IssueV1/logic/utils";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName, unknownDatabase } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import type { ComposedPlan } from "@/types/v1/issue/plan";
import { extractDatabaseGroupName } from "@/utils";

export const databaseForSpec = (plan: ComposedPlan, spec: Plan_Spec) => {
  // Now we only handle changeDatabaseConfig specs.
  const { changeDatabaseConfig } = spec;
  if (changeDatabaseConfig !== undefined) {
    const target = changeDatabaseConfig.target;
    const db = useDatabaseV1Store().getDatabaseByName(target);
    if (!isValidDatabaseName(db.name)) {
      return mockDatabase(plan.projectEntity, target);
    }
    return db;
  }
  return unknownDatabase();
};

/**
 *
 * @returns empty string if no sheet found
 */
export const sheetNameForSpec = (spec: Plan_Spec): string => {
  return spec.changeDatabaseConfig?.sheet ?? "";
};

export const databaseEngineForSpec = async (
  specOrTarget?: Plan_Spec | string
) => {
  if (!specOrTarget) return Engine.ENGINE_UNSPECIFIED;

  const getTarget = (specOrTarget: Plan_Spec | string) => {
    if (typeof specOrTarget === "string") {
      return specOrTarget;
    }
    const config = specOrTarget.changeDatabaseConfig;
    if (!config) {
      return Engine.ENGINE_UNSPECIFIED;
    }
    return config.target;
  };
  const target = getTarget(specOrTarget);

  if (isValidDatabaseName(target)) {
    const db = await useDatabaseV1Store().getOrFetchDatabaseByName(
      target,
      true /* silent */
    );
    if (isValidDatabaseName(db.name)) {
      return db.instanceResource.engine;
    }
  }

  return Engine.ENGINE_UNSPECIFIED;
};

export const isDatabaseChangeSpec = (spec?: Plan_Spec) => {
  if (!spec) return false;
  return isValidDatabaseName(spec.changeDatabaseConfig?.target);
};

export const isGroupingChangeSpec = (spec?: Plan_Spec) => {
  if (!spec) return false;
  const databaseGroup = extractDatabaseGroupName(
    spec.changeDatabaseConfig?.target ?? ""
  );
  return databaseGroup !== "";
};
