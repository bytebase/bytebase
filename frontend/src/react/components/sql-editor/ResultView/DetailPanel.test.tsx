import { create } from "@bufbuild/protobuf";
import type { ReactElement } from "react";
import { act, useEffect } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  QueryRowSchema,
  type RowValue,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import {
  SQLResultViewProvider,
  useSelectionContext,
  useSQLResultViewContext,
} from "./context";
import { DetailPanel } from "./DetailPanel";
import type { ResultTableColumn, ResultTableRow } from "./types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
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

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const textValue = (value: string): RowValue =>
  create(RowValueSchema, {
    kind: { case: "stringValue", value },
  });

const columns: ResultTableColumn[] = [
  {
    id: "payload",
    name: "payload",
    columnType: "json",
  },
];

const detailContent = '{"customer":"longbridge","allowed":true}';
const rows: ResultTableRow[] = [
  {
    key: 0,
    item: create(QueryRowSchema, {
      values: [textValue(detailContent)],
    }),
  },
];

function OpenDetailOnMount() {
  const { setDetail } = useSQLResultViewContext();
  useEffect(() => {
    setDetail({ row: 0, col: 0 });
  }, [setDetail]);
  return null;
}

function SelectCellOnMount() {
  const { toggleSelectCell } = useSelectionContext();
  useEffect(() => {
    toggleSelectCell(0, 0);
  }, [toggleSelectCell]);
  return null;
}

function TestDetailPanel({
  disallowCopyingData = false,
  panelRows = rows,
  panelColumns = columns,
}: {
  disallowCopyingData?: boolean;
  panelRows?: ResultTableRow[];
  panelColumns?: ResultTableColumn[];
}) {
  return (
    <SQLResultViewProvider
      engine={Engine.POSTGRES}
      rows={panelRows}
      columns={panelColumns}
      disallowCopyingData={disallowCopyingData}
    >
      <OpenDetailOnMount />
      <DetailPanel rows={panelRows} columns={panelColumns} />
    </SQLResultViewProvider>
  );
}

function TestProviderWithGridSelection() {
  return (
    <SQLResultViewProvider
      engine={Engine.POSTGRES}
      rows={rows}
      columns={columns}
    >
      <SelectCellOnMount />
    </SQLResultViewProvider>
  );
}

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
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

const setInputValue = (input: HTMLInputElement, value: string) => {
  const valueSetter = Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    "value"
  )?.set;
  valueSetter?.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
};

const getDetailContentRegion = (expectedText = "longbridge") => {
  const candidates = Array.from(document.body.querySelectorAll("div"));
  const contentRegion = candidates.find(
    (element) =>
      element.textContent?.includes(expectedText) &&
      element.className.includes("overflow-auto") &&
      element.className.includes("font-mono")
  );
  expect(contentRegion).toBeInstanceOf(HTMLDivElement);
  return contentRegion as HTMLDivElement;
};

beforeEach(() => {
  localStorage.clear();
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
    configurable: true,
    writable: true,
  });
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("DetailPanel", () => {
  test("keeps allowed drawer text selectable", () => {
    const { render, unmount } = renderIntoContainer(<TestDetailPanel />);
    render();

    expect(getDetailContentRegion().className).toContain("select-text");

    unmount();
  });

  test("keeps copy-restricted drawer text non-selectable", () => {
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel disallowCopyingData />
    );
    render();

    expect(getDetailContentRegion().className).toContain("select-none");

    unmount();
  });

  test("does not bubble a click that finishes native drawer text selection", () => {
    const onDocumentClick = vi.fn();
    document.addEventListener("click", onDocumentClick);
    const { render, unmount } = renderIntoContainer(<TestDetailPanel />);
    render();

    const getSelectionSpy = vi.spyOn(window, "getSelection").mockReturnValue({
      toString: () => "selected drawer text",
    } as Selection);

    act(() => {
      getDetailContentRegion().dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });

    expect(onDocumentClick).not.toHaveBeenCalled();

    getSelectionSpy.mockRestore();
    document.removeEventListener("click", onDocumentClick);
    unmount();
  });

  test("lets native text selection handle copy even when grid selection exists", () => {
    const { render, unmount } = renderIntoContainer(
      <TestProviderWithGridSelection />
    );
    render();

    const getSelectionSpy = vi.spyOn(window, "getSelection").mockReturnValue({
      toString: () => "selected drawer text",
    } as Selection);

    const event = new KeyboardEvent("keydown", {
      key: "c",
      metaKey: true,
      bubbles: true,
      cancelable: true,
    });
    act(() => {
      document.dispatchEvent(event);
    });

    expect(event.defaultPrevented).toBe(false);
    expect(navigator.clipboard.writeText).not.toHaveBeenCalled();

    getSelectionSpy.mockRestore();
    unmount();
  });

  test("highlights plain text search matches and jumps between them", () => {
    const textRows: ResultTableRow[] = [
      {
        key: 0,
        item: create(QueryRowSchema, {
          values: [textValue("CREATE TABLE users;\nCREATE INDEX users_name;")],
        }),
      },
    ];
    const { render, unmount } = renderIntoContainer(
      <TestDetailPanel panelRows={textRows} />
    );
    render();

    const input = document.body.querySelector(
      "input[aria-label='sql-editor.result-detail.search']"
    ) as HTMLInputElement | null;
    expect(input).toBeInstanceOf(HTMLInputElement);

    act(() => {
      setInputValue(input!, "create");
    });

    const marks = Array.from(
      getDetailContentRegion("CREATE TABLE").querySelectorAll("mark")
    );
    expect(marks.map((mark) => mark.textContent)).toEqual(["CREATE", "CREATE"]);
    expect(document.body.textContent).toContain("1 / 2");
    expect(marks[0]?.className).toContain("bg-accent");
    const searchControl = input!.closest(
      "[data-testid='detail-search-control']"
    );
    expect(searchControl).toBeInstanceOf(HTMLDivElement);
    expect(searchControl?.textContent).toContain("1 / 2");
    expect(searchControl?.querySelectorAll("button")).toHaveLength(3);
    expect(searchControl?.className).toContain("rounded-xs");
    expect(searchControl?.className).toContain("bg-transparent");
    expect(searchControl?.className).not.toContain("rounded-md");
    expect(searchControl?.className).not.toContain("shadow");
    expect(searchControl?.parentElement?.className).not.toContain("flex-1");

    act(() => {
      input!.dispatchEvent(
        new KeyboardEvent("keydown", {
          key: "Enter",
          bubbles: true,
          cancelable: true,
        })
      );
    });

    const nextMarks = Array.from(
      getDetailContentRegion("CREATE TABLE").querySelectorAll("mark")
    );
    expect(nextMarks[1]?.className).toContain("bg-accent");

    unmount();
  });

  test("focuses detail search from the native find shortcut", () => {
    const { render, unmount } = renderIntoContainer(<TestDetailPanel />);
    render();

    const event = new KeyboardEvent("keydown", {
      key: "f",
      metaKey: true,
      bubbles: true,
      cancelable: true,
    });
    act(() => {
      document.dispatchEvent(event);
    });

    expect(event.defaultPrevented).toBe(true);
    expect(document.activeElement?.getAttribute("aria-label")).toBe(
      "sql-editor.result-detail.search"
    );

    unmount();
  });
});
