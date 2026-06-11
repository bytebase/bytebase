import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  // Per-test controllable state read by the migrated Zustand hooks.
  currentTabDatabase: "db/test" as string | undefined,
  project: "projects/proj1" as string,
  addTab: vi.fn(),
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
  externalUrl: "https://example.com" as string | undefined,
  linkedQueryHistory: undefined as
    | { name: string; statement: string; createTime: unknown }
    | undefined,
  linkedQueryHistoryTabId: "tab1" as string | undefined,
  setLinkedQueryHistory: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/stores/app", () => {
  // `useAppStore` is used both as a selector hook (for
  // `serverInfo.externalUrl`) and via `.getState().notify(...)`. Route
  // `notify` through the same vi.fn the tests asserted against under the
  // legacy `pushNotification` API so the assertion bodies stay readable.
  const state = () => ({
    notify: mocks.pushNotification,
    serverInfo: { externalUrl: mocks.externalUrl },
  });
  const useAppStore = (selector?: (s: ReturnType<typeof state>) => unknown) =>
    selector ? selector(state()) : state();
  useAppStore.getState = state;
  return { useAppStore };
});

vi.mock("@/react/router", () => ({
  router: {
    resolve: ({
      params,
    }: {
      params: { project: string; queryHistory: string };
    }) => ({
      href: `/sql-editor/projects/${params.project}/queryHistories/${params.queryHistory}`,
    }),
  },
}));

vi.mock("@/react/router/handles", () => ({
  SQL_EDITOR_QUERY_HISTORY_MODULE: "sql-editor.query-history",
}));

// Zustand tab store — selector hook + imperative getter.
vi.mock("@/react/stores/sqlEditor/tab", () => ({
  useSQLEditorTabState: (
    selector: (s: {
      currentTabId: string;
      tabsById: Map<string, { connection: { database: string | undefined } }>;
    }) => unknown
  ) =>
    selector({
      currentTabId: "tab1",
      tabsById: new Map([
        ["tab1", { connection: { database: mocks.currentTabDatabase } }],
      ]),
    }),
  getSQLEditorTabsState: () => ({
    currentTabId: "tab1",
    tabsById: new Map([["tab1", { connection: {} }]]),
    addTab: mocks.addTab,
  }),
}));

// Zustand editor store — active project read.
vi.mock("@/react/stores/sqlEditor/editor", () => ({
  useSQLEditorEditorState: (selector: (s: { project: string }) => unknown) =>
    selector({ project: mocks.project }),
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  selectQueryHistoryEntry: () => () => mocks.historyEntry,
  useSQLEditorStore: (
    selector: (s: {
      fetchQueryHistoryList: typeof mocks.fetchQueryHistoryList;
      resetPageToken: typeof mocks.resetPageToken;
      linkedQueryHistory: typeof mocks.linkedQueryHistory;
      linkedQueryHistoryTabId: typeof mocks.linkedQueryHistoryTabId;
      setLinkedQueryHistory: typeof mocks.setLinkedQueryHistory;
    }) => unknown
  ) =>
    selector({
      fetchQueryHistoryList: mocks.fetchQueryHistoryList,
      resetPageToken: mocks.resetPageToken,
      linkedQueryHistory: mocks.linkedQueryHistory,
      linkedQueryHistoryTabId: mocks.linkedQueryHistoryTabId,
      setLinkedQueryHistory: mocks.setLinkedQueryHistory,
    }),
}));

vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: {
    emit: mocks.sqlEditorEventsEmit,
    on: vi.fn().mockReturnValue(() => {}),
    off: vi.fn(),
  },
}));

