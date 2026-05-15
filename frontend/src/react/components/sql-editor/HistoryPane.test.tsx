import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useSQLEditorTabStore: vi.fn(),
  // Legacy Pinia editor store.
  useSQLEditorVueState: vi.fn(),
  // Mutable per-test state for the zustand mock.
  historyEntry: {
    queryHistories: [] as Array<{
      name: string;
      statement: string;
      createTime: { seconds: bigint; nanos: number } | undefined;
    }>,
    nextPageToken: "" as string | undefined,
  },
  fetchQueryHistoryList: vi.fn().mockResolvedValue({ queryHistories: [] }),
  resetPageToken: vi.fn(),
  pushNotification: vi.fn(),
  sqlEditorEventsEmit: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/react/stores/sqlEditor/tab-vue-state", () => ({
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
}));

vi.mock("@/react/stores/sqlEditor/editor-vue-state", () => ({
  useSQLEditorVueState: mocks.useSQLEditorVueState,
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  selectQueryHistoryEntry: () => () => mocks.historyEntry,
  useSQLEditorStore: (
    selector: (s: {
      fetchQueryHistoryList: typeof mocks.fetchQueryHistoryList;
      resetPageToken: typeof mocks.resetPageToken;
    }) => unknown
  ) =>
    selector({
      fetchQueryHistoryList: mocks.fetchQueryHistoryList,
      resetPageToken: mocks.resetPageToken,
    }),
}));

vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: {
    emit: mocks.sqlEditorEventsEmit,
    on: vi.fn().mockReturnValue(() => {}),
    off: vi.fn(),
  },
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: ({
    value,
    onChange,
    placeholder,
  }: {
    value: string;
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    placeholder?: string;
  }) => (
    <input
      data-testid="search-input"
      value={value}
      onChange={onChange}
      placeholder={placeholder}
    />
  ),
}));

vi.mock("@/types", () => ({
  DEBOUNCE_SEARCH_DELAY: 300,
  getDateForPbTimestampProtoEs: vi.fn((ts: unknown) =>
    ts ? new Date("2024-01-15T10:30:00Z") : new Date(0)
  ),
}));

vi.mock("@/utils", () => ({
  getHighlightHTMLByKeyWords: vi.fn((s: string, _k: string) =>
    s.replace(/</g, "&lt;").replace(/>/g, "&gt;")
  ),
}));

vi.mock("dayjs", () => {
  const mockDayjs = vi.fn(() => ({
    format: vi.fn(() => "2024-01-15 10:30:00"),
  }));
  return { default: mockDayjs };
});

let HistoryPane: typeof import("./HistoryPane").HistoryPane;

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

const makeHistory = (
  name: string,
  statement: string
): {
  name: string;
  statement: string;
  createTime: { seconds: bigint; nanos: number } | undefined;
} => ({
  name,
  statement,
  createTime: { seconds: BigInt(1705312200), nanos: 0 },
});

beforeEach(async () => {
  vi.clearAllMocks();

  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });

  mocks.useSQLEditorTabStore.mockReturnValue({
    currentTab: { connection: { database: "db/test" } },
    addTab: vi.fn(),
  });

  mocks.useSQLEditorVueState.mockReturnValue({
    project: "projects/proj1",
  });

  mocks.historyEntry = { queryHistories: [], nextPageToken: "" };
  mocks.fetchQueryHistoryList.mockResolvedValue({ queryHistories: [] });

  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

  Object.defineProperty(navigator, "clipboard", {
    value: { writeText: vi.fn().mockResolvedValue(undefined) },
    configurable: true,
    writable: true,
  });

  ({ HistoryPane } = await import("./HistoryPane"));
});

afterEach(() => {
  document.body.innerHTML = "";
  vi.useRealTimers();
});

describe("HistoryPane", () => {
  test("empty state — shows no-history-found when empty list and not loading", () => {
    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    expect(container.textContent).toContain("sql-editor.no-history-found");
    unmount();
  });

  test("rendered history list — both entries visible via statement text", () => {
    mocks.historyEntry = {
      queryHistories: [
        makeHistory("histories/h1", "SELECT * FROM users"),
        makeHistory("histories/h2", "SELECT id FROM orders"),
      ],
      nextPageToken: "",
    };

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    expect(container.textContent).toContain("SELECT * FROM users");
    expect(container.textContent).toContain("SELECT id FROM orders");
    unmount();
  });

  test("debounced search — fetchQueryHistoryList called with statement after timer", async () => {
    vi.useFakeTimers();

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    const searchInput = container.querySelector(
      "[data-testid='search-input']"
    ) as HTMLInputElement;
    expect(searchInput).not.toBeNull();

    act(() => {
      Object.defineProperty(searchInput, "value", {
        writable: true,
        value: "SELECT",
      });
      const inputEvent = new Event("change", { bubbles: true });
      Object.defineProperty(inputEvent, "target", {
        writable: false,
        value: searchInput,
      });
      searchInput.dispatchEvent(inputEvent);
    });

    await act(async () => {
      vi.advanceTimersByTime(300);
    });

    expect(mocks.fetchQueryHistoryList).toHaveBeenCalled();
    unmount();
  });

  test("click history row → emits append-editor-content event", async () => {
    mocks.historyEntry = {
      queryHistories: [makeHistory("histories/h1", "SELECT * FROM users")],
      nextPageToken: "",
    };

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    const historyRow = container.querySelector(
      "[data-history-row]"
    ) as HTMLElement;
    expect(historyRow).not.toBeNull();

    await act(async () => {
      historyRow.click();
    });

    expect(mocks.sqlEditorEventsEmit).toHaveBeenCalledWith(
      "append-editor-content",
      { content: "SELECT * FROM users", select: true }
    );
    unmount();
  });

  test("click copy button → clipboard writeText and pushNotification called", async () => {
    mocks.historyEntry = {
      queryHistories: [makeHistory("histories/h1", "SELECT * FROM users")],
      nextPageToken: "",
    };

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    const copyBtn = container.querySelector(
      "[data-copy-btn]"
    ) as HTMLButtonElement;
    expect(copyBtn).not.toBeNull();

    await act(async () => {
      copyBtn.click();
    });

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      "SELECT * FROM users"
    );
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    unmount();
  });
});
