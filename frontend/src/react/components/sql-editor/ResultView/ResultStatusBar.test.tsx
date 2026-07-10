import { render, screen } from "@testing-library/react";
import { act } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { ResultStatusBar, RichDatabaseName } from "./ResultStatusBar";

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/EngineIcon", () => ({
  EngineIcon: () => <span data-testid="engine-icon" />,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      notify: vi.fn(),
    }),
  },
}));

vi.mock("@/utils/v1/database", () => ({
  extractDatabaseResourceName: () => ({
    instance: "instances/prod",
    database: "instances/prod/databases/very-long-database-name",
    instanceName: "prod",
    databaseName: "very-long-database-name",
  }),
  getDatabaseEnvironment: () => ({ title: "Prod" }),
  getInstanceResource: () => ({
    name: "instances/prod",
    title: "bytebase-3.17.11-selfhost",
    engine: Engine.POSTGRES,
  }),
}));

vi.mock("@/utils/v1/instance", () => ({
  instanceV1Name: () => "bytebase-3.17.11-selfhost",
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
  element: HTMLElement,
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

describe("ResultStatusBar", () => {
  beforeEach(() => {
    resizeCallbacks = [];
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

  test("does not cap rich database names outside the status bar", () => {
    render(<RichDatabaseName database={database} />);

    const databaseLabel = screen.getByTestId("result-status-database");
    expect(databaseLabel.className).not.toContain("max-w-[45%]");
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
  });

  test("hides the database label when it would constrain the statement", () => {
    render(
      <ResultStatusBar
        database={database}
        statement="EXPLAIN SELECT db.environment as env, db.instance as ins FROM db JOIN project on db.project = project.resource_id WHERE project.resource_id = 'a' AND db.deleted = false"
        queryTime="6 ms"
        showVisualizeButton
      />
    );

    const statusLeft = screen.getByTestId("result-status-left");
    const databaseLabel = screen.getByTestId("result-status-database");
    const statement = screen.getByTestId("result-status-statement");

    setElementWidth(statusLeft, { clientWidth: 520, scrollWidth: 520 });
    setElementWidth(databaseLabel, { clientWidth: 210, scrollWidth: 210 });
    setElementWidth(statement, { clientWidth: 310, scrollWidth: 900 });

    act(() => {
      for (const callback of resizeCallbacks) {
        callback([] as unknown as ResizeObserverEntry[], {
          disconnect: vi.fn(),
          observe: vi.fn(),
          unobserve: vi.fn(),
        });
      }
    });

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
    setElementWidth(statement, { clientWidth: 250, scrollWidth: 250 });

    act(() => {
      for (const callback of resizeCallbacks) {
        callback([] as unknown as ResizeObserverEntry[], {
          disconnect: vi.fn(),
          observe: vi.fn(),
          unobserve: vi.fn(),
        });
      }
    });

    expect(databaseLabel.classList.contains("hidden")).toBe(false);
  });
});
