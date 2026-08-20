import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  SQLResultViewProvider,
  useSQLResultViewContext,
} from "./context";
import type { ResultTableColumn, ResultTableRow } from "./types";
import { VirtualDataTable } from "./VirtualDataTable";

vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: (options: {
    count: number;
    estimateSize: (index: number) => number;
  }) => ({
    getVirtualItems: () =>
      Array.from({ length: options.count }, (_, index) => ({
        index,
        key: index,
        start: index * options.estimateSize(index),
      })),
    getTotalSize: () => options.count * options.estimateSize(0),
    scrollToIndex: vi.fn(),
  }),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/modules/sql-editor/components/MaskingReasonPopover", () => ({
  MaskingReasonPopover: () => null,
}));

vi.mock("@/stores/app", () => {
  const state = () => ({ notify: vi.fn() });
  return {
    useAppStore: Object.assign(
      (selector: (value: ReturnType<typeof state>) => unknown) =>
        selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("@/utils/v1/database", () => ({
  getInstanceResource: () => ({
    name: "instances/prod",
    engine: Engine.COSMOSDB,
  }),
}));

class ResizeObserverStub implements ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

globalThis.ResizeObserver = ResizeObserverStub;

const columns: ResultTableColumn[] = [
  { id: "id", name: "id", columnType: "TEXT" },
];
const rows: ResultTableRow[] = [
  {
    key: 0,
    item: create(QueryRowSchema, {
      values: [
        create(RowValueSchema, {
          kind: { case: "stringValue", value: "one" },
        }),
      ],
    }),
  },
];
const database = {
  name: "instances/prod/databases/main",
  project: "projects/prod",
  instanceResource: {
    name: "instances/prod",
    engine: Engine.COSMOSDB,
  },
} as Database;

function DetailProbe() {
  const { detail } = useSQLResultViewContext();
  return (
    <div data-testid="detail-state">
      {detail ? `${detail.row}:${detail.col}:${detail.view}` : "closed"}
    </div>
  );
}

describe("VirtualDataTable row detail action", () => {
  test("replaces the hovered row number with a View detail action", () => {
    const { container } = render(
      <SQLResultViewProvider
        engine={Engine.COSMOSDB}
        rows={rows}
        columns={columns}
      >
        <VirtualDataTable
          rows={rows}
          columns={columns}
          activeRowIndex={-1}
          database={database}
          search={{ query: "", scopes: [] }}
          onToggleSort={() => undefined}
        />
        <DetailProbe />
      </SQLResultViewProvider>
    );

    const rowNumber = container.querySelector(
      '[data-row-index="0"] .textinfolabel'
    );
    expect(rowNumber).toHaveClass(
      "group-hover:opacity-0",
      "group-hover:pointer-events-none"
    );
    const action = screen.getByRole("button", {
      name: "sql-editor.view-detail",
    });
    expect(action).toHaveClass(
      "border",
      "size-6",
      "rounded-full",
      "shadow"
    );
    expect(action).not.toHaveClass("absolute");
    expect(action.parentElement).toHaveClass(
      "absolute",
      "left-3",
      "top-1/2",
      "opacity-0",
      "group-hover:opacity-100",
      "group-hover:pointer-events-auto"
    );
    expect(action.querySelector(".lucide-expand")).toBeInTheDocument();

    fireEvent.click(action);
    expect(screen.getByTestId("detail-state")).toHaveTextContent("0:0:row");
  });

  test("releases row detail action focus after pointer activation", () => {
    render(
      <SQLResultViewProvider
        engine={Engine.COSMOSDB}
        rows={rows}
        columns={columns}
      >
        <VirtualDataTable
          rows={rows}
          columns={columns}
          activeRowIndex={-1}
          database={database}
          search={{ query: "", scopes: [] }}
          onToggleSort={() => undefined}
        />
        <DetailProbe />
      </SQLResultViewProvider>
    );

    const action = screen.getByRole("button", {
      name: "sql-editor.view-detail",
    });
    action.focus();
    expect(action).toHaveFocus();

    fireEvent.click(action, { detail: 1 });

    expect(action).not.toHaveFocus();
    expect(screen.getByTestId("detail-state")).toHaveTextContent("0:0:row");
  });

  test("keeps row detail action focus after keyboard activation", () => {
    render(
      <SQLResultViewProvider
        engine={Engine.COSMOSDB}
        rows={rows}
        columns={columns}
      >
        <VirtualDataTable
          rows={rows}
          columns={columns}
          activeRowIndex={-1}
          database={database}
          search={{ query: "", scopes: [] }}
          onToggleSort={() => undefined}
        />
      </SQLResultViewProvider>
    );

    const action = screen.getByRole("button", {
      name: "sql-editor.view-detail",
    });
    action.focus();

    fireEvent.click(action, { detail: 0 });

    expect(action).toHaveFocus();
  });

  test("opens the entire relational row when a data cell is double-clicked", () => {
    const { container } = render(
      <SQLResultViewProvider
        engine={Engine.POSTGRES}
        rows={rows}
        columns={columns}
      >
        <VirtualDataTable
          rows={rows}
          columns={columns}
          activeRowIndex={-1}
          database={database}
          search={{ query: "", scopes: [] }}
          onToggleSort={() => undefined}
        />
        <DetailProbe />
      </SQLResultViewProvider>
    );

    const cell = container.querySelector(
      '[data-row-index="0"] [data-col-index="1"]'
    );
    expect(cell).toBeInstanceOf(HTMLDivElement);
    const cellValue = screen.getByText("one");

    fireEvent.doubleClick(cellValue);
    expect(screen.getByTestId("detail-state")).toHaveTextContent("0:0:row");
  });

  test("supports selecting a row without the row detail action", () => {
    const onRowClick = vi.fn();
    const { container } = render(
      <SQLResultViewProvider
        engine={Engine.COSMOSDB}
        rows={rows}
        columns={columns}
      >
        <VirtualDataTable
          rows={rows}
          columns={columns}
          activeRowIndex={0}
          database={database}
          search={{ query: "", scopes: [] }}
          onToggleSort={() => undefined}
          onRowClick={onRowClick}
          showRowDetailAction={false}
        />
      </SQLResultViewProvider>
    );

    fireEvent.click(
      container.querySelector(
        '[data-row-index="0"] [data-col-index="1"]'
      )!
    );

    expect(onRowClick).toHaveBeenCalledWith(0);
    expect(
      screen.queryByRole("button", { name: "sql-editor.view-detail" })
    ).not.toBeInTheDocument();
    expect(container.querySelector(".textinfolabel")).not.toHaveClass(
      "group-hover:opacity-0"
    );
  });

  test("activates a row without creating a copy selection", () => {
    const onRowClick = vi.fn();
    const { container } = render(
      <SQLResultViewProvider
        engine={Engine.COSMOSDB}
        rows={rows}
        columns={columns}
      >
        <VirtualDataTable
          rows={rows}
          columns={columns}
          activeRowIndex={0}
          database={database}
          search={{ query: "", scopes: [] }}
          onToggleSort={() => undefined}
          onRowClick={onRowClick}
          showRowDetailAction={false}
          allowSelection={false}
          activeRowHighlight="strong"
        />
      </SQLResultViewProvider>
    );

    fireEvent.click(screen.getByText("one"));

    expect(onRowClick).toHaveBeenCalledWith(0);
    expect(
      screen.queryByRole("button", { name: "Select row 1" })
    ).not.toBeInTheDocument();
    expect(
      container.querySelector('[data-col-index="1"] > div')
    ).not.toHaveClass("cursor-pointer");
    expect(container.querySelector('[data-col-index="1"]')).toHaveClass(
      "bg-accent/20!"
    );
  });
});
