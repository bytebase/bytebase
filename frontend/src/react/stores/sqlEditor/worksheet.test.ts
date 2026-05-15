import { beforeEach, describe, expect, test, vi } from "vitest";
import { create, type StoreApi } from "zustand";
import type {
  QueryHistorySlice,
  SQLEditorStoreState,
  TreeSlice,
  UIStateSlice,
  WebTerminalSlice,
} from "./types";
import { createWorksheetSaveSlice } from "./worksheet";

// Stub all the other slices so the composed store satisfies
// `SQLEditorStoreState` without dragging in real implementations
// (the worksheet slice itself dynamic-imports the Pinia stores it
// needs, so we only have to mock those via vi.mock below).
const stubUIStateSlice = (): UIStateSlice => ({
  asidePanelTab: "WORKSHEET",
  showConnectionPanel: false,
  showAIPanel: false,
  pendingInsertAtCaret: undefined,
  highlightAccessGrantName: undefined,
  isShowingCode: false,
  aiPanelSize: 0.3,
  setAsidePanelTab: vi.fn(),
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
  worksheetStore: {
    getWorksheetByName: vi.fn(),
    patchWorksheet: vi.fn(),
    upsertWorksheetOrganizer: vi.fn(),
    createWorksheet: vi.fn(),
  },
}));

vi.mock("@/store", () => ({
  useProjectV1Store: () => piniaMocks.projectStore,
  useWorkSheetStore: () => piniaMocks.worksheetStore,
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: () => piniaMocks.tabStore,
}));

vi.mock("./editor-vue-state", () => ({
  useSQLEditorVueState: () => piniaMocks.editorStore,
}));

vi.mock("@/store/modules/v1/projectIamPolicy", () => ({
  useProjectIamPolicyStore: () => piniaMocks.projectIamPolicyStore,
}));

vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: { emit: vi.fn() },
}));

vi.mock("@/views/sql-editor/Sheet", () => ({
  openWorksheetByName: vi.fn(),
}));

vi.mock("@/utils", async (importOriginal) => {
  const actual = (await importOriginal()) as Record<string, unknown>;
  return {
    ...actual,
    extractWorksheetConnection: vi.fn(
      async ({ database }: { database: string }) => ({
        database,
        instance: "",
      })
    ),
  };
});

const makeStore = (): StoreApi<SQLEditorStoreState> =>
  create<SQLEditorStoreState>()((...args) => ({
    ...stubUIStateSlice(),
    ...stubQueryHistorySlice(),
    ...stubTreeSlice(),
    ...stubWebTerminalSlice(),
    ...createWorksheetSaveSlice(...args),
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
});

describe("worksheet save slice — autoSaveController", () => {
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

describe("worksheet save slice — maybeSwitchProject", () => {
  test("with an invalid project name returns undefined without setting project", async () => {
    const store = makeStore();
    const result = await store.getState().maybeSwitchProject("not-a-project");
    expect(result).toBeUndefined();
    expect(piniaMocks.editorStore.setProject).not.toHaveBeenCalled();
  });
});
