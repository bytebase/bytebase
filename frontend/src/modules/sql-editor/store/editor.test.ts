import { afterEach, beforeEach, describe, expect, test } from "vitest";
import {
  STORAGE_KEY_SQL_EDITOR_REDIS_NODE,
  STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT,
  storageKeySqlEditorLastProject,
} from "@/utils/storage-keys";
import {
  getSQLEditorEditorState,
  subscribeSQLEditorEditorState,
  useSQLEditorEditorStore,
} from "./editor";

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
  // Reset the store to its initial state derived from localStorage.
  useSQLEditorEditorStore.setState({
    project: "",
    projectContextReady: false,
    resultRowsLimit: 1000,
    redisCommandOption: 1,
    isShowExecutingHint: false,
    executingHintDatabase: undefined,
  });
});

afterEach(() => {
  Object.defineProperty(globalThis, "localStorage", {
    value: originalLocalStorage,
    configurable: true,
    writable: true,
  });
});

describe("useSQLEditorEditorStore", () => {
  test("setProject updates state, marks context ready, and persists", () => {
    getSQLEditorEditorState().setProject("projects/alpha");

    const next = getSQLEditorEditorState();
    expect(next.project).toBe("projects/alpha");
    expect(next.projectContextReady).toBe(true);
    expect(JSON.parse(storage.get(storageKeySqlEditorLastProject(""))!)).toBe(
      "projects/alpha"
    );
  });

  test("setProjectContextReady toggles the readiness flag", () => {
    getSQLEditorEditorState().setProjectContextReady(true);
    expect(getSQLEditorEditorState().projectContextReady).toBe(true);

    getSQLEditorEditorState().setProjectContextReady(false);
    expect(getSQLEditorEditorState().projectContextReady).toBe(false);
  });

  test("setResultRowsLimit persists numeric limits to localStorage", () => {
    getSQLEditorEditorState().setResultRowsLimit(2500);

    expect(getSQLEditorEditorState().resultRowsLimit).toBe(2500);
    expect(JSON.parse(storage.get(STORAGE_KEY_SQL_EDITOR_RESULT_LIMIT)!)).toBe(
      2500
    );
  });

  test("setRedisCommandOption persists the selected node target", () => {
    // 2 = ALL_NODES per the proto enum.
    getSQLEditorEditorState().setRedisCommandOption(2);

    expect(getSQLEditorEditorState().redisCommandOption).toBe(2);
    expect(JSON.parse(storage.get(STORAGE_KEY_SQL_EDITOR_REDIS_NODE)!)).toBe(2);
  });

  test("setShowExecutingHint and setExecutingHintDatabase are reactive", () => {
    expect(getSQLEditorEditorState().isShowExecutingHint).toBe(false);
    getSQLEditorEditorState().setShowExecutingHint(true);
    expect(getSQLEditorEditorState().isShowExecutingHint).toBe(true);

    const fakeDb = { name: "instances/x/databases/y" } as never;
    getSQLEditorEditorState().setExecutingHintDatabase(fakeDb);
    expect(getSQLEditorEditorState().executingHintDatabase).toBe(fakeDb);
  });

  test("subscribeSQLEditorEditorState notifies on every mutation", () => {
    let calls = 0;
    const unsubscribe = subscribeSQLEditorEditorState(() => {
      calls++;
    });

    getSQLEditorEditorState().setProject("projects/beta");
    getSQLEditorEditorState().setResultRowsLimit(50);
    getSQLEditorEditorState().setShowExecutingHint(true);

    expect(calls).toBeGreaterThanOrEqual(3);
    unsubscribe();

    getSQLEditorEditorState().setShowExecutingHint(false);
    expect(calls).toBeGreaterThanOrEqual(3);
  });
});
