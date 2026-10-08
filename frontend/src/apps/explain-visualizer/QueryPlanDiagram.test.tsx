import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import noJoinPredicate from "./test-data/mssql/no-join-predicate.xml?raw";
import hashJoinAggregateSort from "./test-data/postgres/hash-join-aggregate-sort.json";
import seqScanFilter from "./test-data/postgres/seq-scan-filter.json";
import spannerHashJoin from "./test-data/spanner/hash-join.json";
import { parseMssqlPlan } from "./mssql-plan";
import { PLAN_READABLE_SCALE } from "./plan-layout";
import {
  PLAN_EDGE_MAX_WIDTH,
  PLAN_EDGE_MIN_WIDTH,
  type PlanParseResult,
  type PlanTree,
} from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import { QueryPlanDiagram } from "./QueryPlanDiagram";
import { parseSpannerPlan } from "./spanner-plan";

const treeOf = (result: PlanParseResult): PlanTree => {
  if (!result.ok) throw new Error(result.message);
  return result.tree;
};

const treeFrom = (fixture: unknown): PlanTree =>
  treeOf(parsePostgresPlan(JSON.stringify(fixture)));

const tree = (): PlanTree => treeFrom(hashJoinAggregateSort);

/**
 * The card whose accessible name matches, which is not simply the button whose
 * name does: a collapse toggle names the node it belongs to as well.
 */
const cardFor = (label: string) => {
  const pattern = new RegExp(label);
  const card = screen
    .getAllByTestId("plan-node-card")
    .find((entry) => pattern.test(entry.getAttribute("aria-label") ?? ""));
  if (!card) throw new Error(`no plan card named ${label}`);
  return card;
};

/** The collapse control belonging to a card, which sits beside it. */
const toggleFor = (label: string) => {
  const toggle = cardFor(label).parentElement?.querySelector(
    "[data-testid='plan-node-collapse']"
  );
  if (!toggle) throw new Error(`no collapse control on ${label}`);
  return toggle;
};

const canvasTransform = () =>
  screen.getByTestId("plan-diagram-canvas").style.transform;

const scaleNow = () =>
  Number(/scale\(([\d.]+)\)/.exec(canvasTransform())?.[1] ?? "0");

const renderDiagram = (
  overrides: Partial<Parameters<typeof QueryPlanDiagram>[0]> = {}
) => {
  const onSelect = vi.fn();
  const onToggleCollapse = vi.fn();
  const props = {
    tree: tree(),
    selectedId: "0",
    onSelect,
    highlight: "off" as const,
    collapsedIds: new Set<string>(),
    onToggleCollapse,
    ...overrides,
  };
  return {
    onSelect,
    onToggleCollapse,
    ...render(<QueryPlanDiagram {...props} />),
  };
};

/** Makes the viewport a real size, which jsdom otherwise reports as zero. */
const sizeViewport = (width: number, height: number) => {
  const rect = {
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    right: width,
    bottom: height,
    width,
    height,
    toJSON: () => "",
  };
  vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(
    rect as DOMRect
  );
};

afterEach(() => {
  vi.restoreAllMocks();
});

