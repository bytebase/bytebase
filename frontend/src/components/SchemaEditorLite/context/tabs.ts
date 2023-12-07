import { v1 as uuidv1 } from "uuid";
import { computed, ref } from "vue";
import { CoreTabContext, TabContext } from "../types";

export const useTabs = () => {
  const tabMap = ref(new Map<string, TabContext>());
  const tabList = computed(() => {
    return Array.from(tabMap.value.values());
  });
  const currentTabId = ref<string>("");
  const currentTab = computed(() => {
    if (!currentTabId.value) return undefined;
    return tabMap.value.get(currentTabId.value);
  });

  const addTab = (coreTab: CoreTabContext, setAsCurrentTab = true) => {
    const existedTab = findTab(coreTab);
    if (existedTab) {
      if (setAsCurrentTab) {
        currentTabId.value = existedTab.id;
      }
      return;
    }

    const id = uuidv1();
    const tab: TabContext = {
      id,
      ...coreTab,
    };

    tabMap.value.set(id, tab);
    if (setAsCurrentTab) {
      currentTabId.value = id;
    }
  };

  const setCurrentTab = (id: string) => {
    if (tabMap.value.has(id)) {
      currentTabId.value = id;
    } else {
      currentTabId.value = "";
    }
  };

  const closeTab = (id: string) => {
    const index = tabList.value.findIndex((tab) => tab.id === id);
    if (currentTabId.value === id) {
      // Find next tab for showing.
      let nextIndex = -1;
      if (index === 0) {
        nextIndex = 1;
      } else {
        nextIndex = index - 1;
      }
      const nextTab = tabList.value[nextIndex];
      if (nextTab) {
        setCurrentTab(nextTab.id);
      } else {
        setCurrentTab("");
      }
    }
    tabMap.value.delete(id);
  };
  const findTab = (target: CoreTabContext) => {
    return tabList.value.find((tab) => {
      if (tab.type !== target.type) return false;
      if (tab.database.name !== target.database.name) return false;
      if (tab.type === "database" && target.type === "database") {
        return tab.metadata.database.name === target.database.name;
      }
      if (tab.type === "table" && target.type === "table") {
        return (
          tab.metadata.schema.name === target.metadata.schema.name &&
          tab.metadata.table.name === target.metadata.table.name
        );
      }
      return false;
    });
  };

  return {
    tabMap,
    tabList,
    currentTabId,
    currentTab,
    addTab,
    setCurrentTab,
    closeTab,
    findTab,
  };
};
