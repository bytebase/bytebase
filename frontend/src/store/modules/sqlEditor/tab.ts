import type { MaybeRef } from "@vueuse/core";
import { head, pick } from "lodash-es";
import { defineStore, storeToRefs } from "pinia";
import { computed, reactive, unref, watch } from "vue";
import type {
  BatchQueryContext,
  CoreSQLEditorTab,
  SQLEditorConnection,
  SQLEditorDatabaseQueryContext,
  SQLEditorTab,
} from "@/types";
import { isValidDatabaseName } from "@/types";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  defaultSQLEditorTab,
  emptySQLEditorConnection,
  extractWorksheetConnection,
  getSheetStatement,
  isDisconnectedSQLEditorTab,
  isSimilarSQLEditorTab,
  useDynamicLocalStorage,
} from "@/utils";
import {
  extractUserId,
  hasFeature,
  useDatabaseV1ByName,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useWorkSheetStore,
} from "../v1";
import { useCurrentUserV1 } from "../v1/auth";
import { useSQLEditorStore } from "./editor";
import { useWebTerminalStore } from "./webTerminal";

const PERSISTENT_TAB_FIELDS = [
  "id",
  "worksheet",
  "mode",
  "batchQueryContext",
  "treeState",
] as const;
export type PersistentTab = Pick<
  SQLEditorTab,
  (typeof PERSISTENT_TAB_FIELDS)[number]
>;

const LOCAL_STORAGE_KEY_PREFIX = "bb.sql-editor-tab";

