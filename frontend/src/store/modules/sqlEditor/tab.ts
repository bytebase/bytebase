import { MaybeRef, useLocalStorage, watchThrottled } from "@vueuse/core";
import { head, pick, uniqBy } from "lodash-es";
import { defineStore, storeToRefs } from "pinia";
import { computed, reactive, unref } from "vue";
import {
  SQLEditorConnection,
  SQLEditorTreeNodeMeta,
  CoreSQLEditorTab,
  SQLEditorTab,
  UNKNOWN_ID,
} from "@/types";
import {
  WebStorageHelper,
  defaultSQLEditorTab,
  emptySQLEditorConnection,
  isDisconnectedSQLEditorTab,
  isSimilarSQLEditorTab,
} from "@/utils";
import { useDatabaseV1Store, useInstanceV1Store } from "../v1";
import { useSQLEditorStore } from "./editor";
import { useWebTerminalStore } from "./webTerminal";

const LOCAL_STORAGE_KEY_PREFIX = "bb.sql-editor-tab";
const KEYS = {
  tabIdList: "tab-id-list",
  currentTabId: "current-tab-id",
  tab: (id: string) => `tab.${id}`,
};

// Only store the core fields of a tab.
// Don't store anything which might be too large.
const PERSISTENT_TAB_FIELDS = [
  "id",
  "title",
  "connection",
  "statement",
  "mode",
  "sheet",
  "status",
  "batchQueryContext",
] as const;
type PersistentTab = Pick<SQLEditorTab, (typeof PERSISTENT_TAB_FIELDS)[number]>;

// `tabsById` stores all PersistentTab items across all projects
const tabsById = reactive(new Map<string, SQLEditorTab>());

