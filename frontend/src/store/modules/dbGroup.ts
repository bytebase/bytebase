import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { head } from "lodash-es";
import { defineStore } from "pinia";
import type { MaybeRef } from "vue";
import { computed, ref, unref, watch, watchEffect } from "vue";
import { databaseGroupServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { buildCELExpr } from "@/plugins/cel";
import { isValidDatabaseGroupName, unknownDatabaseGroup } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import {
  CreateDatabaseGroupRequestSchema,
  DatabaseGroupSchema,
  DatabaseGroupView,
  DeleteDatabaseGroupRequestSchema,
  GetDatabaseGroupRequestSchema,
  ListDatabaseGroupsRequestSchema,
  UpdateDatabaseGroupRequestSchema,
} from "@/types/proto-es/v1/database_group_service_pb";
import { batchConvertParsedExprToCELString } from "@/utils";
import { useCache } from "../cache";
import { databaseGroupNamePrefix } from "./v1/common";

type DatabaseGroupCacheKey = [string /* name */, DatabaseGroupView];

export const useDBGroupStore = defineStore("db-group", () => {
  const cacheByName = useCache<DatabaseGroupCacheKey, DatabaseGroup>(
    "bb.database-group.by-name"
  );

  const getCacheWithFallback = (
    name: string,
    view: DatabaseGroupView = DatabaseGroupView.UNSPECIFIED
  ): DatabaseGroup | undefined => {
    let views: DatabaseGroupView[] = [];
    switch (view) {
      case DatabaseGroupView.UNSPECIFIED:
      case DatabaseGroupView.BASIC:
        views = [DatabaseGroupView.BASIC, DatabaseGroupView.FULL];
        break;
      case DatabaseGroupView.FULL:
        views = [DatabaseGroupView.FULL];
        break;
    }

    for (const v of views) {
      const entity = cacheByName.getEntity([name, v]);
      if (entity) {
        return entity;
      }
    }
  };

  const removeDatabaseGroupCache = (name: string) => {
    const views = [DatabaseGroupView.BASIC, DatabaseGroupView.FULL];
    for (const view of views) {
      cacheByName.invalidateEntity([name, view]);
    }
  };

  // Cache utils
  const setDatabaseGroupCache = (
    databaseGroup: DatabaseGroup,
    view: DatabaseGroupView
  ) => {
    removeDatabaseGroupCache(databaseGroup.name);
    cacheByName.setEntity([databaseGroup.name, view], databaseGroup);
  };

  const getOrFetchDBGroupByName = async (
    name: string,
    options?: {
      skipCache?: boolean;
      silent?: boolean;
      view?: DatabaseGroupView;
    }
  ) => {
    const {
      skipCache = false,
      silent = false,
      view = DatabaseGroupView.BASIC,
    } = options ?? {};
    if (!skipCache) {
      const cached = getDBGroupByName(name, view);
      if (isValidDatabaseGroupName(cached.name)) {
        return cached;
      }
    }

    const request = create(GetDatabaseGroupRequestSchema, {
      name,
      view,
    });
    const databaseGroup =
      await databaseGroupServiceClientConnect.getDatabaseGroup(request, {
        contextValues: createContextValues().set(silentContextKey, silent),
      });
    setDatabaseGroupCache(databaseGroup, view);
    return databaseGroup;
  };

  const fetchDBGroupListByProjectName = async (
    projectName: string,
    view: DatabaseGroupView
  ) => {
    const request = create(ListDatabaseGroupsRequestSchema, {
      parent: projectName,
      view,
    });
    const { databaseGroups } =
      await databaseGroupServiceClientConnect.listDatabaseGroups(request);
    for (const databaseGroup of databaseGroups) {
      setDatabaseGroupCache(databaseGroup, view);
    }
    return databaseGroups;
  };

  const getDBGroupByName = (
    name: string,
    view?: DatabaseGroupView
  ): DatabaseGroup => {
    return getCacheWithFallback(name, view) ?? unknownDatabaseGroup();
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
      "$typeName" | "name" | "title" | "databaseExpr"
    >;
    databaseGroupId: string;
    validateOnly?: boolean;
  }) => {
    const newDatabaseGroup = create(DatabaseGroupSchema, {
      name: databaseGroup.name,
      title: databaseGroup.title,
      databaseExpr: databaseGroup.databaseExpr,
      matchedDatabases: [],
    });
    const request = create(CreateDatabaseGroupRequestSchema, {
      parent: projectName,
      databaseGroup: newDatabaseGroup,
      databaseGroupId,
      validateOnly,
    });
    const createdDatabaseGroup =
      await databaseGroupServiceClientConnect.createDatabaseGroup(request, {
        contextValues: createContextValues().set(
          silentContextKey,
          validateOnly
        ),
      });
    if (!validateOnly) {
      setDatabaseGroupCache(createdDatabaseGroup, DatabaseGroupView.FULL);
    }
    return createdDatabaseGroup;
  };

  const updateDatabaseGroup = async (
    databaseGroup: DatabaseGroup,
    updateMask: string[]
  ) => {
    const request = create(UpdateDatabaseGroupRequestSchema, {
      databaseGroup,
      updateMask: { paths: updateMask },
    });
    const updatedDatabaseGroup =
      await databaseGroupServiceClientConnect.updateDatabaseGroup(request);
    setDatabaseGroupCache(updatedDatabaseGroup, DatabaseGroupView.FULL);
    return updatedDatabaseGroup;
  };

  const deleteDatabaseGroup = async (name: string) => {
    const request = create(DeleteDatabaseGroupRequestSchema, {
      name: name,
    });
    await databaseGroupServiceClientConnect.deleteDatabaseGroup(request);
    removeDatabaseGroupCache(name);
  };

  const fetchDatabaseGroupMatchList = async ({
    projectName,
    expr,
  }: {
    projectName: string;
    expr: ConditionGroupExpr;
  }) => {
    const celexpr = await buildCELExpr(expr);
    if (!celexpr) {
      return [];
    }
    const celStrings = await batchConvertParsedExprToCELString([celexpr]);
    const expression = head(celStrings) || "true"; // Fallback to true.
    const validateOnlyResourceId = `creating-database-group-${Date.now()}`;

    const result = await createDatabaseGroup({
      projectName: projectName,
      databaseGroup: create(DatabaseGroupSchema, {
        name: `${projectName}/${databaseGroupNamePrefix}${validateOnlyResourceId}`,
        title: validateOnlyResourceId,
        databaseExpr: create(ExprSchema, {
          expression,
        }),
      }),
      databaseGroupId: validateOnlyResourceId,
      validateOnly: true,
    });

    return result.matchedDatabases.map((item) => item.name);
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

export const useDBGroupListByProject = (
  project: MaybeRef<string>,
  view: DatabaseGroupView = DatabaseGroupView.BASIC
) => {
  const store = useDBGroupStore();
  const ready = ref(false);
  const dbGroupList = ref<DatabaseGroup[]>([]);

  watchEffect(() => {
    ready.value = false;
    dbGroupList.value = [];
    store
      .fetchDBGroupListByProjectName(unref(project), view)
      .then((response) => {
        ready.value = true;
        dbGroupList.value = response;
      });
  });

  return { dbGroupList, ready };
};

export const useDatabaseGroupByName = (
  name: MaybeRef<string>,
  view: DatabaseGroupView
) => {
  const store = useDBGroupStore();
  const ready = ref(true);
  watch(
    () => unref(name),
    (name) => {
      ready.value = false;
      store.getOrFetchDBGroupByName(name, { view }).then(() => {
        ready.value = true;
      });
    },
    { immediate: true }
  );
  const databaseGroup = computed(() => store.getDBGroupByName(unref(name)));

  return {
    databaseGroup,
    ready,
  };
};
