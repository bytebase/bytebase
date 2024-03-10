import { useLocalStorage, watchThrottled } from "@vueuse/core";
import { head, pick } from "lodash-es";
import { defineStore, storeToRefs } from "pinia";
import { computed, reactive, watch } from "vue";
import "@/types";
import { CoreSQLEditorTab, SQLEditorTab } from "@/types/sqlEditorTab";
import {
  WebStorageHelper,
  defaultSQLEditorTab,
  isDisconnectedSQLEditorTab,
  isSimilarSQLEditorTab,
} from "@/utils";
import { useSQLEditorV2Store } from "./sqlEditorV2";
import { useWebTerminalV1Store } from "./v1";

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
  "batchContext",
] as const;
type PersistentTab = Pick<SQLEditorTab, typeof PERSISTENT_TAB_FIELDS[number]>;

// const keyPrefixWithProject = (project: string) => {
//   return `${LOCAL_STORAGE_KEY_PREFIX}.${project || "ALL"}`;
// };
// const getStorage = (project: string) => {
//   return new WebStorageHelper(keyPrefixWithProject(project), localStorage);
// };

// `tabsById` stores all PersistentTab items across all projects
const tabsById = new Map<string, SQLEditorTab>();
// const watchedTabIds = new Set<string>();

// const useSQLEditorTabsByProject = (project: string) => {
//   const storage = getStorage(project);
//   // We store the tabIdList and the tabs separately.
//   // This index-entity modeling enables us to update one tab entity at a time,
//   // and reduce the performance costing while writing localStorage.
//   // `tabIdList` stores all `tab.id`s in the project
//   const tabIdList = ref<string[]>([]);
//   // `currentTabId` stores current `tab.id` in the project
//   // default to empty string, which means no tab is selected
//   const currentTabId = ref<string>("");

//   const tabById = (id: string) => {
//     return tabsById.get(id);
//   };
//   const tabList = computed(() => {
//     return tabIdList.value.map((id) => {
//       return tabById(id) ?? defaultSQLEditorTab();
//     });
//   });
//   const currentTab = computed(() => {
//     return tabsById.get(currentTabId.value);
//   });

//   // actions
//   /**
//    *
//    * @param payload
//    * @param beside `true` to add the tab beside currentTab, `false` to add the tab to the last, default to `false`
//    * @returns
//    */
//   const addTab = (payload?: Partial<SQLEditorTab>, beside = false) => {
//     const newTab = reactive<SQLEditorTab>({
//       ...defaultSQLEditorTab(),
//       ...payload,
//     });

//     const { id } = newTab;
//     const position = tabIdList.value.indexOf(currentTabId.value ?? "");
//     if (beside && position >= 0) {
//       tabIdList.value.splice(position + 1, 0, id);
//     } else {
//       tabIdList.value.push(id);
//     }
//     currentTabId.value = id;
//     tabsById.set(id, newTab);

//     watchTab(newTab, true /* immediate */);
//     throw new Error("not implemented");
//   };
//   const removeTab = (tab: SQLEditorTab) => {
//     const { id } = tab;
//     const position = tabIdList.value.indexOf(id);
//     if (position < 0) return;
//     tabIdList.value.splice(position, 1);
//     tabsById.delete(id);
//     storage.remove(KEYS.tab(id));

//     if (tab.mode === "ADMIN") {
//       useWebTerminalV1Store().clearQueryStateByTab(id);
//     }
//   };
//   const updateTab = (id: string, payload: Partial<SQLEditorTab>) => {
//     const tab = tabById(id);
//     if (!tab) return;
//     Object.assign(tab, payload);
//   };
//   const updateCurrent = (payload: Partial<SQLEditorTab>) => {
//     const id = currentTabId.value;
//     if (!id) return;
//     updateTab(id, payload);
//   };
//   const setCurrent = (id: string) => {
//     currentTabId.value = id;
//   };
//   const selectOrAddSimilarNewTab = (
//     tab: CoreSQLEditorTab,
//     beside = false,
//     defaultTitle?: string
//   ) => {
//     const curr = currentTab.value;
//     if (curr) {
//       if (isDisconnectedSQLEditorTab(curr)) {
//         if (defaultTitle) {
//           curr.title = defaultTitle;
//         }
//         return;
//       }

