import { computedAsync } from "@vueuse/core";
import { head } from "lodash-es";
import { defineStore } from "pinia";
import type { MaybeRef } from "vue";
import { computed, ref, unref, watchEffect } from "vue";
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
import { useCache } from "../cache";
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

type DatabaseGroupCacheKey = [string /* name */, DatabaseGroupView];

export const useDBGroupStore = defineStore("db-group", () => {
  const cacheByName = useCache<DatabaseGroupCacheKey, ComposedDatabaseGroup>(
    "bb.database-group.by-name"
  );

  // Cache utils
  const setDatabaseGroupCache = (
    databaseGroup: ComposedDatabaseGroup,
    view: DatabaseGroupView
  ) => {
    if (view === DatabaseGroupView.DATABASE_GROUP_VIEW_FULL) {
      cacheByName.invalidateEntity([
        databaseGroup.name,
        DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC,
      ]);
    }
    cacheByName.setEntity([databaseGroup.name, view], databaseGroup);
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
      const cached = cacheByName.getEntity([name, view]);
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
    setDatabaseGroupCache(response, view);
    return response;
  };

  const fetchDBGroupListByProjectName = async (projectName: string) => {
    const { databaseGroups } =
      await databaseGroupServiceClient.listDatabaseGroups({
        parent: projectName,
      });
    const composedList = [];
    const composeDatabaseGroups =
      await batchComposeDatabaseGroup(databaseGroups);
    for (const composedData of composeDatabaseGroups) {
      setDatabaseGroupCache(
        composedData,
        DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC
      );
      composedList.push(composedData);
    }
    return composedList;
  };

  const getDBGroupByName = (
    name: string,
    view?: DatabaseGroupView
  ): ComposedDatabaseGroup | undefined => {
    if (!view) {
      return (
        cacheByName.getEntity([
          name,
          DatabaseGroupView.DATABASE_GROUP_VIEW_FULL,
        ]) ??
        cacheByName.getEntity([
          name,
          DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC,
        ])
      );
    }
    return cacheByName.getEntity([name, view]);
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
      setDatabaseGroupCache(
        composedData[0],
        DatabaseGroupView.DATABASE_GROUP_VIEW_FULL
      );
    }
    return createdDatabaseGroup;
  };

  const updateDatabaseGroup = async (
    databaseGroup: DatabaseGroup,
    updateMask: string[]
  ) => {
    const updatedDatabaseGroup =
      await databaseGroupServiceClient.updateDatabaseGroup({
        databaseGroup,
        updateMask,
      });
    const composedData = await batchComposeDatabaseGroup([
      updatedDatabaseGroup,
    ]);
    setDatabaseGroupCache(
      composedData[0],
      DatabaseGroupView.DATABASE_GROUP_VIEW_FULL
    );
    return updatedDatabaseGroup;
  };

  const deleteDatabaseGroup = async (name: string) => {
    await databaseGroupServiceClient.deleteDatabaseGroup({
      name: name,
    });
    cacheByName.invalidateEntity([
      name,
      DatabaseGroupView.DATABASE_GROUP_VIEW_FULL,
    ]);
    cacheByName.invalidateEntity([
      name,
      DatabaseGroupView.DATABASE_GROUP_VIEW_BASIC,
    ]);
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

  return {
    getOrFetchDBGroupByName,
    fetchDBGroupListByProjectName,
    getDBGroupByName,
    createDatabaseGroup,
    updateDatabaseGroup,
    deleteDatabaseGroup,
    fetchDatabaseGroupMatchList,
  };
});

export const useDBGroupListByProject = (project: MaybeRef<string>) => {
  const store = useDBGroupStore();
  const ready = ref(false);
  const dbGroupList = ref<ComposedDatabaseGroup[]>([]);

  watchEffect(() => {
    ready.value = false;
    dbGroupList.value = [];
    store.fetchDBGroupListByProjectName(unref(project)).then((response) => {
      ready.value = true;
      dbGroupList.value = response;
    });
  });

  return { dbGroupList, ready };
};

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
