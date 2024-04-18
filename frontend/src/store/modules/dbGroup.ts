import { computedAsync } from "@vueuse/core";
import { isEqual } from "lodash-es";
import { defineStore } from "pinia";
import type { MaybeRef } from "vue";
import { computed, ref, unref } from "vue";
import { projectServiceClient } from "@/grpcweb";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import type {
  ComposedSchemaGroupTable,
  ComposedDatabaseGroup,
  ComposedSchemaGroup,
  ComposedDatabase,
  ComposedProject,
} from "@/types";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import type {
  DatabaseGroup,
  SchemaGroup,
} from "@/types/proto/v1/project_service";
import {
  DatabaseGroupView,
  TenantMode,
} from "@/types/proto/v1/project_service";
import {
  batchConvertParsedExprToCELString,
  batchConvertCELStringToParsedExpr,
} from "@/utils";
import { useProjectV1Store, useDatabaseV1Store } from "./v1";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
  getProjectNameAndDatabaseGroupNameAndSchemaGroupName,
  projectNamePrefix,
  schemaGroupNamePrefix,
} from "./v1/common";

const batchComposeDatabaseGroup = async (
  databaseGroupList: DatabaseGroup[]
): Promise<ComposedDatabaseGroup[]> => {
  const composedDatabaseGroupMap: Map<string, ComposedDatabaseGroup> =
    new Map();
  const expressions: string[] = [];
  const composedDatabaseGroupNameList: string[] = [];

  for (const databaseGroup of databaseGroupList) {
    const [projectName, databaseGroupName] = getProjectNameAndDatabaseGroupName(
      databaseGroup.name
    );
    const project = await useProjectV1Store().getOrFetchProjectByName(
      `${projectNamePrefix}${projectName}`
    );

    composedDatabaseGroupMap.set(databaseGroup.name, {
      ...databaseGroup,
      databaseGroupName,
      projectName,
      projectEntity: project,
      simpleExpr: emptySimpleExpr(),
    });

    if (databaseGroup.databaseExpr?.expression) {
      expressions.push(databaseGroup.databaseExpr.expression);
      composedDatabaseGroupNameList.push(databaseGroup.name);
    }
  }

  const exprList = await batchConvertCELStringToParsedExpr(expressions);
  for (let i = 0; i < exprList.length; i++) {
    const databaseGroupName = composedDatabaseGroupNameList[i];

    const celExpr = exprList[i];
    if (celExpr.expr) {
      const simpleExpr = resolveCELExpr(celExpr.expr);
      composedDatabaseGroupMap.get(databaseGroupName)!.simpleExpr =
        wrapAsGroup(simpleExpr);
    }
  }

  return [...composedDatabaseGroupMap.values()];
};

