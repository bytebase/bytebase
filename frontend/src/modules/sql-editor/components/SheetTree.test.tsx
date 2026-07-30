import type { ReactElement } from "react";
import { act, forwardRef } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { WorksheetFolderNode } from "@/modules/sql-editor/model/Sheet";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// Stub ResizeObserver — not provided by jsdom
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

// ---- hoisted mocks ----------------------------------------------------------

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  appStore: {
    getWorksheetByName: vi.fn(),
    deleteWorksheetByName: vi.fn(),
    patchWorksheet: vi.fn(),
    upsertWorksheetOrganizer: vi.fn(),
  } as {
    getWorksheetByName: ReturnType<typeof vi.fn>;
    deleteWorksheetByName: ReturnType<typeof vi.fn>;
    patchWorksheet: ReturnType<typeof vi.fn>;
    upsertWorksheetOrganizer: ReturnType<typeof vi.fn>;
  },
  getSQLEditorTabsState: vi.fn(),
  project: "projects/proj1",
  // The new zustand store mock — only `createWorksheet` is used by SheetTree.
  createWorksheet: vi.fn().mockResolvedValue({}),
  useSheetContext: vi.fn(),
  useSheetContextByView: vi.fn(),
  useDropdown: vi.fn(),
  openWorksheetByName: vi.fn(),
  pushNotification: vi.fn(),
  treeProps: {} as {
    disableDrag?: unknown;
    disableDrop?: unknown;
    onMove?: unknown;
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => mocks.appStore },
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getSQLEditorTabsState: mocks.getSQLEditorTabsState,
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (selector: (s: { project: string }) => unknown) =>
    selector({ project: mocks.project }),
}));

vi.mock("@/modules/sql-editor/store", () => ({
  useSQLEditorStore: (
    selector: (s: { createWorksheet: typeof mocks.createWorksheet }) => unknown
  ) => selector({ createWorksheet: mocks.createWorksheet }),
}));

vi.mock("@/modules/sql-editor/model/Sheet", () => ({
  useSheetContext: mocks.useSheetContext,
  useSheetContextByView: mocks.useSheetContextByView,
  openWorksheetByName: mocks.openWorksheetByName,
  revealNodes: (
    node: WorksheetFolderNode,
    cb: (n: WorksheetFolderNode) => unknown
  ) => {
    const results: unknown[] = [];
    const walk = (n: WorksheetFolderNode) => {
      const r = cb(n);
      if (r !== undefined) results.push(r);
      for (const c of n.children) walk(c);
    };
    walk(node);
    return results;
  },
  revealWorksheets: (
    node: WorksheetFolderNode,
    cb: (n: WorksheetFolderNode) => unknown
  ) => {
    const results: unknown[] = [];
    const walk = (n: WorksheetFolderNode) => {
      if (n.worksheet) {
        const r = cb(n);
        if (r !== undefined) results.push(r);
      }
      for (const c of n.children) walk(c);
    };
    walk(node);
    return results;
  },
}));

vi.mock("./useDropdown", () => ({
  useDropdown: mocks.useDropdown,
}));

vi.mock("./filterNode", () => ({
  filterNode: () => () => true,
}));

// Mock Tree primitive — renders all nodes (recursively) via renderNode
type MockTreeItem = {
  id: string;
  data: WorksheetFolderNode;
  children?: MockTreeItem[];
};
type MockRenderArgs = {
  node: {
    id: string;
    data: MockTreeItem;
    isSelected: boolean;
    isOpen?: boolean;
  };
  style: React.CSSProperties;
};

vi.mock("@/components/ui/tree", () => ({
  Tree: ({
    data,
    renderNode,
    className,
    disableDrag,
    disableDrop,
    onMove,
  }: {
    data: MockTreeItem[];
    renderNode: (args: MockRenderArgs) => React.ReactNode;
    className?: string;
    disableDrag?: unknown;
    disableDrop?: unknown;
    onMove?: unknown;
  }) => {
    mocks.treeProps = { disableDrag, disableDrop, onMove };
    const renderAll = (items: MockTreeItem[]): React.ReactNode[] =>
      items.flatMap((item) => [
        renderNode({
          node: { id: item.id, data: item, isSelected: false, isOpen: false },
          style: {},
        }),
        ...(item.children ? renderAll(item.children) : []),
      ]);
    return (
      <div data-testid="tree" className={className}>
        {renderAll(data)}
      </div>
    );
  },
}));

vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({
    children,
    open,
  }: {
    children: React.ReactNode;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
  }) => (
    <div data-testid="dropdown-menu" data-open={String(open ?? false)}>
      {children}
    </div>
  ),
  DropdownMenuTrigger: ({ render }: { render?: React.ReactElement }) => (
    <div data-testid="dropdown-menu-trigger">{render}</div>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dropdown-menu-content">{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => (
    <div data-testid="dropdown-menu-item" onClick={onClick}>
      {children}
    </div>
  ),
  DropdownMenuSeparator: () => <hr data-testid="dropdown-menu-separator" />,
}));

// Mock AlertDialog — renders children when open
vi.mock("@/components/ui/alert-dialog", () => ({
  AlertDialog: ({
    children,
    open,
  }: {
    children: React.ReactNode;
    open?: boolean;
  }) => (
    <div data-testid="alert-dialog" data-open={String(open ?? false)}>
      {open ? children : null}
    </div>
  ),
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="alert-dialog-content">{children}</div>
  ),
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2 data-testid="alert-dialog-title">{children}</h2>
  ),
  AlertDialogDescription: ({ children }: { children?: React.ReactNode }) => (
    <p data-testid="alert-dialog-description">{children ?? null}</p>
  ),
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="alert-dialog-footer">{children}</div>
  ),
}));

// Mock Button
vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    className,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    className?: string;
  }) => (
    <button
      data-testid="button"
      className={className}
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

// Mock Input
vi.mock("@/components/ui/input", () => ({
  Input: forwardRef<
    HTMLInputElement,
    {
      value?: string;
      onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
      onBlur?: () => void;
      onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
      id?: string;
      className?: string;
    }
  >(({ value, onChange, onBlur, onKeyDown, id, className }, ref) => (
    <input
      ref={ref}
      data-testid="rename-input"
      id={id}
      className={className}
      value={value ?? ""}
      onChange={onChange}
      onBlur={onBlur}
      onKeyDown={onKeyDown}
      readOnly={!onChange}
    />
  )),
}));

// Mock HighlightLabelText
vi.mock("@/components/HighlightLabelText", () => ({
  HighlightLabelText: ({
    text,
    className,
  }: {
    text: string;
    className?: string;
  }) => (
    <span data-testid="highlight-label" className={className}>
      {text}
    </span>
  ),
}));

// Mock TreeNodePrefix
vi.mock("./TreeNodePrefix", () => ({
  TreeNodePrefix: () => <span data-testid="tree-node-prefix" />,
}));

// Mock SharePopoverBody — its router/store imports pull in native deps that
// jsdom can't load; the share popover behavior is covered by its own suite.
vi.mock("./SharePopoverBody", () => ({
  SharePopoverBody: () => <div data-testid="share-popover-body" />,
}));

// Mock Popover primitives to avoid Base UI Popover portal internals.
vi.mock("@/components/ui/popover", () => ({
  Popover: ({
    children,
    open,
  }: {
    children: React.ReactNode;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
  }) => (
    <div data-testid="popover" data-open={String(open ?? false)}>
      {children}
    </div>
  ),
  PopoverTrigger: () => <div data-testid="popover-trigger" />,
  PopoverContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="popover-content">{children}</div>
  ),
}));

// Mock TreeNodeSuffix
vi.mock("./TreeNodeSuffix", () => ({
  TreeNodeSuffix: ({
    onToggleStar,
    node,
  }: {
    node: WorksheetFolderNode;
    view: string;
    onToggleStar?: (args: { worksheet: string; starred: boolean }) => void;
    onSharePanelShow?: (e: React.MouseEvent, node: WorksheetFolderNode) => void;
    onContextMenuShow?: (
      e: React.MouseEvent,
      node: WorksheetFolderNode
    ) => void;
  }) => (
    <div
      data-testid="tree-node-suffix"
      onClick={() => {
        if (node.worksheet) {
          onToggleStar?.({ worksheet: node.worksheet.name, starred: true });
        }
      }}
    />
  ),
}));

// ---- helpers ----------------------------------------------------------------

const makeFolderNode = (
  key: string,
  children: WorksheetFolderNode[] = [],
  editable = false
): WorksheetFolderNode => ({
  key,
  label: key.split("/").slice(-1)[0],
  editable,
  children,
  empty: children.length === 0,
});

const makeWorksheetNode = (
  key: string,
  name = "worksheets/ws1"
): WorksheetFolderNode => ({
  key,
  label: key.split("/").slice(-1)[0],
  editable: false,
  children: [],
  empty: true,
  worksheet: {
    name,
    title: "My Query",
    folders: [],
    type: "worksheet",
  },
});

const makeLoadMoreNode = (key: string): WorksheetFolderNode => ({
  key,
  label: "common.load-more",
  editable: false,
  isLeaf: true,
  loadMore: true,
  children: [],
});

