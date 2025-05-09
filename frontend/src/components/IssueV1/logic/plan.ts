import { head } from "lodash-es";
import { mockDatabase } from "@/components/IssueV1/logic/utils";
import {
  useDatabaseV1Store,
  useDBGroupStore,
  useEnvironmentV1Store,
  useInstanceResourceByName,
} from "@/store";
import {
  UNKNOWN_ID,
  type ComposedIssue,
  unknownDatabase,
  type ComposedDatabase,
  isValidDatabaseName,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { Plan_Spec } from "@/types/proto/v1/plan_service";
import { extractDatabaseResourceName, extractDatabaseGroupName } from "@/utils";

export const databaseForSpec = (
  issue: ComposedIssue,
  spec: Plan_Spec
): ComposedDatabase => {
  const environmentStore = useEnvironmentV1Store();
  const { createDatabaseConfig, changeDatabaseConfig, exportDataConfig } = spec;
  if (createDatabaseConfig !== undefined) {
    const instanceName = createDatabaseConfig.target;
    const databaseName = createDatabaseConfig.database;
    const { instance } = useInstanceResourceByName(instanceName);
    return {
      ...unknownDatabase(),
      name: `${instanceName}/databases/${databaseName}`,
      databaseName,
      instance: instanceName,
      project: issue.project,
      projectEntity: issue.projectEntity,
      effectiveEnvironment: instance.value.environment,
      effectiveEnvironmentEntity: environmentStore.getEnvironmentByName(
        instance.value.environment
      ),
      instanceResource: instance.value,
    };
  } else if (
    changeDatabaseConfig !== undefined ||
    exportDataConfig !== undefined
  ) {
    // TODO(steven): handle db group and deployment config.
    const target = (changeDatabaseConfig?.target ??
      exportDataConfig?.target) as string;
    const db = useDatabaseV1Store().getDatabaseByName(target);
    if (!isValidDatabaseName(db.name)) {
      return mockDatabase(issue.projectEntity, target);
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
