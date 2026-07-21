import { useCallback, useMemo, useRef, useState } from "react";
import { v1 as uuidv1 } from "uuid";
import type { CoreTabContext, TabContext } from "./core/types";
import type { TabsContext } from "./types";

export function useTabs(): TabsContext {
  const tabMapRef = useRef(new Map<string, TabContext>());
  const [currentTabId, setCurrentTabId] = useState("");
  // Version counter to trigger re-renders when tabMap mutates.
  const [, setVersion] = useState(0);
  const bump = useCallback(() => setVersion((v) => v + 1), []);

  // tabMapRef.current.size and currentTabId are deps to re-derive on mutations
  const tabList = useMemo(
    () => Array.from(tabMapRef.current.values()),
    [tabMapRef.current.size, currentTabId]
  );

  const currentTab = useMemo(
    () => (currentTabId ? tabMapRef.current.get(currentTabId) : undefined),
    [currentTabId]
  );

  const findTab = useCallback(
    (target: CoreTabContext): TabContext | undefined => {
      for (const tab of tabMapRef.current.values()) {
        if (tab.type !== target.type) continue;
        if (tab.database.name !== target.database.name) continue;
        if (tab.type === "database") {
          if (tab.metadata.database.name === target.metadata.database.name) {
            return tab;
          }
        }
        // After the type guard above, both tab and target have the same type.
        // We use `as any` on target.metadata to access the narrowed fields
        // since TS can't narrow two variables from the same discriminant check.
        const tabMeta = tab.metadata as Record<string, { name: string }>;
        const targetMeta = target.metadata as Record<string, { name: string }>;
        if (tab.type === "table") {
          if (
            tabMeta.schema.name === targetMeta.schema.name &&
            tabMeta.table.name === targetMeta.table.name
          ) {
            return tab;
          }
        }
        if (tab.type === "view") {
          if (
            tabMeta.schema.name === targetMeta.schema.name &&
            tabMeta.view.name === targetMeta.view.name
          ) {
            return tab;
          }
        }
        if (tab.type === "procedure") {
          if (
            tabMeta.schema.name === targetMeta.schema.name &&
            tabMeta.procedure.name === targetMeta.procedure.name
          ) {
            return tab;
          }
        }
        if (tab.type === "function") {
          if (
            tabMeta.schema.name === targetMeta.schema.name &&
            tabMeta.function.name === targetMeta.function.name
          ) {
            return tab;
          }
        }
      }
      return undefined;
    },
    []
  );

  const addTab = useCallback(
    (coreTab: CoreTabContext, setAsCurrentTab = true) => {
      const existedTab = findTab(coreTab);
      if (existedTab) {
        Object.assign(existedTab, coreTab);
        if (setAsCurrentTab) {
          setCurrentTabId(existedTab.id);
        }
        bump();
        return;
      }

      const id = uuidv1();
      const tab: TabContext = { id, ...coreTab } as TabContext;
      tabMapRef.current.set(id, tab);

      if (setAsCurrentTab) {
        requestAnimationFrame(() => {
          setCurrentTabId(id);
        });
      }
      bump();
    },
    [findTab, bump]
  );

  const setCurrentTab = useCallback((id: string) => {
    if (tabMapRef.current.has(id)) {
      setCurrentTabId(id);
    } else {
      setCurrentTabId("");
    }
  }, []);

  const closeTab = useCallback(
    (id: string) => {
      const list = Array.from(tabMapRef.current.values());
      const index = list.findIndex((tab) => tab.id === id);

      if (currentTabId === id) {
        let nextIndex = -1;
        if (index === 0) {
          nextIndex = 1;
        } else {
          nextIndex = index - 1;
        }
        const nextTab = list[nextIndex];
        setCurrentTabId(nextTab ? nextTab.id : "");
      }

      tabMapRef.current.delete(id);
      bump();
    },
    [currentTabId, bump]
  );

  const clearTabs = useCallback(() => {
    tabMapRef.current.clear();
    setCurrentTabId("");
    bump();
  }, [bump]);

  return {
    tabList,
    currentTabId,
    currentTab,
    addTab,
    setCurrentTab,
    closeTab,
    findTab,
    clearTabs,
  };
}
