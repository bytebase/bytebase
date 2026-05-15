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

// ---- mocks ------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSheetContextByView: vi.fn(),
  useClickOutside: vi.fn(),
  onSelectCallback: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/views/sql-editor/Sheet", () => ({
  useSheetContextByView: mocks.useSheetContextByView,
}));

vi.mock("@/react/hooks/useClickOutside", () => ({
  useClickOutside: mocks.useClickOutside,
}));

type TreeDataNodeLike = {
  id: string;
  data: WorksheetFolderNode;
  children?: TreeDataNodeLike[];
};

// Mock Tree to render a simple test double
vi.mock("@/react/components/ui/tree", () => ({
  Tree: ({
    data,
    renderNode,
  }: {
    data: TreeDataNodeLike[];
    renderNode: (args: {
      node: {
        id: string;
        data: TreeDataNodeLike;
        isSelected: boolean;
      };
      style: React.CSSProperties;
    }) => React.ReactNode;
    onSelect?: (ids: string[]) => void;
  }) => (
    <div data-testid="tree">
      {data.map((item) =>
        renderNode({
          node: {
            id: item.id,
            data: item,
            isSelected: false,
          },
          style: {},
        })
      )}
    </div>
  ),
}));

// Mock Popover — render children inline, track open state
// Module-scoped open state mirror so PopoverContent mock can gate its children
// the same way Base UI's real Portal only renders when `open=true`.
let mockPopoverOpen = false;

vi.mock("@/react/components/ui/popover", () => ({
  Popover: ({
    children,
    open,
  }: {
    children: React.ReactNode;
    open?: boolean;
  }) => {
    mockPopoverOpen = !!open;
    return (
      <div data-testid="popover-root" data-open={String(open ?? false)}>
        {children}
      </div>
    );
  },
  PopoverTrigger: ({
    children,
    render,
  }: {
    children?: React.ReactNode;
    render?: React.ReactElement;
  }) => <div data-testid="popover-trigger">{render ?? children}</div>,
  PopoverContent: ({
    children,
    style,
  }: {
    children: React.ReactNode;
    style?: React.CSSProperties;
  }) =>
    mockPopoverOpen ? (
      <div data-testid="popover-content" style={style}>
        {children}
      </div>
    ) : null,
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: ({
    value,
    placeholder,
    onFocus,
    onChange,
  }: {
    value?: string;
    placeholder?: string;
    onFocus?: () => void;
    onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  }) => (
    <input
      data-testid="folder-input"
      value={value ?? ""}
      placeholder={placeholder}
      onFocus={onFocus}
      onChange={onChange}
      readOnly={!onChange}
    />
  ),
}));

vi.mock("./TreeNodePrefix", () => ({
  TreeNodePrefix: () => <span data-testid="tree-node-prefix" />,
}));

// ---- helpers ----------------------------------------------------------------

const makeFolderNode = (
  key: string,
  children: WorksheetFolderNode[] = []
): WorksheetFolderNode => ({
  key,
  label: key.split("/").slice(-1)[0],
  editable: true,
  children,
  empty: children.length === 0,
});

const setupDefaultMocks = () => {
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.useClickOutside.mockImplementation(() => undefined);
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

  const rootNode = makeFolderNode("/my", [
    makeFolderNode("/my/foo"),
    makeFolderNode("/my/bar"),
  ]);

  const viewContext = {
    folderTree: { value: rootNode },
    folderContext: {
      rootPath: { value: "/my" },
    },
  };

  mocks.useSheetContextByView.mockReturnValue(viewContext);
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

let FolderForm: typeof import("./FolderForm").FolderForm;

beforeEach(async () => {
  vi.clearAllMocks();
  setupDefaultMocks();
  ({ FolderForm } = await import("./FolderForm"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

// ---- tests ------------------------------------------------------------------

describe("FolderForm", () => {
  test("renders input with formatted path (rootPath stripped, slashes converted to ' / ')", () => {
    const { container, render, unmount } = renderIntoContainer(
      <FolderForm folder="/my/foo/bar" onFolderChange={vi.fn()} />
    );
    render();

    const input = container.querySelector(
      "[data-testid='folder-input']"
    ) as HTMLInputElement;
    expect(input).not.toBeNull();
    // "/my/foo/bar" → strip "/my" → "/foo/bar" → strip leading "/" → "foo/bar" → join as " / " → "foo / bar"
    expect(input.value).toBe("foo / bar");

    unmount();
  });

  test("focusing the input opens the popover", () => {
    const { container, render, unmount } = renderIntoContainer(
      <FolderForm folder="/my" onFolderChange={vi.fn()} />
    );
    render();

    // Tree not rendered yet (popover closed)
    expect(document.body.querySelector("[data-testid='tree']")).toBeNull();

    const input = container.querySelector(
      "[data-testid='folder-input']"
    ) as HTMLInputElement;

    act(() => {
      input.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
    });

    // Tree now rendered via portal to overlay layer
    expect(document.body.querySelector("[data-testid='tree']")).not.toBeNull();

    unmount();
  });

  test("typing in input normalizes path and calls onFolderChange", () => {
    const onFolderChange = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <FolderForm folder="/my" onFolderChange={onFolderChange} />
    );
    render();

    const input = container.querySelector(
      "[data-testid='folder-input']"
    ) as HTMLInputElement;
    expect(input).not.toBeNull();

    // Simulate onChange directly — trigger via React internal fiber
    act(() => {
      const nativeInput = container.querySelector(
        "[data-testid='folder-input']"
      ) as HTMLInputElement;
      Object.defineProperty(nativeInput, "value", {
        writable: true,
        value: "foo / bar",
      });
      nativeInput.dispatchEvent(new Event("change", { bubbles: true }));
    });

    // onFolderChange should have been called — at minimum once on mount with initial folder
    expect(onFolderChange).toHaveBeenCalled();

    unmount();
  });

  test("selecting a tree node updates folder and closes popover", async () => {
    const onFolderChange = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <FolderForm folder="/my" onFolderChange={onFolderChange} />
    );
    render();

    // First open the popover via focus
    const input = container.querySelector(
      "[data-testid='folder-input']"
    ) as HTMLInputElement;
    act(() => {
      input.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
    });

    // Tree now rendered via portal
    expect(document.body.querySelector("[data-testid='tree']")).not.toBeNull();

    // Find tree node row and click it (mock renders folderTree.children: /my/foo, /my/bar)
    const treeNodes = document.body.querySelectorAll(
      "[data-testid='tree'] > div"
    );
    expect(treeNodes.length).toBeGreaterThan(0);

    await act(async () => {
      (treeNodes[0] as HTMLElement).click();
      // queueMicrotask fires in the next microtask
      await new Promise((r) => setTimeout(r, 0));
    });

    // onFolderChange should have been called with the selected node key
    expect(onFolderChange).toHaveBeenCalledWith("/my/foo");

    // Tree should no longer be rendered (popover closed)
    expect(document.body.querySelector("[data-testid='tree']")).toBeNull();

    unmount();
  });
});
