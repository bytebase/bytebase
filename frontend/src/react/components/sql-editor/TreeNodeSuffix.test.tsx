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

// ---- mocks ------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useWorkSheetStore: vi.fn(),
  useSQLEditorTabStore: vi.fn(),
  useUserStore: vi.fn(),
  useSheetContext: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useWorkSheetStore: mocks.useWorkSheetStore,
  useUserStore: mocks.useUserStore,
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/views/sql-editor/Sheet", () => ({
  useSheetContext: mocks.useSheetContext,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({
    children,
    content,
  }: {
    children: React.ReactNode;
    content: React.ReactNode;
  }) => (
    <div>
      <div data-testid="tooltip-content">{content}</div>
      {children}
    </div>
  ),
}));

vi.mock("@/types/proto-es/v1/worksheet_service_pb", () => ({
  Worksheet_Visibility: {
    PRIVATE: 0,
    PROJECT_READ: 1,
    PROJECT_WRITE: 2,
  },
}));

// ---- helpers ----------------------------------------------------------------

const makeNode = (
  overrides?: Partial<WorksheetFolderNode>
): WorksheetFolderNode => ({
  key: "/my/folder",
  label: "folder",
  editable: true,
  children: [],
  empty: false,
  ...overrides,
});

const makeWorksheetNode = (
  overrides?: Partial<WorksheetFolderNode>
): WorksheetFolderNode =>
  makeNode({
    key: "/my/folder/ws1",
    label: "My Query",
    worksheet: {
      name: "worksheets/ws1",
      title: "My Query",
      folders: [],
      type: "worksheet",
    },
    ...overrides,
  });

const renderIntoContainer = (element: React.ReactElement) => {
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

let TreeNodeSuffix: typeof import("./TreeNodeSuffix").TreeNodeSuffix;

beforeEach(async () => {
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.useWorkSheetStore.mockReturnValue({
    getWorksheetByName: (name: string) => ({
      name,
      starred: false,
      visibility: 0, // PRIVATE
      creator: "users/test@example.com",
    }),
  });
  mocks.useSQLEditorTabStore.mockReturnValue({
    closeTab: vi.fn(),
  });
  mocks.useUserStore.mockReturnValue({
    getUserByIdentifier: vi.fn(() => ({ title: "Test User" })),
  });
  mocks.useSheetContext.mockReturnValue({
    isWorksheetCreator: vi.fn(() => true),
  });

  ({ TreeNodeSuffix } = await import("./TreeNodeSuffix"));
});

afterEach(() => {
  document.body.innerHTML = "";
  vi.resetModules();
});

describe("TreeNodeSuffix", () => {
  test("renders star icon for worksheet node; click fires onToggleStar with correct args", () => {
    const node = makeWorksheetNode();
    const onToggleStar = vi.fn();
    const onSharePanelShow = vi.fn();
    const onContextMenuShow = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <TreeNodeSuffix
        node={node}
        view="my"
        onToggleStar={onToggleStar}
        onSharePanelShow={onSharePanelShow}
        onContextMenuShow={onContextMenuShow}
      />
    );
    render();

    // Finds the star SVG by its lucide class
    const starSvg = container.querySelector("svg.lucide-star");
    expect(starSvg).not.toBeNull();

    // Click the star
    act(() => {
      starSvg?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(onToggleStar).toHaveBeenCalledWith({
      worksheet: "worksheets/ws1",
      starred: true, // was false, toggled to true
    });

    unmount();
  });

  test("does not render star icon for folder nodes", () => {
    const node = makeNode(); // folder, no worksheet
    const onToggleStar = vi.fn();
    const onSharePanelShow = vi.fn();
    const onContextMenuShow = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <TreeNodeSuffix
        node={node}
        view="my"
        onToggleStar={onToggleStar}
        onSharePanelShow={onSharePanelShow}
        onContextMenuShow={onContextMenuShow}
      />
    );
    render();

    const starSvg = container.querySelector("svg.lucide-star");
    expect(starSvg).toBeNull();

    unmount();
  });

  test('"More" button fires onContextMenuShow with the node', () => {
    const node = makeWorksheetNode();
    const onContextMenuShow = vi.fn();
    const onToggleStar = vi.fn();
    const onSharePanelShow = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <TreeNodeSuffix
        node={node}
        view="my"
        onToggleStar={onToggleStar}
        onSharePanelShow={onSharePanelShow}
        onContextMenuShow={onContextMenuShow}
      />
    );
    render();

    // lucide-react renders MoreHorizontal as "lucide-ellipsis"
    const moreSvg = container.querySelector("svg.lucide-ellipsis");
    expect(moreSvg).not.toBeNull();

    act(() => {
      moreSvg?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    // React wraps native events in SyntheticBaseEvent (not native MouseEvent)
    expect(onContextMenuShow).toHaveBeenCalledWith(
      expect.objectContaining({ type: "click" }),
      node
    );

    unmount();
  });
});
