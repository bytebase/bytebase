import { render, screen } from "@testing-library/react";
import { act, cloneElement } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { formatQueryTime, ResultStatusBar } from "./ResultStatusBar";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/DatabaseTargetDisplay", () => ({
  DatabaseTargetDisplay: ({
    database,
    showEnvironment,
  }: {
    database: Database;
    showEnvironment?: boolean;
  }) => (
    <span
      data-testid="database-target-display"
      data-database={database.name}
      data-show-environment={String(showEnvironment)}
    />
  ),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({
    children,
    content,
  }: {
    children: React.ReactElement<{ title?: string }>;
    content?: string;
  }) => (content ? cloneElement(children, { title: content }) : children),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      notify: vi.fn(),
    }),
  },
}));

const database = {
  name: "instances/prod/databases/very-long-database-name",
  project: "projects/prod",
  effectiveEnvironment: "environments/prod",
} as Database;

let resizeCallbacks: ResizeObserverCallback[] = [];

globalThis.ResizeObserver = class ResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    resizeCallbacks.push(callback);
  }
  observe() {}
  unobserve() {}
  disconnect() {}
} as typeof ResizeObserver;

const setElementWidth = (
  element: Element,
  width: { clientWidth: number; scrollWidth: number }
) => {
  Object.defineProperty(element, "clientWidth", {
    configurable: true,
    value: width.clientWidth,
  });
  Object.defineProperty(element, "scrollWidth", {
    configurable: true,
    value: width.scrollWidth,
  });
};

// The element the bar measures: the truncating span rendered by EllipsisText.
const statementTextOf = (statement: HTMLElement) => {
  const text = statement.querySelector("span");
  if (!text) throw new Error("statement text span not rendered");
  return text;
};

// jsdom reports every rect as zero, so the bar measures no trailing controls
// unless a test places them. Only `right` is read.
const setRightEdge = (element: Element, right: number) => {
  Object.defineProperty(element, "getBoundingClientRect", {
    configurable: true,
    value: () => ({ right, left: 0, width: right }) as DOMRect,
  });
};

const flushResize = () => {
  act(() => {
    for (const callback of resizeCallbacks) {
      callback([] as unknown as ResizeObserverEntry[], {
        disconnect: vi.fn(),
        observe: vi.fn(),
        unobserve: vi.fn(),
      });
    }
  });
};

