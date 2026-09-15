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

const renderViewer = (query?: string) =>
  render(<QueryPlanViewer tree={tree()} rawPlan={rawPlan} query={query} />);

/** Renders a plan the way the entry does: parsed, beside its source. */
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

/** Loads the page at a fragment, the way following a shared link would. */
const startAtFragment = (fragment: string) =>
  history.replaceState(null, "", `${location.pathname}${fragment}`);

const collapseToggle = () => screen.getByTestId("plan-node-collapse");
const cards = () => screen.queryAllByTestId("plan-node-card");

describe("QueryPlanViewer", () => {
  beforeEach(() => {
    mockMatchMedia(false);
    // The viewer keeps the selection in the fragment, so each test has to
    // start from a page that has none.
    startAtFragment("");
  });

  test("opens on the diagram with the root node already selected", () => {
    renderViewer();

    expect(screen.getByTestId("plan-diagram-viewport")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Bitmap Heap Scan" })
    ).toBeInTheDocument();
    expect(screen.getByText("2 nodes · estimated cost 39.18")).toBeVisible();
  });

  test("moves the detail pane to whichever node is clicked", () => {
    renderViewer();

    fireEvent.click(
      screen.getByRole("button", { name: /Bitmap Index Scan/ })
    );
    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();
    expect(screen.getByText("(customer_id = 42)")).toBeInTheDocument();
  });

  test("lists the same plan as a table on the grid tab", () => {
    renderViewer();

    fireEvent.click(screen.getByRole("tab", { name: "Grid" }));
    expect(screen.getByTestId("plan-diagram-viewport")).not.toBeVisible();
    expect(
      screen
        .getAllByTestId("plan-grid-node-type")
        .map((cell) => cell.textContent)
    ).toEqual(["Bitmap Heap Scan", "Bitmap Index Scan"]);
    // The diagram stays mounted behind this tab, so the totals it also draws
    // are in the document; this one belongs to the panel on screen.
    expect(
      within(screen.getByRole("tabpanel")).getByText(
        "2 nodes · estimated cost 39.18"
      )
    ).toBeVisible();
  });

  test("keeps the selection when moving between the diagram and grid", () => {
    renderViewer();

    fireEvent.click(screen.getByRole("tab", { name: "Grid" }));
    const rows = screen.getAllByTestId("plan-grid-row");
    expect(rows[0]).toHaveAttribute("aria-selected", "true");

    fireEvent.click(rows[1]);
    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Diagram" }));
    expect(
      screen.getByRole("button", { name: /Bitmap Index Scan/ })
    ).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(screen.getByRole("button", { name: /^Bitmap Heap Scan/ }));
    fireEvent.click(screen.getByRole("tab", { name: "Grid" }));
    expect(screen.getAllByTestId("plan-grid-row")[0]).toHaveAttribute(
      "aria-selected",
      "true"
    );
    expect(
      screen.getByRole("heading", { name: "Bitmap Heap Scan" })
    ).toBeInTheDocument();
  });

  test("answers where the cost goes on the summary tab", () => {
    renderViewer();

    fireEvent.click(screen.getByRole("tab", { name: "Summary" }));
    expect(screen.getByTestId("plan-diagram-viewport")).not.toBeVisible();
    expect(screen.getByTestId("plan-summary")).toBeInTheDocument();
    expect(
      screen
        .getAllByTestId("plan-timeline-node-type")
        .map((cell) => cell.textContent)
    ).toEqual(["Bitmap Heap Scan", "Bitmap Index Scan"]);
    // 34.81 of the plan's 39.18 cost units are the heap scan's own.
    expect(screen.getAllByTestId("plan-costliest-row")[0]).toHaveTextContent(
      "88.8%"
    );
  });

  test("shares the selection between the summary and the other tabs", () => {
    renderViewer();

    fireEvent.click(screen.getByRole("tab", { name: "Summary" }));
    const timelineRows = screen.getAllByTestId("plan-timeline-row");
    expect(timelineRows[0]).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(timelineRows[1]);
    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Diagram" }));
    expect(
      screen.getByRole("button", { name: /Bitmap Index Scan/ })
    ).toHaveAttribute("aria-pressed", "true");

    fireEvent.click(screen.getByRole("tab", { name: "Grid" }));
    fireEvent.click(screen.getAllByTestId("plan-grid-row")[0]);
    fireEvent.click(screen.getByRole("tab", { name: "Summary" }));
    expect(screen.getAllByTestId("plan-timeline-row")[0]).toHaveAttribute(
      "aria-pressed",
      "true"
    );
  });

  test("switches to the raw plan and back to the diagram", () => {
    renderViewer();

    fireEvent.click(screen.getByRole("tab", { name: "Raw plan" }));
    expect(screen.getByText(/"Node Type": "Bitmap Heap Scan"/)).toBeVisible();
    expect(screen.getByTestId("plan-diagram-viewport")).not.toBeVisible();

    fireEvent.click(screen.getByRole("tab", { name: "Diagram" }));
    expect(screen.getByTestId("plan-diagram-viewport")).toBeVisible();
  });

  test("shows the statement on the query tab", () => {
    renderViewer("SELECT * FROM orders WHERE customer_id = 42;");

    fireEvent.click(screen.getByRole("tab", { name: "Query" }));
    expect(
      screen.getByText("SELECT * FROM orders WHERE customer_id = 42;")
    ).toBeVisible();
  });

  test("offers a copy of whichever text a tab is showing", () => {
    renderViewer("SELECT * FROM orders WHERE customer_id = 42;");

    openTab("Raw plan");
    expect(screen.getByTestId("plan-copy-button")).toHaveTextContent(
      "Copy plan"
    );

    openTab("Query");
    expect(screen.getByTestId("plan-copy-button")).toHaveTextContent(
      "Copy query"
    );
  });

  test("offers no copy of a statement that was never captured", () => {
    renderViewer();

    openTab("Query");
    expect(screen.queryByTestId("plan-copy-button")).toBeNull();
  });

  test("explains an absent statement instead of rendering an empty tab", () => {
    renderViewer();

    fireEvent.click(screen.getByRole("tab", { name: "Query" }));
    expect(
      screen.getByText("The statement was not captured with this plan.")
    ).toBeVisible();
  });

  test("shades the diagram once a highlight mode is chosen", () => {
    renderViewer();
    expect(screen.queryAllByTestId("plan-node-tint")).toHaveLength(0);

    fireEvent.click(screen.getByRole("button", { name: "Cost" }));
    expect(screen.getAllByTestId("plan-node-tint").length).toBeGreaterThan(0);
  });

  test("offers every tab and highlight for a plan with cost and row estimates", () => {
    renderParsed(parseMssqlPlan, procedureTwoStatements);

    expect(tabNames()).toEqual([
      "Diagram",
      "Grid",
      "Summary",
      "Raw plan",
      "Query",
    ]);
    expect(
      within(screen.getByRole("group", { name: "Highlight nodes by" }))
        .getAllByRole("button")
        .map((option) => option.textContent)
    ).toEqual(["Off", "Cost", "Rows"]);
    expect(screen.getByText("10 nodes · estimated cost 1.64")).toBeVisible();
  });

  test("keeps to what a plan without estimates can show", () => {
    renderParsed(parseSpannerPlan, JSON.stringify(spannerHashJoin));

    // The summary is where the cost goes, and nothing can be shaded.
    expect(tabNames()).toEqual(["Diagram", "Grid", "Raw plan", "Query"]);
    expect(screen.queryByRole("group", { name: "Highlight nodes by" })).toBeNull();
    expect(screen.getByText("7 nodes")).toBeVisible();
    expect(screen.queryByText(/card's bar/)).toBeNull();
    expect(screen.getAllByTestId("plan-node-card")).toHaveLength(7);
  });

  test("indents an XML plan one element per line on the raw tab", () => {
    renderParsed(parseMssqlPlan, procedureTwoStatements);

    openTab("Raw plan");
    const raw = within(screen.getByRole("tabpanel")).getByText(
      /^<ShowPlanXML/
    ).textContent;
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

  test("stacks the detail pane below the diagram on a narrow screen", () => {
    mockMatchMedia(true);
    renderViewer();
    expect(screen.getByTestId("plan-split")).toHaveAttribute(
      "data-orientation",
      "vertical"
    );
  });

  test("names the selected node in the URL, which is what shares it", () => {
    renderViewer();

    fireEvent.click(screen.getByRole("button", { name: /Bitmap Index Scan/ }));
    expect(location.hash).toBe("#node-0.0");
  });

  test("leaves the URL alone until the reader picks a node", () => {
    renderViewer();

    // A fragment written on load would make the next reload look like a
    // followed deep link, which reveals its node over the view the page fits.
    expect(location.hash).toBe("");
  });

  test("opens on the node a fragment names", () => {
    startAtFragment("#node-0.0");
    renderViewer();

    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Bitmap Index Scan/ })
    ).toHaveAttribute("aria-pressed", "true");
  });

  test("falls back to the root when a fragment names no node in this plan", () => {
    startAtFragment("#node-9.9.9");
    renderViewer();

    expect(
      screen.getByRole("heading", { name: "Bitmap Heap Scan" })
    ).toBeInTheDocument();
    // The fragment is the reader's; a load neither follows nor rewrites it.
    expect(location.hash).toBe("#node-9.9.9");
  });

  test("ignores a fragment that is not about a node at all", () => {
    startAtFragment("#somewhere-else");
    renderViewer();

    expect(
      screen.getByRole("heading", { name: "Bitmap Heap Scan" })
    ).toBeInTheDocument();
  });

  test("follows a fragment that changes without a reload", () => {
    renderViewer();

    startAtFragment("#node-0.0");
    fireEvent(window, new Event("hashchange"));

    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();
  });

  test("keeps the diagram's zoom while the reader moves between tabs", () => {
    renderViewer();
    const canvas = () => screen.getByTestId("plan-diagram-canvas").style.transform;

    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    const zoomed = canvas();
    expect(zoomed).toContain("scale(1.25)");

    openTab("Grid");
    openTab("Diagram");

    expect(canvas()).toBe(zoomed);
  });

  test("keeps a subtree folded while the reader moves between tabs", () => {
    renderViewer();
    expect(cards()).toHaveLength(2);

    fireEvent.click(collapseToggle());
    expect(cards()).toHaveLength(1);

    openTab("Grid");
    openTab("Diagram");
    expect(cards()).toHaveLength(1);
    expect(collapseToggle()).toHaveAttribute("aria-expanded", "false");
  });

  test("moves the selection out of a subtree it folds away", () => {
    renderViewer();

    fireEvent.click(screen.getByRole("button", { name: /Bitmap Index Scan/ }));
    expect(
      screen.getByRole("heading", { name: "Bitmap Index Scan" })
    ).toBeInTheDocument();

    fireEvent.click(collapseToggle());

    // The selected node has no card any more, so the collapse that swallowed
    // it takes the selection — and the URL follows.
    expect(
      screen.getByRole("heading", { name: "Bitmap Heap Scan" })
    ).toBeInTheDocument();
    expect(location.hash).toBe("#node-0");
  });

  test("unfolds a subtree when a node inside it is selected elsewhere", () => {
    renderViewer();
    fireEvent.click(collapseToggle());

    openTab("Grid");
    // The grid still lists every node, folded or not.
    fireEvent.click(screen.getAllByTestId("plan-grid-row")[1]);

    openTab("Diagram");
    expect(cards()).toHaveLength(2);
    expect(
      screen.getByRole("button", { name: /Bitmap Index Scan/ })
    ).toHaveAttribute("aria-pressed", "true");
  });
});
