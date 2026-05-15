import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { WorksheetFolderNode } from "@/views/sql-editor/Sheet";

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
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useWorkSheetStore: vi.fn(),
  useSQLEditorTabStore: vi.fn(),
  useSQLEditorVueState: vi.fn(),
  // The new zustand store mock — only `createWorksheet` is used by SheetTree.
  createWorksheet: vi.fn().mockResolvedValue({}),
  useSheetContext: vi.fn(),
  useSheetContextByView: vi.fn(),
  useDropdown: vi.fn(),
  openWorksheetByName: vi.fn(),
  pushNotification: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useWorkSheetStore: mocks.useWorkSheetStore,
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: mocks.useSQLEditorVueState,
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: (
    selector: (s: { createWorksheet: typeof mocks.createWorksheet }) => unknown
  ) => selector({ createWorksheet: mocks.createWorksheet }),
}));

vi.mock("@/views/sql-editor/Sheet", () => ({
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

vi.mock("@/react/components/ui/tree", () => ({
  Tree: ({
    data,
    renderNode,
  }: {
    data: MockTreeItem[];
    renderNode: (args: MockRenderArgs) => React.ReactNode;
  }) => {
    const renderAll = (items: MockTreeItem[]): React.ReactNode[] =>
      items.flatMap((item) => [
        renderNode({
          node: { id: item.id, data: item, isSelected: false, isOpen: false },
          style: {},
        }),
        ...(item.children ? renderAll(item.children) : []),
      ]);
    return <div data-testid="tree">{renderAll(data)}</div>;
  },
}));

vi.mock("@/react/components/ui/dropdown-menu", () => ({
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
vi.mock("@/react/components/ui/alert-dialog", () => ({
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
vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button data-testid="button" onClick={onClick}>
      {children}
    </button>
  ),
}));

// Mock Input
vi.mock("@/react/components/ui/input", () => ({
  Input: ({
    value,
    onChange,
    onBlur,
    onKeyDown,
    id,
  }: {
    value?: string;
    onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
    onBlur?: () => void;
    onKeyDown?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
    id?: string;
  }) => (
    <input
      data-testid="rename-input"
      id={id}
      value={value ?? ""}
      onChange={onChange}
      onBlur={onBlur}
      onKeyDown={onKeyDown}
      readOnly={!onChange}
    />
  ),
}));

// Mock HighlightLabelText
vi.mock("@/react/components/HighlightLabelText", () => ({
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
vi.mock("@/react/components/ui/popover", () => ({
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

  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

  const folderContext = {
    rootPath: { value: "/my" },
    isSubFolder: vi.fn(() => false),
    moveFolder: vi.fn(),
    removeFolder: vi.fn(),
    addFolder: vi.fn((path: string) => path),
    ensureFolderPath: vi.fn((path: string) => path),
  };

  const sheetContext = {
    filter: { value: { keyword: "", onlyShowStarred: false } },
    selectedKeys: selectedKeys,
    expandedKeys: expandedKeys,
    editingNode: editingNode,
    batchUpdateWorksheetFolders: vi.fn(),
  };

  const viewContext = {
    isInitialized: { value: true },
    isLoading: { value: false },
    sheetTree: { value: rootNode },
    fetchSheetList: vi.fn(),
    folderContext,
    getFoldersForWorksheet: vi.fn((path: string) => [path]),
    events: {
      on: vi.fn(() => () => {}),
      emit: vi.fn(),
    },
  };

  mocks.useSheetContext.mockReturnValue(sheetContext);
  mocks.useSheetContextByView.mockReturnValue(viewContext);
  mocks.useSQLEditorVueState.mockReturnValue({
    project: "projects/proj1",
  });
  mocks.useWorkSheetStore.mockReturnValue({
    getWorksheetByName: vi.fn((name: string) => ({
      name,
      title: "My Query",
      folders: [],
      database: "",
      starred: false,
      creator: "users/test@example.com",
    })),
    deleteWorksheetByName: vi.fn().mockResolvedValue(undefined),
    patchWorksheet: vi.fn().mockResolvedValue({}),
    upsertWorksheetOrganizer: vi.fn().mockResolvedValue(undefined),
  });
  mocks.useSQLEditorTabStore.mockReturnValue({
    getTabByWorksheet: vi.fn(() => null),
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
    defaultMocks.viewContext.sheetTree.value = rootNode;

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
    defaultMocks.viewContext.sheetTree.value = rootNode;
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

  test("4. Multi-select mode renders checkboxes; toggle fires onCheckedNodesChange", () => {
    const defaultMocks = setupDefaultMocks();
    const wsNode = makeWorksheetNode("/my/ws2");
    const rootNode = makeFolderNode("/my", [wsNode]);
    defaultMocks.viewContext.sheetTree.value = rootNode;

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

  test("5. Right-click → opens context menu with items", () => {
    const defaultMocks = setupDefaultMocks();
    const wsNode = makeWorksheetNode("/my/ws2");
    const rootNode = makeFolderNode("/my", [wsNode]);
    defaultMocks.viewContext.sheetTree.value = rootNode;

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
    defaultMocks.viewContext.sheetTree.value = rootNode;

    const deleteWorksheetByName = vi.fn().mockResolvedValue(undefined);
    mocks.useWorkSheetStore.mockReturnValue({
      getWorksheetByName: vi.fn((name: string) => ({
        name,
        title: "My Query",
        folders: [],
        database: "",
        starred: false,
        creator: "users/test@example.com",
      })),
      deleteWorksheetByName,
      patchWorksheet: vi.fn().mockResolvedValue({}),
      upsertWorksheetOrganizer: vi.fn().mockResolvedValue(undefined),
    });

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
});
