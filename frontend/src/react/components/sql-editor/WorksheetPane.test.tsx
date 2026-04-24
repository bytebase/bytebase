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
  useSheetContext: vi.fn(),
  useSheetContextByView: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/views/sql-editor/Sheet", () => ({
  useSheetContext: mocks.useSheetContext,
  useSheetContextByView: mocks.useSheetContextByView,
}));

// ---- primitive mocks --------------------------------------------------------

vi.mock("@/react/components/ui/button", () => ({
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

vi.mock("@/react/components/ui/search-input", () => ({
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
  DropdownMenuTrigger: ({
    children,
    className,
  }: {
    children: React.ReactNode;
    className?: string;
    "aria-label"?: string;
  }) => (
    <button data-testid="dropdown-menu-trigger" className={className}>
      {children}
    </button>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dropdown-menu-content">{children}</div>
  ),
}));

vi.mock("@/react/components/ui/dialog", () => ({
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
  }: {
    folder: string;
    onFolderChange: (f: string) => void;
  }) => (
    <div data-testid="folder-form" data-folder={folder}>
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
    checkedNodes?: WorksheetFolderNode[];
    onMultiSelectModeChange: (m: boolean) => void;
    onCheckedNodesChange: (n: WorksheetFolderNode[]) => void;
    ref?: React.Ref<{
      handleMultiDelete: (nodes: WorksheetFolderNode[]) => Promise<void>;
    }>;
  }) => {
    // Expose the imperative handle on the ref so WorksheetPane can call it
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
                worksheet: {
                  name: "worksheets/ws1",
                  title: "ws1",
                  folders: [],
                  type: "worksheet",
                },
              } as WorksheetFolderNode,
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

  const filterRef = { value: filter };
  const filterChanged = { value: false };

  const batchUpdateWorksheetFolders = vi.fn().mockResolvedValue(undefined);
  const getFoldersForWorksheet = vi.fn((path: string): string[] =>
    path ? [path] : []
  );

  mocks.useVueState.mockImplementation((getter) => getter());

  mocks.useSheetContext.mockReturnValue({
    filter: filterRef,
    filterChanged,
    batchUpdateWorksheetFolders,
  });

  mocks.useSheetContextByView.mockReturnValue({
    getFoldersForWorksheet,
  });

  return { filterRef, batchUpdateWorksheetFolders, getFoldersForWorksheet };
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

let WorksheetPane: typeof import("./WorksheetPane").WorksheetPane;

beforeEach(async () => {
  vi.clearAllMocks();
  ({ WorksheetPane } = await import("./WorksheetPane"));
});

afterEach(() => {
  document.body.innerHTML = "";
  vi.resetModules();
});

// ---- tests ------------------------------------------------------------------

describe("WorksheetPane", () => {
  test("1. Renders SheetTree for each enabled view", () => {
    setupDefaultMocks();
    const { container, render, unmount } = renderIntoContainer(
      <WorksheetPane />
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
      <WorksheetPane />
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
      <WorksheetPane />
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
      <WorksheetPane />
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

    // Toolbar buttons rendered — filter by button labels
    const toolbarButtons = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).map((el) => el.textContent?.trim());
    expect(toolbarButtons).toEqual(
      expect.arrayContaining([
        "common.delete",
        "sheet.move-worksheets",
        "common.cancel",
      ])
    );

    unmount();
  });

  test("5. Move modal opens and submit calls batchUpdateWorksheetFolders", async () => {
    const { batchUpdateWorksheetFolders, getFoldersForWorksheet } =
      setupDefaultMocks();
    const { container, render, unmount } = renderIntoContainer(
      <WorksheetPane />
    );
    render();

    // Step 1: enter multi-select and check a worksheet
    act(() => {
      container
        .querySelector("[data-testid='sheet-tree-my-enter-multi-select']")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    act(() => {
      container
        .querySelector("[data-testid='sheet-tree-my-check-ws']")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    // Step 2: click the "move-worksheets" toolbar button to open the modal
    const moveButton = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).find((el) => el.textContent?.trim() === "sheet.move-worksheets") as
      | HTMLButtonElement
      | undefined;
    expect(moveButton).not.toBeUndefined();

    act(() => {
      moveButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    // Dialog should be open now — FolderForm is rendered
    const folderForm = container.querySelector("[data-testid='folder-form']");
    expect(folderForm).not.toBeNull();

    // Step 3: change the folder target via the mocked FolderForm's setter
    act(() => {
      container
        .querySelector("[data-testid='folder-form-set-target']")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    const saveButton = Array.from(
      container.querySelectorAll("[data-testid='button']")
    ).find((el) => el.textContent?.trim() === "common.save");
    expect(saveButton).not.toBeUndefined();

    await act(async () => {
      saveButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(getFoldersForWorksheet).toHaveBeenCalledWith("/some/folder");
    expect(batchUpdateWorksheetFolders).toHaveBeenCalledWith([
      { name: "worksheets/ws1", folders: ["/some/folder"] },
    ]);

    unmount();
  });
});
