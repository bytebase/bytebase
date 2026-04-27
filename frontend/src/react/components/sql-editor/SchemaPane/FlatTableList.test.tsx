import { create } from "@bufbuild/protobuf";
import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  ColumnMetadataSchema,
  DatabaseMetadataSchema,
  IndexMetadataSchema,
  SchemaMetadataSchema,
  TableMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  hoverState: undefined as unknown,
  hoverUpdate: vi.fn(),
  hoverSetPosition: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
}));

vi.mock("./hover-state", () => ({
  useHoverState: () => ({
    state: mocks.hoverState,
    setPosition: mocks.hoverSetPosition,
    update: mocks.hoverUpdate,
    cancel: vi.fn(),
  }),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    className,
  }: {
    children: React.ReactNode;
    onClick?: (e: React.MouseEvent) => void;
    className?: string;
  }) => (
    <button
      type="button"
      data-testid="btn"
      data-class={className}
      onClick={onClick}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/utils/dom", () => ({
  findAncestor: () => null,
}));

let FlatTableList: typeof import("./FlatTableList").FlatTableList;

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

const buildMetadata = (
  schemas: {
    name: string;
    tables: { name: string; columns?: string[]; indexes?: string[] }[];
  }[]
) =>
  create(DatabaseMetadataSchema, {
    schemas: schemas.map((s) =>
      create(SchemaMetadataSchema, {
        name: s.name,
        tables: s.tables.map((t) =>
          create(TableMetadataSchema, {
            name: t.name,
            columns: (t.columns ?? []).map((c) =>
              create(ColumnMetadataSchema, { name: c, type: "varchar" })
            ),
            indexes: (t.indexes ?? []).map((i) =>
              create(IndexMetadataSchema, { name: i })
            ),
          })
        ),
      })
    ),
  });

beforeEach(async () => {
  mocks.hoverState = undefined;
  mocks.hoverUpdate.mockReset();
  mocks.hoverSetPosition.mockReset();
  ({ FlatTableList } = await import("./FlatTableList"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

const noopProps = {
  database: "instances/i/databases/db",
  onSelect: vi.fn(),
  onSelectAll: vi.fn(),
  onContextMenu: vi.fn(),
};

describe("FlatTableList", () => {
  test("renders one row per table across all schemas", () => {
    const md = buildMetadata([
      { name: "public", tables: [{ name: "users" }, { name: "orders" }] },
      { name: "audit", tables: [{ name: "events" }] },
    ]);
    const { container, render, unmount } = renderInto(
      <FlatTableList metadata={md} {...noopProps} />
    );
    render();
    expect(container.querySelectorAll(".bb-flat-table-row").length).toBe(3);
    unmount();
  });

  test("filters by search keyword (case-insensitive substring match on schema/table key)", () => {
    const md = buildMetadata([
      { name: "public", tables: [{ name: "users" }, { name: "orders" }] },
    ]);
    const { container, render, unmount } = renderInto(
      <FlatTableList metadata={md} search="ORD" {...noopProps} />
    );
    render();
    const rows = container.querySelectorAll(".bb-flat-table-row");
    expect(rows.length).toBe(1);
    expect(rows[0].textContent).toContain("orders");
    unmount();
  });

  test("renders the empty placeholder when no tables match the search", () => {
    const md = buildMetadata([{ name: "public", tables: [{ name: "users" }] }]);
    const { container, render, unmount } = renderInto(
      <FlatTableList metadata={md} search="missing" {...noopProps} />
    );
    render();
    expect(container.textContent).toContain("No tables found");
    unmount();
  });

  test("expand toggle reveals columns + indexes", () => {
    const md = buildMetadata([
      {
        name: "public",
        tables: [
          { name: "users", columns: ["id", "email"], indexes: ["ix_email"] },
        ],
      },
    ]);
    const { container, render, unmount } = renderInto(
      <FlatTableList metadata={md} {...noopProps} />
    );
    render();
    const expandBtn = container.querySelectorAll("[data-testid='btn']")[0];
    act(() => {
      (expandBtn as HTMLButtonElement).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(container.textContent).toContain("id");
    expect(container.textContent).toContain("email");
    expect(container.textContent).toContain("ix_email");
    unmount();
  });

  test("clicking a row fires onSelect with the table item", () => {
    const md = buildMetadata([{ name: "public", tables: [{ name: "users" }] }]);
    const onSelect = vi.fn();
    const { container, render, unmount } = renderInto(
      <FlatTableList
        metadata={md}
        database="instances/i/databases/db"
        onSelect={onSelect}
        onSelectAll={vi.fn()}
        onContextMenu={vi.fn()}
      />
    );
    render();
    const row = container.querySelector(".bb-flat-table-row") as HTMLElement;
    act(() => {
      row.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onSelect.mock.calls[0][0].metadata.name).toBe("users");
    unmount();
  });

  test("right-click fires onContextMenu and defaults are prevented", () => {
    const md = buildMetadata([{ name: "public", tables: [{ name: "users" }] }]);
    const onContextMenu = vi.fn();
    const { container, render, unmount } = renderInto(
      <FlatTableList
        metadata={md}
        database="instances/i/databases/db"
        onSelect={vi.fn()}
        onSelectAll={vi.fn()}
        onContextMenu={onContextMenu}
      />
    );
    render();
    const row = container.querySelector(".bb-flat-table-row") as HTMLElement;
    const event = new MouseEvent("contextmenu", {
      bubbles: true,
      cancelable: true,
    });
    act(() => {
      row.dispatchEvent(event);
    });
    expect(onContextMenu).toHaveBeenCalled();
    expect(event.defaultPrevented).toBe(true);
    unmount();
  });
});