export const useSQLEditorTabStore = defineStore("sqlEditorTab", () => {
  // re-expose selected project in sqlEditorStore for shortcut
  const { project } = storeToRefs(useSQLEditorStore());

  // states
  const storage = new WebStorageHelper(LOCAL_STORAGE_KEY_PREFIX);
  const tabIdListMapByProject = useLocalStorage<Record<string, string[]>>(
    `${LOCAL_STORAGE_KEY_PREFIX}.${KEYS.tabIdList}`,
    {}
  );
  const currentTabIdMapByProject = useLocalStorage<
    Record<string, string | undefined>
  >(`${LOCAL_STORAGE_KEY_PREFIX}.${KEYS.currentTabId}`, {});
  const initializedProjects = new Set<string>();
  const maybeInitProject = (project: string) => {
    if (initializedProjects.has(project)) {
      return;
    }

    const storedTabIdList = tabIdListMapByProject.value[project] ?? [];
    // Load tabs
    const validTabIdList: string[] = [];
    storedTabIdList.forEach((id) => {
      const storedTab = storage.load<PersistentTab | undefined>(
        KEYS.tab(id),
        undefined
      );
      if (!storedTab) return;
      const tab = reactive<SQLEditorTab>({
        ...defaultSQLEditorTab(),
        ...storedTab,
        id,
      });
      watchTab(tab, false /* !immediate */);
      tabsById.set(id, tab);
      validTabIdList.push(id);
    });

    tabIdListMapByProject.value[project] = validTabIdList;

    // Load currentTabId
    const storedCurrentTabId = currentTabIdMapByProject.value[project];
    if (!storedCurrentTabId || !validTabIdList.includes(storedCurrentTabId)) {
      // storedCurrentTabId is not in tabIdList
      // fallback to the first tab or nothing
      currentTabIdMapByProject.value[project] = head(validTabIdList) ?? "";
    }

    initializedProjects.add(project);
  };

  // computed states
  // `tabIdList` is the tabIdList in current project
  // it's a combination of `project` and `tabIdListMapByProject`
  const tabIdList = computed({
    get() {
      // _maybeInitProject(project.value);
      return tabIdListMapByProject.value[project.value] ?? [];
    },
    set(list) {
      tabIdListMapByProject.value[project.value] = list;
    },
  });
  // `currentTabId` is the currentTabId in current project
  // it's a combination of `project` and `currentTabIdMapByProject`
  const currentTabId = computed({
    get() {
      // _maybeInitProject(project.value);
      return currentTabIdMapByProject.value[project.value] ?? "";
    },
    set(id) {
      currentTabIdMapByProject.value[project.value] = id;
    },
  });
  const tabById = (id: string) => {
    return tabsById.get(id);
  };
  const tabList = computed(() => {
    return tabIdList.value.map((id) => {
      return tabById(id) ?? defaultSQLEditorTab();
    });
  });
  const currentTab = computed(() => {
    // _maybeInitProject(project.value);
    const currId = currentTabId.value;
    if (!currId) return undefined;
    return tabsById.get(currId);
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

    const { id } = newTab;
    const position = tabIdList.value.indexOf(currentTabId.value ?? "");
    if (beside && position >= 0) {
      tabIdList.value.splice(position + 1, 0, id);
    } else {
      tabIdList.value.push(id);
    }
    currentTabId.value = id;
    tabsById.set(id, newTab);

    watchTab(newTab, true /* immediate */);
  };
  const removeTab = (tab: SQLEditorTab) => {
    const { id } = tab;
    const position = tabIdList.value.indexOf(id);
    if (position < 0) return;
    tabIdList.value.splice(position, 1);
    tabsById.delete(id);
    storage.remove(KEYS.tab(id));

    if (tab.mode === "ADMIN") {
      useWebTerminalStore().clearQueryStateByTab(id);
    }
  };
  const updateTab = (id: string, payload: Partial<SQLEditorTab>) => {
    const tab = tabById(id);
    if (!tab) return;
    Object.assign(tab, payload);
  };
  const updateCurrentTab = (payload: Partial<SQLEditorTab>) => {
    const id = currentTabId.value;
    if (!id) return;
    updateTab(id, payload);
  };
  const setCurrentTabId = (id: string) => {
    currentTabId.value = id;
  };
  const selectOrAddSimilarNewTab = (
    tab: CoreSQLEditorTab,
    beside = false,
    defaultTitle?: string
  ) => {
    const curr = currentTab.value;
    if (curr) {
      if (
        isDisconnectedSQLEditorTab(curr) ||
        isSimilarSQLEditorTab(tab, curr)
      ) {
        curr.connection = tab.connection;
        curr.sheet = tab.sheet;
        curr.mode = tab.mode;
        if (defaultTitle) {
          curr.title = defaultTitle;
        }
        return;
      }
    }
    const similarNewTab = tabList.value.find(
      (tmp) => tmp.status === "NEW" && isSimilarSQLEditorTab(tmp, tab)
    );
    if (similarNewTab) {
      setCurrentTabId(similarNewTab.id);
    } else {
      addTab(
        {
          ...tab,
          title: defaultTitle,
        },
        beside
      );
    }
  };
  // clean persistent tabs that are not in the `tabIdList` anymore
  const _cleanup = () => {
    const prefix = `${LOCAL_STORAGE_KEY_PREFIX}.tab.`;
    const keys = storage.keys().filter((key) => key.startsWith(prefix));
    const flattenTabIdSet = new Set(
      Object.keys(tabIdListMapByProject.value).flatMap(
        (project) => tabIdListMapByProject.value[project]
      )
    );
    keys.forEach((key) => {
      const id = key.substring(prefix.length);
      if (!flattenTabIdSet.has(id)) {
        storage.remove(KEYS.tab(id));
      }
    });
  };
  // watch the field changes of a tab, store it to localStorage
  // when needed, but not to frequently (for performance consideration)
  const watchTab = (tab: SQLEditorTab, immediate: boolean) => {
    // Use a throttled watcher to reduce the performance overhead when writing.
    watchThrottled(
      () => pick(tab, ...PERSISTENT_TAB_FIELDS) as PersistentTab,
      (persistentTab) => {
        storage.save<PersistentTab>(KEYS.tab(persistentTab.id), persistentTab);
      },
      { deep: true, immediate, throttle: 100, trailing: true }
    );
  };
  // Load tabs session from localStorage
  // Reset if failed
  const initAll = () => {
    const projects = Object.keys(tabIdListMapByProject.value);
    // initialize all stored projects
    projects.forEach((project) => {
      maybeInitProject(project);
    });
    // initialize current project if needed (when it's not stored)
    maybeInitProject(project.value);

    _cleanup();
  };
  initAll();

  const reset = () => {
    storage.clear();
    tabIdListMapByProject.value = {};
    currentTabIdMapByProject.value = {};
    initAll();
  };

  // some shortcuts
  const isDisconnected = computed(() => {
    const tab = currentTab.value;
    if (!tab) return true;
    return isDisconnectedSQLEditorTab(tab);
  });

  return {
    tabIdList,
    tabList,
    currentTabId,
    currentTab,
    tabById,
    addTab,
    removeTab,
    updateTab,
    updateCurrentTab,
    setCurrentTabId,
    selectOrAddSimilarNewTab,
    maybeInitProject,
    reset,
    tabIdListMapByProject,
    currentTabIdMapByProject,
    isDisconnected,
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
    return !!tab.sheet;
  }
  return false;
};

export const useSQLEditorConnectionDetail = (
  connection: MaybeRef<SQLEditorConnection>
) => {
  const instance = computed(() => {
    return useInstanceV1Store().getInstanceByName(unref(connection).instance);
  });

  const database = computed(() => {
    return useDatabaseV1Store().getDatabaseByName(unref(connection).database);
  });

  const environment = computed(() => {
    if (database.value.uid !== String(UNKNOWN_ID)) {
      return database.value.effectiveEnvironmentEntity;
    }

    return instance.value.environmentEntity;
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
  return uniqBy(
    tabList.flatMap<SQLEditorTreeNodeMeta<"database">>((tab) => {
      const { database } = tab.connection;
      if (database) {
        const db = useDatabaseV1Store().getDatabaseByName(database);
        return [{ type: "database", target: db }];
      }
      return [];
    }),
    (meta) => meta.target.name
  );
};
