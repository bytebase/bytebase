import { beforeEach, describe, expect, test, vi } from "vitest";
import { create, type StoreApi } from "zustand";
import {
  createQueryHistorySlice,
  getQueryHistoryCacheKey,
  selectQueryHistoryEntry,
} from "./queryHistory";
import type {
  QueryHistoryFilter,
  SQLEditorStoreState,
  TreeSlice,
  UIStateSlice,
  WebTerminalSlice,
  WorksheetSaveSlice,
} from "./types";

const searchQueryHistoriesMock = vi.fn();

// Stub the connect client so the slice's `mergeLatest` /
// `fetchQueryHistoryList` calls hit a deterministic in-memory fake.
vi.mock("@/api", () => ({
  queryHistoryServiceClientConnect: {
    searchQueryHistories: (...args: unknown[]) =>
      searchQueryHistoriesMock(...args),
  },
}));

// Stub the UIState half so the composed store satisfies the
// `SQLEditorStoreState` shape without dragging in the real uiState
// slice (which reads localStorage on creation).
const stubUIStateSlice = (): UIStateSlice => ({
  asidePanelTab: "WORKSHEET",
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

const stubWorksheetSaveSlice = (): WorksheetSaveSlice => ({
  autoSaveController: null,
  setAutoSaveController: vi.fn(),
  abortAutoSave: vi.fn(),
  maybeSwitchProject: vi.fn(async () => undefined),
  maybeUpdateWorksheet: vi.fn(async () => undefined),
  createWorksheet: vi.fn(async () => undefined),
});

const makeStore = (): StoreApi<SQLEditorStoreState> =>
  create<SQLEditorStoreState>()((...args) => ({
    ...stubUIStateSlice(),
    ...stubTreeSlice(),
    ...stubWebTerminalSlice(),
    ...stubWorksheetSaveSlice(),
    ...createQueryHistorySlice(...args),
  }));

const filter: QueryHistoryFilter = {
  project: "projects/p1",
  database: "instances/inst1/databases/db1",
};

const history = (name: string) => ({
  name,
  statement: `-- ${name}`,
  createTime: undefined,
});

beforeEach(() => {
  searchQueryHistoriesMock.mockReset();
});

describe("getQueryHistoryCacheKey", () => {
  test("composes project + database when both are valid", () => {
    expect(
      getQueryHistoryCacheKey({
        project: "projects/p1",
        database: "instances/i/databases/d",
      })
    ).toBe("projects/p1.instances/i/databases/d");
  });

  test("omits invalid resource segments", () => {
    expect(getQueryHistoryCacheKey({ project: "p1", database: "" })).toBe("");
  });
});

describe("queryHistory slice — request parent scoping", () => {
  test("passes the project as parent and keeps project out of the CEL filter", async () => {
    searchQueryHistoriesMock.mockResolvedValue({
      queryHistories: [],
      nextPageToken: "",
    });
    const store = makeStore();
    await store.getState().fetchQueryHistoryList(filter);
    await store.getState().mergeLatest(filter);
    expect(searchQueryHistoriesMock).toHaveBeenCalledTimes(2);
    for (const call of searchQueryHistoriesMock.mock.calls) {
      const request = call[0] as { parent: string; filter: string };
      expect(request.parent).toBe("projects/p1");
      expect(request.filter).not.toContain("project ==");
    }
  });

  test("skips the request entirely without a valid project", async () => {
    const store = makeStore();
    const emptyFilter: QueryHistoryFilter = { project: "", database: "" };
    const fetchResp = await store.getState().fetchQueryHistoryList(emptyFilter);
    const mergeResp = await store.getState().mergeLatest(emptyFilter);
    expect(searchQueryHistoriesMock).not.toHaveBeenCalled();
    expect(fetchResp.queryHistories).toEqual([]);
    expect(mergeResp.queryHistories).toEqual([]);
    const entry = selectQueryHistoryEntry(emptyFilter)(store.getState());
    expect(entry.queryHistories).toEqual([]);
  });
});

describe("queryHistory slice — fetchQueryHistoryList", () => {
  test("replaces the list on first page (no pageToken)", async () => {
    searchQueryHistoriesMock.mockResolvedValue({
      queryHistories: [history("a"), history("b")],
      nextPageToken: "tok-1",
    });
    const store = makeStore();
    await store.getState().fetchQueryHistoryList(filter);
    const entry = selectQueryHistoryEntry(filter)(store.getState());
    expect(entry.queryHistories.map((h) => h.name)).toEqual(["a", "b"]);
    expect(entry.nextPageToken).toBe("tok-1");
  });

  test("appends + dedupes by `name` on subsequent pages", async () => {
    searchQueryHistoriesMock.mockResolvedValueOnce({
      queryHistories: [history("a"), history("b")],
      nextPageToken: "tok-2",
    });
    const store = makeStore();
    await store.getState().fetchQueryHistoryList(filter);

    searchQueryHistoriesMock.mockResolvedValueOnce({
      // `b` is a duplicate (cursor drift) and must be filtered.
      queryHistories: [history("b"), history("c")],
      nextPageToken: "tok-3",
    });
    await store.getState().fetchQueryHistoryList(filter);

    const entry = selectQueryHistoryEntry(filter)(store.getState());
    expect(entry.queryHistories.map((h) => h.name)).toEqual(["a", "b", "c"]);
    expect(entry.nextPageToken).toBe("tok-3");
  });
});

describe("queryHistory slice — mergeLatest", () => {
  test("prepends fresh entries and preserves the existing nextPageToken", async () => {
    // Seed page 1.
    searchQueryHistoriesMock.mockResolvedValueOnce({
      queryHistories: [history("a"), history("b")],
      nextPageToken: "tok-1",
    });
    const store = makeStore();
    await store.getState().fetchQueryHistoryList(filter);

    // mergeLatest returns the just-executed query at the top.
    searchQueryHistoriesMock.mockResolvedValueOnce({
      queryHistories: [history("z"), history("a")],
      nextPageToken: "tok-fresh",
    });
    await store.getState().mergeLatest(filter);

    const entry = selectQueryHistoryEntry(filter)(store.getState());
    // `z` is new and prepended; `a` is deduped.
    expect(entry.queryHistories.map((h) => h.name)).toEqual(["z", "a", "b"]);
    // Cursor is preserved (existing wins).
    expect(entry.nextPageToken).toBe("tok-1");
  });

  test("falls back to the response cursor when no cached entry exists yet", async () => {
    searchQueryHistoriesMock.mockResolvedValue({
      queryHistories: [history("z")],
      nextPageToken: "tok-from-resp",
    });
    const store = makeStore();
    await store.getState().mergeLatest(filter);
    const entry = selectQueryHistoryEntry(filter)(store.getState());
    expect(entry.nextPageToken).toBe("tok-from-resp");
  });
});

describe("queryHistory slice — resetPageToken", () => {
  test("clears the entry for the given filter only", async () => {
    searchQueryHistoriesMock.mockResolvedValueOnce({
      queryHistories: [history("a")],
      nextPageToken: "tok-1",
    });
    const store = makeStore();
    await store.getState().fetchQueryHistoryList(filter);

    store.getState().resetPageToken(filter);
    const entry = selectQueryHistoryEntry(filter)(store.getState());
    expect(entry.queryHistories).toEqual([]);
    expect(entry.nextPageToken).toBeUndefined();
  });
});

describe("selectQueryHistoryEntry", () => {
  test("returns the same stable empty reference when the entry is missing", () => {
    const store = makeStore();
    const a = selectQueryHistoryEntry(filter)(store.getState());
    const b = selectQueryHistoryEntry(filter)(store.getState());
    // Reference equality — a stable empty default keeps zustand selectors
    // from re-rendering on every store change for unseen filters.
    expect(a).toBe(b);
    expect(a.queryHistories).toEqual([]);
  });
});
