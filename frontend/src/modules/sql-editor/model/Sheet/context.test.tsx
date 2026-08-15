import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  SavedQuerySchema,
  type SavedQuery,
} from "@/types/proto-es/v1/saved_query_service_pb";
import {
  storageKeySqlEditorSavedQueryFilter,
  storageKeySqlEditorSavedQueryFolder,
} from "@/utils";

type AppState = {
  currentUser: { email: string; workspace: string };
  savedQueriesByKey: Record<string, SavedQuery>;
  isSaaSMode: () => boolean;
  getSavedQueryByName: (name: string) => SavedQuery | undefined;
  savedQueryList: () => SavedQuery[];
  patchSavedQueryFolderInCache: ReturnType<typeof vi.fn>;
  searchSavedQueryFolders: ReturnType<typeof vi.fn>;
  fetchSavedQueryList: ReturnType<typeof vi.fn>;
  updateSavedQueryStar: ReturnType<typeof vi.fn>;
  moveMySavedQueries: ReturnType<typeof vi.fn>;
  patchSavedQuery: ReturnType<typeof vi.fn>;
};

const mocks = vi.hoisted(() => {
  let appState: AppState;
  const appSubscribers = new Set<(state: AppState, prev: AppState) => void>();
  const keyForSavedQuery = (name: string) => `${name.split("/").pop()}:FULL`;

  const setSavedQueries = (savedQueries: SavedQuery[]) => {
    const prev = {
      ...appState,
      savedQueriesByKey: appState.savedQueriesByKey,
    };
    appState = {
      ...appState,
      savedQueriesByKey: {
        ...appState.savedQueriesByKey,
        ...Object.fromEntries(
          savedQueries.map((savedQuery) => [
            keyForSavedQuery(savedQuery.name),
            savedQuery,
          ])
        ),
      },
    };
    for (const subscriber of appSubscribers) {
      subscriber(appState, prev);
    }
  };

  return {
    getAppState: () => appState,
    resetAppState: () => {
      appSubscribers.clear();
      appState = {
        currentUser: {
          email: "creator@example.com",
          workspace: "workspaces/default",
        },
        savedQueriesByKey: {},
        isSaaSMode: () => false,
        getSavedQueryByName: (name: string) =>
          appState.savedQueriesByKey[keyForSavedQuery(name)],
        savedQueryList: () => Object.values(appState.savedQueriesByKey),
        searchSavedQueryFolders: vi.fn(async () => []),
        fetchSavedQueryList: vi.fn(async (_project, _filter, _params) => {
          const savedQuery = create(SavedQuerySchema, {
            name: "projects/proj1/savedQueries/1",
            project: "projects/proj1",
            creator: "users/creator@example.com",
            title: "Existing saved query",
          });
          setSavedQueries([savedQuery]);
          return { savedQueries: [savedQuery], nextPageToken: "next-page" };
        }),
        updateSavedQueryStar: vi.fn(),
        moveMySavedQueries: vi.fn(async () => 0),
        patchSavedQuery: vi.fn(async () => undefined),
        patchSavedQueryFolderInCache: vi.fn(),
      };
    },
    addSavedQueries: setSavedQueries,
    useAppStore: {
      getState: () => appState,
      subscribe: (subscriber: (state: AppState, prev: AppState) => void) => {
        appSubscribers.add(subscriber);
        return () => appSubscribers.delete(subscriber);
      },
    },
    getSQLEditorTabsState: vi.fn(() => ({
      currentTabId: "",
      openTmpTabList: [],
      tabsById: new Map(),
    })),
    subscribeSQLEditorEditorState: vi.fn(),
    subscribeSQLEditorTabsState: vi.fn(),
  };
});

