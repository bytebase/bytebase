import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  SQLResultViewProvider,
  useSQLResultViewContext,
} from "./context";
import { TableCell } from "./TableCell";
import type { ResultTableColumn, ResultTableRow } from "./types";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

class ResizeObserverStub implements ResizeObserver {
  private readonly callback: ResizeObserverCallback;

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }
  observe() {
    this.callback([], this);
  }
  unobserve() {}
  disconnect() {}
  takeRecords(): ResizeObserverEntry[] {
    return [];
  }
}

globalThis.ResizeObserver = ResizeObserverStub;

const columns: ResultTableColumn[] = [
  { id: "created_at", name: "created_at", columnType: "TEXT" },
];

const longValue = "1.779696812227E+12";
const value = create(RowValueSchema, {
  kind: { case: "stringValue", value: longValue },
});
const indentedValue = create(RowValueSchema, {
  kind: { case: "stringValue", value: "  -> Seq Scan on project" },
});
const jsonValue = create(RowValueSchema, {
  kind: { case: "stringValue", value: '{"name":"Ada"}' },
});

const rows: ResultTableRow[] = [
  {
    key: 0,
    item: create(QueryRowSchema, {
      values: [value],
    }),
  },
];

function DetailProbe() {
  const { detail } = useSQLResultViewContext();
  return (
    <div data-testid="detail-state">
      {detail ? `${detail.row}:${detail.col}:${detail.view}` : "closed"}
    </div>
  );
}

describe("TableCell", () => {
  beforeEach(() => {
    vi.spyOn(HTMLElement.prototype, "scrollWidth", "get").mockReturnValue(80);
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(80);
    vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockReturnValue(20);
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(20);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  test("reserves space for the expand action when cell content is truncated", async () => {
    vi.spyOn(HTMLElement.prototype, "scrollWidth", "get").mockReturnValue(160);

    const { container } = render(
      <SQLResultViewProvider
        engine={Engine.POSTGRES}
        rows={rows}
        columns={columns}
      >
        <TableCell
          value={value}
          rowIndex={0}
          colIndex={0}
          allowSelect
          columnType="TEXT"
          keyword=""
        />
      </SQLResultViewProvider>
    );

    expect(await screen.findByRole("button")).toBeInTheDocument();
    expect(container.querySelector(".line-clamp-3")).toHaveClass(
      "max-w-[calc(100%-1.5rem)]"
    );
  });

  test("shows the expand action for non-truncated JSON content", async () => {
    render(
      <SQLResultViewProvider
        engine={Engine.POSTGRES}
        rows={rows}
        columns={columns}
      >
        <TableCell
          value={jsonValue}
          rowIndex={0}
          colIndex={0}
          allowSelect
          columnType="JSON"
          keyword=""
        />
        <DetailProbe />
      </SQLResultViewProvider>
    );

    const action = await screen.findByRole("button");
    fireEvent.click(action);

    expect(screen.getByTestId("detail-state")).toHaveTextContent("0:0:cell");
  });

  test("preserves leading whitespace without soft-wrapping string cells", () => {
    const { container } = render(
      <SQLResultViewProvider
        engine={Engine.POSTGRES}
        rows={rows}
        columns={columns}
      >
        <TableCell
          value={indentedValue}
          rowIndex={0}
          colIndex={0}
          allowSelect
          columnType="TEXT"
          keyword=""
        />
      </SQLResultViewProvider>
    );

    const content = container.querySelector(".line-clamp-3");
    expect(content?.textContent).toBe("  -> Seq Scan on project");
    expect(content).toHaveClass("whitespace-pre");
    expect(content).not.toHaveClass("whitespace-pre-wrap");
    expect(content).not.toHaveClass("wrap-break-word");
  });
});
