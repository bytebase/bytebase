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
}: {
  disallowCopyingData?: boolean;
}) {
  return (
    <SQLResultViewProvider
      engine={Engine.POSTGRES}
      rows={rows}
      columns={columns}
      disallowCopyingData={disallowCopyingData}
    >
      <OpenDetailOnMount />
      <DetailPanel rows={rows} columns={columns} />
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

const getDetailContentRegion = () => {
  const candidates = Array.from(document.body.querySelectorAll("div"));
  const contentRegion = candidates.find(
    (element) =>
      element.textContent?.includes("longbridge") &&
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
});