describe("QueryPlanDiagram", () => {
  test("renders one card per plan node with its estimates", () => {
    renderDiagram();

    expect(screen.getAllByRole("button")).toHaveLength(
      // six plan node cards, the four collapse toggles their parents carry,
      // and the three zoom controls
      6 + 4 + 3
    );
    expect(cardFor("Hash Join")).toHaveTextContent(
      "cost 1,090.86 · rows 50,000"
    );
    // The hash side of the join is the only node the planner marks "Inner".
    expect(cardFor("^Hash, Inner")).toHaveTextContent("cost 78 · rows 5,000");
  });

  test("draws an edge for every parent-child link", () => {
    renderDiagram();
    expect(
      screen.getByTestId("plan-diagram-edges").querySelectorAll("path")
    ).toHaveLength(5);
  });

  test("thickens an edge with the rows its child is estimated to return", () => {
    renderDiagram();
    const widthByRows = new Map(
      screen
        .getAllByTestId("plan-diagram-edge")
        .map((edge) => [
          Number(edge.getAttribute("data-plan-edge-rows")),
          Number(edge.getAttribute("stroke-width")),
        ])
    );

    // The plan's widest estimate saturates; the narrower ones rank below it and
    // stay above the floor that keeps a one-row edge visible.
    expect(widthByRows.get(50000)).toBe(PLAN_EDGE_MAX_WIDTH);
    const small = widthByRows.get(3) ?? 0;
    const middling = widthByRows.get(5000) ?? 0;
    expect(small).toBeGreaterThan(PLAN_EDGE_MIN_WIDTH);
    expect(small).toBeLessThan(middling);
    expect(middling).toBeLessThan(PLAN_EDGE_MAX_WIDTH);
  });

  test("names the row estimate an edge carries", () => {
    renderDiagram();
    const titles = screen
      .getAllByTestId("plan-diagram-edge")
      .map((edge) => edge.querySelector("title")?.textContent);

    expect(titles).toContain("50,000 estimated rows");
    expect(titles).toContain("3 estimated rows");
  });

  test("shows each node's share of the plan's cost on its card", () => {
    renderDiagram();
    const barOf = (label: string) =>
      cardFor(label).querySelector<HTMLElement>(
        "[data-testid='plan-cost-share-bar']"
      );

    // The scan of `orders` owns 819 of the plan's 1,465.93 cost units.
    expect(cardFor("^Seq Scan, orders")).toHaveTextContent("55.9%");
    expect(barOf("^Seq Scan, orders")?.style.width).toBe(
      `${(819 / 1465.93) * 100}%`
    );
    // A Hash adds nothing on top of the scan it hashes.
    expect(cardFor("^Hash, Inner")).toHaveTextContent("0%");
    expect(barOf("^Hash, Inner")?.style.width).toBe("0%");
  });

  test("marks a node the parser warns about", () => {
    renderDiagram();

    expect(screen.getAllByTestId("plan-node-warning")).toHaveLength(1);
    expect(
      cardFor("^Seq Scan, orders").querySelector(
        "[data-testid='plan-node-warning']"
      )
    ).not.toBeNull();
    expect(cardFor("^Seq Scan, orders")).toHaveAccessibleName(
      /Full table scan/
    );
    // `customers` is small enough that reading all of it costs nothing worth
    // acting on.
    expect(
      cardFor("^Seq Scan, customers").querySelector(
        "[data-testid='plan-node-warning']"
      )
    ).toBeNull();
  });

  test("names each of a node's warnings on its icon and its label", () => {
    renderDiagram({ tree: treeOf(parseMssqlPlan(noJoinPredicate)) });
    const join = cardFor("^Nested Loops \\(Inner Join\\)");

    expect(join).toHaveAccessibleName(/No join predicate/);
    expect(
      join.querySelector("[data-testid='plan-node-warning'] title")
    ).toHaveTextContent("No join predicate");
  });

  test("gives a lone node the whole plan and no warning", () => {
    renderDiagram({ tree: treeFrom(seqScanFilter) });

    expect(cardFor("^Seq Scan, customers")).toHaveTextContent("100%");
    expect(screen.queryAllByTestId("plan-node-warning")).toHaveLength(0);
    expect(screen.queryAllByTestId("plan-diagram-edge")).toHaveLength(0);
  });

  test("draws a plan without estimates as its operators alone", () => {
    renderDiagram({ tree: treeOf(parseSpannerPlan(JSON.stringify(spannerHashJoin))) });

    expect(screen.getAllByTestId("plan-node-card")).toHaveLength(7);
    // No cost or row line, no cost bar, and no share in the label.
    expect(cardFor("^Hash Join")).not.toHaveTextContent(/cost|rows/);
    expect(cardFor("^Hash Join")).not.toHaveAccessibleName(/of plan cost/);
    expect(screen.queryAllByTestId("plan-cost-share-bar")).toHaveLength(0);
    // Without row estimates every edge is the same hairline, with no count.
    for (const edge of screen.getAllByTestId("plan-diagram-edge")) {
      expect(Number(edge.getAttribute("stroke-width"))).toBe(
        PLAN_EDGE_MIN_WIDTH
      );
      expect(edge.querySelector("title")).toBeNull();
    }
    expect(cardFor("^Table Scan")).toHaveAccessibleName(/Full table scan/);
    expect(cardFor("^Index Scan")).toHaveAccessibleName(/Full index scan/);
  });

  test("gives the subject the lines a plan without estimates leaves free", () => {
    renderDiagram({ tree: treeOf(parseSpannerPlan(JSON.stringify(spannerHashJoin))) });

    expect(
      screen.getByText("(($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))")
    ).toHaveClass("line-clamp-3");
    renderDiagram();
    expect(screen.getByText("(o.customer_id = c.id)")).toHaveClass("truncate");
  });

  test("marks the selected card and reports clicks on the others", () => {
    const { onSelect } = renderDiagram();

    expect(cardFor("Sort")).toHaveAttribute("aria-pressed", "true");
    expect(cardFor("Hash Join")).toHaveAttribute("aria-pressed", "false");

    fireEvent.click(cardFor("Hash Join"));
    expect(onSelect).toHaveBeenCalledWith("0.0.0");
  });

  test("does not select a card when the click ends a pan", () => {
    const { onSelect } = renderDiagram();
    const viewport = screen.getByTestId("plan-diagram-viewport");

    fireEvent.pointerDown(viewport, { button: 0, clientX: 0, clientY: 0 });
    fireEvent.pointerMove(window, { clientX: 120, clientY: 40 });
    fireEvent.pointerUp(window);
    // detail 1: a real mouse click. A keyboard activation reports 0 and is
    // deliberately exempt from the drag guard.
    fireEvent.click(cardFor("Hash Join"), { detail: 1 });

    expect(onSelect).not.toHaveBeenCalled();
  });

  test("pans the canvas while the pointer is held down", () => {
    renderDiagram();
    const viewport = screen.getByTestId("plan-diagram-viewport");
    const before = canvasTransform();

    fireEvent.pointerDown(viewport, { button: 0, clientX: 10, clientY: 10 });
    fireEvent.pointerMove(window, { clientX: 90, clientY: 60 });

    expect(canvasTransform()).not.toBe(before);
    expect(canvasTransform()).toContain("translate(80px, 50px)");
  });

  test("zooms in and back out from the controls", () => {
    renderDiagram();

    expect(canvasTransform()).toContain("scale(1)");
    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    expect(canvasTransform()).toContain("scale(1.25)");
    fireEvent.click(screen.getByRole("button", { name: "Zoom out" }));
    expect(canvasTransform()).toContain("scale(1)");
  });

  test("zooms the same amount whether a notch arrives in pixels or lines", () => {
    sizeViewport(600, 400);
    renderDiagram();
    const viewport = screen.getByTestId("plan-diagram-viewport");
    fireEvent.click(screen.getByRole("button", { name: "Fit to view" }));
    const fitted = scaleNow();

    // Chrome sends one notch as pixels; Firefox sends the same notch as lines.
    fireEvent.wheel(viewport, { deltaY: 120, deltaMode: 0 });
    const byPixels = scaleNow();
    expect(byPixels).toBeLessThan(fitted);

    fireEvent.click(screen.getByRole("button", { name: "Fit to view" }));
    fireEvent.wheel(viewport, { deltaY: 3, deltaMode: 1 });

    expect(scaleNow()).toBeCloseTo(byPixels, 10);
  });

  test("keeps a view the reader placed when a collapse re-lays out the plan", () => {
    sizeViewport(600, 400);
    const plan = tree();
    const { rerender } = renderDiagram({ tree: plan });
    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    const placed = canvasTransform();

    rerender(
      <QueryPlanDiagram
        tree={plan}
        selectedId="0"
        onSelect={() => undefined}
        highlight="off"
        collapsedIds={new Set(["0.0.0"])}
        onToggleCollapse={() => undefined}
      />
    );

    expect(screen.getAllByTestId("plan-node-card")).toHaveLength(3);
    expect(canvasTransform()).toBe(placed);
  });

  test("places the view again for a plan the reader has not moved when it re-lays out", () => {
    sizeViewport(600, 400);
    const plan = tree();
    const { rerender } = renderDiagram({ tree: plan });
    const opened = canvasTransform();

    rerender(
      <QueryPlanDiagram
        tree={plan}
        selectedId="0"
        onSelect={() => undefined}
        highlight="off"
        collapsedIds={new Set(["0.0.0"])}
        onToggleCollapse={() => undefined}
      />
    );

    // What is left is small enough to read whole, so the view changes to show it.
    expect(canvasTransform()).not.toBe(opened);
    expect(scaleNow()).toBeGreaterThan(PLAN_READABLE_SCALE);
    expect(screen.getAllByTestId("plan-node-card")).toHaveLength(3);
  });

  test("opens a plan too deep to read whole at a readable scale, from its root down", () => {
    sizeViewport(600, 400);
    renderDiagram();

    // Shown whole, this plan would open at about half size.
    expect(scaleNow()).toBe(PLAN_READABLE_SCALE);
    expect(canvasTransform()).toMatch(/translate\([\d.]+px, 0px\)/);
    expect(screen.getByTestId("plan-mini-map")).toBeInTheDocument();
  });

  test("opens a plan small enough to read whole by showing all of it", () => {
    sizeViewport(1200, 900);
    renderDiagram();

    expect(scaleNow()).toBe(1);
    expect(screen.queryByTestId("plan-mini-map")).toBeNull();
  });

  test("keeps showing the whole plan after Fit to view when it re-lays out", () => {
    sizeViewport(600, 400);
    const plan = tree();
    const { rerender } = renderDiagram({ tree: plan });
    fireEvent.click(screen.getByRole("button", { name: "Fit to view" }));

    // Folding the hash's scan away leaves a plan still too deep to read whole.
    rerender(
      <QueryPlanDiagram
        tree={plan}
        selectedId="0"
        onSelect={() => undefined}
        highlight="off"
        collapsedIds={new Set(["0.0.0.1"])}
        onToggleCollapse={() => undefined}
      />
    );

    expect(scaleNow()).toBeLessThan(PLAN_READABLE_SCALE);
    expect(screen.queryByTestId("plan-mini-map")).toBeNull();
  });

  test("tints cards only while a highlight mode is on", () => {
    const { rerender } = renderDiagram();
    expect(screen.queryAllByTestId("plan-node-tint")).toHaveLength(0);

    rerender(
      <QueryPlanDiagram
        tree={tree()}
        selectedId="0"
        onSelect={() => undefined}
        highlight="cost"
        collapsedIds={new Set()}
        onToggleCollapse={() => undefined}
      />
    );
    const tintOf = (label: string) =>
      cardFor(label).querySelector("[data-testid='plan-node-tint']");
    // The sequential scan of `orders` owns the plan's largest exclusive cost,
    // so it saturates and every other node shades below it.
    expect(tintOf("^Seq Scan, orders")).toHaveClass("bg-warning");
    expect(tintOf("^Seq Scan, orders")).toHaveStyle({ opacity: "0.45" });
    expect(
      Number(
        (tintOf("Hash Join") as HTMLElement).style.opacity
      )
    ).toBeLessThan(0.45);

    rerender(
      <QueryPlanDiagram
        tree={tree()}
        selectedId="0"
        onSelect={() => undefined}
        highlight="rows"
        collapsedIds={new Set()}
        onToggleCollapse={() => undefined}
      />
    );
    expect(
      cardFor("Hash Join").querySelector("[data-testid='plan-node-tint']")
    ).toHaveClass("bg-info");
  });

  test("offers a collapse only on the nodes that have something below", () => {
    renderDiagram();

    expect(screen.getAllByTestId("plan-node-collapse")).toHaveLength(4);
    expect(
      cardFor("^Seq Scan, orders").parentElement?.querySelector(
        "[data-testid='plan-node-collapse']"
      )
    ).toBeNull();
    expect(toggleFor("Hash Join")).toHaveAccessibleName(
      "Collapse 3 nodes under Hash Join"
    );
  });

  test("reports the node whose subtree should fold away", () => {
    const { onToggleCollapse, onSelect } = renderDiagram();

    fireEvent.click(toggleFor("Hash Join"));

    expect(onToggleCollapse).toHaveBeenCalledWith("0.0.0");
    // The control belongs to the card without being part of it, so using it
    // does not also move the selection.
    expect(onSelect).not.toHaveBeenCalled();
  });

  test("drops a collapsed node's descendants and says how many", () => {
    renderDiagram({ collapsedIds: new Set(["0.0.0"]) });

    expect(screen.getAllByTestId("plan-node-card")).toHaveLength(3);
    expect(screen.queryAllByTestId("plan-diagram-edge")).toHaveLength(2);
    expect(toggleFor("Hash Join")).toHaveTextContent("3");
    expect(toggleFor("Hash Join")).toHaveAccessibleName(
      "Expand 3 nodes under Hash Join"
    );
    expect(toggleFor("Hash Join")).toHaveAttribute("aria-expanded", "false");
    // Nothing left the diagram quietly: the card says what it is holding.
    expect(cardFor("Hash Join")).toHaveAccessibleName(/3 nodes hidden/);
  });

  test("brings a node the caller asks for into the middle of the view", () => {
    sizeViewport(600, 400);
    const onRevealed = vi.fn();
    render(
      <QueryPlanDiagram
        tree={tree()}
        selectedId="0"
        onSelect={() => undefined}
        highlight="off"
        collapsedIds={new Set()}
        onToggleCollapse={() => undefined}
        revealId="0.0.0.0"
        onRevealed={onRevealed}
      />
    );

    const scale = Number(
      /scale\(([\d.]+)\)/.exec(canvasTransform())?.[1] ?? "0"
    );
    const [, x, y] = /translate\((-?[\d.]+)px, (-?[\d.]+)px\)/
      .exec(canvasTransform())
      ?.map(Number) ?? [0, 0, 0];
    const card = cardFor("^Seq Scan, orders");
    const wrapper = card.parentElement as HTMLElement;
    // The card's middle, in content space, lands on the viewport's middle.
    const centerX = Number.parseFloat(wrapper.style.left) + 216 / 2;
    const centerY = Number.parseFloat(wrapper.style.top) + 96 / 2;

    expect(x + centerX * scale).toBeCloseTo(300, 4);
    expect(y + centerY * scale).toBeCloseTo(200, 4);
    expect(onRevealed).toHaveBeenCalled();
  });

  test("ignores a reveal for a node the plan does not have", () => {
    sizeViewport(600, 400);
    const onRevealed = vi.fn();
    const { rerender } = renderDiagram();
    const before = canvasTransform();

    rerender(
      <QueryPlanDiagram
        tree={tree()}
        selectedId="0"
        onSelect={() => undefined}
        highlight="off"
        collapsedIds={new Set()}
        onToggleCollapse={() => undefined}
        revealId="nope"
        onRevealed={onRevealed}
      />
    );

    expect(canvasTransform()).toBe(before);
    // The caller still hears back, so a bad fragment does not leave a reveal
    // pending forever.
    expect(onRevealed).toHaveBeenCalled();
  });

  test("shows a mini-map only while part of the plan is off screen", () => {
    sizeViewport(600, 400);
    renderDiagram();
    fireEvent.click(screen.getByRole("button", { name: "Fit to view" }));
    expect(screen.queryByTestId("plan-mini-map")).toBeNull();

    // Zooming in pushes the plan past the edges of the viewport.
    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    expect(screen.getByTestId("plan-mini-map")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Fit to view" }));
    expect(screen.queryByTestId("plan-mini-map")).toBeNull();
  });

  test("moves the view when the mini-map is clicked", () => {
    sizeViewport(600, 400);
    renderDiagram();
    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    fireEvent.click(screen.getByRole("button", { name: "Zoom in" }));
    const before = canvasTransform();

    fireEvent.pointerDown(screen.getByTestId("plan-mini-map"), {
      button: 0,
      clientX: 120,
      clientY: 60,
    });

    expect(canvasTransform()).not.toBe(before);
  });

  test("leaves the mini-map out of an unmeasured viewport", () => {
    renderDiagram();
    expect(screen.queryByTestId("plan-mini-map")).toBeNull();
  });

  test("leaves the mini-map out of a viewport too small to spare room for it", () => {
    sizeViewport(400, 300);
    renderDiagram();

    // The plan opens partly off screen, which in a roomier viewport would
    // bring the mini-map up over it.
    expect(scaleNow()).toBe(PLAN_READABLE_SCALE);
    expect(screen.queryByTestId("plan-mini-map")).toBeNull();
  });

  test("still selects a node from the keyboard after a drag", () => {
    const onSelect = vi.fn();
    renderDiagram({ onSelect });
    const viewport = screen.getByTestId("plan-diagram-viewport");

    // Drag the canvas: the click that ends this must not select.
    fireEvent.pointerDown(viewport, { button: 0, clientX: 0, clientY: 0 });
    fireEvent.pointerMove(window, { clientX: 120, clientY: 60 });
    fireEvent.pointerUp(window);

    const card = screen.getAllByRole("button", { name: /Seq Scan/ })[0];
    // A pointer click still loses to the guard...
    fireEvent.click(card, { detail: 1 });
    expect(onSelect).not.toHaveBeenCalled();
    // ...but Enter or Space, which arrive with detail 0, must not.
    fireEvent.click(card, { detail: 0 });
    expect(onSelect).toHaveBeenCalled();
  });
});