// The migrated `useSheetContext()` exposes `expandedKeys` / `selectedKeys`
// as plain values plus setters. We model the live state behind a `value`
// holder (so the existing per-test assertions like
// `expandedKeys.value.has(...)` keep working) and wire the component's
// `setExpandedKeys` setter to mutate that same holder.
const makeExpandedKeysRef = (keys: string[] = []) => ({
  value: new Set(keys),
});
const makeSelectedKeysRef = (keys: string[] = []) => ({
  value: new Set(keys),
});
const makeEditingNodeRef = () => ({ value: undefined as unknown });

const setupDefaultMocks = () => {
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });

  const rootNode = makeFolderNode("/my", [
    makeFolderNode("/my/folder1", [makeWorksheetNode("/my/folder1/ws1")]),
    makeWorksheetNode("/my/ws2"),
  ]);

  const expandedKeys = makeExpandedKeysRef(["/my"]);
  const selectedKeys = makeSelectedKeysRef();
  const editingNode = makeEditingNodeRef();

  const folderContext = {
    rootPath: "/my",
    listSubFolders: vi.fn((_parent: string): string[] => []),
    isSubFolder: vi.fn(() => false),
    moveFolder: vi.fn(),
    removeFolder: vi.fn(),
    addFolder: vi.fn((path: string) => path),
    ensureFolderPath: vi.fn((path: string) => path),
  };

  const sheetContext = {
    filter: { keyword: "", onlyShowStarred: false },
    get selectedKeys() {
      return Array.from(selectedKeys.value);
    },
    get expandedKeys() {
      return expandedKeys.value;
    },
    get editingNode() {
      return editingNode.value;
    },
    batchUpdateWorksheetFolders: vi.fn(),
    batchUpdateWorksheetFolderPaths: vi.fn(),
    setExpandedKeys: vi.fn(
      (next: Set<string> | ((prev: Set<string>) => Set<string>)) => {
        expandedKeys.value =
          typeof next === "function" ? next(expandedKeys.value) : next;
      }
    ),
    setSelectedKeys: vi.fn((next: string[]) => {
      selectedKeys.value = new Set(next);
    }),
    setEditingNode: vi.fn((next: unknown) => {
      editingNode.value = next;
    }),
  };

  const viewContext = {
    isInitialized: true,
    isLoading: false,
    isFetchingNextPage: false,
    hasMore: false,
    hasMoreForFolder: vi.fn((_folderKey: string) => false),
    get sheetTree() {
      return viewContext._sheetTree.value;
    },
    _sheetTree: { value: rootNode } as { value: WorksheetFolderNode },
    fetchSheetList: vi.fn(),
    fetchNextPage: vi.fn(),
    fetchWorksheetsByFolder: vi.fn(),
    rebuildTree: vi.fn(),
    folderContext,
    getFoldersForWorksheet: vi.fn((path: string) => [path]),
    events: {
      on: vi.fn(() => () => {}),
      emit: vi.fn(),
    },
  };

  mocks.useSheetContext.mockReturnValue(sheetContext);
  mocks.useSheetContextByView.mockReturnValue(viewContext);
  mocks.appStore.getWorksheetByName.mockImplementation((name: string) => ({
    name,
    title: "My Query",
    folders: [],
    database: "",
    starred: false,
    creator: "users/test@example.com",
  }));
  mocks.appStore.deleteWorksheetByName.mockResolvedValue(undefined);
  mocks.appStore.patchWorksheet.mockResolvedValue({});
  mocks.appStore.upsertWorksheetOrganizer.mockResolvedValue(undefined);
  mocks.getSQLEditorTabsState.mockReturnValue({
    tabsById: new Map(),
    closeTab: vi.fn(),
    updateTab: vi.fn(),
    setCurrentTabId: vi.fn(),
  });
  mocks.createWorksheet.mockResolvedValue({});
  mocks.useDropdown.mockReturnValue({
    currentNode: undefined,
    options: [],
    worksheetEntity: undefined,
    showSharePanel: false,
    handleContextMenu: vi.fn(),
    handleSharePanelShow: vi.fn(),
    handleClickOutside: vi.fn(),
  });
  mocks.openWorksheetByName.mockResolvedValue(undefined);

  return {
    rootNode,
    expandedKeys,
    selectedKeys,
    editingNode,
    folderContext,
    sheetContext,
    viewContext,
  };
};