const composeSchemaGroup = async (
  schemaGroup: SchemaGroup
): Promise<ComposedSchemaGroup> => {
  const [projectName, databaseGroupName] =
    getProjectNameAndDatabaseGroupNameAndSchemaGroupName(schemaGroup.name);
  const databaseGroup = await useDBGroupStore().getOrFetchDBGroupByName(
    `${projectNamePrefix}${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`
  );

  const composedData: ComposedSchemaGroup = {
    ...schemaGroup,
    databaseGroup,
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
    const composeDatabaseGroups =
      await batchComposeDatabaseGroup(databaseGroups);
    for (const composedData of composeDatabaseGroups) {
      dbGroupMapByName.value.set(composedData.name, composedData);
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
    const composedData = await batchComposeDatabaseGroup([databaseGroup]);
    const response = composedData[0];
    dbGroupMapByName.value.set(name, response);
    return response;
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
    const composeDatabaseGroups =
      await batchComposeDatabaseGroup(databaseGroups);
    for (const composedData of composeDatabaseGroups) {
      dbGroupMapByName.value.set(composedData.name, composedData);
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
      const composedData = await batchComposeDatabaseGroup([
        createdDatabaseGroup,
      ]);
      dbGroupMapByName.value.set(createdDatabaseGroup.name, composedData[0]);
    }
    return createdDatabaseGroup;
  };

  const fetchDatabaseGroupMatchList = async ({
    projectName,
    expr,
  }: {
    projectName: string;
    expr: ConditionGroupExpr;
  }) => {
    const celStrings = await batchConvertParsedExprToCELString([
      ParsedExpr.fromJSON({
        expr: buildCELExpr(expr),
      }),
    ]);
    const expression = celStrings[0] || "true";
    const validateOnlyResourceId = `creating-database-group-${Date.now()}`;

    const result = await createDatabaseGroup({
      projectName: projectName,
      databaseGroup: {
        name: `${projectName}/${databaseGroupNamePrefix}${validateOnlyResourceId}`,
        databasePlaceholder: validateOnlyResourceId,
        databaseExpr: Expr.fromJSON({
          expression,
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
      if (database) {
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
    const composedData = await batchComposeDatabaseGroup([
      updatedDatabaseGroup,
    ]);
    dbGroupMapByName.value.set(updatedDatabaseGroup.name, composedData[0]);
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
    if (!buildCELExpr(expr)) {
      return {
        matchedTableList: [],
        unmatchedTableList: [],
      };
    }
    const celStrings = await batchConvertParsedExprToCELString([
      ParsedExpr.fromJSON({
        expr: buildCELExpr(expr),
      }),
    ]);
    const validateOnlyResourceId = `creating-schema-group-${Date.now()}`;
    const parent = `${projectName}/${databaseGroupNamePrefix}${databaseGroupName}`;

    try {
      const result = await createSchemaGroup({
        dbGroupName: parent,
        schemaGroup: {
          name: `${databaseGroupName}/${schemaGroupNamePrefix}${validateOnlyResourceId}`,
          tablePlaceholder: validateOnlyResourceId,
          tableExpr: Expr.fromJSON({
            expression: celStrings[0] || "true",
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

export const useDatabaseInGroupFilter = (
  project: MaybeRef<ComposedProject>,
  referenceDatabase: MaybeRef<ComposedDatabase | undefined>
) => {
  const isPreparingDatabaseGroups = ref(false);

  const databaseGroups = computedAsync(
    async () => {
      if (unref(project).tenantMode !== TenantMode.TENANT_MODE_ENABLED) {
        return [];
      }

      const response = await projectServiceClient.listDatabaseGroups({
        parent: unref(project).name,
      });
      return Promise.all(
        response.databaseGroups.map((group) => {
          return projectServiceClient.getDatabaseGroup({
            name: group.name,
            view: DatabaseGroupView.DATABASE_GROUP_VIEW_FULL,
          });
        })
      );
    },
    [],
    {
      evaluating: isPreparingDatabaseGroups,
    }
  );

  const groupsContainRefDB = computed(() => {
    const dbGroups = databaseGroups.value;
    const referenceDB = unref(referenceDatabase);
    if (!referenceDB) {
      return dbGroups;
    }
    return dbGroups.filter((group) => {
      return !!group.matchedDatabases.find(
        (match) => match.name === referenceDB.name
      );
    });
  });

  const databaseFilter = (db: ComposedDatabase) => {
    if (isPreparingDatabaseGroups.value) {
      return false;
    }
    const dbGroups = databaseGroups.value;
    if (!dbGroups) {
      // dbGroups not configured
      // allow all databases
      return true;
    }
    const referenceDB = unref(referenceDatabase);
    if (!referenceDB) {
      // No ref DB
      // allow all databases
      return true;
    }

    if (groupsContainRefDB.value.length === 0) {
      // the referenced DB is not in any group
      // allow all databases
      return true;
    }
    return groupsContainRefDB.value
      .flatMap((group) => group.matchedDatabases.map((match) => match.name))
      .includes(db.name);
  };
  return { isPreparingDatabaseGroups, databaseFilter };
};
