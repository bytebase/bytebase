import { beforeEach, describe, expect, test, vi } from "vitest";
import { create, type StoreApi } from "zustand";
import { getSQLEditorEditorState } from "./editor";
import { createSavedQuerySaveSlice } from "./savedQuery";
import { getSQLEditorTabsState } from "./tab";
import type {
  QueryHistorySlice,
  SQLEditorStoreState,
  TreeSlice,
  UIStateSlice,
  WebTerminalSlice,
} from "./types";

// Stub all the other slices so the composed store satisfies
// `SQLEditorStoreState` without dragging in real implementations
// (the saved query slice itself dynamic-imports the Pinia stores it
// needs, so we only have to mock those via vi.mock below).
const stubUIStateSlice = (): UIStateSlice => ({
  asidePanelTab: "SAVED_QUERY",
  showConnectionPanel: false,
  showAIPanel: false,
  pendingInsertAtCaret: undefined,
  highlightAccessGrantName: undefined,
  isShowingCode: false,
  aiPanelSize: 0.3,
  linkedQueryHistory: undefined,
  linkedQueryHistoryTabId: undefined,
  linkedQueryHistoryBaseline: undefined,
  setAsidePanelTab: vi.fn(),
  setLinkedQueryHistory: vi.fn(),
  setShowConnectionPanel: vi.fn(),
  setShowAIPanel: vi.fn(),
  setPendingInsertAtCaret: vi.fn(),
  setHighlightAccessGrantName: vi.fn(),
  setIsShowingCode: vi.fn(),
  handleEditorPanelResize: vi.fn(),
});

const stubQueryHistorySlice = (): QueryHistorySlice => ({
  queryHistoryByKey: {},
  fetchQueryHistoryList: vi.fn(async () => undefined as never),
  resetPageToken: vi.fn(),
  mergeLatest: vi.fn(async () => undefined as never),
  fetchQueryHistory: vi.fn(async () => undefined as never),
});

const stubTreeSlice = (): TreeSlice => ({
  treeState: "UNSET",
  treeNodeKeysById: {},
  setTreeState: vi.fn(),
  collectTreeNode: vi.fn(),
  treeNodeKeysByTarget: vi.fn(() => []),
});

const stubWebTerminalSlice = (): WebTerminalSlice => ({
  webTerminalQueryItemsByTabId: {},
  ensureWebTerminalQueryState: vi.fn(),
  clearWebTerminalQueryState: vi.fn(),
  replaceWebTerminalQueryItems: vi.fn(),
  pushWebTerminalQueryItem: vi.fn(),
  updateWebTerminalQueryItem: vi.fn(),
});

const piniaMocks = vi.hoisted(() => ({
  editorStore: {
    project: "projects/default",
    projectContextReady: true,
    setProject: vi.fn(),
  },
  projectStore: {
    getOrFetchProjectByName: vi.fn(),
  },
  projectIamPolicyStore: {
    getOrFetchProjectIamPolicy: vi.fn(),
  },
  tabStore: {
    updateTab: vi.fn(),
  },
  savedQueryStore: {
    getSavedQueryByName: vi.fn(),
    patchSavedQuery: vi.fn(),
    createSavedQuery: vi.fn(),
    fetchProject: vi.fn(),
    loadProjectIamPolicy: vi.fn(),
  },
}));

vi.mock("@/stores", () => ({
  useProjectV1Store: () => piniaMocks.projectStore,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => piniaMocks.savedQueryStore },
}));

vi.mock("@/modules/sql-editor/store/tab-vue-state", () => ({
  useSQLEditorTabStore: () => piniaMocks.tabStore,
}));

vi.mock("./editor-vue-state", () => ({
  useSQLEditorVueState: () => piniaMocks.editorStore,
}));

vi.mock("@/stores/modules/v1/projectIamPolicy", () => ({
  useProjectIamPolicyStore: () => piniaMocks.projectIamPolicyStore,
}));

vi.mock("@/modules/sql-editor/model/events", () => ({
  sqlEditorEvents: { emit: vi.fn() },
}));

vi.mock("@/modules/sql-editor/model/Sheet", () => ({
  openSavedQueryByName: vi.fn(),
}));

