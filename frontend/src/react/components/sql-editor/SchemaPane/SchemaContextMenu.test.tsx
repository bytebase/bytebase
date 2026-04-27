import { act, createRef, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { TreeNode } from "./schemaTree";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useSchemaPaneContextMenu: vi.fn(),
}));

vi.mock("./actions", () => ({
  useSchemaPaneContextMenu: mocks.useSchemaPaneContextMenu,
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
  DropdownMenuSubmenu: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dropdown-submenu">{children}</div>
  ),
  DropdownMenuSubmenuContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dropdown-submenu-content">{children}</div>
  ),
  DropdownMenuSubmenuTrigger: ({ children }: { children: React.ReactNode }) => (
    <button data-testid="dropdown-submenu-trigger" type="button">
      {children}
    </button>
  ),
}));

vi.mock("@base-ui/react/menu", () => ({
  Menu: {
    Trigger: ({
      ref,
      ...rest
    }: {
      ref?: React.Ref<HTMLButtonElement>;
    } & React.HTMLAttributes<HTMLButtonElement>) => (
      <button ref={ref} data-testid="menu-trigger" {...rest} />
    ),
  },
}));

let SchemaContextMenu: typeof import("./SchemaContextMenu").SchemaContextMenu;
type Handle = import("./SchemaContextMenu").SchemaContextMenuHandle;

const renderInto = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  return {
    container,
    render: () => act(() => root.render(element)),
    unmount: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

const makeNode = (): TreeNode =>
  ({
    key: "table/users",
    meta: {
      type: "table",
      target: {
        database: "instances/i/databases/db",
        schema: "",
        table: "users",
      },
    },
  }) as unknown as TreeNode;

const noopDeps = {
  availableActions: [],
  setSchemaViewer: () => {},
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ SchemaContextMenu } = await import("./SchemaContextMenu"));
});

describe("SchemaContextMenu", () => {
  test("renders nothing when the menu hook returns no items", () => {
    mocks.useSchemaPaneContextMenu.mockReturnValue([]);
    const ref = createRef<Handle>();
    const { container, render, unmount } = renderInto(
      <SchemaContextMenu ref={ref} {...noopDeps} />
    );
    render();
    expect(container.textContent).toBe("");
    unmount();
  });

  test("renders one item per hook result and clicking invokes onSelect", () => {
    const select = vi.fn();
    mocks.useSchemaPaneContextMenu.mockReturnValue([
      { key: "copy-name", label: "Copy", icon: null, onSelect: select },
    ]);
    const ref = createRef<Handle>();
    const { container, render, unmount } = renderInto(
      <SchemaContextMenu ref={ref} {...noopDeps} />
    );
    render();
    // Force a show so the hook is called with a non-null node.
    act(() => {
      ref.current?.show(makeNode(), {
        preventDefault: () => {},
        stopPropagation: () => {},
        clientX: 10,
        clientY: 20,
      } as React.MouseEvent);
    });
    const items = container.querySelectorAll("[data-testid='dropdown-item']");
    expect(items.length).toBe(1);
    act(() => {
      (items[0] as HTMLButtonElement).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(select).toHaveBeenCalled();
    unmount();
  });

  test("renders nested submenu for items with children", () => {
    mocks.useSchemaPaneContextMenu.mockReturnValue([
      {
        key: "generate-sql",
        label: "Generate SQL",
        icon: null,
        children: [
          {
            key: "generate-sql--select",
            label: "SELECT",
            icon: null,
            onSelect: vi.fn(),
          },
        ],
      },
    ]);
    const ref = createRef<Handle>();
    const { container, render, unmount } = renderInto(
      <SchemaContextMenu ref={ref} {...noopDeps} />
    );
    render();
    act(() => {
      ref.current?.show(makeNode(), {
        preventDefault: () => {},
        stopPropagation: () => {},
        clientX: 0,
        clientY: 0,
      } as React.MouseEvent);
    });
    expect(
      container.querySelector("[data-testid='dropdown-submenu']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='dropdown-submenu-trigger']")
    ).not.toBeNull();
    expect(
      container.querySelector("[data-testid='dropdown-submenu-content']")
    ).not.toBeNull();
    unmount();
  });
});
