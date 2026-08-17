import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { SavedQueryFolderNode } from "@/modules/sql-editor/model/Sheet";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// Stub ResizeObserver — not provided by jsdom
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};
window.matchMedia = ((query: string) => ({
  matches: true,
  media: query,
  onchange: null,
  addEventListener: () => {},
  removeEventListener: () => {},
  addListener: () => {},
  removeListener: () => {},
  dispatchEvent: () => true,
})) as unknown as typeof window.matchMedia;

// ---- hoisted mocks ----------------------------------------------------------

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useSheetContext: vi.fn(),
  useSheetContextByView: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/modules/sql-editor/model/Sheet", () => ({
  useSheetContext: mocks.useSheetContext,
  useSheetContextByView: mocks.useSheetContextByView,
}));

// ---- primitive mocks --------------------------------------------------------

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

vi.mock("@/components/ui/search-input", () => ({
  SearchInput: ({
    value,
    onChange,
    placeholder,
  }: {
    value?: string;
    onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
    placeholder?: string;
  }) => (
    <input
      data-testid="search-input"
      value={value ?? ""}
      onChange={onChange}
      placeholder={placeholder}
    />
  ),
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
  DropdownMenuTrigger: ({
    children,
    className,
    render,
  }: {
    children?: React.ReactNode;
    className?: string;
    render?: React.ReactNode;
    "aria-label"?: string;
  }) => (
    <button data-testid="dropdown-menu-trigger" className={className}>
      {render ?? children}
    </button>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dropdown-menu-content">{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
    disabled,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
  }) => (
    <button
      data-testid="dropdown-menu-item"
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/components/ui/dialog", () => ({
  Dialog: ({
    children,
    open,
    onOpenChange,
  }: {
    children: React.ReactNode;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
  }) => (
    <div
      data-testid="dialog"
      data-open={String(open ?? false)}
      data-close-handler={onOpenChange ? "true" : "false"}
    >
      {open ? children : null}
    </div>
  ),
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dialog-content">{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2 data-testid="dialog-title">{children}</h2>
  ),
}));

vi.mock("./FilterMenuItem", () => ({
  FilterMenuItem: ({
    label,
    value,
    onValueChange,
  }: {
    label: string;
    value: boolean;
    onValueChange: (v: boolean) => void;
  }) => (
    <button
      data-testid="filter-menu-item"
      data-label={label}
      data-value={String(value)}
      onClick={() => onValueChange(!value)}
    >
      {label}
    </button>
  ),
}));

vi.mock("./FolderForm", () => ({
  FolderForm: ({
    folder,
    onFolderChange,
    includeRoot,
  }: {
    folder: string;
    onFolderChange: (f: string) => void;
    includeRoot?: boolean;
  }) => (
    <div
      data-testid="folder-form"
      data-folder={folder}
      data-include-root={String(!!includeRoot)}
    >
      {includeRoot && (
        <button
          data-testid="folder-form-set-root"
          onClick={() => onFolderChange("/my")}
        >
          set-root
        </button>
      )}
      <button
        data-testid="folder-form-set-target"
        onClick={() => onFolderChange("/some/folder")}
      >
        set-target
      </button>
    </div>
  ),
}));

// Mock SheetTree — expose buttons to trigger the callbacks that SheetTree
// would fire in the real component (enter multi-select, check a node).
vi.mock("./SheetTree", () => ({
  SheetTree: ({
    view,
    multiSelectMode,
    checkedNodes,
    onMultiSelectModeChange,
    onCheckedNodesChange,
    ref,
  }: {
    view: string;
    multiSelectMode?: boolean;
    checkedNodes?: SavedQueryFolderNode[];
    onMultiSelectModeChange: (m: boolean) => void;
    onCheckedNodesChange: (n: SavedQueryFolderNode[]) => void;
    ref?: React.Ref<{
      handleMultiDelete: (nodes: SavedQueryFolderNode[]) => Promise<void>;
    }>;
  }) => {
    // Expose the imperative handle on the ref so SavedQueryPane can call it
    if (ref && typeof ref === "object") {
      (ref as { current: unknown }).current = {
        handleMultiDelete: vi.fn().mockResolvedValue(undefined),
      };
    }
    return (
      <div
        data-testid="sheet-tree"
        data-view={view}
        data-multi-select-mode={String(multiSelectMode ?? false)}
        data-checked-count={String(checkedNodes?.length ?? 0)}
      >
        <button
          data-testid={`sheet-tree-${view}-enter-multi-select`}
          onClick={() => onMultiSelectModeChange(true)}
        >
          enter-multi-select
        </button>
        <button
          data-testid={`sheet-tree-${view}-check-ws`}
          onClick={() =>
            onCheckedNodesChange([
              {
                key: "/my/ws1",
                label: "ws1",
                editable: false,
                children: [],
                empty: true,
                savedQuery: {
                  name: "savedQueries/ws1",
                  title: "ws1",
                  folders: [],
                  type: "savedQuery",
                },
              } as SavedQueryFolderNode,
            ])
          }
        >
          check-ws
        </button>
      </div>
    );
  },
}));

