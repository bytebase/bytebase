import { fireEvent, render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import procedureTwoStatements from "./test-data/mssql/procedure-two-statements.xml?raw";
import bitmapIndexScan from "./test-data/postgres/bitmap-index-scan.json";
import spannerHashJoin from "./test-data/spanner/hash-join.json";
import { parseMssqlPlan } from "./mssql-plan";
import type { PlanTree } from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import { QueryPlanViewer } from "./QueryPlanViewer";
import { parseSpannerPlan } from "./spanner-plan";

vi.mock("react-resizable-panels", () => ({
  Group: ({
    children,
    orientation,
  }: {
    children: ReactNode;
    orientation: string;
  }) => (
    <div data-testid="plan-split" data-orientation={orientation}>
      {children}
    </div>
  ),
  Panel: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Separator: () => <div />,
}));

// jsdom's window exposes no matchMedia under vitest, so the viewer's
// narrow-screen query needs one.
function mockMatchMedia(matches: boolean) {
  window.matchMedia = ((query: string) => ({
    matches,
    media: query,
    onchange: null,
    addEventListener: () => {},
    removeEventListener: () => {},
    addListener: () => {},
    removeListener: () => {},
    dispatchEvent: () => true,
  })) as unknown as typeof window.matchMedia;
}

const rawPlan = JSON.stringify(bitmapIndexScan);

const tree = (): PlanTree => {
  const result = parsePostgresPlan(rawPlan);
  if (!result.ok) throw new Error(result.message);
  return result.tree;
};

const renderViewer = () =>
  render(<QueryPlanViewer tree={tree()} rawPlan={rawPlan} />);

/** Renders a plan the way the result pane does: parsed, beside its source. */
const renderParsed = (
  parse: (source: string) => ReturnType<typeof parsePostgresPlan>,
  source: string
) => {
  const result = parse(source);
  if (!result.ok) throw new Error(result.message);
  return render(<QueryPlanViewer tree={result.tree} rawPlan={source} />);
};

const openTab = (name: string) =>
  fireEvent.click(screen.getByRole("tab", { name }));

const tabNames = () =>
  screen.getAllByRole("tab").map((entry) => entry.textContent);

/** The panel on screen; the diagram's stays mounted behind the others. */
const visiblePanel = () => within(screen.getByRole("tabpanel"));

/** A card on the diagram, not the ranking row of the same name. */
const diagramCard = (name: RegExp) =>
  within(screen.getByTestId("plan-diagram-viewport")).getByRole("button", {
    name,
  });

const collapseToggle = () => screen.getByTestId("plan-node-collapse");
const cards = () => screen.queryAllByTestId("plan-node-card");

describe("QueryPlanViewer", () => {
  beforeEach(() => {
    mockMatchMedia(false);
  });

  test("opens on the diagram with the costliest operator selected", () => {
    renderParsed(parseMssqlPlan, procedureTwoStatements);

    expect(screen.getByTestId("plan-diagram-viewport")).toBeVisible();
    const [first] = visiblePanel().getAllByTestId("plan-costliest-row");
    expect(first).toHaveAttribute("aria-pressed", "true");
    const name = first.querySelector(".truncate")?.textContent ?? "";
    expect(screen.getByRole("heading", { name })).toBeInTheDocument();
  });

  test("offers the diagram, a table and the text, in that order", () => {
    renderViewer();

    expect(tabNames()).toEqual(["Diagram", "Table", "Text"]);
    expect(screen.getByText("2 nodes · estimated cost 39.18")).toBeVisible();
  });

  test("ranks the costliest operators beside the diagram", () => {
    renderViewer();

    const rows = visiblePanel().getAllByTestId("plan-costliest-row");
    // 34.81 of the plan's 39.18 cost units are the heap scan's own.
    expect(rows[0]).toHaveTextContent("Bitmap Heap Scan");
    expect(rows[0]).toHaveTextContent("88.8%");
  });

  test("selects an operator from the ranking", () => {
    renderViewer();

    fireEvent.click(visiblePanel().getAllByTestId("plan-costliest-row")[1]);
    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();
    expect(
      diagramCard(/Bitmap Index Scan/)
    ).toHaveAttribute("aria-pressed", "true");
  });

  test("moves the detail pane to whichever node is clicked", () => {
    renderViewer();

    fireEvent.click(
      diagramCard(/Bitmap Index Scan/)
    );
    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();
    expect(screen.getByText("(customer_id = 42)")).toBeInTheDocument();
  });

  test("lists the same plan as a table, beside the same ranking", () => {
    renderViewer();

    openTab("Table");
    expect(screen.getByTestId("plan-diagram-viewport")).not.toBeVisible();
    expect(
      visiblePanel()
        .getAllByTestId("plan-grid-node-type")
        .map((cell) => cell.textContent)
    ).toEqual(["Bitmap Heap Scan", "Bitmap Index Scan"]);
    expect(
      visiblePanel().getByText("2 nodes · estimated cost 39.18")
    ).toBeVisible();
    expect(visiblePanel().getAllByTestId("plan-costliest-row")).toHaveLength(2);
  });

  test("keeps the selection when moving between the diagram and table", () => {
    renderViewer();

    openTab("Table");
    const rows = screen.getAllByTestId("plan-grid-row");
    expect(rows[0]).toHaveAttribute("aria-selected", "true");

    fireEvent.click(rows[1]);
    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();

    openTab("Diagram");
    expect(
      diagramCard(/Bitmap Index Scan/)
    ).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(diagramCard(/^Bitmap Heap Scan/));
    openTab("Table");
    expect(screen.getAllByTestId("plan-grid-row")[0]).toHaveAttribute(
      "aria-selected",
      "true"
    );
  });

  test("switches to the text plan and back to the diagram", () => {
    renderViewer();

    openTab("Text");
    expect(screen.getByText(/"Node Type": "Bitmap Heap Scan"/)).toBeVisible();
    expect(screen.getByTestId("plan-diagram-viewport")).not.toBeVisible();
    expect(screen.getByTestId("plan-copy-button")).toHaveTextContent(
      "Copy plan"
    );

    openTab("Diagram");
    expect(screen.getByTestId("plan-diagram-viewport")).toBeVisible();
  });

  test("shows a separate text plan in the same tab row", () => {
    render(
      <QueryPlanViewer
        tree={tree()}
        rawPlan={rawPlan}
        textPlan="Seq Scan on orders"
      />
    );

    openTab("Text");
    expect(screen.getByText("Seq Scan on orders")).toBeVisible();
    expect(screen.queryByText(/"Node Type"/)).toBeNull();
  });

  test("shades the diagram once a highlight mode is chosen", () => {
    renderViewer();
    expect(screen.queryAllByTestId("plan-node-tint")).toHaveLength(0);

    fireEvent.click(screen.getByRole("button", { name: "Cost" }));
    expect(screen.getAllByTestId("plan-node-tint").length).toBeGreaterThan(0);
  });

  test("offers every highlight for a plan with cost and row estimates", () => {
    renderParsed(parseMssqlPlan, procedureTwoStatements);

    expect(
      within(screen.getByRole("group", { name: "Highlight" }))
        .getAllByRole("button")
        .map((option) => option.textContent)
    ).toEqual(["Off", "Cost", "Rows"]);
    expect(screen.getByText("10 nodes · estimated cost 1.64")).toBeVisible();
    // What the bars and edges mean is a hover or a focus away, not a line of
    // its own.
    expect(screen.getByRole("button", { name: /card's bar/ })).toBeVisible();
  });

  test("keeps to what a plan without estimates can show", () => {
    renderParsed(parseSpannerPlan, JSON.stringify(spannerHashJoin));

    // Nothing can be ranked or shaded.
    expect(tabNames()).toEqual(["Diagram", "Table", "Text"]);
    expect(screen.queryByTestId("plan-costliest")).toBeNull();
    expect(screen.queryByRole("group", { name: "Highlight" })).toBeNull();
    expect(screen.getByText("7 nodes")).toBeVisible();
    expect(screen.queryByLabelText(/card's bar/)).toBeNull();
    expect(screen.getAllByTestId("plan-node-card")).toHaveLength(7);
    // With no cost to judge relevance by, the scan warnings still reach the
    // panel rather than being dropped for want of a number to weigh them.
    const warnings = screen.getAllByTestId("plan-summary-warning");
    expect(warnings).toHaveLength(2);
    expect(warnings[0]).toHaveTextContent("Full table scan");
    expect(warnings[0]).toHaveTextContent("Albums");
    expect(warnings[1]).toHaveTextContent("Full index scan");
  });

  test("indents an XML plan one element per line on the text tab", () => {
    renderParsed(parseMssqlPlan, procedureTwoStatements);

    openTab("Text");
    const raw = visiblePanel().getByText(/^<ShowPlanXML/).textContent;
    const lines = raw?.split("\n") ?? [];

    expect(lines[0]).toBe(
      '<ShowPlanXML xmlns="http://schemas.microsoft.com/sqlserver/2004/07/showplan" Version="1.564" Build="16.0.4275.2">'
    );
    expect(lines[1]).toBe("  <BatchSequence>");
    expect(lines.at(-1)).toBe("</ShowPlanXML>");
    // A line break inside an attribute stays escaped, so a copy keeps it.
    expect(raw).toContain(
      'StatementText="&#10;CREATE   PROCEDURE dbo.region_report @region VARCHAR(16) AS&#10;BEGIN&#10;'
    );
    // The indented plan is still the same plan.
    const reparsed = parseMssqlPlan(raw ?? "");
    const original = parseMssqlPlan(procedureTwoStatements);
    expect(reparsed.ok && reparsed.tree).toEqual(original.ok && original.tree);
  });

  test("splits side by side on a wide screen", () => {
    renderViewer();
    expect(screen.getByTestId("plan-split")).toHaveAttribute(
      "data-orientation",
      "horizontal"
    );
  });

  test("stacks the side panel below the diagram on a narrow screen", () => {
    mockMatchMedia(true);
    renderViewer();
    expect(screen.getByTestId("plan-split")).toHaveAttribute(
      "data-orientation",
      "vertical"
    );
  });

  test("hides copy controls when copying is disabled", () => {
    render(
      <QueryPlanViewer tree={tree()} rawPlan={rawPlan} disallowCopyingData />
    );

    openTab("Text");
    expect(screen.queryByTestId("plan-copy-button")).toBeNull();
    expect(screen.getByRole("tablist").parentElement).toHaveClass(
      "select-none"
    );
  });

  test("keeps the diagram's zoom while the reader moves between tabs", () => {
    renderViewer();
    const canvas = () => screen.getByTestId("plan-diagram-canvas").style.transform;

    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    const zoomed = canvas();
    expect(zoomed).toContain("scale(1.25)");

    openTab("Table");
    openTab("Diagram");

    expect(canvas()).toBe(zoomed);
  });

  test("keeps a subtree folded while the reader moves between tabs", () => {
    renderViewer();
    expect(cards()).toHaveLength(2);

    fireEvent.click(collapseToggle());
    expect(cards()).toHaveLength(1);

    openTab("Table");
    openTab("Diagram");
    expect(cards()).toHaveLength(1);
    expect(collapseToggle()).toHaveAttribute("aria-expanded", "false");
  });

  test("moves the selection out of a subtree it folds away", () => {
    renderViewer();

    fireEvent.click(diagramCard(/Bitmap Index Scan/));
    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();

    fireEvent.click(collapseToggle());

    // The selected node has no card any more, so the collapse that swallowed
    // it takes the selection.
    expect(
      screen.getByRole("heading", { name: "Bitmap Heap Scan" })
    ).toBeInTheDocument();
  });

  test("unfolds a subtree when a node inside it is selected elsewhere", () => {
    renderViewer();
    fireEvent.click(collapseToggle());

    openTab("Table");
    // The table still lists every node, folded or not.
    fireEvent.click(screen.getAllByTestId("plan-grid-row")[1]);

    openTab("Diagram");
    expect(cards()).toHaveLength(2);
    expect(
      diagramCard(/Bitmap Index Scan/)
    ).toHaveAttribute("aria-pressed", "true");
  });

  test("shows a note at the end of the tab row", () => {
    render(
      <QueryPlanViewer tree={tree()} rawPlan={rawPlan} note="Estimated now" />
    );

    const note = screen.getByText("Estimated now");
    expect(note).toHaveClass("ml-auto");
    expect(note.closest('[role="tablist"]')).not.toBeNull();
  });
});

describe("QueryPlanViewer on a measured plan", () => {
  const measuredPlan = JSON.stringify([
    {
      Plan: {
        "Node Type": "Nested Loop",
        "Total Cost": 100,
        "Plan Rows": 10,
        "Actual Total Time": 24.5,
        "Actual Rows": 10,
        "Actual Loops": 1,
        Plans: [
          {
            "Node Type": "Seq Scan",
            "Parent Relationship": "Outer",
            "Relation Name": "orders",
            "Total Cost": 90,
            "Plan Rows": 4,
            "Actual Total Time": 20,
            "Actual Rows": 4,
            "Actual Loops": 1,
          },
        ],
      },
    },
  ]);

  beforeEach(() => mockMatchMedia(false));

  test("offers shading by the time it measured, ahead of the estimate", () => {
    renderParsed(parsePostgresPlan, measuredPlan);

    expect(
      within(screen.getByRole("group", { name: "Highlight" }))
        .getAllByRole("button")
        .map((option) => option.textContent)
    ).toEqual(["Off", "Time", "Cost", "Rows"]);
  });

  test("reports the measured total and says the bars follow it", () => {
    renderParsed(parsePostgresPlan, measuredPlan);

    expect(
      screen.getByText("2 nodes · measured time 24.5 ms · estimated cost 100")
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: /share of the plan's measured time/ })
    ).toBeVisible();
  });

  test("shades by measured time once asked", () => {
    renderParsed(parsePostgresPlan, measuredPlan);

    fireEvent.click(screen.getByRole("button", { name: "Time" }));
    expect(screen.getAllByTestId("plan-node-tint").length).toBeGreaterThan(0);
  });
});
