import { isEqual } from "lodash-es";
import { defineStore } from "pinia";
import { ref } from "vue";
import { projectServiceClient } from "@/grpcweb";
import { ConditionGroupExpr, buildCELExpr } from "@/plugins/cel";
import {
  ComposedSchemaGroupTable,
  ComposedDatabaseGroup,
  ComposedSchemaGroup,
  ComposedDatabase,
} from "@/types";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import { Environment } from "@/types/proto/v1/environment_service";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { convertParsedExprToCELString } from "@/utils";
import { convertDatabaseGroupExprFromCEL } from "@/utils/databaseGroup/cel";
import { buildDatabaseGroupExpr } from "@/utils/databaseGroup/cel";
import {
  useEnvironmentV1Store,
  useProjectV1Store,
  useDatabaseV1Store,
} from "./v1";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
  getProjectNameAndDatabaseGroupNameAndSchemaGroupName,
  projectNamePrefix,
  schemaGroupNamePrefix,
} from "./v1/common";

const composeDatabaseGroup = async (
  databaseGroup: DatabaseGroup
): Promise<ComposedDatabaseGroup> => {
  const [projectName, databaseGroupName] = getProjectNameAndDatabaseGroupName(
    databaseGroup.name
  );
  const expression = databaseGroup.databaseExpr?.expression ?? "";
  const convertResult = await convertDatabaseGroupExprFromCEL(expression);
  const project = await useProjectV1Store().getOrFetchProjectByName(
    `${projectNamePrefix}${projectName}`
  );
  const environment = useEnvironmentV1Store().getEnvironmentByName(
    convertResult.environmentId
  ) as Environment;

  return {
    ...databaseGroup,
    databaseGroupName: databaseGroupName,
    projectName: projectName,
    project: project,
    environmentName: convertResult.environmentId ?? "",
    environment: environment,
    simpleExpr: convertResult.conditionGroupExpr,
  };
};

const composeSchemaGroup = async (
  schemaGroup: SchemaGroup
): Promise<ComposedSchemaGroup> => {
  const [projectName, databaseGroupName] =
    getProjectNameAndDatabaseGroupNameAndSchemaGroupName(schemaGroup.name);
  const databaseGroup = await composeDatabaseGroup(
    await useDBGroupStore().getOrFetchDBGroupByName(
      `${projectNamePrefix}${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`
    )
  );
  const composedData: ComposedSchemaGroup = {
    ...schemaGroup,
    databaseGroup: databaseGroup,
  };
  return composedData;
};

