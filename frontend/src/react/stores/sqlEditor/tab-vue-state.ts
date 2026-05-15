import { cloneDeep, head, isUndefined, omitBy, pick } from "lodash-es";
import { computed, type MaybeRef, reactive, toRefs, unref, watch } from "vue";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import {
  hasFeature,
  useCurrentUserV1,
  useDatabaseV1ByName,
  useEnvironmentV1Store,
  useWorkSheetStore,
} from "@/store";
import {
  migrateDraftsFromCache,
  migrateTabViewState,
} from "@/store/modules/sqlEditor/legacy/migration";
import type {
  BatchQueryContext,
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
  getDatabaseEnvironment,
  getInstanceResource,
  getSheetStatement,
  isConnectedSQLEditorTab,
  storageKeySqlEditorCurrentTab,
  storageKeySqlEditorTabs,
  useDynamicLocalStorage,
} from "@/utils";

/**
 * Vue-reactive SQL Editor tab state. Replaces the Pinia
 * `useSQLEditorTabStore` (`store/modules/sqlEditor/tab.ts`) — same
 * shape and field semantics, but lives as a module-level lazy
 * singleton instead of a Pinia store. The store is still heavily
 * Vue-reactive (`reactive(new Map())`, `computed`, `watch`,
 * `useDynamicLocalStorage`), so consumers continue to use
 * `useVueState(() => tabStore.X)` for cross-framework reads.
 */
const PERSISTENT_TAB_FIELDS = [
  "id",
  "worksheet",
  "mode",
  "batchQueryContext",
  "treeState",
  "viewState",
] as const;
type PersistentTab = Pick<SQLEditorTab, (typeof PERSISTENT_TAB_FIELDS)[number]>;

