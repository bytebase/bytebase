import { afterEach, beforeEach, describe, expect, test } from "vitest";
import {
  __resetTabStoreProjectCursor,
  getSQLEditorTabsState,
  subscribeSQLEditorTabsState,
  useSQLEditorTabsStore,
} from "./tab";

const storage = new Map<string, string>();
const originalLocalStorage = globalThis.localStorage;

const localStorageMock = {
  getItem: (key: string) => storage.get(key) ?? null,
  setItem: (key: string, value: string) => {
    storage.set(key, value);
  },
  removeItem: (key: string) => {
    storage.delete(key);
  },
  clear: () => {
    storage.clear();
  },
  key: (index: number) => Array.from(storage.keys())[index] ?? null,
  get length() {
    return storage.size;
  },
};

beforeEach(() => {
  storage.clear();
  Object.defineProperty(globalThis, "localStorage", {
    value: localStorageMock,
    configurable: true,
    writable: true,
  });
  getSQLEditorTabsState().reset();
  __resetTabStoreProjectCursor();
});

afterEach(() => {
  Object.defineProperty(globalThis, "localStorage", {
    value: originalLocalStorage,
    configurable: true,
    writable: true,
  });
});

describe("useSQLEditorTabsStore", () => {
  test("addTab seeds a new tab, sets it current, and tracks open order", () => {
    const tab = getSQLEditorTabsState().addTab({ title: "first" });

    const state = useSQLEditorTabsStore.getState();
    expect(state.currentTabId).toBe(tab.id);
    expect(state.tabsById.get(tab.id)?.title).toBe("first");
    expect(state.openTmpTabList.map((t) => t.id)).toEqual([tab.id]);
  });

  test("addTab appends or inserts beside the current tab based on flag", () => {
    const a = getSQLEditorTabsState().addTab({ title: "a" });
    const c = getSQLEditorTabsState().addTab({ title: "c" });
    // Insert "b" beside the current tab (c) — should land after c.
    const b = getSQLEditorTabsState().addTab({ title: "b" }, true);

    const order = useSQLEditorTabsStore
      .getState()
      .openTmpTabList.map((t) => t.id);
    expect(order).toEqual([a.id, c.id, b.id]);
  });

  test("updateTab merges fields into the existing tab", () => {
    const tab = getSQLEditorTabsState().addTab({ title: "before" });
    getSQLEditorTabsState().updateTab(tab.id, {
      title: "after",
      status: "DIRTY",
    });

    const stored = useSQLEditorTabsStore.getState().tabsById.get(tab.id);
    expect(stored?.title).toBe("after");
    expect(stored?.status).toBe("DIRTY");
  });

  test("updateCurrentTab is a shorthand over updateTab", () => {
    const tab = getSQLEditorTabsState().addTab({ statement: "SELECT 1" });
    getSQLEditorTabsState().updateCurrentTab({ statement: "SELECT 2" });

    const stored = useSQLEditorTabsStore.getState().tabsById.get(tab.id);
    expect(stored?.statement).toBe("SELECT 2");
  });

  test("closeTab removes the tab and advances currentTabId", () => {
    const a = getSQLEditorTabsState().addTab({ title: "a" });
    const b = getSQLEditorTabsState().addTab({ title: "b" });

    getSQLEditorTabsState().closeTab(b.id);

    const state = useSQLEditorTabsStore.getState();
    expect(state.tabsById.has(b.id)).toBe(false);
    expect(state.openTmpTabList.map((t) => t.id)).toEqual([a.id]);
    expect(state.currentTabId).toBe(a.id);
  });

  test("closeTab on the only tab clears currentTabId", () => {
    const a = getSQLEditorTabsState().addTab({ title: "lonely" });

    getSQLEditorTabsState().closeTab(a.id);

    const state = useSQLEditorTabsStore.getState();
    expect(state.openTmpTabList).toHaveLength(0);
    expect(state.currentTabId).toBe("");
  });

  test("setOpenTabListOrder rewrites the persisted order without losing tabs", () => {
    const a = getSQLEditorTabsState().addTab({ title: "a" });
    const b = getSQLEditorTabsState().addTab({ title: "b" });

    const reordered = [
      ...useSQLEditorTabsStore.getState().openTmpTabList,
    ].reverse();
    getSQLEditorTabsState().setOpenTabListOrder(reordered);

    const order = useSQLEditorTabsStore
      .getState()
      .openTmpTabList.map((t) => t.id);
    expect(order).toEqual([b.id, a.id]);
    // Live tab map unchanged.
    expect(useSQLEditorTabsStore.getState().tabsById.size).toBe(2);
  });

  test("cloneTab copies statement and connection into a new tab", () => {
    const original = getSQLEditorTabsState().addTab({
      title: "orig",
      statement: "SELECT *",
    });
    const cloned = getSQLEditorTabsState().cloneTab(original.id, {
      title: "clone",
    });

    expect(cloned.id).not.toBe(original.id);
    expect(cloned.statement).toBe("SELECT *");
    expect(useSQLEditorTabsStore.getState().currentTabId).toBe(cloned.id);
  });

  test("updateBatchQueryContext seeds and merges batch context", () => {
    const tab = getSQLEditorTabsState().addTab({ title: "batch" });

    getSQLEditorTabsState().updateBatchQueryContext({
      databases: ["instances/x/databases/y"],
    });

    const stored = useSQLEditorTabsStore.getState().tabsById.get(tab.id);
    expect(stored?.batchQueryContext.databases).toEqual([
      "instances/x/databases/y",
    ]);
  });

  test("database query context add/update/remove flow", () => {
    const tab = getSQLEditorTabsState().addTab({ title: "ctx" });
    const db = "instances/x/databases/y";
    // Seed the context map via updateTab (the public action) so the
    // mutation flows through the same immer path that production
    // callers use.
    getSQLEditorTabsState().updateTab(tab.id, {
      databaseQueryContexts: new Map([
        [
          db,
          [
            // The tests below don't read params/engine/etc., so cast a
            // shape-light fixture rather than constructing a full
            // SQLEditorQueryParams (which would require monaco types).
            { id: "c1", status: "PENDING" } as never,
            { id: "c2", status: "PENDING" } as never,
          ],
        ],
      ]),
    });

    getSQLEditorTabsState().updateDatabaseQueryContext({
      database: db,
      contextId: "c1",
      context: { status: "DONE" },
    });
    expect(
      useSQLEditorTabsStore
        .getState()
        .tabsById.get(tab.id)
        ?.databaseQueryContexts?.get(db)
        ?.find((c) => c.id === "c1")?.status
    ).toBe("DONE");

    getSQLEditorTabsState().removeDatabaseQueryContext({
      database: db,
      contextId: "c1",
    });
    const remainingIds = useSQLEditorTabsStore
      .getState()
      .tabsById.get(tab.id)
      ?.databaseQueryContexts?.get(db)
      ?.map((c) => c.id);
    expect(remainingIds).toEqual(["c2"]);

    getSQLEditorTabsState().deleteDatabaseQueryContext(db);
    expect(
      useSQLEditorTabsStore
        .getState()
        .tabsById.get(tab.id)
        ?.databaseQueryContexts?.has(db)
    ).toBe(false);
  });

  test("subscribeSQLEditorTabsState fires on every mutation", () => {
    let count = 0;
    const unsubscribe = subscribeSQLEditorTabsState(() => {
      count++;
    });

    const a = getSQLEditorTabsState().addTab({ title: "a" });
    getSQLEditorTabsState().updateTab(a.id, { title: "a2" });
    // Force a distinct mutation so immer produces a new state and the
    // subscription fires (setting currentTabId to its current value is
    // a no-op that immer collapses away).
    getSQLEditorTabsState().setCurrentTabId("");

    expect(count).toBeGreaterThanOrEqual(3);
    unsubscribe();
  });
});