vi.mock("@/lib/i18n", () => ({
  default: { t: (key: string) => key },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: mocks.useAppStore,
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  getSQLEditorEditorState: () => ({ project: "projects/proj1" }),
  subscribeSQLEditorEditorState: mocks.subscribeSQLEditorEditorState,
  useSQLEditorEditorStore: vi.fn(),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getSQLEditorTabsState: mocks.getSQLEditorTabsState,
  subscribeSQLEditorTabsState: mocks.subscribeSQLEditorTabsState,
  useSQLEditorTabsStore: vi.fn(),
}));

describe("sheet context", () => {
  let root: Root;
  let container: HTMLDivElement;

  beforeEach(() => {
    vi.useFakeTimers();
    mocks.resetAppState();
    window.localStorage.clear();
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  afterEach(() => {
    root?.unmount();
    container.remove();
    vi.useRealTimers();
    vi.resetModules();
  });

  test("adds newly created saved queries to the initialized paged my view", async () => {
    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return (
        <div>
          {viewContext.sheetTree.children.map((child) => child.label).join(",")}
        </div>
      );
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });

    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    expect(container.textContent).toContain("Existing saved query");
    expect(viewContext!.sheetTree.children.map((child) => child.label)).toContain(
      "common.load-more"
    );
    expect(viewContext!.folderTree.children.map((child) => child.label)).not.toContain(
      "common.load-more"
    );

    mocks.addSavedQueries([
      create(SavedQuerySchema, {
        name: "projects/proj1/savedQueries/2",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Created saved query",
      }),
    ]);
    await act(async () => {
      vi.runAllTimers();
    });

    expect(container.textContent).toContain("Created saved query");
  });

  test("does not restore saved query filter from localStorage", async () => {
    window.localStorage.setItem(
      storageKeySqlEditorSavedQueryFilter(
        "",
        "projects/proj1",
        "creator@example.com"
      ),
      JSON.stringify({
        keyword: "payroll",
        showMine: true,
        showShared: false,
        showDraft: true,
        onlyShowStarred: true,
      })
    );

    const { provideSheetContext, useSheetContext } = await import("./context");
    let sheetContext: ReturnType<typeof useSheetContext> | undefined;

    const Probe = () => {
      provideSheetContext();
      sheetContext = useSheetContext();
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });

    expect(sheetContext!.filter).toEqual({
      keyword: "",
      showMine: true,
      showShared: true,
      showDraft: true,
      onlyShowStarred: false,
    });
  });

  test("does not persist saved query filter to localStorage", async () => {
    const key = storageKeySqlEditorSavedQueryFilter(
      "",
      "projects/proj1",
      "creator@example.com"
    );

    const { provideSheetContext, useSheetContext } = await import("./context");
    let sheetContext: ReturnType<typeof useSheetContext> | undefined;

    const Probe = () => {
      provideSheetContext();
      sheetContext = useSheetContext();
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });

    act(() => {
      sheetContext!.setFilter({
        ...sheetContext!.filter,
        keyword: "payroll",
        onlyShowStarred: true,
      });
    });

    expect(window.localStorage.getItem(key)).toBeNull();
  });

  test("uses a collision-free key for load-more rows", async () => {
    mocks.getAppState().searchSavedQueryFolders.mockResolvedValueOnce([
      "__load-more",
    ]);

    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });

    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    const keys = viewContext!.sheetTree.children.map((child) => child.key);
    expect(keys).toContain("/my/__load-more");
    expect(keys).toContain("__savedQuery_load_more__:/my");
  });

  test("loads caller saved query folders before the root saved query page", async () => {
    mocks
      .getAppState()
      .searchSavedQueryFolders.mockResolvedValueOnce(["alpha", "alpha/beta"]);
    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => ({
      savedQueries: [],
      nextPageToken: "",
    }));

    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return (
        <div>
          {viewContext.sheetTree.children.map((child) => child.label).join(",")}
        </div>
      );
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    expect(mocks.getAppState().searchSavedQueryFolders).toHaveBeenCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com"'
    );
    expect(mocks.getAppState().fetchSavedQueryList).toHaveBeenCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == ""',
      expect.objectContaining({ pageToken: "" })
    );
    expect(container.textContent).toContain("alpha");
    expect(container.textContent).not.toContain("shared");
  });

  test("builds the shared folder tree from the server, not the row cache", async () => {
    // Regression: deriving shared folders from cached rows found nothing on a
    // cold cache, because the root page only fetches unfiled rows -- so every
    // foldered shared saved query was unreachable.
    mocks
      .getAppState()
      .searchSavedQueryFolders.mockResolvedValueOnce(["theirs", "theirs/deep"]);
    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => ({
      savedQueries: [],
      nextPageToken: "",
    }));

    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("shared");
      return (
        <div>
          {viewContext.sheetTree.children.map((child) => child.label).join(",")}
        </div>
      );
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    // Shared is "reached through a binding", not "somebody else created it":
    // the latter also matches saved queries an admin can read but nobody
    // shared with them.
    expect(mocks.getAppState().searchSavedQueryFolders).toHaveBeenCalledWith(
      "projects/proj1",
      "shared == true"
    );
    expect(container.textContent).toContain("theirs");
  });

  test("merges persisted empty folders with backend folders", async () => {
    window.localStorage.setItem(
      storageKeySqlEditorSavedQueryFolder(
        "",
        "projects/proj1",
        "shared",
        "creator@example.com"
      ),
      JSON.stringify(["/shared/alpha", "/shared/local-empty"])
    );
    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => ({
      savedQueries: [],
      nextPageToken: "",
    }));

    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("shared");
      return (
        <div>
          {viewContext.sheetTree.children.map((child) => child.label).join(",")}
        </div>
      );
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    expect(container.textContent).toContain("alpha");
    expect(container.textContent).toContain("local-empty");
  });

  test("persists locally created empty folders", async () => {
    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });

    act(() => {
      viewContext!.folderContext.addFolder("/my/local-empty");
    });

    expect(
      JSON.parse(
        window.localStorage.getItem(
          storageKeySqlEditorSavedQueryFolder(
            "",
            "projects/proj1",
            "my",
            "creator@example.com"
          )
        ) ?? "[]"
      )
    ).toEqual(["/my/local-empty"]);
  });

  test("uses server-side keyword and starred filters when fetching saved queries", async () => {
    const { provideSheetContext, useSheetContext, useSheetContextByView } =
      await import("./context");
    let sheetContext: ReturnType<typeof useSheetContext> | undefined;
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      sheetContext = useSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });
    mocks.getAppState().fetchSavedQueryList.mockClear();

    act(() => {
      sheetContext!.setFilter({
        ...sheetContext!.filter,
        keyword: "Payroll",
        onlyShowStarred: true,
      });
    });
    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(mocks.getAppState().fetchSavedQueryList).toHaveBeenLastCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == "" && title.contains("payroll") && starred == true',
      expect.objectContaining({ pageToken: "" })
    );

    mocks.getAppState().fetchSavedQueryList.mockClear();
    await act(async () => {
      await viewContext!.fetchNextPage();
    });

    expect(mocks.getAppState().fetchSavedQueryList).toHaveBeenLastCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == "" && title.contains("payroll") && starred == true',
      expect.objectContaining({ pageToken: "next-page" })
    );
  });

  test("keeps backend-known folders visible when starred filter is active", async () => {
    mocks
      .getAppState()
      .searchSavedQueryFolders.mockResolvedValue(["alpha"]);
    mocks.getAppState().fetchSavedQueryList.mockResolvedValue({
      savedQueries: [],
      nextPageToken: "",
    });

    const { provideSheetContext, useSheetContext, useSheetContextByView } =
      await import("./context");
    let sheetContext: ReturnType<typeof useSheetContext> | undefined;
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      sheetContext = useSheetContext();
      viewContext = useSheetContextByView("my");
      return (
        <div>
          {viewContext.sheetTree.children.map((child) => child.label).join(",")}
        </div>
      );
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });
    mocks.getAppState().fetchSavedQueryList.mockClear();

    act(() => {
      sheetContext!.setFilter({
        ...sheetContext!.filter,
        onlyShowStarred: true,
      });
    });
    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(mocks.getAppState().fetchSavedQueryList).toHaveBeenLastCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == "" && starred == true',
      expect.objectContaining({ pageToken: "" })
    );
    expect(container.textContent).toContain("alpha");
  });

  test("fetches saved query descendants for a folder without replacing the paged view", async () => {
    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return (
        <div>
          {viewContext.sheetTree.children.map((child) => child.label).join(",")}
        </div>
      );
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      const savedQuery = create(SavedQuerySchema, {
          name: "projects/proj1/savedQueries/3",
          project: "projects/proj1",
          creator: "users/creator@example.com",
          title: "Folder saved query",
          folder: "alpha",
        });
      mocks.addSavedQueries([savedQuery]);
      return {
        savedQueries: [savedQuery],
        nextPageToken: "",
      };
    });

    await act(async () => {
      await viewContext!.fetchSavedQueriesByFolder("/my/alpha");
    });

    expect(mocks.getAppState().fetchSavedQueryList).toHaveBeenLastCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == "alpha"',
      expect.objectContaining({ pageToken: "" })
    );
    expect(container.textContent).toContain("Existing saved query");
    expect(container.textContent).toContain("alpha");
  });

  test("fetches the next page for a folder without using the root page token", async () => {
    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      const savedQuery = create(SavedQuerySchema, {
        name: "projects/proj1/savedQueries/3",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Folder saved query",
        folder: "alpha",
      });
      mocks.addSavedQueries([savedQuery]);
      return {
        savedQueries: [savedQuery],
        nextPageToken: "folder-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchSavedQueriesByFolder("/my/alpha");
    });

    const alpha = viewContext!.sheetTree.children.find(
      (child) => child.key === "/my/alpha"
    );
    expect(alpha?.children.map((child) => child.label)).toContain(
      "common.load-more"
    );

    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => ({
      savedQueries: [],
      nextPageToken: "",
    }));
    await act(async () => {
      await viewContext!.fetchNextPage("/my/alpha");
    });

    expect(mocks.getAppState().fetchSavedQueryList).toHaveBeenLastCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == "alpha"',
      expect.objectContaining({ pageToken: "folder-next" })
    );
  });

  test("appends newly fetched saved query pages without frontend reordering", async () => {
    const firstSavedQuery = create(SavedQuerySchema, {
      name: "projects/proj1/savedQueries/first",
      project: "projects/proj1",
      creator: "users/creator@example.com",
      title: "Zeta saved query",
    });
    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      mocks.addSavedQueries([firstSavedQuery]);
      return {
        savedQueries: [firstSavedQuery],
        nextPageToken: "next-page",
      };
    });

    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    const secondSavedQuery = create(SavedQuerySchema, {
      name: "projects/proj1/savedQueries/second",
      project: "projects/proj1",
      creator: "users/creator@example.com",
      title: "Alpha saved query",
    });
    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      mocks.addSavedQueries([secondSavedQuery]);
      return {
        savedQueries: [secondSavedQuery],
        nextPageToken: "",
      };
    });
    await act(async () => {
      await viewContext!.fetchNextPage();
    });

    expect(
      viewContext!.sheetTree.children
        .filter((child) => child.savedQuery)
        .map((child) => child.label)
    ).toEqual(["Zeta saved query", "Alpha saved query"]);
  });

  test("keeps folder and descendant load-more state after renaming a folder", async () => {
    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      const savedQuery = create(SavedQuerySchema, {
        name: "projects/proj1/savedQueries/3",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Folder saved query",
        folder: "alpha",
      });
      mocks.addSavedQueries([savedQuery]);
      return {
        savedQueries: [savedQuery],
        nextPageToken: "folder-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchSavedQueriesByFolder("/my/alpha");
    });

    expect(viewContext!.hasMoreForFolder("/my/alpha")).toBe(true);

    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      const savedQuery = create(SavedQuerySchema, {
        name: "projects/proj1/savedQueries/4",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Child folder saved query",
        folder: "alpha/child",
      });
      mocks.addSavedQueries([savedQuery]);
      return {
        savedQueries: [savedQuery],
        nextPageToken: "child-folder-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchSavedQueriesByFolder("/my/alpha/child");
    });

    expect(viewContext!.hasMoreForFolder("/my/alpha/child")).toBe(true);

    act(() => {
      viewContext!.folderContext.moveFolder("/my/alpha", "/my/beta");
      viewContext!.rebuildTree();
    });

    expect(viewContext!.hasMoreForFolder("/my/beta")).toBe(true);
    expect(viewContext!.hasMoreForFolder("/my/beta/child")).toBe(true);
    expect(viewContext!.hasMoreForFolder("/my/alpha/child")).toBe(false);
  });

  test("invalidates folder pagination state when merging folders", async () => {
    const { provideSheetContext, useSheetContextByView } = await import(
      "./context"
    );
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    viewContext!.folderContext.addFolder("/my/beta");

    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      const savedQuery = create(SavedQuerySchema, {
        name: "projects/proj1/savedQueries/3",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Alpha saved query",
        folder: "alpha",
      });
      mocks.addSavedQueries([savedQuery]);
      return {
        savedQueries: [savedQuery],
        nextPageToken: "alpha-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchSavedQueriesByFolder("/my/alpha");
    });

    expect(viewContext!.hasMoreForFolder("/my/alpha")).toBe(true);

    act(() => {
      viewContext!.folderContext.moveFolder("/my/alpha", "/my/beta");
      viewContext!.rebuildTree();
    });

    expect(viewContext!.hasMoreForFolder("/my/beta")).toBe(false);
    expect(viewContext!.hasMoreForFolder("/my/alpha")).toBe(false);
  });

  test("invalidates affected page tokens after moving saved queries", async () => {
    const { provideSheetContext, useSheetContext, useSheetContextByView } =
      await import("./context");
    let sheetContext: ReturnType<typeof useSheetContext> | undefined;
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      sheetContext = useSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });

    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      const savedQuery = create(SavedQuerySchema, {
        name: "projects/proj1/savedQueries/3",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Folder saved query",
        folder: "alpha",
      });
      mocks.addSavedQueries([savedQuery]);
      return {
        savedQueries: [savedQuery],
        nextPageToken: "folder-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchSavedQueriesByFolder("/my/alpha");
    });

    expect(viewContext!.hasMore).toBe(true);
    expect(viewContext!.hasMoreForFolder("/my/alpha")).toBe(true);

    await act(async () => {
      await sheetContext!.batchUpdateSavedQueryFolders([
        { name: "projects/proj1/savedQueries/1", folders: ["alpha"] },
      ]);
    });

    expect(viewContext!.hasMore).toBe(false);
    expect(viewContext!.hasMoreForFolder("/my/alpha")).toBe(false);
  });

  test("batch updates saved query folders via UpdateSavedQuery per row", async () => {
    const { provideSheetContext, useSheetContext } = await import("./context");
    let sheetContext: ReturnType<typeof useSheetContext> | undefined;

    const Probe = () => {
      provideSheetContext();
      sheetContext = useSheetContext();
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });

    // Re-filing patches the cached row, so the rows must be in the cache and
    // owned by the current user.
    mocks.addSavedQueries([
      create(SavedQuerySchema, {
        name: "projects/proj1/savedQueries/1",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "one",
      }),
      create(SavedQuerySchema, {
        name: "projects/proj1/savedQueries/2",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "two",
      }),
    ]);

    await act(async () => {
      await sheetContext!.batchUpdateSavedQueryFolders([
        { name: "projects/proj1/savedQueries/1", folders: ["target"] },
        { name: "projects/proj1/savedQueries/2", folders: ["target"] },
      ]);
    });

    expect(mocks.getAppState().moveMySavedQueries).not.toHaveBeenCalled();
    expect(mocks.getAppState().patchSavedQuery).toHaveBeenCalledTimes(2);
    expect(mocks.getAppState().patchSavedQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "projects/proj1/savedQueries/1",
        folder: "target",
      }),
      ["folder"]
    );
    expect(mocks.getAppState().patchSavedQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "projects/proj1/savedQueries/2",
        folder: "target",
      }),
      ["folder"]
    );
    // Both cache views are re-filed per row so the tree rebuild does not snap
    // the rows back to the old folder.
    expect(
      mocks.getAppState().patchSavedQueryFolderInCache
    ).toHaveBeenCalledWith(["projects/proj1/savedQueries/1"], "target");
    expect(
      mocks.getAppState().patchSavedQueryFolderInCache
    ).toHaveBeenCalledWith(["projects/proj1/savedQueries/2"], "target");
  });

  test("moves cached rows that the starred filter hides", async () => {
    // Regression: the server moves every row in the folder (the batch filter
    // drops display filters), so patching only the visible rows left the
    // hidden ones under their old folder once the starred filter cleared.
    const starred = create(SavedQuerySchema, {
      name: "projects/proj1/savedQueries/starred",
      project: "projects/proj1",
      creator: "users/creator@example.com",
      title: "Starred",
      folder: "old",
      starred: true,
    });
    const unstarred = create(SavedQuerySchema, {
      name: "projects/proj1/savedQueries/unstarred",
      project: "projects/proj1",
      creator: "users/creator@example.com",
      title: "Unstarred",
      folder: "old",
      starred: false,
    });
    mocks.getAppState().fetchSavedQueryList.mockImplementationOnce(async () => {
      mocks.addSavedQueries([starred, unstarred]);
      return { savedQueries: [starred, unstarred], nextPageToken: "" };
    });

    const { provideSheetContext, useSheetContext, useSheetContextByView } =
      await import("./context");
    let sheetContext: ReturnType<typeof useSheetContext> | undefined;
    let viewContext: ReturnType<typeof useSheetContextByView> | undefined;

    const Probe = () => {
      provideSheetContext();
      sheetContext = useSheetContext();
      viewContext = useSheetContextByView("my");
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    await act(async () => {
      await viewContext!.fetchSheetList();
    });
    act(() => {
      sheetContext!.setFilter({
        ...sheetContext!.filter,
        onlyShowStarred: true,
      });
    });

    await act(async () => {
      await sheetContext!.batchUpdateSavedQueryFolderPaths("my", [
        { sourceFolder: ["old"], targetFolder: ["new"] },
      ]);
    });

    expect(
      mocks.getAppState().patchSavedQueryFolderInCache
    ).toHaveBeenCalledWith([starred.name, unstarred.name], "new");
  });

  test("batch updates saved query folder paths without display filters", async () => {
    const { provideSheetContext, useSheetContext } = await import("./context");
    let sheetContext: ReturnType<typeof useSheetContext> | undefined;

    const Probe = () => {
      provideSheetContext();
      sheetContext = useSheetContext();
      return null;
    };

    root = createRoot(container);
    await act(async () => {
      root.render(<Probe />);
    });
    act(() => {
      sheetContext!.setFilter({
        ...sheetContext!.filter,
        keyword: "Payroll",
        onlyShowStarred: true,
      });
    });

    await act(async () => {
      await sheetContext!.batchUpdateSavedQueryFolderPaths("my", [
        { sourceFolder: ["old"], targetFolder: ["new"] },
        { sourceFolder: ["old", "child"], targetFolder: ["new", "child"] },
      ]);
    });

    // The server moves descendants with the folder, so only the top-level move
    // is sent -- "old/child" would otherwise be rewritten twice.
    expect(mocks.getAppState().moveMySavedQueries).toHaveBeenCalledTimes(1);
    expect(mocks.getAppState().moveMySavedQueries).toHaveBeenCalledWith(
      "projects/proj1",
      { sourceFolder: "old", targetFolder: "new" }
    );
  });
});