// ---- helpers ----------------------------------------------------------------

type Filter = {
  keyword: string;
  showMine: boolean;
  showShared: boolean;
  showDraft: boolean;
  onlyShowStarred: boolean;
};

const setupDefaultMocks = (overrides: Partial<Filter> = {}) => {
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });

  const filter: Filter = {
    keyword: "",
    showMine: true,
    showShared: true,
    showDraft: true,
    onlyShowStarred: false,
    ...overrides,
  };

  // `filterRef.value` mirrors the migrated context's plain `filter` object;
  // the component reads `filter` (a plain value) and writes via `setFilter`,
  // which we wire to mutate the same holder so per-test assertions on
  // `filterRef.value.*` observe the writes.
  const filterRef = { value: filter };
  const filterChanged = false;

  const batchUpdateSavedQueryFolders = vi.fn().mockResolvedValue(undefined);
  const getFoldersForSavedQuery = vi.fn((path: string): string[] =>
    !path || path === "/my" ? [] : [path]
  );
  const sheetTree = {
    key: "/my",
    label: "Mine",
    editable: false,
    children: [
      {
        key: "/my/ws1",
        label: "ws1",
        editable: false,
        children: [],
        empty: true,
        savedQuery: {
          name: "savedQueries/ws1",
          title: "ws1",
          folders: [],
          type: "savedQuery",
        },
      } as SavedQueryFolderNode,
    ],
  } as SavedQueryFolderNode;

  mocks.useSheetContext.mockReturnValue({
    get filter() {
      return filterRef.value;
    },
    filterChanged,
    batchUpdateSavedQueryFolders,
    setFilter: vi.fn((next: Filter | ((prev: Filter) => Filter)) => {
      filterRef.value =
        typeof next === "function" ? next(filterRef.value) : next;
    }),
  });

  mocks.useSheetContextByView.mockReturnValue({
    getFoldersForSavedQuery,
    sheetTree,
  });

  return { filterRef, batchUpdateSavedQueryFolders, getFoldersForSavedQuery };
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

let SavedQueryPane: typeof import("./SavedQueryPane").SavedQueryPane;

beforeEach(async () => {
  vi.clearAllMocks();
  ({ SavedQueryPane } = await import("./SavedQueryPane"));
});

afterEach(() => {
  document.body.innerHTML = "";
  vi.resetModules();
});

// ---- tests ------------------------------------------------------------------

describe("SavedQueryPane", () => {
  test("1. Renders SheetTree for each enabled view", () => {
    setupDefaultMocks();
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryPane />
    );
    render();

    const trees = container.querySelectorAll("[data-testid='sheet-tree']");
    const views = Array.from(trees).map((t) => t.getAttribute("data-view"));
    expect(views).toEqual(["my", "shared", "draft"]);

    unmount();
  });

  test("2. Hides 'my' SheetTree when showMine is false", () => {
    setupDefaultMocks({ showMine: false });
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryPane />
    );
    render();

    const trees = container.querySelectorAll("[data-testid='sheet-tree']");
    const views = Array.from(trees).map((t) => t.getAttribute("data-view"));
    expect(views).toEqual(["shared", "draft"]);

    unmount();
  });

  test("3. Filter menu item toggle writes back to filter ref", () => {
    const { filterRef } = setupDefaultMocks();
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryPane />
    );
    render();

    // Find the "show-draft" filter menu item (by label)
    const items = Array.from(
      container.querySelectorAll("[data-testid='filter-menu-item']")
    ) as HTMLElement[];
    const showDraftItem = items.find(
      (el) => el.getAttribute("data-label") === "sheet.filter.show-draft"
    );
    expect(showDraftItem).not.toBeUndefined();

    act(() => {
      showDraftItem?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(filterRef.value.showDraft).toBe(false);

    unmount();
  });

  test("4. Multi-select toolbar appears when a SheetTree enters multi-select", () => {
    setupDefaultMocks();
    const { container, render, unmount } = renderIntoContainer(
      <SavedQueryPane />
    );
    render();

    // Before: toolbar hidden (no TrashIcon button — only sheet-tree buttons)
    const beforeToolbar = container.querySelector(
      "[data-testid='sheet-tree-my-enter-multi-select']"
    );
    expect(beforeToolbar).not.toBeNull();

    // Enter multi-select via the mocked "my" SheetTree
    act(() => {
      beforeToolbar?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    // Now: the "my" tree should report multi-select on
    const myTree = container.querySelector(
      "[data-testid='sheet-tree'][data-view='my']"
    );
    expect(myTree?.getAttribute("data-multi-select-mode")).toBe("true");

    // Toolbar buttons rendered — Cancel and Delete both fit inline now that
    // the move-to-folder action is gone (no peer offers bulk query move).
    const toolbarButtons = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).map((el) => el.textContent?.trim());
    expect(toolbarButtons).toEqual(
      expect.arrayContaining(["common.cancel", "common.delete"])
    );
    expect(toolbarButtons).not.toContain("sheet.move-saved-queries");
    expect(toolbarButtons).not.toContain("common.n-selected");

    unmount();
  });

});