export const useSQLEditorTabStore = defineStore("sqlEditorTab", () => {
  // re-expose selected project in sqlEditorStore for shortcut
  const { project } = storeToRefs(useSQLEditorStore());
  const tabsById = reactive(new Map<string, SQLEditorTab>());
  const worksheetStore = useWorkSheetStore();

  const me = useCurrentUserV1();
  const userUID = computed(() => extractUserId(me.value.name));
  const keyNamespace = computed(
    () => `${LOCAL_STORAGE_KEY_PREFIX}.${project.value}.${userUID.value}`
  );

  const loadStoredTabs = async () => {
    const validTabList: PersistentTab[] = [];
    for (const tab of openTabList.value) {
      let fullTab: SQLEditorTab | undefined;
      if (tab.worksheet) {
        const worksheet = await worksheetStore.getOrFetchWorksheetByName(
          tab.worksheet,
          true
        );
        if (!worksheet) {
          continue;
        }
        const statement = getSheetStatement(worksheet);
        const connection = await extractWorksheetConnection(worksheet);

        fullTab = {
          ...defaultSQLEditorTab(),
          ...tab,
          connection,
          worksheet: worksheet.name,
          title: worksheet.title,
          statement,
          status: "CLEAN",
        };
      } else {
        const draft = draftTabList.value.find((item) => item.id === tab.id);
        if (!draft) {
          continue;
        }
        fullTab = draft;
      }
      if (!fullTab) {
        continue;
      }

      validTabList.push(tab);
      tabsById.set(tab.id, fullTab);
    }

    openTabList.value = validTabList;
  };

  const draftTabList = useDynamicLocalStorage<SQLEditorTab[]>(
    computed(() => `${keyNamespace.value}.draft-tab-list`),
    [],
    localStorage,
    {
      listenToStorageChanges: true,
    }
  );

  const openTabList = useDynamicLocalStorage<
    PersistentTab[]
  >(
    computed(() => `${keyNamespace.value}.opening-tab-list`),
    [],
    localStorage,
    {
      listenToStorageChanges: false,
    }
  );

  const currentTabId = useDynamicLocalStorage<string>(
    computed(() => `${keyNamespace.value}.current-tab-id`),
    "",
    localStorage,
    {
      listenToStorageChanges: false,
    }
  );

  const maybeInitProject = async () => {
    tabsById.clear();
    await loadStoredTabs();
    currentTabId.value = head(openTabList.value)?.id ?? "";
  };

  const tabById = (id: string) => {
    return tabsById.get(id);
  };

  const tabList = computed(() => {
    return openTabList.value.map((item) => {
      return tabById(item.id) ?? defaultSQLEditorTab();
    });
  });
  const currentTab = computed(() => {
    const currId = currentTabId.value;
    if (!currId) return undefined;
    return tabById(currId);
  });

  const supportBatchMode = computed(() => currentTab.value?.mode !== "ADMIN");

  const isInBatchMode = computed(() => {
    if (!currentTab.value) {
      return false;
    }
    if (!supportBatchMode.value) {
      return false;
    }
    if (!hasFeature(PlanFeature.FEATURE_BATCH_QUERY)) {
      return false;
    }
    const { batchQueryContext } = currentTab.value;
    if (!batchQueryContext) {
      return false;
    }
    const { databaseGroups = [], databases = [] } = batchQueryContext;
    if (!hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS)) {
      return databases.length > 1;
    }
    return databaseGroups.length > 0 || databases.length > 1;
  });

  // actions
  /**
   *
   * @param payload
   * @param beside `true` to add the tab beside currentTab, `false` to add the tab to the last, default to `false`
   * @returns
   */
  const addTab = (payload?: Partial<SQLEditorTab>, beside = false) => {
    const newTab = reactive<SQLEditorTab>({
      ...defaultSQLEditorTab(),
      ...payload,
    });
    const { id, worksheet } = newTab;

    const persistentTab = pick(
      newTab,
      ...PERSISTENT_TAB_FIELDS
    ) as PersistentTab
    const position = openTabList.value.findIndex(
      (item) => item.id === currentTabId.value
    );
    if (beside && position >= 0) {
      openTabList.value.splice(position + 1, 0, persistentTab);
    } else {
      openTabList.value.push(persistentTab);
    }
    if (!worksheet) {
      draftTabList.value.push(newTab);
    }

    setCurrentTabId(id);
    tabsById.set(id, newTab);

    return newTab;
  };

  const removeDraft = (tab: SQLEditorTab) => {
    const draftIndex = draftTabList.value.findIndex(
      (item) => item.id === tab.id
    );
    if (draftIndex >= 0) {
      draftTabList.value.splice(draftIndex, 1);
    }
  };

  const closeTab = (tab: SQLEditorTab) => {
    const { id } = tab;
    const position = openTabList.value.findIndex((item) => item.id === id);
    if (position < 0) {
      return;
    }
    openTabList.value.splice(position, 1);
    tabsById.delete(id);

    if (tab.mode === "ADMIN") {
      useWebTerminalStore().clearQueryStateByTab(id);
    }

    if (id === currentTabId.value) {
      const nextIndex = Math.min(position, openTabList.value.length - 1);
      const nextTab = openTabList.value[nextIndex];
      setCurrentTabId(nextTab?.id ?? "");
    }
  };

  const updateTab = (id: string, payload: Partial<SQLEditorTab>) => {
    const tab = tabById(id);
    if (!tab) return;
    Object.assign(tab, payload);

    if (payload.worksheet) {
      removeDraft(tab);
    }
  };

  const updateCurrentTab = (payload: Partial<SQLEditorTab>) => {
    const id = currentTabId.value;
    if (!id) return;
    updateTab(id, payload);
  };

  const updateBatchQueryContext = (payload: Partial<BatchQueryContext>) => {
    const tab = currentTab.value;
    if (!tab) {
      return;
    }
    updateTab(tab.id, {
      batchQueryContext: {
        databases: tab.batchQueryContext?.databases ?? [],
        dataSourceType:
          tab.batchQueryContext?.dataSourceType ?? DataSourceType.READ_ONLY,
        ...tab.batchQueryContext,
        ...payload,
      },
    });
  };

  // removeDatabaseQueryContext remove the context by id, and returns the next context.
  const removeDatabaseQueryContext = ({
    database,
    contextId,
  }: {
    database: string;
    contextId: string;
  }): SQLEditorDatabaseQueryContext | undefined => {
    const tab = tabById(currentTabId.value);
    if (!tab || !tab.databaseQueryContexts) {
      return;
    }
    if (!tab.databaseQueryContexts.has(database)) {
      return;
    }
    const contexts = tab.databaseQueryContexts.get(database)!;
    const index = contexts.findIndex((context) => context.id === contextId);
    if (index < 0) {
      return;
    }
    contexts.splice(index, 1);
    return contexts[index] || contexts[index - 1];
  };

  const batchRemoveDatabaseQueryContext = ({
    database,
    contextIds,
  }: {
    database: string;
    contextIds: string[];
  }) => {
    const tab = tabById(currentTabId.value);
    if (!tab || !tab.databaseQueryContexts) {
      return;
    }
    if (!tab.databaseQueryContexts.has(database)) {
      return;
    }
    // Early exit if no contexts to remove
    if (contextIds.length === 0) {
      return;
    }

    const target = new Set(contextIds);
    const contexts = tab.databaseQueryContexts.get(database)!;
    const newContexts = contexts.filter((ctx) => !target.has(ctx.id));

    // Only update if something actually changed
    if (newContexts.length !== contexts.length) {
      tab.databaseQueryContexts.set(database, newContexts);
    }
  };

  const deleteDatabaseQueryContext = (database: string) => {
    const tab = tabById(currentTabId.value);
    if (!tab || !tab.databaseQueryContexts) {
      return;
    }
    tab.databaseQueryContexts.delete(database);
  };

  const updateDatabaseQueryContext = ({
    database,
    contextId,
    context,
  }: {
    database: string;
    contextId: string;
    context: Partial<SQLEditorDatabaseQueryContext>;
  }) => {
    const tab = tabById(currentTabId.value);
    if (!tab || !tab.databaseQueryContexts) {
      return;
    }
    if (!tab.databaseQueryContexts.has(database)) {
      return;
    }
    const target = tab.databaseQueryContexts
      .get(database)
      ?.find((c) => c.id === contextId);
    if (!target) {
      return;
    }
    Object.assign(target, context);
    return target;
  };

  const setCurrentTabId = (id: string) => {
    currentTabId.value = id;
  };

  const selectOrAddSimilarNewTab = (
    tab: CoreSQLEditorTab,
    beside = false,
    defaultTitle?: string,
    ignoreMode?: boolean
  ) => {
    const curr = currentTab.value;
    if (curr) {
      if (
        isDisconnectedSQLEditorTab(curr) ||
        isSimilarSQLEditorTab(tab, curr, ignoreMode)
      ) {
        updateTab(curr.id, {
          connection: tab.connection,
          worksheet: tab.worksheet,
          mode: tab.mode,
        });
        return curr;
      }
    }
    const similarNewTab = tabList.value.find(
      (tmp) => tmp.status === "CLEAN" && isSimilarSQLEditorTab(tmp, tab)
    );
    if (similarNewTab) {
      setCurrentTabId(similarNewTab.id);
      return similarNewTab;
    } else {
      return addTab(
        {
          ...tab,
          title: defaultTitle,
        },
        beside
      );
    }
  };

  watch(
    () => project.value,
    () => {
      maybeInitProject();
    }
  );

  // some shortcuts
  const isDisconnected = computed(() => {
    const tab = currentTab.value;
    if (!tab) return true;
    return isDisconnectedSQLEditorTab(tab);
  });

  return {
    project,
    tabList,
    draftList: computed(() => draftTabList.value),
    currentTabId,
    currentTab,
    tabById,
    addTab,
    removeDraft,
    closeTab,
    updateTab,
    updateCurrentTab,
    updateBatchQueryContext,
    updateDatabaseQueryContext,
    removeDatabaseQueryContext,
    batchRemoveDatabaseQueryContext,
    deleteDatabaseQueryContext,
    setCurrentTabId,
    selectOrAddSimilarNewTab,
    maybeInitProject,
    isDisconnected,
    isInBatchMode,
    supportBatchMode,
  };
});

