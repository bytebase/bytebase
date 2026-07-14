import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { SQLResultViewProvider } from "./context";
import { TableCell } from "./TableCell";
import type { ResultTableColumn, ResultTableRow } from "./types";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/utils/v1/database", () => ({
  getInstanceResource: () => ({
    name: "instances/prod",
    engine: Engine.POSTGRES,
  }),
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

const database = {
  name: "instances/prod/databases/main",
  project: "projects/prod",
  instanceResource: {
    name: "instances/prod",
    engine: Engine.POSTGRES,
  },
} as Database;

const columns: ResultTableColumn[] = [
  { id: "created_at", name: "created_at", columnType: "TEXT" },
];

const longValue = "1.779696812227E+12";
const value = create(RowValueSchema, {
  kind: { case: "stringValue", value: longValue },
});

const rows: ResultTableRow[] = [
  {
    key: 0,
    item: create(QueryRowSchema, {
      values: [value],
    }),
  },
];

describe("TableCell", () => {
  beforeEach(() => {
    vi.spyOn(HTMLElement.prototype, "scrollWidth", "get").mockReturnValue(160);
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(80);
    vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockReturnValue(20);
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(20);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  test("reserves space for the expand action when cell content is truncated", async () => {
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
          database={database}
          keyword=""
        />
      </SQLResultViewProvider>
    );

    expect(await screen.findByRole("button")).toBeInTheDocument();
    expect(container.querySelector(".line-clamp-3")).toHaveClass(
      "max-w-[calc(100%-1.5rem)]"
    );
  });
});