// HistorySearchInput is exercised in its own test; here it's stubbed to a
// plain input wired to the string-based onChange.
vi.mock("./HistorySearchInput", () => ({
  HistorySearchInput: ({
    value,
    onChange,
    placeholder,
  }: {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
  }) => (
    <input
      data-testid="search-input"
      value={value}
      onChange={(e) => onChange(e.target.value)}
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
  extractProjectResourceName: (name: string) =>
    name.match(/(?:^|\/)projects\/([^/]+)(?:$|\/)/)?.[1] ?? "",
  extractQueryHistoryUID: (name: string) =>
    name.match(/(?:^|\/)queryHistories\/([^/]+)(?:$|\/)/)?.[1] ?? "-1",
}));

// The real HighlightLabelText pulls in `@/utils/util` (DOMPurify + i18n init);
// stub it to render the text so textContent assertions still hold.
vi.mock("@/react/components/HighlightLabelText", () => ({
  HighlightLabelText: ({
    text,
    className,
  }: {
    text: string;
    className?: string;
  }) => <span className={className}>{text}</span>,
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

  mocks.currentTabDatabase = "db/test";
  mocks.project = "projects/proj1";
  mocks.externalUrl = "https://example.com";
  mocks.linkedQueryHistory = undefined;
  mocks.linkedQueryHistoryTabId = "tab1";

  mocks.historyEntry = { queryHistories: [], nextPageToken: "" };
  mocks.fetchQueryHistoryList.mockResolvedValue({ queryHistories: [] });

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

  test("click copy-link button → clipboard writes the external deep link", async () => {
    mocks.historyEntry = {
      queryHistories: [
        makeHistory("projects/proj1/queryHistories/h1", "SELECT * FROM users"),
      ],
      nextPageToken: "",
    };

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    const copyLinkBtn = container.querySelector(
      "[data-copy-link-btn]"
    ) as HTMLButtonElement;
    expect(copyLinkBtn).not.toBeNull();

    await act(async () => {
      copyLinkBtn.click();
    });

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      "https://example.com/sql-editor/projects/proj1/queryHistories/h1"
    );
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
    unmount();
  });

  test("copy-link button falls back to window origin when externalUrl unset", async () => {
    mocks.externalUrl = undefined;
    mocks.historyEntry = {
      queryHistories: [
        makeHistory("projects/proj1/queryHistories/h1", "SELECT * FROM users"),
      ],
      nextPageToken: "",
    };

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    const copyLinkBtn = container.querySelector(
      "[data-copy-link-btn]"
    ) as HTMLButtonElement;

    await act(async () => {
      copyLinkBtn.click();
    });

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      `${window.location.origin}/sql-editor/projects/proj1/queryHistories/h1`
    );
    unmount();
  });

  test("opened-from-link section renders the linked history and recent header", () => {
    mocks.linkedQueryHistory = makeHistory(
      "projects/proj1/queryHistories/h1",
      "SELECT * FROM salary"
    );
    mocks.historyEntry = {
      queryHistories: [makeHistory("histories/h2", "SELECT * FROM employee")],
      nextPageToken: "",
    };

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    const linkedSection = container.querySelector("[data-linked-history]");
    expect(linkedSection).not.toBeNull();
    expect(container.textContent).toContain("sql-editor.opened-from-link");
    expect(container.textContent).toContain("SELECT * FROM salary");
    // Recent header appears because there is a non-linked recent entry.
    expect(container.textContent).toContain("sql-editor.recent");
    expect(container.textContent).toContain("SELECT * FROM employee");
    unmount();
  });

  test("linked history is not duplicated in the recent list", () => {
    mocks.linkedQueryHistory = makeHistory(
      "projects/proj1/queryHistories/h1",
      "SELECT * FROM salary"
    );
    // The recent list also contains the linked entry (same name) plus another.
    mocks.historyEntry = {
      queryHistories: [
        makeHistory("projects/proj1/queryHistories/h1", "SELECT * FROM salary"),
        makeHistory("histories/h2", "SELECT * FROM employee"),
      ],
      nextPageToken: "",
    };

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    // One linked row + one recent row (the duplicate is filtered out).
    expect(container.querySelectorAll("[data-history-row]").length).toBe(2);
    unmount();
  });

  test("opened-from-link section is hidden when another tab is active", () => {
    mocks.linkedQueryHistory = makeHistory(
      "projects/proj1/queryHistories/h1",
      "SELECT * FROM salary"
    );
    // Linked draft tab is not the active tab (active is "tab1").
    mocks.linkedQueryHistoryTabId = "other-tab";
    mocks.historyEntry = {
      queryHistories: [makeHistory("histories/h2", "SELECT * FROM employee")],
      nextPageToken: "",
    };

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    expect(container.querySelector("[data-linked-history]")).toBeNull();
    expect(container.textContent).not.toContain("sql-editor.opened-from-link");
    // Recent header is also hidden — it only pairs with the linked section.
    expect(container.textContent).not.toContain("sql-editor.recent");
    expect(container.textContent).toContain("SELECT * FROM employee");
    unmount();
  });

  test("dismiss button clears the linked history", async () => {
    mocks.linkedQueryHistory = makeHistory(
      "projects/proj1/queryHistories/h1",
      "SELECT * FROM salary"
    );

    const { container, render, unmount } = renderIntoContainer(<HistoryPane />);
    render();

    const dismissBtn = container.querySelector(
      "[data-dismiss-linked-history]"
    ) as HTMLButtonElement;
    expect(dismissBtn).not.toBeNull();

    await act(async () => {
      dismissBtn.click();
    });

    expect(mocks.setLinkedQueryHistory).toHaveBeenCalledWith(undefined);
    unmount();
  });
});
