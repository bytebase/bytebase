import { v1 as uuidv1 } from "uuid";
import { useSchemaEditorContext } from "./context";
import { CoreTabContext, TabContext } from "./types";

export const useTabs = () => {
  const { tabMap, tabList, currentTabId } = useSchemaEditorContext();

  const addTab = (coreTab: CoreTabContext, setAsCurrentTab = true) => {
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
  const findTab = () => {
    // TBD
  };

  return {
    addTab,
    setCurrentTab,
    closeTab,
    findTab,
  };
};

// findTab(parentName: string, tableId?: string) {
//   let tabType = SchemaEditorTabType.TabForDatabase;
//   if (tableId !== undefined) {
//     tabType = SchemaEditorTabType.TabForTable;
//   }

//   const tab = this.tabList.find((tab) => {
//     if (
//       tab.type === tabType &&
//       tab.parentName === parentName &&
//       (tab.type === SchemaEditorTabType.TabForDatabase ||
//         (tab.type === SchemaEditorTabType.TabForTable &&
//           tab.tableId === tableId))
//     ) {
//       return true;
//     }
//     return false;
//   });

//   return tab;
// }

// addTab(tab: TabContext, setAsCurrentTab = true) {
//   const tabCache = this.tabList.find((item) => {
//     if (
//       item.type === tab.type &&
//       item.parentName === tab.parentName &&
//       (item.type === SchemaEditorTabType.TabForDatabase ||
//         (item.type === SchemaEditorTabType.TabForTable &&
//           item.tableId === (tab as TableTabContext).tableId))
//     ) {
//       return true;
//     }
//     return false;
//   });

//   if (tabCache !== undefined) {
//     tab = {
//       ...tabCache,
//       ...tab,
//       id: tabCache.id,
//     };
//   }
//   this.tabState.tabMap.set(tab.id, tab);

//   if (setAsCurrentTab) {
//     this.setCurrentTab(tab.id);
//   }
// },
// setCurrentTab(tabId: string) {
//   if (isUndefined(this.tabState.tabMap.get(tabId))) {
//     this.tabState.currentTabId = undefined;
//   } else {
//     this.tabState.currentTabId = tabId;
//   }
// },
// closeTab(tabId: string) {
//   const tabList = Array.from(this.tabState.tabMap.values());
//   const tabIndex = tabList.findIndex((tab) => tab.id === tabId);
//   // Find next tab for showing.
//   if (this.tabState.currentTabId === tabId) {
//     let nextTabIndex = -1;
//     if (tabIndex === 0) {
//       nextTabIndex = 1;
//     } else {
//       nextTabIndex = tabIndex - 1;
//     }
//     const nextTab = tabList[nextTabIndex];
//     if (nextTab) {
//       this.setCurrentTab(nextTab.id);
//     } else {
//       this.setCurrentTab("");
//     }
//   }
//   this.tabState.tabMap.delete(tabId);
// },
// findTab(parentName: string, tableId?: string) {
//   let tabType = SchemaEditorTabType.TabForDatabase;
//   if (tableId !== undefined) {
//     tabType = SchemaEditorTabType.TabForTable;
//   }

//   const tab = this.tabList.find((tab) => {
//     if (
//       tab.type === tabType &&
//       tab.parentName === parentName &&
//       (tab.type === SchemaEditorTabType.TabForDatabase ||
//         (tab.type === SchemaEditorTabType.TabForTable &&
//           tab.tableId === tableId))
//     ) {
//       return true;
//     }
//     return false;
//   });

//   return tab;
// }
