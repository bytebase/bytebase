import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { createExplainToken } from "@/utils/explainToken";
import { ExplainVisualizerApp } from "./ExplainVisualizerApp";
import { MSSQL_PLAN_INVALID_XML_MESSAGE } from "./mssql-plan";
import {
  POSTGRES_PLAN_EMPTY_MESSAGE,
  POSTGRES_PLAN_INVALID_JSON_MESSAGE,
} from "./postgres-plan";
import indexSeekKeyLookup from "./test-data/mssql/index-seek-key-lookup.xml?raw";
import cteNestedLoopInitplan from "./test-data/postgres/cte-nested-loop-initplan.json";
import spannerHashJoin from "./test-data/spanner/hash-join.json";

vi.mock("react-resizable-panels", () => ({
  Group: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Panel: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Separator: () => <div />,
}));

/** Opens the page the way the SQL editor does: a stored plan and its token. */
const openPlan = (engine: Engine, explain: string, statement = "SELECT 1") => {
  const token = createExplainToken({ engine, explain, statement });
  history.replaceState(null, "", `/explain-visualizer.html?token=${token}`);
  return render(<ExplainVisualizerApp />);
};

const card = (name: RegExp) =>
  screen
    .getAllByTestId("plan-node-card")
    .find((entry) => name.test(entry.getAttribute("aria-label") ?? ""));

describe("ExplainVisualizerApp", () => {
  beforeEach(() => {
    sessionStorage.clear();
    // jsdom's window exposes no matchMedia under vitest; the viewer queries it
    // to decide whether to stack the detail pane.
    window.matchMedia = ((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => true,
    })) as unknown as typeof window.matchMedia;
  });

  test("draws a PostgreSQL plan", () => {
    openPlan(Engine.POSTGRES, JSON.stringify(cteNestedLoopInitplan));

    for (const nodeType of [/^Limit/, /^Nested Loop/, /^Index Scan/]) {
      expect(card(nodeType)).toBeDefined();
    }
    expect(screen.getByText("InitPlan 1")).toBeInTheDocument();
  });

  test("draws a SQL Server plan in the same viewer", () => {
    openPlan(Engine.MSSQL, indexSeekKeyLookup);

    for (const nodeType of [/^SELECT/, /^Index Seek/, /^Key Lookup/]) {
      expect(card(nodeType)).toBeDefined();
    }
    expect(screen.getByRole("tab", { name: "Summary" })).toBeInTheDocument();
  });

  test("draws a Spanner plan in the same viewer", () => {
    openPlan(Engine.SPANNER, JSON.stringify(spannerHashJoin));

    for (const nodeType of [/^Hash Join/, /^Table Scan/, /^Index Scan/]) {
      expect(card(nodeType)).toBeDefined();
    }
    expect(screen.getByRole("tab", { name: "Diagram" })).toBeInTheDocument();
  });

  test("explains a plan it cannot read and still shows what the server returned", () => {
    openPlan(Engine.POSTGRES, "Seq Scan on customers");

    expect(screen.getByRole("alert")).toHaveTextContent(
      POSTGRES_PLAN_INVALID_JSON_MESSAGE
    );
    expect(screen.getByText("Seq Scan on customers")).toBeInTheDocument();
    expect(screen.queryByTestId("plan-diagram-viewport")).toBeNull();
  });

  test("reads each engine's plan with that engine's parser", () => {
    // Valid JSON, but SQL Server's plan is XML.
    openPlan(Engine.MSSQL, JSON.stringify(cteNestedLoopInitplan));

    expect(screen.getByRole("alert")).toHaveTextContent(
      MSSQL_PLAN_INVALID_XML_MESSAGE
    );
  });

  test("explains an empty plan without echoing an empty block", () => {
    const { container } = openPlan(Engine.POSTGRES, "");

    expect(screen.getByRole("alert")).toHaveTextContent(
      POSTGRES_PLAN_EMPTY_MESSAGE
    );
    expect(container.querySelector("pre")).toBeNull();
  });

  test("says the session expired when the plan is no longer stored", () => {
    history.replaceState(null, "", "/explain-visualizer.html?token=explain-gone");
    render(<ExplainVisualizerApp />);

    expect(screen.getByRole("alert")).toHaveTextContent("Session expired");
  });

  test("turns away an engine the visualizer cannot draw", () => {
    openPlan(Engine.MYSQL, "-> Table scan on t");

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Unsupported database engine"
    );
  });
});