export const useCurrentSQLEditorTab = () => {
  return storeToRefs(useSQLEditorTabStore()).currentTab;
};

export const isSQLEditorTabClosable = (tab: SQLEditorTab) => {
  const { tabList } = useSQLEditorTabStore();

  if (tabList.length > 1) {
    // Not the only one tab
    return true;
  }
  if (tabList.length === 1) {
    // It's the only one tab, and it's closable if it's a sheet tab
    return !!tab.worksheet;
  }
  return false;
};

export const useSQLEditorConnectionDetail = (
  connection: MaybeRef<SQLEditorConnection>
) => {
  const { database } = useDatabaseV1ByName(
    computed(() => unref(connection).database)
  );

  const instance = computed(() => {
    return database.value.instanceResource;
  });

  const environment = computed(() => {
    if (isValidDatabaseName(database.value.name)) {
      return database.value.effectiveEnvironmentEntity;
    }

    return useEnvironmentV1Store().getEnvironmentByName(
      instance.value.environment ?? ""
    );
  });

  return { connection, instance, database, environment };
};

export const useConnectionOfCurrentSQLEditorTab = () => {
  const store = useSQLEditorTabStore();
  const connection = computed(() => {
    return store.currentTab?.connection ?? emptySQLEditorConnection();
  });

  const { instance, database, environment } =
    useSQLEditorConnectionDetail(connection);

  return { connection, instance, database, environment };
};

export const resolveOpeningDatabaseListFromSQLEditorTabList = () => {
  const { tabList } = useSQLEditorTabStore();
  const databaseStore = useDatabaseV1Store();
  const databaseSet = new Set<string>();

  for (const tab of tabList) {
    const { database } = tab.connection;
    if (database) {
      const db = databaseStore.getDatabaseByName(database);
      databaseSet.add(db.name);
    }
  }
  return databaseSet;
};