export const useDBGroupStore = defineStore("db-group", () => {
  const dbGroupMapByName = ref<Map<string, ComposedDatabaseGroup>>(new Map());
  const schemaGroupMapByName = ref<Map<string, ComposedSchemaGroup>>(new Map());
  const cachedProjectNameSet = ref<Set<string>>(new Set());
  const cachedDatabaseGroupNameSet = ref<Set<string>>(new Set());

  const fetchAllDatabaseGroupList = async () => {
    const { databaseGroups } = await projectServiceClient.listDatabaseGroups({
      parent: `${projectNamePrefix}-`,
    });
    const composedList = [];
    for (const dbGroup of databaseGroups) {
      const composedData = await composeDatabaseGroup(dbGroup);
      dbGroupMapByName.value.set(dbGroup.name, composedData);
      composedList.push(composedData);
    }
    return composedList;
  };

  const getAllDatabaseGroupList = () => {
    return Array.from(dbGroupMapByName.value.values());
  };

  const getOrFetchDBGroupByName = async (name: string, silent = false) => {
    const cached = dbGroupMapByName.value.get(name);
    if (cached) return cached;

    const databaseGroup = await projectServiceClient.getDatabaseGroup(
      {
        name: name,
      },
      { silent }
    );
    const composedData = await composeDatabaseGroup(databaseGroup);
    dbGroupMapByName.value.set(name, composedData);
    return composedData;
  };

  const getOrFetchDBGroupListByProjectName = async (projectName: string) => {
    const hasCache = cachedProjectNameSet.value.has(projectName);
    if (hasCache) {
      return Array.from(dbGroupMapByName.value.values()).filter((dbGroup) =>
        dbGroup.name.startsWith(projectName)
      );
    }

    const { databaseGroups } = await projectServiceClient.listDatabaseGroups({
      parent: projectName,
    });
    const composedList = [];
    for (const dbGroup of databaseGroups) {
      const composedData = await composeDatabaseGroup(dbGroup);
      dbGroupMapByName.value.set(dbGroup.name, composedData);
      composedList.push(composedData);
    }
    cachedProjectNameSet.value.add(projectName);
    return composedList;
  };

  const getDBGroupListByProjectName = (projectName: string) => {
    return Array.from(dbGroupMapByName.value.values()).filter((dbGroup) =>
      dbGroup.name.startsWith(projectName)
    );
  };

  const getDBGroupByName = (name: string) => {
    return dbGroupMapByName.value.get(name);
  };

  const createDatabaseGroup = async ({
    projectName,
    databaseGroup,
    databaseGroupId,
    validateOnly = false,
  }: {
    projectName: string;
    databaseGroup: Pick<
      DatabaseGroup,
      "name" | "databasePlaceholder" | "databaseExpr"
    >;
    databaseGroupId: string;
    validateOnly?: boolean;
  }) => {
    const createdDatabaseGroup = await projectServiceClient.createDatabaseGroup(
      {
        parent: projectName,
        databaseGroup,
        databaseGroupId,
        validateOnly,
      }
    );

    if (!validateOnly) {
      const composedData = await composeDatabaseGroup(createdDatabaseGroup);
      dbGroupMapByName.value.set(createdDatabaseGroup.name, composedData);
    }
    return createdDatabaseGroup;
  };

  const fetchDatabaseGroupMatchList = async ({
    projectName,
    environmentId,
    expr,
  }: {
    projectName: string;
    environmentId: string;
    expr: ConditionGroupExpr;
  }) => {
    const environment =
      useEnvironmentV1Store().getEnvironmentByUID(environmentId);

    const celString = await convertParsedExprToCELString(
      ParsedExpr.fromJSON({
        expr: buildCELExpr(
          buildDatabaseGroupExpr({
            environmentId: environment.name,
            conditionGroupExpr: expr,
          })
        ),
      })
    );

    const validateOnlyResourceId = `creating-database-group-${Date.now()}`;

    const result = await createDatabaseGroup({
      projectName: projectName,
      databaseGroup: {
        name: `${projectName}/${databaseGroupNamePrefix}${validateOnlyResourceId}`,
        databasePlaceholder: validateOnlyResourceId,
        databaseExpr: Expr.fromJSON({
          expression: celString,
        }),
      },
      databaseGroupId: validateOnlyResourceId,
      validateOnly: true,
    });

    const matchedDatabaseList: ComposedDatabase[] = [];
    const unmatchedDatabaseList: ComposedDatabase[] = [];
    const databaseStore = useDatabaseV1Store();

    for (const item of result.matchedDatabases) {
      const database = await databaseStore.getOrFetchDatabaseByName(item.name);
      if (!database) {
        continue;
      }

      matchedDatabaseList.push(database);
    }
    for (const item of result.unmatchedDatabases) {
      const database = await databaseStore.getOrFetchDatabaseByName(item.name);
      if (
        database &&
        database.instanceEntity.environmentEntity.uid === environmentId
      ) {
        unmatchedDatabaseList.push(database);
      }
    }

    return {
      matchedDatabaseList,
      unmatchedDatabaseList,
    };
  };

  const updateDatabaseGroup = async (
    databaseGroup: Pick<
      DatabaseGroup,
      "name" | "databasePlaceholder" | "databaseExpr"
    >
  ) => {
    const rawDatabaseGroup = dbGroupMapByName.value.get(databaseGroup.name);
    if (!rawDatabaseGroup) {
      throw new Error("Database group not found");
    }
    const updateMask: string[] = [];
    if (
      !isEqual(
        rawDatabaseGroup.databasePlaceholder,
        databaseGroup.databasePlaceholder
      )
    ) {
      updateMask.push("database_placeholder");
    }
    if (!isEqual(rawDatabaseGroup.databaseExpr, databaseGroup.databaseExpr)) {
      updateMask.push("database_expr");
    }
    const updatedDatabaseGroup = await projectServiceClient.updateDatabaseGroup(
      {
        databaseGroup,
        updateMask,
      }
    );
    const composedData = await composeDatabaseGroup(updatedDatabaseGroup);
    dbGroupMapByName.value.set(updatedDatabaseGroup.name, composedData);
    return updatedDatabaseGroup;
  };

  const deleteDatabaseGroup = async (name: string) => {
    await projectServiceClient.deleteDatabaseGroup({
      name: name,
    });
    dbGroupMapByName.value.delete(name);
  };

  const getOrFetchSchemaGroupByName = async (name: string, silent = false) => {
    const cached = schemaGroupMapByName.value.get(name);
    if (cached) return cached;

    const schemaGroup = await projectServiceClient.getSchemaGroup(
      {
        name: name,
      },
      { silent }
    );
    const composedData = await composeSchemaGroup(schemaGroup);
    schemaGroupMapByName.value.set(name, composedData);
    return schemaGroup;
  };

  const getOrFetchSchemaGroupListByDBGroupName = async (
    dbGroupName: string
  ) => {
    const hasCache = cachedDatabaseGroupNameSet.value.has(dbGroupName);
    if (hasCache) {
      return Array.from(schemaGroupMapByName.value.values()).filter(
        (schemaGroup) => schemaGroup.name.startsWith(dbGroupName)
      );
    }
    const { schemaGroups } = await projectServiceClient.listSchemaGroups({
      parent: dbGroupName,
    });
    const composedList: ComposedSchemaGroup[] = [];
    for (const schemaGroup of schemaGroups) {
      const composedData = await composeSchemaGroup(schemaGroup);
      schemaGroupMapByName.value.set(schemaGroup.name, composedData);
      composedList.push(composedData);
    }
    cachedDatabaseGroupNameSet.value.add(dbGroupName);
    return composedList;
  };

  const getSchemaGroupListByDBGroupName = (dbGroupName: string) => {
    return Array.from(schemaGroupMapByName.value.values()).filter(
      (schemaGroup) => schemaGroup.name.startsWith(dbGroupName)
    );
  };

  const getSchemaGroupByName = (name: string) => {
    return schemaGroupMapByName.value.get(name);
  };

  const createSchemaGroup = async ({
    dbGroupName,
    schemaGroup,
    schemaGroupId,
    validateOnly = false,
  }: {
    dbGroupName: string;
    schemaGroup: Pick<SchemaGroup, "name" | "tablePlaceholder" | "tableExpr">;
    schemaGroupId: string;
    validateOnly?: boolean;
  }) => {
    const createdSchemaGroup = await projectServiceClient.createSchemaGroup({
      parent: dbGroupName,
      schemaGroup,
      schemaGroupId,
      validateOnly,
    });
    const composedData = await composeSchemaGroup(createdSchemaGroup);
    if (!validateOnly) {
      schemaGroupMapByName.value.set(composedData.name, composedData);
    }
    return composedData;
  };

  const fetchSchemaGroupMatchList = async ({
    projectName,
    databaseGroupName,
    expr,
  }: {
    projectName: string;
    databaseGroupName: string;
    expr: ConditionGroupExpr;
  }) => {
    const celString = await convertParsedExprToCELString(
      ParsedExpr.fromJSON({
        expr: buildCELExpr(expr),
      })
    );
    const validateOnlyResourceId = `creating-schema-group-${Date.now()}`;
    const parent = `${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`;

    try {
      const result = await createSchemaGroup({
        dbGroupName: parent,
        schemaGroup: {
          name: `${databaseGroupName}/${schemaGroupNamePrefix}${validateOnlyResourceId}`,
          tablePlaceholder: validateOnlyResourceId,
          tableExpr: Expr.fromJSON({
            expression: celString || "true",
          }),
        },
        schemaGroupId: validateOnlyResourceId,
        validateOnly: true,
      });

      const matchedTableList: ComposedSchemaGroupTable[] = [];
      const unmatchedTableList: ComposedSchemaGroupTable[] = [];
      const databaseStore = useDatabaseV1Store();

      for (const item of result.matchedTables) {
        const database = await databaseStore.getOrFetchDatabaseByName(
          item.database
        );
        if (!database) {
          continue;
        }

        matchedTableList.push({
          ...item,
          databaseEntity: database,
        });
      }
      for (const item of result.unmatchedTables) {
        const database = await databaseStore.getOrFetchDatabaseByName(
          item.database
        );
        unmatchedTableList.push({
          ...item,
          databaseEntity: database,
        });
      }

      return {
        matchedTableList,
        unmatchedTableList,
      };
    } catch (e) {
      console.error(e);
      return {
        matchedTableList: [],
        unmatchedTableList: [],
      };
    }
  };

  const updateSchemaGroup = async (
    schemaGroup: Pick<SchemaGroup, "name" | "tablePlaceholder" | "tableExpr">
  ) => {
    const rawSchemaGroup = schemaGroupMapByName.value.get(schemaGroup.name);
    if (!rawSchemaGroup) {
      throw new Error("Schema group not found");
    }
    const updateMask: string[] = [];
    if (
      !isEqual(rawSchemaGroup.tablePlaceholder, schemaGroup.tablePlaceholder)
    ) {
      updateMask.push("table_placeholder");
    }
    if (!isEqual(rawSchemaGroup.tableExpr, schemaGroup.tableExpr)) {
      updateMask.push("table_expr");
    }
    const updatedSchemaGroup = await projectServiceClient.updateSchemaGroup({
      schemaGroup,
      updateMask,
    });
    const composedData = await composeSchemaGroup(updatedSchemaGroup);
    schemaGroupMapByName.value.set(composedData.name, composedData);
    return composedData;
  };

  const deleteSchemaGroup = async (name: string) => {
    await projectServiceClient.deleteSchemaGroup({
      name: name,
    });
    schemaGroupMapByName.value.delete(name);
  };

  return {
    fetchAllDatabaseGroupList,
    getAllDatabaseGroupList,
    getOrFetchDBGroupByName,
    getOrFetchDBGroupListByProjectName,
    getDBGroupListByProjectName,
    getDBGroupByName,
    createDatabaseGroup,
    updateDatabaseGroup,
    deleteDatabaseGroup,
    getOrFetchSchemaGroupByName,
    getOrFetchSchemaGroupListByDBGroupName,
    getSchemaGroupListByDBGroupName,
    getSchemaGroupByName,
    createSchemaGroup,
    updateSchemaGroup,
    deleteSchemaGroup,
    fetchDatabaseGroupMatchList,
    fetchSchemaGroupMatchList,
  };
});