test("keeps the initialized tree visible while refetching", () => {
  const { viewContext } = setupDefaultMocks();
  viewContext.isLoading = true;

  const { container, render, unmount } = renderIntoContainer(
    <SheetTree view="my" />
  );
  render();

  expect(container.textContent).toContain("folder1");
  expect(container.textContent).toContain("ws2");
  unmount();
});

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    update: (next: ReactElement) => {
      act(() => {
        root.render(next);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

let SheetTree: typeof import("./SheetTree").SheetTree;

beforeEach(async () => {
  vi.clearAllMocks();
  setupDefaultMocks();
  ({ SheetTree } = await import("./SheetTree"));
});

afterEach(() => {
  document.body.innerHTML = "";
  vi.resetModules();
});

// ---- tests ------------------------------------------------------------------

describe("SheetTree", () => {
  test("1. Renders tree from store data", () => {
    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    // Tree primitive is rendered
    const tree = document.body.querySelector("[data-testid='tree']");
    expect(tree).not.toBeNull();
    expect(tree?.className).toContain("[&_[role=treeitem]]:!min-w-0");

    // HighlightLabelText nodes are rendered (root node + children)
    const labels = container.querySelectorAll(
      "[data-testid='highlight-label']"
    );
    expect(labels.length).toBeGreaterThan(0);

    unmount();
  });

  test("2. Click worksheet → fires openWorksheetByName", () => {
    const defaultMocks = setupDefaultMocks();
    // Root node is a folder with a worksheet child
    const wsNode = makeWorksheetNode("/my/ws2");
    const rootNode = makeFolderNode("/my", [wsNode]);
    defaultMocks.viewContext._sheetTree.value = rootNode;

    // Make useDropdown return the worksheet node as current
    mocks.useDropdown.mockReturnValue({
      currentNode: undefined,
      options: [],
      worksheetEntity: undefined,
      showSharePanel: false,
      handleContextMenu: vi.fn(),
      handleSharePanelShow: vi.fn(),
      handleClickOutside: vi.fn(),
    });

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    // Find tree row for the worksheet node — it has data-item-key
    const row = container.querySelector(
      `[data-item-key="/my/ws2"]`
    ) as HTMLElement | null;
    expect(row).not.toBeNull();

    // Click on the label area
    const label = row?.querySelector("[data-testid='highlight-label']");
    act(() => {
      label?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(mocks.openWorksheetByName).toHaveBeenCalledWith(
      expect.objectContaining({
        worksheet: "worksheets/ws1",
        forceNewTab: false,
      })
    );

    unmount();
  });

  test("3. Click folder → toggles expand in Pinia store", () => {
    const defaultMocks = setupDefaultMocks();
    const folder = makeFolderNode("/my/folder1", []);
    const rootNode = makeFolderNode("/my", [folder]);
    defaultMocks.viewContext._sheetTree.value = rootNode;
    // expandedKeys starts without /my/folder1
    defaultMocks.expandedKeys.value = new Set(["/my"]);

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const row = container.querySelector(
      `[data-item-key="/my/folder1"]`
    ) as HTMLElement | null;
    expect(row).not.toBeNull();

    const prefix = row?.querySelector("[data-testid='tree-node-prefix']");
    act(() => {
      prefix?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    // expandedKeys should now contain /my/folder1
    expect(defaultMocks.expandedKeys.value.has("/my/folder1")).toBe(true);

    unmount();
  });

  test("3b. Opening a locally nonempty folder still fetches its server members", () => {
    const defaultMocks = setupDefaultMocks();
    const folder = makeFolderNode("/my/folder1", [
      makeWorksheetNode("/my/folder1/new-local-ws"),
    ]);
    const rootNode = makeFolderNode("/my", [folder]);
    defaultMocks.viewContext._sheetTree.value = rootNode;
    defaultMocks.expandedKeys.value = new Set(["/my"]);

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const row = container.querySelector(
      `[data-item-key="/my/folder1"]`
    ) as HTMLElement | null;
    const prefix = row?.querySelector("[data-testid='tree-node-prefix']");
    act(() => {
      prefix?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(defaultMocks.viewContext.fetchWorksheetsByFolder).toHaveBeenCalledWith(
      "/my/folder1"
    );

    unmount();
  });

  test("3c. Already expanded locally nonempty folders are fetched", () => {
    const defaultMocks = setupDefaultMocks();
    const folder = makeFolderNode("/my/folder1", [
      makeWorksheetNode("/my/folder1/new-local-ws"),
    ]);
    const rootNode = makeFolderNode("/my", [folder]);
    defaultMocks.viewContext._sheetTree.value = rootNode;
    defaultMocks.expandedKeys.value = new Set(["/my", "/my/folder1"]);

    const { render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    expect(defaultMocks.viewContext.fetchWorksheetsByFolder).toHaveBeenCalledWith(
      "/my/folder1"
    );

    unmount();
  });

  test("4. Multi-select mode renders checkboxes; toggle fires onCheckedNodesChange", () => {
    const defaultMocks = setupDefaultMocks();
    const wsNode = makeWorksheetNode("/my/ws2");
    const rootNode = makeFolderNode("/my", [wsNode]);
    defaultMocks.viewContext._sheetTree.value = rootNode;

    const onCheckedNodesChange = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        multiSelectMode={true}
        checkedNodes={[]}
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={onCheckedNodesChange}
      />
    );
    render();

    // Checkboxes should be present
    const checkboxes = container.querySelectorAll("input[type='checkbox']");
    expect(checkboxes.length).toBeGreaterThan(0);

    // The important assertion is that checkboxes are rendered when in
    // multi-select mode.
    expect(checkboxes.length).toBeGreaterThan(0);

    // Trigger the onChange handler via React's nativeEvent mechanism.
    // React uses the nativeInputValueSetter for value-based inputs but for
    // checkboxes, `click()` fires a click + change sequence.
    act(() => {
      const cb = checkboxes[0] as HTMLInputElement;
      // Use React's reconciler-aware click dispatch
      cb.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    // onCheckedNodesChange should have been called by the checkbox onChange
    expect(onCheckedNodesChange).toHaveBeenCalled();

    unmount();
  });

  test("4b. Multi-select folder selection skips load-more nodes", () => {
    const defaultMocks = setupDefaultMocks();
    const wsNode = makeWorksheetNode("/my/folder1/ws1");
    const loadMoreNode = makeLoadMoreNode("/my/folder1/__load-more");
    const folderNode = makeFolderNode(
      "/my/folder1",
      [wsNode, loadMoreNode],
      true
    );
    defaultMocks.viewContext._sheetTree.value = makeFolderNode("/my", [
      folderNode,
    ]);

    const onCheckedNodesChange = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        multiSelectMode={true}
        checkedNodes={[]}
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={onCheckedNodesChange}
      />
    );
    render();

    const row = container.querySelector(
      `[data-item-key="/my/folder1"]`
    ) as HTMLElement | null;
    const checkbox = row?.querySelector("input[type='checkbox']");

    act(() => {
      checkbox?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(onCheckedNodesChange).toHaveBeenCalledWith(
      expect.not.arrayContaining([loadMoreNode])
    );

    unmount();
  });

  test("5. Right-click → opens context menu with items", () => {
    const defaultMocks = setupDefaultMocks();
    const wsNode = makeWorksheetNode("/my/ws2");
    const rootNode = makeFolderNode("/my", [wsNode]);
    defaultMocks.viewContext._sheetTree.value = rootNode;

    const handleContextMenu = vi.fn();
    mocks.useDropdown.mockReturnValue({
      currentNode: wsNode,
      options: [
        { type: "item", key: "rename", label: "Rename" },
        { type: "item", key: "delete", label: "Delete" },
      ],
      worksheetEntity: undefined,
      showSharePanel: false,
      handleContextMenu,
      handleSharePanelShow: vi.fn(),
      handleClickOutside: vi.fn(),
    });

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    // One DropdownMenu lives at the SheetTree root with items derived from
    // useDropdown's options. Verify both items render.
    const menuItems = container.querySelectorAll(
      "[data-testid='dropdown-menu-item']"
    );
    expect(menuItems.length).toBe(2);
    expect(menuItems[0].textContent).toBe("Rename");
    expect(menuItems[1].textContent).toBe("Delete");

    // Right-click on a row fires handleContextMenu (via openMenuAtPoint).
    const row = container.querySelector(
      `[data-item-key="/my/ws2"]`
    ) as HTMLElement | null;
    expect(row).not.toBeNull();
    act(() => {
      row?.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true }));
    });

    expect(handleContextMenu).toHaveBeenCalled();

    unmount();
  });

  test("6. Delete confirm → fires worksheetV1Store.deleteWorksheetByName", async () => {
    const defaultMocks = setupDefaultMocks();
    const wsNode = makeWorksheetNode("/my/ws2", "worksheets/ws2");
    const rootNode = makeFolderNode("/my", [wsNode]);
    defaultMocks.viewContext._sheetTree.value = rootNode;

    const deleteWorksheetByName = vi.fn().mockResolvedValue(undefined);
    mocks.appStore.deleteWorksheetByName = deleteWorksheetByName;

    mocks.useDropdown.mockReturnValue({
      currentNode: wsNode,
      options: [{ type: "item", key: "delete", label: "Delete" }],
      worksheetEntity: { name: "worksheets/ws2" },
      showSharePanel: false,
      handleContextMenu: vi.fn(),
      handleSharePanelShow: vi.fn(),
      handleClickOutside: vi.fn(),
    });

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    // Find and click the "Delete" menu item → opens the delete-sheet dialog
    const deleteItem = Array.from(
      container.querySelectorAll("[data-testid='dropdown-menu-item']")
    ).find((el) => el.textContent === "Delete") as HTMLElement | undefined;
    expect(deleteItem).not.toBeUndefined();

    await act(async () => {
      deleteItem?.click();
    });

    // The AlertDialog for delete-sheet should now be open
    const dialogs = document.body.querySelectorAll(
      '[data-testid="alert-dialog"][data-open="true"]'
    );
    expect(dialogs.length).toBeGreaterThan(0);

    // Click the "confirm delete" button
    const buttons = document.body.querySelectorAll(
      "[data-testid='alert-dialog-content'] [data-testid='button']"
    );
    const deleteButton = Array.from(buttons).slice(-1)[0] as
      | HTMLElement
      | undefined;
    expect(deleteButton).not.toBeUndefined();

    await act(async () => {
      deleteButton?.click();
      await new Promise((r) => setTimeout(r, 10));
    });

    expect(deleteWorksheetByName).toHaveBeenCalledWith("worksheets/ws2");

    unmount();
  });

  test("7. Load more button fetches the next page", async () => {
    const defaultMocks = setupDefaultMocks();
    defaultMocks.viewContext._sheetTree.value = makeFolderNode("/my", [
      {
        key: "/my/__load-more",
        label: "common.load-more",
        editable: false,
        isLeaf: true,
        loadMore: true,
        children: [],
      },
    ]);

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const loadMoreRow = container.querySelector(
      `[data-item-key="/my/__load-more"]`
    ) as HTMLElement | null;
    expect(loadMoreRow).not.toBeNull();
    const loadMoreWrapper = loadMoreRow?.querySelector(
      "[data-testid='load-more-wrapper']"
    );
    const loadMoreButton = loadMoreRow?.querySelector("[data-testid='button']");
    expect(loadMoreButton).not.toBeNull();
    expect(loadMoreWrapper?.classList.contains("flex-1")).toBe(true);
    expect(loadMoreWrapper?.classList.contains("text-left")).toBe(true);
    expect(loadMoreButton?.classList.contains("tree-label")).toBe(true);
    expect(loadMoreButton?.classList.contains("flex-1")).toBe(false);
    expect(loadMoreButton?.classList.contains("text-control")).toBe(true);
    expect(loadMoreButton?.classList.contains("text-control-light")).toBe(false);
    expect(loadMoreButton?.classList.contains("text-xs")).toBe(true);
    expect(loadMoreButton?.classList.contains("cursor-pointer")).toBe(true);

    await act(async () => {
      loadMoreWrapper?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(defaultMocks.viewContext.fetchNextPage).not.toHaveBeenCalled();

    await act(async () => {
      loadMoreButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(defaultMocks.viewContext.fetchNextPage).toHaveBeenCalledWith(
      undefined
    );

    unmount();
  });

  test("7b. Folder load more button fetches that folder's next page", async () => {
    const defaultMocks = setupDefaultMocks();
    const folder = makeFolderNode("/my/alpha", [
      makeWorksheetNode("/my/alpha/ws1"),
      {
        key: "/my/alpha/__load-more",
        label: "common.load-more",
        editable: false,
        isLeaf: true,
        loadMore: true,
        loadMoreFolderKey: "/my/alpha",
        children: [],
      },
    ]);
    defaultMocks.viewContext._sheetTree.value = makeFolderNode("/my", [folder]);
    defaultMocks.expandedKeys.value = new Set(["/my", "/my/alpha"]);

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const loadMoreRow = container.querySelector(
      `[data-item-key="/my/alpha/__load-more"]`
    ) as HTMLElement | null;
    expect(loadMoreRow).not.toBeNull();
    const loadMoreWrapper = loadMoreRow?.querySelector(
      "[data-testid='load-more-wrapper']"
    );
    const loadMoreButton = loadMoreRow?.querySelector("[data-testid='button']");
    expect(loadMoreButton).not.toBeNull();
    expect(loadMoreWrapper?.classList.contains("flex-1")).toBe(true);
    expect(loadMoreWrapper?.classList.contains("text-left")).toBe(true);
    expect(loadMoreButton?.classList.contains("tree-label")).toBe(true);
    expect(loadMoreButton?.classList.contains("flex-1")).toBe(false);
    expect(loadMoreButton?.classList.contains("text-control")).toBe(true);
    expect(loadMoreButton?.classList.contains("text-control-light")).toBe(false);
    expect(loadMoreButton?.classList.contains("text-xs")).toBe(true);
    expect(loadMoreButton?.classList.contains("cursor-pointer")).toBe(true);

    await act(async () => {
      loadMoreWrapper?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(defaultMocks.viewContext.fetchNextPage).not.toHaveBeenCalled();

    await act(async () => {
      loadMoreButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(defaultMocks.viewContext.fetchNextPage).toHaveBeenCalledWith(
      "/my/alpha"
    );

    unmount();
  });

  test("7c. Load more rows are inert for tree actions", () => {
    const defaultMocks = setupDefaultMocks();
    const loadMoreNode = makeLoadMoreNode("/my/__load-more");
    const folderNode = makeFolderNode("/my/folder", [], true);
    defaultMocks.viewContext._sheetTree.value = makeFolderNode("/my", [
      folderNode,
      loadMoreNode,
    ]);

    const handleContextMenu = vi.fn();
    mocks.useDropdown.mockReturnValue({
      currentNode: undefined,
      options: [],
      worksheetEntity: undefined,
      showSharePanel: false,
      handleContextMenu,
      handleSharePanelShow: vi.fn(),
      handleClickOutside: vi.fn(),
    });

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const loadMoreRow = container.querySelector(
      `[data-item-key="/my/__load-more"]`
    ) as HTMLElement | null;
    expect(loadMoreRow).not.toBeNull();

    act(() => {
      loadMoreRow?.dispatchEvent(
        new MouseEvent("contextmenu", { bubbles: true })
      );
    });
    expect(handleContextMenu).not.toHaveBeenCalled();

    const disableDrag = mocks.treeProps.disableDrag as
      | ((data: { data: WorksheetFolderNode }) => boolean)
      | undefined;
    expect(disableDrag?.({ data: loadMoreNode })).toBe(true);
    expect(disableDrag?.({ data: folderNode })).toBe(false);

    const disableDrop = mocks.treeProps.disableDrop as
      | ((args: {
          parentNode?: { data: { data: WorksheetFolderNode } } | null;
          dragNodes: { data: { data: WorksheetFolderNode } }[];
        }) => boolean)
      | undefined;
    expect(
      disableDrop?.({
        parentNode: { data: { data: loadMoreNode } },
        dragNodes: [{ data: { data: folderNode } }],
      })
    ).toBe(true);
    expect(
      disableDrop?.({
        parentNode: { data: { data: folderNode } },
        dragNodes: [{ data: { data: loadMoreNode } }],
      })
    ).toBe(true);

    unmount();
  });

  test("7d. Add folder ignores load-more nodes when generating a name", async () => {
    const defaultMocks = setupDefaultMocks();
    const existingFolder = makeFolderNode("/my/new folder", [], true);
    const loadMoreNode = makeLoadMoreNode("/my/__load-more");
    const rootNode = makeFolderNode(
      "/my",
      [existingFolder, loadMoreNode],
      true
    );
    defaultMocks.viewContext._sheetTree.value = rootNode;

    mocks.useDropdown.mockReturnValue({
      currentNode: rootNode,
      options: [{ type: "item", key: "add-folder", label: "Add folder" }],
      worksheetEntity: undefined,
      showSharePanel: false,
      handleContextMenu: vi.fn(),
      handleSharePanelShow: vi.fn(),
      handleClickOutside: vi.fn(),
    });

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const addFolderItem = container.querySelector(
      "[data-testid='dropdown-menu-item']"
    ) as HTMLElement | null;

    await act(async () => {
      addFolderItem?.click();
    });

    expect(defaultMocks.viewContext.folderContext.addFolder).toHaveBeenCalledWith(
      "/my/new folder2"
    );
    expect(defaultMocks.sheetContext.setEditingNode).toHaveBeenCalledWith(
      expect.objectContaining({
        rawLabel: "new folder2",
      })
    );

    unmount();
  });

  test("8. Rename input is not clipped by display-mode row overflow", () => {
    const defaultMocks = setupDefaultMocks();
    const wsNode = makeWorksheetNode("/my/ws-with-long-title");
    const rootNode = makeFolderNode("/my", [wsNode]);
    defaultMocks.viewContext._sheetTree.value = rootNode;
    defaultMocks.editingNode.value = {
      node: wsNode,
      rawLabel: "bytebase-3.12.2 May 9, 2026, 12:12:02 PM GMT+8",
    };

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const input = container.querySelector(
      "[data-testid='rename-input']"
    ) as HTMLInputElement | null;
    expect(input).not.toBeNull();
    expect(input?.className).toContain("w-full");
    expect(input?.className).toContain("h-6");

    const label = input?.closest(".tree-label");
    expect(label?.className).toContain("overflow-visible");

    const row = input?.closest("[data-item-key='/my/ws-with-long-title']");
    expect(row?.className).toContain("overflow-visible");
    expect(row?.className).toContain("py-0");

    const tree = container.querySelector("[data-testid='tree']");
    expect(tree?.className).toContain(
      "[&_[role=treeitem]]:overflow-visible"
    );

    unmount();
  });

  test("8b. Rename input preserves cursor position while typing", async () => {
    const defaultMocks = setupDefaultMocks();
    const folderNode = makeFolderNode("/my/folder1", [], true);
    defaultMocks.viewContext._sheetTree.value = makeFolderNode("/my", [
      folderNode,
    ]);
    defaultMocks.editingNode.value = {
      node: folderNode,
      rawLabel: "abcdef",
    };

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const input = container.querySelector(
      "[data-testid='rename-input']"
    ) as HTMLInputElement | null;
    expect(input).not.toBeNull();
    input?.setSelectionRange(3, 3);

    await act(async () => {
      if (!input) return;
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set;
      valueSetter?.call(input, "abcXdef");
      input.setSelectionRange(4, 4);
      input.dispatchEvent(new Event("input", { bubbles: true }));
      input.setSelectionRange(7, 7);
    });
    render();
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(input?.selectionStart).toBe(4);
    expect(input?.selectionEnd).toBe(4);

    unmount();
  });

  test("9. Folder rename updates known folder paths with exact batch filters", async () => {
    const defaultMocks = setupDefaultMocks();
    const childFolder = makeFolderNode("/my/old/child");
    const oldFolder = makeFolderNode("/my/old", [childFolder]);
    const rootNode = makeFolderNode("/my", [oldFolder]);
    defaultMocks.viewContext._sheetTree.value = rootNode;
    defaultMocks.editingNode.value = {
      node: oldFolder,
      rawLabel: "new",
    };
    defaultMocks.viewContext.getFoldersForWorksheet.mockImplementation(
      (path: string) =>
        path
          .replace("/my", "")
          .split("/")
          .map((part) => part.trim())
          .filter(Boolean)
    );
    defaultMocks.folderContext.listSubFolders.mockImplementation(
      (parent: string) => (parent === "/my/old" ? ["/my/old/child"] : [])
    );

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const input = container.querySelector(
      "[data-testid='rename-input']"
    ) as HTMLInputElement | null;
    expect(input).not.toBeNull();

    await act(async () => {
      input?.dispatchEvent(new FocusEvent("focusout", { bubbles: true }));
      await new Promise((resolve) => setTimeout(resolve, 10));
    });

    expect(
      defaultMocks.sheetContext.batchUpdateWorksheetFolderPaths
    ).toHaveBeenCalledWith("my", [
      { sourceFolder: ["old"], targetFolder: ["new"] },
      { sourceFolder: ["old", "child"], targetFolder: ["new", "child"] },
    ]);
    expect(defaultMocks.viewContext.rebuildTree).toHaveBeenCalled();

    unmount();
  });

  test("10. Opening an empty folder fetches that folder's worksheets", async () => {
    const defaultMocks = setupDefaultMocks();
    const folder = makeFolderNode("/my/empty", []);
    const rootNode = makeFolderNode("/my", [folder]);
    defaultMocks.viewContext._sheetTree.value = rootNode;
    defaultMocks.expandedKeys.value = new Set(["/my"]);

    const { container, render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );
    render();

    const row = container.querySelector(
      `[data-item-key="/my/empty"]`
    ) as HTMLElement | null;
    expect(row).not.toBeNull();
    const prefix = row?.querySelector("[data-testid='tree-node-prefix']");

    await act(async () => {
      prefix?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(
      defaultMocks.viewContext.fetchWorksheetsByFolder
    ).toHaveBeenCalledWith("/my/empty");

    unmount();
  });

  test("11. Already-expanded empty folder fetches after render", async () => {
    const defaultMocks = setupDefaultMocks();
    const folder = makeFolderNode("/my/empty", []);
    const rootNode = makeFolderNode("/my", [folder]);
    defaultMocks.viewContext._sheetTree.value = rootNode;
    defaultMocks.expandedKeys.value = new Set(["/my", "/my/empty"]);

    const { render, unmount } = renderIntoContainer(
      <SheetTree
        view="my"
        onMultiSelectModeChange={vi.fn()}
        onCheckedNodesChange={vi.fn()}
      />
    );

    await act(async () => {
      render();
    });

    expect(
      defaultMocks.viewContext.fetchWorksheetsByFolder
    ).toHaveBeenCalledWith("/my/empty");

    unmount();
  });
});
