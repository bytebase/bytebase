import { computed, type MaybeRef, reactive, ref, toRefs, unref } from "vue";
import {
  getSQLEditorEditorState,
  subscribeSQLEditorEditorState,
} from "@/react/stores/sqlEditor/editor";
import {
  getSQLEditorTabsState,
  subscribeSQLEditorTabsState,
} from "@/react/stores/sqlEditor/tab";
import {
  hasFeature,
  useDatabaseV1ByName,
  useEnvironmentV1Store,
} from "@/store";
import type {
  BatchQueryContext,
  SQLEditorConnection,
  SQLEditorDatabaseQueryContext,
  SQLEditorTab,
} from "@/types";
import { isValidDatabaseName } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  defaultSQLEditorTab,
  emptySQLEditorConnection,
  getDatabaseEnvironment,
  getInstanceResource,
  isConnectedSQLEditorTab,
} from "@/utils";

/**
 * Vue compatibility shim mirroring the historical
 * `useSQLEditorTabStore()` API while the new Zustand store
 * (`@/react/stores/sqlEditor/tab.ts`) holds the source of truth.
 *
 * Lives on the Vue side because the only remaining consumers are
 * Vue-tracked code paths (the AI plugin and the worksheet Pinia
 * store). React consumers should import directly from
 * `@/react/stores/sqlEditor/tab` instead.
 *
 * All reads route through a Vue version `ref` that bumps on every
 * Zustand mutation; writes proxy through the Zustand actions.
 */
const buildSQLEditorTabShim = () => {
  // Vue-reactive cursors bumped on every Zustand mutation.
  const editorVersion = ref(0);
  const tabsVersion = ref(0);
  subscribeSQLEditorEditorState(() => {
    editorVersion.value++;
  });
  subscribeSQLEditorTabsState(() => {
    tabsVersion.value++;
  });

  const touchTabs = () => {
    void tabsVersion.value;
  };

  const project = computed<string>(() => {
    void editorVersion.value;
    return getSQLEditorEditorState().project;
  });

  const currentTabId = computed<string>({
    get: () => {
      touchTabs();
      return getSQLEditorTabsState().currentTabId;
    },
    set: (value) => {
      getSQLEditorTabsState().setCurrentTabId(value);
    },
  });

  const currentTab = computed<SQLEditorTab | undefined>(() => {
    touchTabs();
    const state = getSQLEditorTabsState();
    return state.tabsById.get(state.currentTabId);
  });

  const openTabList = computed<SQLEditorTab[]>({
    get: () => {
      touchTabs();
      const state = getSQLEditorTabsState();
      return state.openTmpTabList.map(
        (persisted) => state.tabsById.get(persisted.id) ?? defaultSQLEditorTab()
      );
    },
    set: (list) => {
      const persistents = list.map((tab) => ({
        id: tab.id,
        worksheet: tab.worksheet,
        mode: tab.mode,
        batchQueryContext: tab.batchQueryContext,
        treeState: tab.treeState,
        viewState: tab.viewState,
      }));
      getSQLEditorTabsState().setOpenTabListOrder(persistents);
    },
  });

  const isDisconnected = computed(() => {
    touchTabs();
    const tab = currentTab.value;
    if (!tab) return true;
    return !isConnectedSQLEditorTab(tab);
  });

  const supportBatchMode = computed(() => {
    touchTabs();
    return currentTab.value?.mode !== "ADMIN";
  });

  const isInBatchMode = computed(() => {
    touchTabs();
    const tab = currentTab.value;
    if (!tab) return false;
    if (!supportBatchMode.value) return false;
    if (!hasFeature(PlanFeature.FEATURE_BATCH_QUERY)) return false;
    const { batchQueryContext } = tab;
    if (!batchQueryContext) return false;
    const { databaseGroups = [], databases = [] } = batchQueryContext;
    if (!hasFeature(PlanFeature.FEATURE_DATABASE_GROUPS)) {
      return databases.length > 1;
    }
    return databaseGroups.length > 0 || databases.length > 1;
  });

  const getTabById = (tabId: string) => {
    touchTabs();
    return getSQLEditorTabsState().tabsById.get(tabId);
  };

  const getTabByWorksheet = (worksheet: string) => {
    return openTabList.value.find((item) => item.worksheet === worksheet);
  };

  // Action forwards.
  const setCurrentTabId = (id: string) =>
    getSQLEditorTabsState().setCurrentTabId(id);
  const addTab = (payload?: Partial<SQLEditorTab>, beside = false) =>
    getSQLEditorTabsState().addTab(payload, beside);
  const cloneTab = (targetId: string, payload?: Partial<SQLEditorTab>) =>
    getSQLEditorTabsState().cloneTab(targetId, payload);
  const closeTab = (tabId: string) => getSQLEditorTabsState().closeTab(tabId);
  const updateTab = (id: string, payload: Partial<SQLEditorTab>) =>
    getSQLEditorTabsState().updateTab(id, payload);
  const updateCurrentTab = (payload: Partial<SQLEditorTab>) =>
    getSQLEditorTabsState().updateCurrentTab(payload);
  const updateBatchQueryContext = (payload: Partial<BatchQueryContext>) =>
    getSQLEditorTabsState().updateBatchQueryContext(payload);
  const removeDatabaseQueryContext = (params: {
    database: string;
    contextId: string;
  }): SQLEditorDatabaseQueryContext | undefined =>
    getSQLEditorTabsState().removeDatabaseQueryContext(params);
  const batchRemoveDatabaseQueryContext = (params: {
    database: string;
    contextIds: string[];
  }) => getSQLEditorTabsState().batchRemoveDatabaseQueryContext(params);
  const deleteDatabaseQueryContext = (database: string) =>
    getSQLEditorTabsState().deleteDatabaseQueryContext(database);
  const updateDatabaseQueryContext = (params: {
    database: string;
    contextId: string;
    context: Partial<SQLEditorDatabaseQueryContext>;
  }) => getSQLEditorTabsState().updateDatabaseQueryContext(params);
  const initProject = (newProject: string) =>
    getSQLEditorTabsState().initProject(newProject);

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

let _shim: ReturnType<typeof buildSQLEditorTabShim> | undefined;
export const useSQLEditorTabStore = () => {
  if (!_shim) _shim = buildSQLEditorTabShim();
  return _shim;
};

export type SQLEditorTabState = ReturnType<typeof useSQLEditorTabStore>;

export const useCurrentSQLEditorTab = () => {
  return toRefs(useSQLEditorTabStore()).currentTab;
};

export { isSQLEditorTabClosable } from "@/react/stores/sqlEditor/tab";

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