//       if (isSimilarSQLEditorTab(tab, curr)) {
//         return;
//       }
//     }
//     const similarNewTab = tabList.value.find(
//       (tmp) => tmp.status === "NEW" && isSimilarSQLEditorTab(tmp, tab)
//     );
//     if (similarNewTab) {
//       setCurrent(similarNewTab.id);
//     } else {
//       addTab(
//         {
//           ...tab,
//           title: defaultTitle,
//         },
//         beside
//       );
//     }
//   };
//   // clean persistent tabs that are not in the `tabIdList` anymore
//   const _cleanup = (tabIdList: string[]) => {
//     const prefix = `${keyPrefixWithProject(project)}.tab.`;
//     const keys = storage.keys().filter((key) => key.startsWith(prefix));
//     keys.forEach((key) => {
//       const id = key.substring(prefix.length);
//       if (tabIdList.indexOf(id) < 0) {
//         storage.remove(KEYS.tab(id));
//       }
//     });
//   };
//   // watch the field changes of a tab, store it to localStorage
//   // when needed, but not to frequently (for performance consideration)
//   const watchTab = (tab: SQLEditorTab, immediate: boolean) => {
//     if (watchedTabIds.has(tab.id)) {
//       return;
//     }
//     const dirtyFields = [
//       () => tab.title,
//       () => tab.sheet,
//       () => tab.statement,
//       () => tab.connection,
//       () => tab.batchContext,
//     ];
//     // set `tab.status` to "DIRTY" when it's changed
//     watch(dirtyFields, () => {
//       tab.status = "DIRTY";
//     });

//     // Use a throttled watcher to reduce the performance overhead when writing.
//     watchThrottled(
//       () => pick(tab, ...PERSISTENT_TAB_FIELDS) as PersistentTab,
//       (persistentTab) => {
//         storage.save<PersistentTab>(KEYS.tab(persistentTab.id), persistentTab);
//       },
//       { deep: true, immediate, throttle: 100, trailing: true }
//     );

//     watchedTabIds.add(tab.id);
//   };
//   // Load tabs session from localStorage
//   // Reset if failed
//   const init = () => {
//     // Load tabIdList
//     const storedTabIdList = storage.load<string[]>(KEYS.tabIdList, []);
//     debugger;

//     // Load tabs
//     const validTabIdList: string[] = [];
//     storedTabIdList.forEach((id) => {
//       const exitedTab = tabsById.get(id);
//       if (exitedTab) {
//         validTabIdList.push(id);
//         return;
//       }

//       const storedTab = storage.load<PersistentTab | undefined>(
//         KEYS.tab(id),
//         undefined
//       );
//       if (!storedTab) return;
//       const tab = reactive<SQLEditorTab>({
//         ...defaultSQLEditorTab(),
//         ...storedTab,
//         id,
//       });
//       watchTab(tab, false /* !immediate */);
//       tabsById.set(id, tab);
//       validTabIdList.push(id);
//     });
//     tabIdList.value = validTabIdList;

//     // if tabIdList is empty, push a default empty tab
//     if (tabIdList.value.length === 0) {
//       const initialTab = defaultSQLEditorTab();
//       watchTab(initialTab, true /* immediate */);
//       tabsById.set(initialTab.id, initialTab);
//       tabIdList.value.push(initialTab.id);
//     }

//     // Load currentTabId
//     const firstTabId = head(tabIdList.value) ?? "";
//     currentTabId.value = storage.load<string>(KEYS.currentTabId, firstTabId);
//     if (!tabIdList.value.includes(currentTabId.value)) {
//       // currentTabId is not in tabIdList
//       // fallback to the first tab or nothing
//       currentTabId.value = firstTabId;
//     }

