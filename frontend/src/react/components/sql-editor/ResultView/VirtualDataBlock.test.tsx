import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryRowSchema,
  type RowValue,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { SQLResultViewProvider } from "./context";
import type { ResultTableColumn, ResultTableRow } from "./types";
import { VirtualDataBlock } from "./VirtualDataBlock";

const virtualizerOptions: Array<{
  count: number;
  estimateSize: (index: number) => number;
  gap?: number;
  paddingEnd?: number;
}> = [];

vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: (options: {
    count: number;
    estimateSize: (index: number) => number;
    gap?: number;
    paddingEnd?: number;
  }) => {
    virtualizerOptions.push(options);
    const itemSize = options.estimateSize(0);
    const gap = options.gap ?? 0;
    const paddingEnd = options.paddingEnd ?? 0;
    return {
      getVirtualItems: () =>
        Array.from({ length: options.count }, (_, index) => ({
          index,
          key: index,
          start: index * (itemSize + gap),
        })),
      getTotalSize: () =>
        options.count * itemSize +
        Math.max(0, options.count - 1) * gap +
        paddingEnd,
      measureElement: vi.fn(),
      scrollToIndex: vi.fn(),
    };
  },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/components/sql-editor/MaskingReasonPopover", () => ({
  MaskingReasonPopover: () => null,
}));

vi.mock("@/react/stores/app", () => {
  const state = () => ({
    notify: vi.fn(),
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("@/types", () => ({
  getDateForPbTimestampProtoEs: (ts: { seconds: bigint; nanos: number }) =>
    new Date(Number(ts.seconds) * 1000 + Math.floor((ts.nanos ?? 0) / 1e6)),
}));

vi.mock("@/utils/v1/database", () => ({
  getInstanceResource: () => ({
    name: "instances/prod",
    engine: Engine.POSTGRES,
  }),
}));

class ResizeObserverStub implements ResizeObserver {
  constructor(_callback: ResizeObserverCallback) {}
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords(): ResizeObserverEntry[] {
    return [];
  }
}

globalThis.ResizeObserver = ResizeObserverStub;

const textValue = (value: string): RowValue =>
  create(RowValueSchema, {
    kind: { case: "stringValue", value },
  });

const columns: ResultTableColumn[] = [
  { id: "company", name: "company", columnType: "TEXT" },
  { id: "status", name: "status", columnType: "TEXT" },
];

const rows: ResultTableRow[] = [
  {
    key: 0,
    item: create(QueryRowSchema, {
      values: [textValue("Acme"), textValue("reviewed")],
    }),
  },
  {
    key: 1,
    item: create(QueryRowSchema, {
      values: [textValue("Bytebase"), textValue("approved")],
    }),
  },
  {
    key: 2,
    item: create(QueryRowSchema, {
      values: [textValue("Longbridge"), textValue("final-row")],
    }),
  },
];

const database = {
  name: "instances/prod/databases/main",
  project: "projects/prod",
  instanceResource: {
    name: "instances/prod",
    engine: Engine.POSTGRES,
  },
} as Database;

const renderBlock = () =>
  render(
    <SQLResultViewProvider
      engine={Engine.POSTGRES}
      rows={rows}
      columns={columns}
    >
      <VirtualDataBlock
        rows={rows}
        columns={columns}
        activeRowIndex={-1}
        isSensitiveColumn={() => false}
        database={database}
        search={{ query: "", scopes: [] }}
      />
    </SQLResultViewProvider>
  );

describe("VirtualDataBlock", () => {
  beforeEach(() => {
    virtualizerOptions.length = 0;
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
      configurable: true,
      writable: true,
    });
  });

  test("reserves end scroll space and keeps the final row copy action clickable", () => {
    const { container } = renderBlock();

    const virtualizerConfig = virtualizerOptions.at(-1);
    expect(virtualizerConfig?.paddingEnd).toBeGreaterThanOrEqual(96);

    const scrollContent = container.querySelector(
      "div[style*='position: relative']"
    );
    expect(scrollContent).toBeInstanceOf(HTMLDivElement);
    const height = Number.parseFloat(
      (scrollContent as HTMLDivElement).style.height
    );
    const rowHeight = virtualizerConfig?.estimateSize(0) ?? 0;
    const gap = virtualizerConfig?.gap ?? 0;
    expect(height).toBeGreaterThanOrEqual(
      rows.length * rowHeight + (rows.length - 1) * gap + 96
    );

    const copyButtons = screen.getAllByRole("button", { name: "common.copy" });
    fireEvent.click(copyButtons[copyButtons.length - 1]);

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      JSON.stringify(
        {
          company: "Longbridge",
          status: "final-row",
        },
        null,
        4
      )
    );
  });
});
