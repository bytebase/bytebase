import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTab } from "@/types/sqlEditor/tab";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorTabStore: vi.fn(),
  closeTab: vi.fn(),
  setCurrentTabId: vi.fn(),
  createWorksheet: vi.fn().mockResolvedValue(undefined),
  tabListEventsOn:
    vi.fn<(event: string, h: (p: unknown) => void) => () => void>(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: (
    selector: (s: { createWorksheet: typeof mocks.createWorksheet }) => unknown
  ) =>
    selector({
      createWorksheet: mocks.createWorksheet,
    }),
}));

vi.mock("@/views/sql-editor/TabList/events", () => ({
  tabListEvents: { on: mocks.tabListEventsOn, emit: vi.fn() },
}));

vi.mock("scroll-into-view-if-needed", () => ({
  default: vi.fn(),
}));

// Minimal @dnd-kit stubs — we're not testing drag behavior here.
vi.mock("@dnd-kit/core", () => ({
  DndContext: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  closestCenter: vi.fn(),
  PointerSensor: class {},
  useSensor: vi.fn(),
  useSensors: vi.fn(() => []),
}));

vi.mock("@dnd-kit/sortable", () => ({
  SortableContext: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  horizontalListSortingStrategy: vi.fn(),
  useSortable: vi.fn(() => ({
    attributes: {},
    listeners: {},
    setNodeRef: vi.fn(),
    transform: null,
    transition: null,
    isDragging: false,
  })),
}));

// Stub the TabItem composed subtree so we can assert on rendered tabs
// without pulling Label/Prefix/Suffix internals.
vi.mock("./TabItem/TabItem", () => ({
  TabItem: ({
    tab,
    onSelect,
    onClose,
    onContextMenu,
  }: {
    tab: SQLEditorTab;
    index: number;
    onSelect: (tab: SQLEditorTab, i: number) => void;
    onClose: (tab: SQLEditorTab, i: number) => void;
    onContextMenu: (tab: SQLEditorTab, i: number, e: React.MouseEvent) => void;
  }) => (
    <div
      data-testid="tab-item"
      data-tab-id={tab.id}
      onClick={() => onSelect(tab, 0)}
      onMouseDown={(e) => {
        if (e.button === 2)
          onContextMenu(tab, 0, e as unknown as React.MouseEvent);
      }}
    >
      <button
        data-testid={`close-${tab.id}`}
        onClick={(e) => {
          e.stopPropagation();
          onClose(tab, 0);
        }}
      />
    </div>
  ),
}));

vi.mock("./TabContextMenu", () => ({
  TabContextMenu: () => <div data-testid="tab-context-menu" />,
}));

vi.mock("@/react/components/HeaderProfileMenuMount", () => ({
  HeaderProfileMenuMount: () => <div data-testid="header-profile-menu" />,
}));

vi.mock("@/react/components/ui/alert-dialog", () => ({
  AlertDialog: ({
    open,
    children,
  }: {
    open?: boolean;
    children: React.ReactNode;
    onOpenChange?: (v: boolean) => void;
  }) => (
    <div data-testid="alert-dialog" data-open={String(open ?? false)}>
      {open ? children : null}
    </div>
  ),
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="alert-dialog-content">{children}</div>
  ),
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h2>{children}</h2>
  ),
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => (
    <p>{children}</p>
  ),
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    "aria-label": ariaLabel,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
    "aria-label"?: string;
  }) => (
    <button
      data-testid="button"
      aria-label={ariaLabel}
      disabled={disabled}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

let TabList: typeof import("./TabList").TabList;

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
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const makeTab = (
  id: string,
  overrides: Partial<SQLEditorTab> = {}
): SQLEditorTab =>
  ({
    id,
    title: `t-${id}`,
    mode: "WORKSHEET",
    status: "CLEAN",
    ...overrides,
  }) as unknown as SQLEditorTab;

const setup = (tabs: SQLEditorTab[], currentTabId = tabs[0]?.id ?? "") => {
  const tabStore = {
    openTabList: tabs,
    currentTabId,
    setCurrentTabId: mocks.setCurrentTabId,
    closeTab: mocks.closeTab,
  };
  mocks.useSQLEditorTabStore.mockReturnValue(tabStore);
  mocks.useVueState.mockImplementation((getter) => getter());
  mocks.tabListEventsOn.mockReturnValue(() => {});
  return { tabStore };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  ({ TabList } = await import("./TabList"));
});

describe("TabList", () => {
  test("renders one TabItem per tab in the store", () => {
    setup([makeTab("a"), makeTab("b"), makeTab("c")]);
    const { container, render, unmount } = renderIntoContainer(<TabList />);
    render();
    const items = container.querySelectorAll("[data-testid='tab-item']");
    expect(items).toHaveLength(3);
    expect(Array.from(items).map((e) => e.getAttribute("data-tab-id"))).toEqual(
      ["a", "b", "c"]
    );
    unmount();
  });

  test("clicking a tab fires setCurrentTabId", () => {
    setup([makeTab("a"), makeTab("b")], "a");
    const { container, render, unmount } = renderIntoContainer(<TabList />);
    render();
    const tabB = container.querySelector("[data-tab-id='b']") as HTMLElement;
    act(() => {
      tabB.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(mocks.setCurrentTabId).toHaveBeenCalledWith("b");
    unmount();
  });

  test("clicking close on a CLEAN tab calls store.closeTab without confirmation", () => {
    setup([makeTab("a"), makeTab("b")]);
    const { container, render, unmount } = renderIntoContainer(<TabList />);
    render();
    act(() => {
      container
        .querySelector("[data-testid='close-a']")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(mocks.closeTab).toHaveBeenCalledWith("a");
    unmount();
  });

  test("closing a DIRTY worksheet opens the confirm dialog", () => {
    setup([makeTab("a", { status: "DIRTY" })]);
    const { container, render, unmount } = renderIntoContainer(<TabList />);
    render();
    act(() => {
      container
        .querySelector("[data-testid='close-a']")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    const dialog = container.querySelector("[data-testid='alert-dialog']");
    expect(dialog?.getAttribute("data-open")).toBe("true");
    expect(mocks.closeTab).not.toHaveBeenCalled();
    unmount();
  });

  test("+ button calls worksheetStore.createWorksheet", () => {
    setup([makeTab("a")]);
    const { container, render, unmount } = renderIntoContainer(<TabList />);
    render();
    const addButton = container.querySelector(
      "[aria-label='common.add']"
    ) as HTMLButtonElement | null;
    expect(addButton).not.toBeNull();
    act(() => {
      addButton?.click();
    });
    expect(mocks.createWorksheet).toHaveBeenCalled();
    unmount();
  });

  test("subscribes to close-tab bus event on mount", () => {
    setup([makeTab("a")]);
    const { render, unmount } = renderIntoContainer(<TabList />);
    render();
    expect(mocks.tabListEventsOn).toHaveBeenCalledWith(
      "close-tab",
      expect.any(Function)
    );
    unmount();
  });
});
