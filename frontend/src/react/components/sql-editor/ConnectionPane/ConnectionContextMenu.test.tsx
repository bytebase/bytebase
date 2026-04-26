import { act, createRef, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTreeNode } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useConnectionMenu: vi.fn(),
  handleSelect: vi.fn(),
}));

vi.mock("./actions", () => ({
  useConnectionMenu: mocks.useConnectionMenu,
}));

vi.mock("@/react/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dropdown-menu">{children}</div>
  ),
  DropdownMenuContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dropdown-content">{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
  }) => (
    <button data-testid="dropdown-item" onClick={onClick} type="button">
      {children}
    </button>
  ),
  DropdownMenuTrigger: ({
    ref,
    ...rest
  }: {
    ref?: React.Ref<HTMLButtonElement>;
  } & React.HTMLAttributes<HTMLButtonElement>) => (
    <button ref={ref} data-testid="dropdown-trigger" {...rest} />
  ),
}));

let ConnectionContextMenu: typeof import("./ConnectionContextMenu").ConnectionContextMenu;
type Handle = import("./ConnectionContextMenu").ConnectionContextMenuHandle;

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

const makeNode = (): SQLEditorTreeNode =>
  ({
    key: "databases/bb",
    meta: {
      type: "database",
      target: { name: "instances/prod/databases/bb" },
    },
  }) as unknown as SQLEditorTreeNode;

beforeEach(async () => {
  vi.clearAllMocks();
  ({ ConnectionContextMenu } = await import("./ConnectionContextMenu"));
});

describe("ConnectionContextMenu", () => {
  test("renders null when the connection menu has no items", () => {
    mocks.useConnectionMenu.mockReturnValue({
      items: [],
      handleSelect: mocks.handleSelect,
    });
    const ref = createRef<Handle>();
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionContextMenu ref={ref} />
    );
    render();
    expect(container.textContent).toBe("");
    unmount();
  });

  test("renders one item per hook result + clicking invokes handleSelect with the key", () => {
    mocks.useConnectionMenu.mockReturnValue({
      items: [
        { key: "connect", label: "Connect", icon: null, onSelect: vi.fn() },
        {
          key: "view-database-detail",
          label: "View",
          icon: null,
          onSelect: vi.fn(),
        },
      ],
      handleSelect: mocks.handleSelect,
    });
    const ref = createRef<Handle>();
    const { container, render, unmount } = renderIntoContainer(
      <ConnectionContextMenu ref={ref} />
    );
    render();
    // Force a show to pass a node to the hook.
    act(() => {
      ref.current?.show(makeNode(), {
        preventDefault: () => {},
        stopPropagation: () => {},
        clientX: 10,
        clientY: 20,
      } as React.MouseEvent);
    });
    const items = container.querySelectorAll("[data-testid='dropdown-item']");
    expect(items.length).toBe(2);
    act(() => {
      (items[0] as HTMLButtonElement).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(mocks.handleSelect).toHaveBeenCalledWith("connect");
    unmount();
  });
});