//     // Unlike legacy tab store, we won't pre fetch all tab's sheet (if exited)
//     // here.
//     // In fact we don't have any reasons to fetch the full sheet since we have
//     // `statement` in the persistent tab.
//     // This is useful when a user opens a lot of sheets and then reopen the page

//     // Clean up stored but unused tabs
//     _cleanup(tabIdList.value);

//     watch(
//       currentTabId,
//       (id) => {
//         storage.save<string>(KEYS.currentTabId, id);
//       },
//       {
//         immediate: true,
//       }
//     );
//     watch(
//       tabIdList,
//       (idList) => {
//         storage.save<string[]>(KEYS.tabIdList, idList);
//       },
//       { deep: true, immediate: true }
//     );
//   };
//   init();

//   const reset = () => {
//     storage.clear();
//     init();
//   };

//   return {
//     tabIdList,
//     tabList,
//     currentTabId,
//     currentTab,
//     tabById,
//     addTab,
//     removeTab,
//     updateTab,
//     updateCurrent,
//     setCurrent,
//     selectOrAddSimilarNewTab,
//     reset,
//   };
// };

export const useSQLEditorTabStore = defineStore("sql-editor-tab", () => {
  // re-expose selected project in sqlEditorStore for shortcut
  const { project } = storeToRefs(useSQLEditorV2Store());

  // states
  const storage = new WebStorageHelper(LOCAL_STORAGE_KEY_PREFIX);
  const tabIdListMapByProject = useLocalStorage<Record<string, string[]>>(
    `${LOCAL_STORAGE_KEY_PREFIX}.${KEYS.tabIdList}`,
    {}
  );
  const currentTabIdMapByProject = useLocalStorage<
    Record<string, string | undefined>
  >(`${LOCAL_STORAGE_KEY_PREFIX}.${KEYS.currentTabId}`, {});
  const _maybeInitProject = (project: string) => {
    if (
      project in tabIdListMapByProject &&
      project in currentTabIdMapByProject
    ) {
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

    // if validTabIdList is empty, push a default empty tab
    if (validTabIdList.length === 0) {
      const initialTab = defaultSQLEditorTab();
      watchTab(initialTab, true /* immediate */);
      tabsById.set(initialTab.id, initialTab);
      validTabIdList.push(initialTab.id);
    }
    tabIdListMapByProject.value[project] = validTabIdList;

    // Load currentTabId
    const firstTabId = head(validTabIdList) ?? "";
    const storedCurrentTabId = currentTabIdMapByProject.value[project];
    if (!storedCurrentTabId || !validTabIdList.includes(storedCurrentTabId)) {
      // storedCurrentTabId is not in tabIdList
      // fallback to the first tab or nothing
      currentTabIdMapByProject.value[project] = firstTabId;
    }
  };

  // computed states
  // `tabIdList` is the tabIdList in current project
  // it's a combination of `project` and `tabIdListMapByProject`
  const tabIdList = computed({
    get() {
      _maybeInitProject(project.value);
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
      _maybeInitProject(project.value);
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
      useWebTerminalV1Store().clearQueryStateByTab(id);
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
      if (isDisconnectedSQLEditorTab(curr)) {
        if (defaultTitle) {
          curr.title = defaultTitle;
        }
        return;
      }

      if (isSimilarSQLEditorTab(tab, curr)) {
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
    const dirtyFields = [
      () => tab.title,
      () => tab.sheet,
      () => tab.statement,
      () => tab.connection,
      () => tab.batchContext,
    ];
    // set `tab.status` to "DIRTY" when it's changed
    watch(dirtyFields, () => {
      tab.status = "DIRTY";
    });

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
      _maybeInitProject(project);
    });
    // initialize current project if needed (when it's not stored)
    _maybeInitProject(project.value);

    _cleanup();
  };
  initAll();

  const reset = () => {
    storage.clear();
    tabIdListMapByProject.value = {};
    currentTabIdMapByProject.value = {};
    initAll();
  };

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
    reset,
  };
});

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