describe("ResultStatusBar", () => {
  beforeEach(() => {
    resizeCallbacks = [];
  });

  test("formats query latency", () => {
    expect(formatQueryTime(undefined)).toBe("-");
    expect(formatQueryTime({ seconds: 0n, nanos: 250_000_000 } as never)).toBe(
      "250 ms"
    );
    expect(formatQueryTime({ seconds: 2n, nanos: 500_000_000 } as never)).toBe(
      "2.50 s"
    );
  });

  test("lets the database label shrink while the statement truncates", () => {
    render(
      <ResultStatusBar
        database={database}
        statement="SELECT db.environment as env, db.instance as ins FROM db JOIN project on db.project = project.resource_id WHERE project.resource_id = 'a' AND db.deleted = false"
        queryTime="4 ms"
      />
    );

    const statusLeft = screen.getByTestId("result-status-left");
    expect(statusLeft.className).toContain("min-w-0");
    expect(statusLeft.className).toContain("flex-1");

    const databaseLabel = screen.getByTestId("result-status-database");
    expect(databaseLabel.className).toContain("min-w-0");
    expect(databaseLabel.className).toContain("overflow-hidden");
    expect(databaseLabel.className).toContain("whitespace-nowrap");

    const statement = screen.getByTestId("result-status-statement");
    expect(statement.className).toContain("min-w-0");
    expect(statement.className).toContain("flex-1");
    expect(statement.className).not.toContain("max-w-3xl");
    expect(statement.textContent).toContain("SELECT db.environment");
  });

  test("renders the database with the unified display", () => {
    render(
      <ResultStatusBar
        database={database}
        statement="SELECT 1"
        queryTime="3 ms"
      />
    );

    const databaseTarget = screen.getByTestId("database-target-display");
    expect(databaseTarget.getAttribute("data-database")).toBe(database.name);
    expect(databaseTarget.getAttribute("data-show-environment")).toBe("true");
  });

  test("lets the statement use available space while keeping copy attached", () => {
    render(
      <ResultStatusBar
        database={database}
        statement="SELECT db.environment as env, db.instance as ins FROM db JOIN project on db.project = project.resource_id WHERE project.resource_id = 'a' AND 2=2 AND 3=3 LIMIT 50"
        queryTime="7 ms"
      />
    );

    const statement = screen.getByTestId("result-status-statement");
    const copyButton = screen.getByRole("button", { name: "common.copy" });
    const statementText = statement.querySelector("span");

    expect(statement.className).toContain("flex-1");
    expect(statement.className).not.toContain("max-w-3xl");
    expect(statementText?.classList.contains("flex-1")).toBe(false);
    expect(copyButton.className).toContain("shrink-0");
    expect(copyButton.className).toContain("h-6");
    expect(copyButton.className).not.toContain("h-auto");
  });

  test("hides the statement copy button when the statement is empty", () => {
    render(
      <ResultStatusBar database={database} statement="" queryTime="3 ms" />
    );

    expect(
      screen.queryByRole("button", { name: "common.copy" })
    ).not.toBeInTheDocument();
  });

  test("hides the database label when it would constrain the statement", () => {
    render(
      <ResultStatusBar
        database={database}
        statement="EXPLAIN SELECT db.environment as env, db.instance as ins FROM db JOIN project on db.project = project.resource_id WHERE project.resource_id = 'a' AND db.deleted = false"
        queryTime="6 ms"
      />
    );

    const statusLeft = screen.getByTestId("result-status-left");
    const databaseLabel = screen.getByTestId("result-status-database");
    const statement = screen.getByTestId("result-status-statement");
    const statementText = statementTextOf(statement);

    setElementWidth(statusLeft, { clientWidth: 520, scrollWidth: 520 });
    setElementWidth(databaseLabel, { clientWidth: 210, scrollWidth: 210 });
    setElementWidth(statement, { clientWidth: 310, scrollWidth: 310 });
    setElementWidth(statementText, { clientWidth: 282, scrollWidth: 900 });

    flushResize();

    expect(databaseLabel.classList.contains("hidden")).toBe(true);
  });

  test("keeps a long database label when the capped width still fits", () => {
    render(
      <ResultStatusBar
        database={database}
        statement="SELECT 1"
        queryTime="3 ms"
      />
    );

    const statusLeft = screen.getByTestId("result-status-left");
    const databaseLabel = screen.getByTestId("result-status-database");
    const statement = screen.getByTestId("result-status-statement");

    setElementWidth(statusLeft, { clientWidth: 520, scrollWidth: 520 });
    setElementWidth(databaseLabel, { clientWidth: 230, scrollWidth: 900 });
    setElementWidth(statement, { clientWidth: 282, scrollWidth: 282 });
    setElementWidth(statementTextOf(statement), {
      clientWidth: 250,
      scrollWidth: 250,
    });

    flushResize();

    expect(databaseLabel.classList.contains("hidden")).toBe(false);
  });

  test("counts the copy control and the row gaps against the space left", () => {
    const { rerender } = render(
      <ResultStatusBar
        database={database}
        statement="SELECT * FROM employee WHERE id = 1"
        queryTime="5 ms"
      />
    );

    const statusLeft = screen.getByTestId("result-status-left");
    const databaseLabel = screen.getByTestId("result-status-database");
    const statement = screen.getByTestId("result-status-statement");
    const statementText = statementTextOf(statement);
    const copyButton = screen.getByRole("button", { name: "common.copy" });

    // 250px of text, then a 4px gap and a 24px copy control; the label is 230px
    // and sits 8px from the statement. The row needs 516px in all.
    statusLeft.style.columnGap = "8px";
    setElementWidth(databaseLabel, { clientWidth: 230, scrollWidth: 230 });
    setElementWidth(statementText, { clientWidth: 250, scrollWidth: 250 });
    setRightEdge(statementText, 250);
    setRightEdge(copyButton, 278);

    setElementWidth(statusLeft, { clientWidth: 500, scrollWidth: 500 });
    flushResize();

    // The text alone would fit beside the label in 500px; the copy control and
    // the gaps are what push the row over, so the label has to yield.
    expect(databaseLabel.classList.contains("hidden")).toBe(true);

    setElementWidth(statusLeft, { clientWidth: 520, scrollWidth: 520 });
    flushResize();

    expect(databaseLabel.classList.contains("hidden")).toBe(false);

    // An empty statement drops the copy control, so nothing trails the text.
    rerender(
      <ResultStatusBar database={database} statement="" queryTime="5 ms" />
    );
    setElementWidth(statementTextOf(statement), {
      clientWidth: 0,
      scrollWidth: 0,
    });
    setElementWidth(statusLeft, { clientWidth: 240, scrollWidth: 240 });
    flushResize();

    expect(databaseLabel.classList.contains("hidden")).toBe(false);
  });

  test("shows the database label again when the container widens back", () => {
    render(
      <ResultStatusBar
        database={database}
        statement="SELECT db.environment as env, db.instance as ins FROM db JOIN project on db.project = project.resource_id WHERE project.resource_id = 'a' AND db.deleted = false"
        queryTime="4 ms"
      />
    );

    const statusLeft = screen.getByTestId("result-status-left");
    const databaseLabel = screen.getByTestId("result-status-database");
    const statement = screen.getByTestId("result-status-statement");
    const statementText = statementTextOf(statement);

    // Widths measured in Chromium at a 1440px viewport. The statement row
    // stretches to fill whatever the label leaves, while the text span stays at
    // its intrinsic 994px in every state below.
    setElementWidth(statusLeft, { clientWidth: 1319, scrollWidth: 1319 });
    setElementWidth(databaseLabel, { clientWidth: 233, scrollWidth: 233 });
    setElementWidth(statement, { clientWidth: 1078, scrollWidth: 1078 });
    setElementWidth(statementText, { clientWidth: 994, scrollWidth: 994 });

    flushResize();

    expect(databaseLabel.classList.contains("hidden")).toBe(false);

    // Narrowed to a 600px viewport: the statement no longer fits beside the
    // label, so the label gives up its room.
    setElementWidth(statusLeft, { clientWidth: 479, scrollWidth: 479 });
    setElementWidth(databaseLabel, { clientWidth: 216, scrollWidth: 216 });
    setElementWidth(statement, { clientWidth: 256, scrollWidth: 256 });
    setElementWidth(statementText, { clientWidth: 224, scrollWidth: 994 });

    flushResize();

    expect(databaseLabel.classList.contains("hidden")).toBe(true);

    // Back to 1440px. The hidden label now measures zero and the statement row
    // stretches across the whole width, so only the text span still reports
    // what the statement actually needs.
    setElementWidth(statusLeft, { clientWidth: 1319, scrollWidth: 1319 });
    setElementWidth(databaseLabel, { clientWidth: 0, scrollWidth: 0 });
    setElementWidth(statement, { clientWidth: 1319, scrollWidth: 1319 });
    setElementWidth(statementText, { clientWidth: 994, scrollWidth: 994 });

    flushResize();

    expect(databaseLabel.classList.contains("hidden")).toBe(false);
  });
});
