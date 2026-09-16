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
}));

vi.mock("@/modules/sql-editor/store", () => ({
  useSQLEditorStore: (
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
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  getSQLEditorEditorState: () => ({ project: "projects/proj1" }),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getSQLEditorTabsState: () => ({
    currentTabId: mocks.tab.id,
    tabsById: new Map([[mocks.tab.id, mocks.tab]]),
    updateTab: mocks.updateTab,
  }),
  useSQLEditorTabState: (
    selector: (state: {
      currentTabId: string;
      tabsById: Map<string, typeof mocks.tab>;
      updateTab: typeof mocks.updateTab;
    }) => unknown
  ) =>
    selector({
      currentTabId: mocks.tab.id,
      tabsById: new Map([[mocks.tab.id, mocks.tab]]),
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
    mocks.createSavedQuery.mockResolvedValue(undefined);
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
});
