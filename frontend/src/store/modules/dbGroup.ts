import { computedAsync } from "@vueuse/core";
import { head, isEqual } from "lodash-es";
import { defineStore } from "pinia";
import type { MaybeRef } from "vue";
import { computed, ref, unref } from "vue";
import { databaseGroupServiceClient } from "@/grpcweb";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import type {
  ComposedDatabaseGroup,
  ComposedDatabase,
  ComposedProject,
} from "@/types";
import { ParsedExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import { DatabaseGroup } from "@/types/proto/v1/database_group_service";
import { DatabaseGroupView } from "@/types/proto/v1/database_group_service";
import {
  batchConvertParsedExprToCELString,
  batchConvertCELStringToParsedExpr,
} from "@/utils";
import { useProjectV1Store, useDatabaseV1Store } from "./v1";
import {
  databaseGroupNamePrefix,
  getProjectNameAndDatabaseGroupName,
  projectNamePrefix,
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

export const useDBGroupStore = defineStore("db-group", () => {
  // TODO(steven): update cache key with view.
  const dbGroupMapByName = ref<Map<string, ComposedDatabaseGroup>>(new Map());
  const cachedProjectNameSet = ref<Set<string>>(new Set());

  const getAllDatabaseGroupList = () => {
    return Array.from(dbGroupMapByName.value.values());
  };

  const getOrFetchDBGroupByName = async (
    name: string,
    options?: Partial<{
      skipCache: boolean;
      silent: boolean;
      view: DatabaseGroupView;
    }>
  ) => {
    const { skipCache, silent, view } = {
      ...{
        skipCache: false,
        silent: false,
        view: DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC,
      },
      ...options,
    };
    if (!skipCache) {
      const cached = dbGroupMapByName.value.get(name);
      if (cached) return cached;
    }

    const databaseGroup = await databaseGroupServiceClient.getDatabaseGroup(
      {
        name,
        view,
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

    const { databaseGroups } =
      await databaseGroupServiceClient.listDatabaseGroups({
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

  const getDBGroupByName = (
    name: string
  ): ComposedDatabaseGroup | undefined => {
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
      "name" | "databasePlaceholder" | "databaseExpr" | "multitenancy"
    >;
    databaseGroupId: string;
    validateOnly?: boolean;
  }) => {
    const createdDatabaseGroup =
      await databaseGroupServiceClient.createDatabaseGroup(
        {
          parent: projectName,
          databaseGroup,
          databaseGroupId,
          validateOnly,
        },
        {
          silent: validateOnly,
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
        expr: await buildCELExpr(expr),
      }),
    ]);
    const expression = head(celStrings) || "true"; // Fallback to true.
    const validateOnlyResourceId = `creating-database-group-${Date.now()}`;

    const result = await createDatabaseGroup({
      projectName: projectName,
      databaseGroup: DatabaseGroup.fromPartial({
        name: `${projectName}/${databaseGroupNamePrefix}${validateOnlyResourceId}`,
        databasePlaceholder: validateOnlyResourceId,
        databaseExpr: Expr.fromJSON({
          expression,
        }),
      }),
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
      "name" | "databasePlaceholder" | "databaseExpr" | "multitenancy"
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
    if (!isEqual(rawDatabaseGroup.multitenancy, databaseGroup.multitenancy)) {
      updateMask.push("multitenancy");
    }
    const updatedDatabaseGroup =
      await databaseGroupServiceClient.updateDatabaseGroup({
        databaseGroup,
        updateMask,
      });
    const composedData = await batchComposeDatabaseGroup([
      updatedDatabaseGroup,
    ]);
    dbGroupMapByName.value.set(updatedDatabaseGroup.name, composedData[0]);
    return updatedDatabaseGroup;
  };

  const deleteDatabaseGroup = async (name: string) => {
    await databaseGroupServiceClient.deleteDatabaseGroup({
      name: name,
    });
    dbGroupMapByName.value.delete(name);
  };

  return {
    getAllDatabaseGroupList,
    getOrFetchDBGroupByName,
    getOrFetchDBGroupListByProjectName,
    getDBGroupListByProjectName,
    getDBGroupByName,
    createDatabaseGroup,
    updateDatabaseGroup,
    deleteDatabaseGroup,
    fetchDatabaseGroupMatchList,
  };
});

export const useDatabaseInGroupFilter = (
  project: MaybeRef<ComposedProject>,
  referenceDatabase: MaybeRef<ComposedDatabase | undefined>
) => {
  const isPreparingDatabaseGroups = ref(false);

  const databaseGroups = computedAsync(
    async () => {
      const response = await databaseGroupServiceClient.listDatabaseGroups({
        parent: unref(project).name,
      });
      return Promise.all(
        response.databaseGroups.map((group) => {
          return databaseGroupServiceClient.getDatabaseGroup({
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