const buildSQLEditorTabState = () => {
  // re-expose selected project in sqlEditorStore for shortcut
  const { project } = toRefs(useSQLEditorVueState());
  const tabsById = reactive(new Map<string, SQLEditorTab>());
  const worksheetStore = useWorkSheetStore();

  const me = useCurrentUserV1();
  const keyNamespace = computed(() =>
    storageKeySqlEditorTabs(project.value, me.value.email)
  );

  const openTmpTabList = useDynamicLocalStorage<PersistentTab[]>(
    computed(() => keyNamespace.value),
    [],
    localStorage,
    {
      listenToStorageChanges: false,
    }
  );

  const currentTabId = useDynamicLocalStorage<string>(
    computed(() =>
      storageKeySqlEditorCurrentTab(project.value, me.value.email)
    ),
    "",
    localStorage,
    {
      listenToStorageChanges: false,
    }
  );

  const loadStoredTabs = async () => {
    const validOpenTabMap: Map<string, PersistentTab> = new Map();
    for (const tab of openTmpTabList.value) {
      if (validOpenTabMap.has(tab.id)) {
        continue;
      }
      if (!tab.worksheet) {
        continue;
      }
      const worksheet = await worksheetStore.getOrFetchWorksheetByName(
        tab.worksheet,
        true
      );
      if (!worksheet) {
        continue;
      }
      const statement = getSheetStatement(worksheet);
      const connection = await extractWorksheetConnection(worksheet);

      const fullTab: SQLEditorTab = {
        ...defaultSQLEditorTab(),
        ...omitBy(tab, isUndefined),
        connection,
        worksheet: worksheet.name,
        title: worksheet.title,
        statement,
        status: "CLEAN",
        databaseQueryContexts: undefined,
      };

      validOpenTabMap.set(tab.id, tab);
      tabsById.set(tab.id, fullTab);
    }

    openTmpTabList.value = [...validOpenTabMap.values()];
  };

  const initProject = async (project: string) => {
    await migrateDraftsFromCache(project);
    migrateTabViewState(project);
    tabsById.clear();
    await loadStoredTabs();
    currentTabId.value = head(openTmpTabList.value)?.id ?? "";
  };

  const getTabById = (tabId: string) => {
    return tabsById.get(tabId);
  };

  const openTabList = computed({
    get() {
      return openTmpTabList.value.map((item) => {
        return getTabById(item.id) ?? defaultSQLEditorTab();
      });
    },
    set(list) {
      openTmpTabList.value = list;
    },
  });

  const getTabByWorksheet = (worksheet: string) => {
    return openTabList.value.find((item) => item.worksheet === worksheet);
  };

  const currentTab = computed(() => {
    const currId = currentTabId.value;
    if (!currId) return undefined;
    return getTabById(currId);
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

  /**
   * Create or update the tab, and ensure the tab is open.
   * @param payload
   * @param beside `true` to add the tab beside currentTab, `false` to add the tab to the last, default to `false`
   * @returns the tab
   */
  const addTab = (
    payload?: Partial<SQLEditorTab>,
    beside = false
  ): SQLEditorTab => {
    const defaultTab: SQLEditorTab = {
      ...defaultSQLEditorTab(),
      ...omitBy(payload, isUndefined),
    };
    const { id } = defaultTab;

    let newTab = getTabById(id);
    if (newTab) {
      Object.assign(newTab, omitBy(payload, isUndefined));
    } else {
      newTab = defaultTab;
      tabsById.set(id, newTab);
    }

    upsertCache(newTab, beside);
    setCurrentTabId(id);
    return newTab;
  };

  const cloneTab = (
    targetId: string,
    payload?: Partial<SQLEditorTab>
  ): SQLEditorTab => {
    const targetTab = getTabById(targetId);
    const clonedTab: Partial<SQLEditorTab> = {
      statement: targetTab?.statement,
      connection: cloneDeep(targetTab?.connection),
      treeState: cloneDeep(targetTab?.treeState),
      editorState: cloneDeep(targetTab?.editorState),
      batchQueryContext: cloneDeep(targetTab?.batchQueryContext),
      // Cloned tabs start untitled; the UI renders an "Untitled" placeholder.
      title: "",
      ...payload,
    };

    return addTab(clonedTab, true);
  };

  const closeTab = (tabId: string) => {
    const position = openTmpTabList.value.findIndex(
      (item) => item.id === tabId
    );
    if (position < 0) {
      return;
    }
    openTmpTabList.value.splice(position, 1);
    tabsById.delete(tabId);
    // Dynamic import avoids a static cycle with the zustand store
    // (which transitively re-imports this module via `@/store`).
    void import("@/react/stores/sqlEditor/webTerminal-service").then(
      ({ disposeWebTerminalQuerySession }) => {
        disposeWebTerminalQuerySession(tabId);
      }
    );

    if (tabId === currentTabId.value) {
      const nextIndex = Math.min(position, openTmpTabList.value.length - 1);
      const nextTab = openTmpTabList.value[nextIndex];
      setCurrentTabId(nextTab?.id ?? "");
    }
  };

  const upsertCache = (tab: SQLEditorTab, beside = false) => {
    const persistentTab = pick(tab, ...PERSISTENT_TAB_FIELDS) as PersistentTab;

    const position = openTmpTabList.value.findIndex(
      (item) => item.id === tab.id
    );
    if (position >= 0) {
      Object.assign(openTmpTabList.value[position], persistentTab);
    } else {
      const currentPosition = openTmpTabList.value.findIndex(
        (item) => item.id === currentTabId.value
      );
      if (beside && currentPosition >= 0) {
        openTmpTabList.value.splice(currentPosition + 1, 0, persistentTab);
      } else {
        openTmpTabList.value.push(persistentTab);
      }
    }
  };

  const updateTab = (
    id: string,
    payload: Partial<SQLEditorTab>
  ): SQLEditorTab | undefined => {
    const tab = getTabById(id);
    if (!tab) return;
    Object.assign(tab, payload);
    upsertCache(tab);
    return tab;
  };

  const updateCurrentTab = (payload: Partial<SQLEditorTab>) => {
    const id = currentTabId.value;
    if (!id) return;
    return updateTab(id, payload);
  };

  const updateBatchQueryContext = (payload: Partial<BatchQueryContext>) => {
    const tab = currentTab.value;
    if (!tab) {
      return;
    }
    return updateTab(tab.id, {
      batchQueryContext: {
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
    const tab = getTabById(currentTabId.value);
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
    const tab = getTabById(currentTabId.value);
    if (!tab || !tab.databaseQueryContexts) {
      return;
    }
    if (!tab.databaseQueryContexts.has(database)) {
      return;
    }
    if (contextIds.length === 0) {
      return;
    }

    const target = new Set(contextIds);
    const contexts = tab.databaseQueryContexts.get(database)!;
    const newContexts = contexts.filter((ctx) => !target.has(ctx.id));

    if (newContexts.length !== contexts.length) {
      tab.databaseQueryContexts.set(database, newContexts);
    }
  };

  const deleteDatabaseQueryContext = (database: string) => {
    const tab = getTabById(currentTabId.value);
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
    const tab = getTabById(currentTabId.value);
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

  watch(
    () => project.value,
    async (project) => {
      await initProject(project);
    }
  );

  const isDisconnected = computed(() => {
    const tab = currentTab.value;
    if (!tab) return true;
    return !isConnectedSQLEditorTab(tab);
  });

  // Wrap with `reactive()` so consumers see the same auto-unwrap /
  // auto-wrap-on-assign ergonomics they had with the Pinia store. Use
  // Vue's `toRefs()` (not Pinia's `storeToRefs`) to extract individual
  // refs when needed.
  return reactive({
    project,
    initProject,
    getTabById,
    getTabByWorksheet,
    openTabList,
    currentTabId,
    currentTab,
    addTab,
    cloneTab,
    closeTab,
    updateTab,
    updateCurrentTab,
    updateBatchQueryContext,
    updateDatabaseQueryContext,
    removeDatabaseQueryContext,
    batchRemoveDatabaseQueryContext,
    deleteDatabaseQueryContext,
    setCurrentTabId,
    isDisconnected,
    isInBatchMode,
    supportBatchMode,
  });
};

let _state: ReturnType<typeof buildSQLEditorTabState> | undefined;
export const useSQLEditorTabStore = () => {
  if (!_state) _state = buildSQLEditorTabState();
  return _state;
};

export type SQLEditorTabState = ReturnType<typeof useSQLEditorTabStore>;

export const useCurrentSQLEditorTab = () => {
  return toRefs(useSQLEditorTabStore()).currentTab;
};

export const isSQLEditorTabClosable = (tab: SQLEditorTab) => {
  const { openTabList } = useSQLEditorTabStore();

  if (openTabList.length > 1) {
    return true;
  }
  if (openTabList.length === 1) {
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
    return getInstanceResource(database.value);
  });

  const environment = computed(() => {
    if (isValidDatabaseName(database.value.name)) {
      return getDatabaseEnvironment(database.value);
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
