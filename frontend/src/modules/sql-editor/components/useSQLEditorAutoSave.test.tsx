import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useSQLEditorAutoSave } from "./useSQLEditorAutoSave";

const mocks = vi.hoisted(() => ({
  abortAutoSave: vi.fn(),
  setAutoSaveController: vi.fn(),
  maybeUpdateSavedQuery: vi.fn(),
  createSavedQuery: vi.fn(),
  updateTab: vi.fn(),
  getSavedQueryByName: vi.fn(),
  notify: vi.fn(),
  canCreateSavedQueryInProject: vi.fn(() => true),
  tab: {
    id: "tab-1",
    savedQuery: "",
    status: "DIRTY",
    statement: "SELECT 1",
    connection: {
      instance: "instances/inst1",
      database: "instances/inst1/databases/db1",
    },
  },
  tabsById: new Map(),
  currentTabId: "tab-1",
}));

vi.mock("@/modules/sql-editor/store", () => ({
  useSQLEditorStore: Object.assign(
    (
      selector: (state: {
        abortAutoSave: typeof mocks.abortAutoSave;
        setAutoSaveController: typeof mocks.setAutoSaveController;
        maybeUpdateSavedQuery: typeof mocks.maybeUpdateSavedQuery;
        createSavedQuery: typeof mocks.createSavedQuery;
      }) => unknown
    ) =>
      selector({
        abortAutoSave: mocks.abortAutoSave,
        setAutoSaveController: mocks.setAutoSaveController,
        maybeUpdateSavedQuery: mocks.maybeUpdateSavedQuery,
        createSavedQuery: mocks.createSavedQuery,
      }),
    { getState: () => ({ autoSaveController: null }) }
  ),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  getSQLEditorEditorState: () => ({ project: "projects/proj1" }),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getSQLEditorTabsState: () => ({
    currentTabId: mocks.currentTabId,
    tabsById: mocks.tabsById,
    updateTab: mocks.updateTab,
  }),
  useSQLEditorTabState: (
    selector: (state: {
      currentTabId: string;
      tabsById: typeof mocks.tabsById;
      updateTab: typeof mocks.updateTab;
    }) => unknown
  ) =>
    selector({
      currentTabId: mocks.currentTabId,
      tabsById: mocks.tabsById,
      updateTab: mocks.updateTab,
    }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      getSavedQueryByName: mocks.getSavedQueryByName,
      notify: mocks.notify,
    }),
  },
}));

vi.mock("@/utils", () => ({
  canCreateSavedQueryInProject: mocks.canCreateSavedQueryInProject,
  isSavedQueryWritableV1: vi.fn(() => true),
}));

describe("useSQLEditorAutoSave", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    mocks.tab.savedQuery = "";
    mocks.tab.status = "DIRTY";
    mocks.tab.statement = "SELECT 1";
    mocks.tab.connection = {
      instance: "instances/inst1",
      database: "instances/inst1/databases/db1",
    };
    mocks.tabsById.clear();
    mocks.tabsById.set(mocks.tab.id, mocks.tab);
    mocks.currentTabId = mocks.tab.id;
    mocks.createSavedQuery.mockResolvedValue(undefined);
    mocks.updateTab.mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test("creates a saved query after an unsaved tab receives SQL", async () => {
    renderHook(() => useSQLEditorAutoSave());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mocks.createSavedQuery).toHaveBeenCalledWith({
      tabId: "tab-1",
      database: "instances/inst1/databases/db1",
      statement: "SELECT 1",
      signal: expect.any(AbortSignal),
    });
  });

  test("does not save whitespace-only drafts", async () => {
    mocks.tab.statement = "  \n\t ";
    renderHook(() => useSQLEditorAutoSave());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mocks.createSavedQuery).not.toHaveBeenCalled();
  });

  test("saves a whitespace-only statement for an existing saved query", async () => {
    mocks.tab.savedQuery = "projects/proj1/savedQueries/query1";
    mocks.tab.statement = "  \n\t ";
    mocks.getSavedQueryByName.mockReturnValue({
      name: mocks.tab.savedQuery,
    });
    mocks.maybeUpdateSavedQuery.mockResolvedValue(undefined);

    renderHook(() => useSQLEditorAutoSave());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mocks.maybeUpdateSavedQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        tabId: "tab-1",
        savedQuery: "projects/proj1/savedQueries/query1",
        statement: "  \n\t ",
      })
    );
  });

  test("does not start a second create while the first one is saving", async () => {
    mocks.tab.status = "SAVING";
    renderHook(() => useSQLEditorAutoSave());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mocks.createSavedQuery).not.toHaveBeenCalled();
  });

  test("saves the edited tab after switching away from an identical clean tab", async () => {
    const cleanTab = {
      ...mocks.tab,
      id: "tab-2",
      status: "CLEAN",
      connection: { ...mocks.tab.connection },
    };
    mocks.tabsById.set(cleanTab.id, cleanTab);
    const { rerender } = renderHook(() => useSQLEditorAutoSave());

    mocks.currentTabId = cleanTab.id;
    rerender();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    expect(mocks.createSavedQuery).not.toHaveBeenCalled();

    mocks.currentTabId = mocks.tab.id;
    rerender();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mocks.createSavedQuery).toHaveBeenCalledWith(
      expect.objectContaining({ tabId: "tab-1" })
    );
  });

  test("saves a newer database selection after the first create completes", async () => {
    let resolveCreate: () => void;
    mocks.createSavedQuery.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveCreate = resolve;
        })
    );
    mocks.updateTab.mockImplementation((_id, payload) => {
      Object.assign(mocks.tab, payload);
    });
    const { rerender } = renderHook(() => useSQLEditorAutoSave());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    mocks.tab.connection = {
      instance: "instances/inst2",
      database: "instances/inst2/databases/db2",
    };
    rerender();
    mocks.tab.savedQuery = "projects/proj1/savedQueries/query1";
    mocks.tab.status = "DIRTY";
    mocks.getSavedQueryByName.mockReturnValue({ name: mocks.tab.savedQuery });

    await act(async () => {
      resolveCreate();
      await Promise.resolve();
    });

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mocks.maybeUpdateSavedQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        database: "instances/inst2/databases/db2",
      })
    );
  });

  test("does not retry a failed save until the tab changes again", async () => {
    let rejectCreate: (reason?: unknown) => void;
    mocks.createSavedQuery.mockImplementation(
      () =>
        new Promise<void>((_resolve, reject) => {
          rejectCreate = reject;
        })
    );
    mocks.updateTab.mockImplementation((_id, payload) => {
      Object.assign(mocks.tab, payload);
    });
    const { rerender } = renderHook(() => useSQLEditorAutoSave());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });
    rerender();

    await act(async () => {
      rejectCreate(new Error("network failed"));
      await Promise.resolve();
    });
    rerender();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mocks.createSavedQuery).toHaveBeenCalledTimes(1);
  });

  test("returns an aborted tab to dirty", async () => {
    mocks.createSavedQuery.mockRejectedValue(
      new DOMException("Aborted", "AbortError")
    );
    mocks.updateTab.mockImplementation((_id, payload) => {
      Object.assign(mocks.tab, payload);
    });
    renderHook(() => useSQLEditorAutoSave());

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2000);
    });

    expect(mocks.updateTab).toHaveBeenLastCalledWith("tab-1", {
      status: "DIRTY",
    });
  });
});
