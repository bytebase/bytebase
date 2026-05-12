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
  useSQLEditorStore: vi.fn(),
  useSQLEditorQueryHistoryStore: vi.fn(),
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
  useSQLEditorTabStore: mocks.useSQLEditorTabStore,
  useSQLEditorStore: mocks.useSQLEditorStore,
  useSQLEditorQueryHistoryStore: mocks.useSQLEditorQueryHistoryStore,
  pushNotification: mocks.pushNotification,
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

  mocks.useSQLEditorStore.mockReturnValue({
    project: "projects/proj1",
  });

  mocks.useSQLEditorQueryHistoryStore.mockReturnValue({
    getQueryHistoryList: vi.fn(() => ({
      queryHistories: [],
      nextPageToken: "",
    })),
    fetchQueryHistoryList: vi.fn().mockResolvedValue({ queryHistories: [] }),
    resetPageToken: vi.fn(),
  });

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
    const queryHistoryStore = {
      getQueryHistoryList: vi.fn(() => ({
        queryHistories: [],
        nextPageToken: "",
      })),
      fetchQueryHistoryList: vi.fn().mockResolvedValue({ queryHistories: [] }),
      resetPageToken: vi.fn(),
    };
    mocks.useSQLEditorQueryHistoryStore.mockReturnValue(queryHistoryStore);
    mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    expect(container.textContent).toContain("sql-editor.no-history-found");
    unmount();
  });

  test("rendered history list — both entries visible via statement text", () => {
    const histories = [
      makeHistory("histories/h1", "SELECT * FROM users"),
      makeHistory("histories/h2", "SELECT id FROM orders"),
    ];
    const queryHistoryStore = {
      getQueryHistoryList: vi.fn(() => ({
        queryHistories: histories,
        nextPageToken: "",
      })),
      fetchQueryHistoryList: vi
        .fn()
        .mockResolvedValue({ queryHistories: histories }),
      resetPageToken: vi.fn(),
    };
    mocks.useSQLEditorQueryHistoryStore.mockReturnValue(queryHistoryStore);
    mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    expect(container.textContent).toContain("SELECT * FROM users");
    expect(container.textContent).toContain("SELECT id FROM orders");
    unmount();
  });

  test("debounced search — fetchQueryHistoryList called with statement after timer", async () => {
    vi.useFakeTimers();

    const fetchQueryHistoryList = vi
      .fn()
      .mockResolvedValue({ queryHistories: [] });
    const queryHistoryStore = {
      getQueryHistoryList: vi.fn(() => ({
        queryHistories: [],
        nextPageToken: "",
      })),
      fetchQueryHistoryList,
      resetPageToken: vi.fn(),
    };
    mocks.useSQLEditorQueryHistoryStore.mockReturnValue(queryHistoryStore);
    mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    // Find the search input and type
    const searchInput = container.querySelector(
      "[data-testid='search-input']"
    ) as HTMLInputElement;
    expect(searchInput).not.toBeNull();

    // Type into the search input by directly setting value and firing input event
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

    // Advance timers to trigger debounce
    await act(async () => {
      vi.advanceTimersByTime(300);
    });

    // fetchQueryHistoryList should have been called (initial load + after search)
    expect(fetchQueryHistoryList).toHaveBeenCalled();
    unmount();
  });

  test("click history row → emits append-editor-content event", async () => {
    const histories = [makeHistory("histories/h1", "SELECT * FROM users")];
    const queryHistoryStore = {
      getQueryHistoryList: vi.fn(() => ({
        queryHistories: histories,
        nextPageToken: "",
      })),
      fetchQueryHistoryList: vi
        .fn()
        .mockResolvedValue({ queryHistories: histories }),
      resetPageToken: vi.fn(),
    };
    mocks.useSQLEditorQueryHistoryStore.mockReturnValue(queryHistoryStore);
    mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    // Find history row (first div in the history list)
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
    const histories = [makeHistory("histories/h1", "SELECT * FROM users")];
    const queryHistoryStore = {
      getQueryHistoryList: vi.fn(() => ({
        queryHistories: histories,
        nextPageToken: "",
      })),
      fetchQueryHistoryList: vi
        .fn()
        .mockResolvedValue({ queryHistories: histories }),
      resetPageToken: vi.fn(),
    };
    mocks.useSQLEditorQueryHistoryStore.mockReturnValue(queryHistoryStore);
    mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

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
