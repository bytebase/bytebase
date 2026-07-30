import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  WorksheetSchema,
  Worksheet_Visibility,
  type Worksheet,
} from "@/types/proto-es/v1/worksheet_service_pb";
import { storageKeySqlEditorWorksheetFolder } from "@/utils";

type AppState = {
  currentUser: { email: string; workspace: string };
  worksheetsByKey: Record<string, Worksheet>;
  isSaaSMode: () => boolean;
  getWorksheetByName: (name: string) => Worksheet | undefined;
  listWorksheetFolders: ReturnType<typeof vi.fn>;
  fetchWorksheetList: ReturnType<typeof vi.fn>;
  upsertWorksheetOrganizer: ReturnType<typeof vi.fn>;
  batchUpdateWorksheetOrganizers: ReturnType<typeof vi.fn>;
};

const mocks = vi.hoisted(() => {
  let appState: AppState;
  const appSubscribers = new Set<(state: AppState, prev: AppState) => void>();
  const keyForWorksheet = (name: string) => `${name.split("/").pop()}:FULL`;

  const setWorksheets = (worksheets: Worksheet[]) => {
    const prev = {
      ...appState,
      worksheetsByKey: appState.worksheetsByKey,
    };
    appState = {
      ...appState,
      worksheetsByKey: {
        ...appState.worksheetsByKey,
        ...Object.fromEntries(
          worksheets.map((worksheet) => [
            keyForWorksheet(worksheet.name),
            worksheet,
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
        worksheetsByKey: {},
        isSaaSMode: () => false,
        getWorksheetByName: (name: string) =>
          appState.worksheetsByKey[keyForWorksheet(name)],
        listWorksheetFolders: vi.fn(async () => []),
        fetchWorksheetList: vi.fn(async (_project, _filter, _params) => {
          const worksheet = create(WorksheetSchema, {
            name: "projects/proj1/worksheets/1",
            project: "projects/proj1",
            creator: "users/creator@example.com",
            title: "Existing worksheet",
            visibility: Worksheet_Visibility.PRIVATE,
          });
          setWorksheets([worksheet]);
          return { worksheets: [worksheet], nextPageToken: "next-page" };
        }),
        upsertWorksheetOrganizer: vi.fn(),
        batchUpdateWorksheetOrganizers: vi.fn(),
      };
    },
    addWorksheets: setWorksheets,
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

  test("adds newly created worksheets to the initialized paged my view", async () => {
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

    expect(container.textContent).toContain("Existing worksheet");
    expect(viewContext!.sheetTree.children.map((child) => child.label)).toContain(
      "common.load-more"
    );

    mocks.addWorksheets([
      create(WorksheetSchema, {
        name: "projects/proj1/worksheets/2",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Created worksheet",
        visibility: Worksheet_Visibility.PRIVATE,
      }),
    ]);
    await act(async () => {
      vi.runAllTimers();
    });

    expect(container.textContent).toContain("Created worksheet");
  });

  test("loads caller worksheet folders before the root worksheet page", async () => {
    mocks
      .getAppState()
      .listWorksheetFolders.mockResolvedValueOnce([
        { folders: ["alpha"], category: "my" },
        { folders: ["alpha", "beta"], category: "my" },
        { folders: ["shared"], category: "shared" },
      ]);
    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => ({
      worksheets: [],
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

    expect(mocks.getAppState().listWorksheetFolders).toHaveBeenCalledWith(
      "projects/proj1"
    );
    expect(mocks.getAppState().fetchWorksheetList).toHaveBeenCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == ""',
      expect.objectContaining({ pageToken: "" })
    );
    expect(container.textContent).toContain("alpha");
    expect(container.textContent).not.toContain("shared");
  });

  test("merges persisted empty folders with backend folders", async () => {
    window.localStorage.setItem(
      storageKeySqlEditorWorksheetFolder(
        "",
        "projects/proj1",
        "shared",
        "creator@example.com"
      ),
      JSON.stringify(["/shared/alpha", "/shared/local-empty"])
    );
    mocks.getAppState().listWorksheetFolders.mockResolvedValueOnce([
      { folders: ["alpha"], category: "my" },
    ]);
    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => ({
      worksheets: [],
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
          storageKeySqlEditorWorksheetFolder(
            "",
            "projects/proj1",
            "my",
            "creator@example.com"
          )
        ) ?? "[]"
      )
    ).toEqual(["/my/local-empty"]);
  });

  test("uses server-side keyword and starred filters when fetching worksheets", async () => {
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
    mocks.getAppState().fetchWorksheetList.mockClear();

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

    expect(mocks.getAppState().fetchWorksheetList).toHaveBeenLastCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && title.contains("payroll") && starred == true',
      expect.objectContaining({ pageToken: "" })
    );
  });

  test("fetches worksheet descendants for a folder without replacing the paged view", async () => {
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

    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => {
      const worksheet = create(WorksheetSchema, {
          name: "projects/proj1/worksheets/3",
          project: "projects/proj1",
          creator: "users/creator@example.com",
          title: "Folder worksheet",
          folders: ["alpha"],
          visibility: Worksheet_Visibility.PRIVATE,
        });
      mocks.addWorksheets([worksheet]);
      return {
        worksheets: [worksheet],
        nextPageToken: "",
      };
    });

    await act(async () => {
      await viewContext!.fetchWorksheetsByFolder("/my/alpha");
    });

    expect(mocks.getAppState().fetchWorksheetList).toHaveBeenLastCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == "alpha"',
      expect.objectContaining({ pageToken: "" })
    );
    expect(container.textContent).toContain("Existing worksheet");
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

    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => {
      const worksheet = create(WorksheetSchema, {
        name: "projects/proj1/worksheets/3",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Folder worksheet",
        folders: ["alpha"],
        visibility: Worksheet_Visibility.PRIVATE,
      });
      mocks.addWorksheets([worksheet]);
      return {
        worksheets: [worksheet],
        nextPageToken: "folder-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchWorksheetsByFolder("/my/alpha");
    });

    const alpha = viewContext!.sheetTree.children.find(
      (child) => child.key === "/my/alpha"
    );
    expect(alpha?.children.map((child) => child.label)).toContain(
      "common.load-more"
    );

    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => ({
      worksheets: [],
      nextPageToken: "",
    }));
    await act(async () => {
      await viewContext!.fetchNextPage("/my/alpha");
    });

    expect(mocks.getAppState().fetchWorksheetList).toHaveBeenLastCalledWith(
      "projects/proj1",
      'creator == "users/creator@example.com" && folder == "alpha"',
      expect.objectContaining({ pageToken: "folder-next" })
    );
  });

  test("appends newly fetched worksheet pages without frontend reordering", async () => {
    const firstWorksheet = create(WorksheetSchema, {
      name: "projects/proj1/worksheets/first",
      project: "projects/proj1",
      creator: "users/creator@example.com",
      title: "Zeta worksheet",
      visibility: Worksheet_Visibility.PRIVATE,
    });
    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => {
      mocks.addWorksheets([firstWorksheet]);
      return {
        worksheets: [firstWorksheet],
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

    const secondWorksheet = create(WorksheetSchema, {
      name: "projects/proj1/worksheets/second",
      project: "projects/proj1",
      creator: "users/creator@example.com",
      title: "Alpha worksheet",
      visibility: Worksheet_Visibility.PRIVATE,
    });
    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => {
      mocks.addWorksheets([secondWorksheet]);
      return {
        worksheets: [secondWorksheet],
        nextPageToken: "",
      };
    });
    await act(async () => {
      await viewContext!.fetchNextPage();
    });

    expect(
      viewContext!.sheetTree.children
        .filter((child) => child.worksheet)
        .map((child) => child.label)
    ).toEqual(["Zeta worksheet", "Alpha worksheet"]);
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

    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => {
      const worksheet = create(WorksheetSchema, {
        name: "projects/proj1/worksheets/3",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Folder worksheet",
        folders: ["alpha"],
        visibility: Worksheet_Visibility.PRIVATE,
      });
      mocks.addWorksheets([worksheet]);
      return {
        worksheets: [worksheet],
        nextPageToken: "folder-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchWorksheetsByFolder("/my/alpha");
    });

    expect(viewContext!.hasMoreForFolder("/my/alpha")).toBe(true);

    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => {
      const worksheet = create(WorksheetSchema, {
        name: "projects/proj1/worksheets/4",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Child folder worksheet",
        folders: ["alpha", "child"],
        visibility: Worksheet_Visibility.PRIVATE,
      });
      mocks.addWorksheets([worksheet]);
      return {
        worksheets: [worksheet],
        nextPageToken: "child-folder-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchWorksheetsByFolder("/my/alpha/child");
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

    mocks.getAppState().fetchWorksheetList.mockImplementationOnce(async () => {
      const worksheet = create(WorksheetSchema, {
        name: "projects/proj1/worksheets/3",
        project: "projects/proj1",
        creator: "users/creator@example.com",
        title: "Alpha worksheet",
        folders: ["alpha"],
        visibility: Worksheet_Visibility.PRIVATE,
      });
      mocks.addWorksheets([worksheet]);
      return {
        worksheets: [worksheet],
        nextPageToken: "alpha-next",
      };
    });
    await act(async () => {
      await viewContext!.fetchWorksheetsByFolder("/my/alpha");
    });

    expect(viewContext!.hasMoreForFolder("/my/alpha")).toBe(true);

    act(() => {
      viewContext!.folderContext.moveFolder("/my/alpha", "/my/beta");
      viewContext!.rebuildTree();
    });

    expect(viewContext!.hasMoreForFolder("/my/beta")).toBe(false);
    expect(viewContext!.hasMoreForFolder("/my/alpha")).toBe(false);
  });

  test("batch updates worksheet folders with one name filter per target folder", async () => {
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

    await act(async () => {
      await sheetContext!.batchUpdateWorksheetFolders([
        { name: "projects/proj1/worksheets/1", folders: ["target"] },
        { name: "projects/proj1/worksheets/2", folders: ["target"] },
      ]);
    });

    expect(mocks.getAppState().upsertWorksheetOrganizer).not.toHaveBeenCalled();
    expect(
      mocks.getAppState().batchUpdateWorksheetOrganizers
    ).toHaveBeenCalledWith([
      {
        parent: "projects/proj1",
        filter:
          'name in ["projects/proj1/worksheets/1","projects/proj1/worksheets/2"]',
        organizer: { folders: ["target"] },
        updateMask: ["folders"],
      },
    ]);
  });

  test("batch updates worksheet folder paths without display filters", async () => {
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
      await sheetContext!.batchUpdateWorksheetFolderPaths("my", [
        { sourceFolder: ["old"], targetFolder: ["new"] },
        { sourceFolder: ["old", "child"], targetFolder: ["new", "child"] },
      ]);
    });

    expect(
      mocks.getAppState().batchUpdateWorksheetOrganizers
    ).toHaveBeenCalledWith([
      {
        parent: "projects/proj1",
        filter:
          'creator == "users/creator@example.com" && folder == "old/child"',
        organizer: { folders: ["new", "child"] },
        updateMask: ["folders"],
      },
      {
        parent: "projects/proj1",
        filter: 'creator == "users/creator@example.com" && folder == "old"',
        organizer: { folders: ["new"] },
        updateMask: ["folders"],
      },
    ]);
  });
});