vi.mock("@/lib/sqlEditorConnection", () => ({
  extractSavedQueryConnection: vi.fn(
    async ({ database }: { database: string }) => ({
      instance: database ? "instances/cosmos" : "",
      database,
    })
  ),
}));

const makeStore = (): StoreApi<SQLEditorStoreState> =>
  create<SQLEditorStoreState>()((...args) => ({
    ...stubUIStateSlice(),
    ...stubQueryHistorySlice(),
    ...stubTreeSlice(),
    ...stubWebTerminalSlice(),
    ...createSavedQuerySaveSlice(...args),
  }));

beforeEach(() => {
  Object.values(piniaMocks).forEach((store) => {
    Object.values(store).forEach((v) => {
      if (typeof v === "function" && "mockReset" in v) {
        (v as { mockReset: () => void }).mockReset();
      }
    });
  });
  piniaMocks.editorStore.project = "projects/default";
  piniaMocks.editorStore.projectContextReady = true;
  getSQLEditorEditorState().setProject("");
  getSQLEditorTabsState().reset();
});

describe("saved query save slice — autoSaveController", () => {
  test("initial value is null", () => {
    const store = makeStore();
    expect(store.getState().autoSaveController).toBeNull();
  });

  test("setAutoSaveController writes the new value", () => {
    const store = makeStore();
    const controller = new AbortController();
    store.getState().setAutoSaveController(controller);
    expect(store.getState().autoSaveController).toBe(controller);
  });

  test("abortAutoSave with no controller is a no-op", () => {
    const store = makeStore();
    expect(() => store.getState().abortAutoSave()).not.toThrow();
    expect(store.getState().autoSaveController).toBeNull();
  });

  test("abortAutoSave with a controller aborts and clears it", () => {
    const store = makeStore();
    const controller = new AbortController();
    const abortSpy = vi.spyOn(controller, "abort");
    store.getState().setAutoSaveController(controller);
    store.getState().abortAutoSave();
    expect(abortSpy).toHaveBeenCalledTimes(1);
    expect(store.getState().autoSaveController).toBeNull();
  });
});

describe("saved query save slice — maybeUpdateSavedQuery", () => {
  test("preserves the selected Cosmos DB container when saving the saved query", async () => {
    const store = makeStore();
    const tab = getSQLEditorTabsState().addTab({
      savedQuery: "projects/default/savedQueries/cosmos-sheet",
      connection: {
        instance: "instances/cosmos",
        database: "instances/cosmos/databases/grs",
        table: "SUPPORDERS_VIS.items",
      },
      statement: "select * from SUPPORDERS_VIS.items",
      status: "DIRTY",
    });
    piniaMocks.savedQueryStore.getSavedQueryByName.mockReturnValue({
      name: tab.savedQuery,
      title: "Cosmos saved query",
      database: tab.connection.database,
    });
    piniaMocks.savedQueryStore.patchSavedQuery.mockResolvedValue({
      name: tab.savedQuery,
    });

    await store.getState().maybeUpdateSavedQuery({
      tabId: tab.id,
      savedQuery: tab.savedQuery,
      database: tab.connection.database,
      statement: tab.statement,
    });

    expect(getSQLEditorTabsState().tabsById.get(tab.id)?.connection).toEqual({
      instance: "instances/cosmos",
      database: "instances/cosmos/databases/grs",
      table: "SUPPORDERS_VIS.items",
    });
  });
});

describe("saved query save slice — maybeSwitchProject", () => {
  test("with an invalid project name returns undefined without setting project", async () => {
    const store = makeStore();
    const result = await store.getState().maybeSwitchProject("not-a-project");
    expect(result).toBeUndefined();
    expect(piniaMocks.editorStore.setProject).not.toHaveBeenCalled();
  });

  test("switches project even when project IAM policy preload fails", async () => {
    piniaMocks.savedQueryStore.fetchProject.mockResolvedValue({
      name: "projects/aaa",
    });
    piniaMocks.savedQueryStore.loadProjectIamPolicy.mockRejectedValue(
      new Error("permission denied")
    );

    const store = makeStore();
    const result = await store.getState().maybeSwitchProject("projects/aaa");

    expect(result).toBe("projects/aaa");
    expect(getSQLEditorEditorState().project).toBe("projects/aaa");
  });
});
