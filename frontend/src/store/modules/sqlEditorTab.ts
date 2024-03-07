import { watchThrottled } from "@vueuse/core";
import { head, pick } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, watch } from "vue";
import "@/types";
import { SQLEditorTab } from "@/types/sqlEditorTab";
import { WebStorageHelper, defaultSQLEditorTab } from "@/utils";
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

const keyPrefixWithProject = (project: string) => {
  return `${LOCAL_STORAGE_KEY_PREFIX}.${project || "ALL"}`;
};
const getStorage = (project: string) => {
  return new WebStorageHelper(keyPrefixWithProject(project), localStorage);
};

export const useSQLEditorTabsByProject = (project: string) => {
  const storage = getStorage(project);
  // We store the tabIdList and the tabs separately.
  // This index-entity modeling enables us to update one tab entity at a time,
  // and reduce the performance costing while writing localStorage.
  // `tabIdList` stores all `tab.id`s in the project
  const tabIdList = ref<string[]>([]);
  // `currentTabId` stores current `tab.id` in the project
  // default to empty string, which means no tab is selected
  const currentTabId = ref<string>("");
  // `tabs` stores all PersistentTabs in the project
  const tabs = ref(new Map<string, SQLEditorTab>());

  const tabById = (id: string) => {
    return tabs.value.get(id);
  };
  const tabList = computed(() => {
    return tabIdList.value.map((id) => {
      return tabById(id) ?? defaultSQLEditorTab();
    });
  });
  const currentTab = computed(() => {
    return tabs.value.get(currentTabId.value);
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
    tabs.value.set(id, newTab);

    watchTab(newTab, true /* immediate */);
    throw new Error("not implemented");
  };
  const removeTab = (tab: SQLEditorTab) => {
    const { id } = tab;
    const position = tabIdList.value.indexOf(id);
    if (position < 0) return;
    tabIdList.value.splice(position, 1);
    tabs.value.delete(id);
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
  const updateCurrent = (payload: Partial<SQLEditorTab>) => {
    const id = currentTabId.value;
    if (!id) return;
    updateTab(id, payload);
  };
  const setCurrent = (id: string) => {
    currentTabId.value = id;
  };
  // clean persistent tabs that are not in the `tabIdList` anymore
  const _cleanup = (tabIdList: string[]) => {
    const prefix = `${keyPrefixWithProject(project)}.tab.`;
    const keys = storage.keys().filter((key) => key.startsWith(prefix));
    keys.forEach((key) => {
      const id = key.substring(prefix.length);
      if (tabIdList.indexOf(id) < 0) {
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
  const init = () => {
    // Load tabIdList
    const storedTabIdList = storage.load<string[]>(KEYS.tabIdList, []);

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
      tabs.value.set(id, tab);
      validTabIdList.push(id);
    });
    tabIdList.value = validTabIdList;

    // Load currentTabId
    const firstTabId = head(tabIdList.value) ?? "";
    currentTabId.value = storage.load<string>(KEYS.currentTabId, firstTabId);
    if (!tabIdList.value.includes(currentTabId.value)) {
      // currentTabId is not in tabIdList
      // fallback to the first tab or nothing
      currentTabId.value = firstTabId;
    }

    // Unlike legacy tab store, we won't pre fetch all tab's sheet (if exited)
    // here.
    // In fact we don't have any reasons to fetch the full sheet since we have
    // `statement` in the persistent tab.
    // This is useful when a user opens a lot of sheets and then reopen the page

    // Clean up stored but unused tabs
    _cleanup(tabIdList.value);

    watch(
      currentTabId,
      (id) => {
        storage.save<string>(KEYS.currentTabId, id);
      },
      {
        immediate: true,
      }
    );
    watch(
      tabIdList,
      (idList) => {
        storage.save<string[]>(KEYS.tabIdList, idList);
      },
      { deep: true, immediate: true }
    );
  };
  init();

  const reset = () => {
    storage.clear();
    init();
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
    updateCurrent,
    setCurrent,
    reset,
  };
};

export const useSQLEditorTabStore = defineStore("sql-editor-tab", () => {
  // states
  const project = ref<string>(""); // empty to "ALL" projects for high-privileged users
  const tabsByProject = new Map<
    string /* projects/{project} */,
    ReturnType<typeof useSQLEditorTabsByProject>
  >();
  const current = computed(() => {
    const initialized = tabsByProject.get(project.value);
    if (initialized) {
      return initialized;
    }
    const tabs = useSQLEditorTabsByProject(project.value);
    tabsByProject.set(project.value, tabs);
    return tabs;
  });
  const resetAll = () => {
    for (const tabs of tabsByProject.values()) {
      tabs.reset();
    }
  };

  return {
    project,
    current,
    resetAll,
  };
});
