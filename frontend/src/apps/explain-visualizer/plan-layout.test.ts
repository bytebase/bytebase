import { describe, expect, test } from "vitest";
import {
  layoutPlan,
  PLAN_NODE_HEIGHT,
  PLAN_NODE_WIDTH,
  planFitsViewport,
  planGuideLevels,
  planIndentPixels,
  planMiniMap,
  planNodeCenter,
  planNodeOnScreen,
  planViewportCenteredOn,
} from "./plan-layout";
import type { PlanNode, PlanTree } from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import hashJoinAggregateSort from "./test-data/hash-join-aggregate-sort.json";
import seqScanFilter from "./test-data/seq-scan-filter.json";

const parseFixture = (fixture: unknown): PlanTree => {
  const result = parsePostgresPlan(JSON.stringify(fixture));
  if (!result.ok) throw new Error(result.message);
  return result.tree;
};

const depthOf = (root: PlanNode, id: string): number => {
  const walk = (node: PlanNode, depth: number): number | undefined => {
    if (node.id === id) return depth;
    for (const child of node.children) {
      const found = walk(child, depth + 1);
      if (found !== undefined) return found;
    }
    return undefined;
  };
  const depth = walk(root, 0);
  if (depth === undefined) throw new Error(`no node ${id}`);
  return depth;
};

describe("layoutPlan", () => {
  test("places a lone node inside a card-sized content box", () => {
    const { root } = parseFixture(seqScanFilter);
    const layout = layoutPlan(root);

    expect(layout.nodes).toHaveLength(1);
    expect(layout.edges).toHaveLength(0);
    expect(layout.nodes[0]).toMatchObject({ x: 32, y: 32 });
    expect(layout.width).toBe(PLAN_NODE_WIDTH + 64);
    expect(layout.height).toBe(PLAN_NODE_HEIGHT + 64);
  });

  test("stacks every node of a plan on one row per depth", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const layout = layoutPlan(tree.root);

    expect(layout.nodes).toHaveLength(tree.nodes.length);
    const rowByDepth = new Map<number, number>();
    for (const placed of layout.nodes) {
      const depth = depthOf(tree.root, placed.node.id);
      const existing = rowByDepth.get(depth);
      if (existing === undefined) {
        rowByDepth.set(depth, placed.y);
      } else {
        expect(placed.y).toBe(existing);
      }
    }
    expect(rowByDepth.size).toBe(5);
  });

  test("keeps every card inside the reported content box", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const layout = layoutPlan(tree.root);

    for (const placed of layout.nodes) {
      expect(placed.x).toBeGreaterThanOrEqual(0);
      expect(placed.y).toBeGreaterThanOrEqual(0);
      expect(placed.x + PLAN_NODE_WIDTH).toBeLessThanOrEqual(layout.width);
      expect(placed.y + PLAN_NODE_HEIGHT).toBeLessThanOrEqual(layout.height);
    }
  });

  test("draws one edge per parent-child link, top-down", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const layout = layoutPlan(tree.root);
    const byId = new Map(
      layout.nodes.map((placed) => [placed.node.id, placed])
    );

    expect(layout.edges.map((edge) => edge.id)).toEqual([
      "0->0.0",
      "0.0->0.0.0",
      "0.0.0->0.0.0.0",
      "0.0.0->0.0.0.1",
      "0.0.0.1->0.0.0.1.0",
    ]);

    const parent = byId.get("0.0.0");
    const child = byId.get("0.0.0.1");
    if (!parent || !child) throw new Error("missing laid-out node");
    const edge = layout.edges.find((entry) => entry.id === "0.0.0->0.0.0.1");
    expect(edge?.path).toBe(
      `M${parent.x + PLAN_NODE_WIDTH / 2},${parent.y + PLAN_NODE_HEIGHT} ` +
        `C${parent.x + PLAN_NODE_WIDTH / 2},${(parent.y + PLAN_NODE_HEIGHT + child.y) / 2} ` +
        `${child.x + PLAN_NODE_WIDTH / 2},${(parent.y + PLAN_NODE_HEIGHT + child.y) / 2} ` +
        `${child.x + PLAN_NODE_WIDTH / 2},${child.y}`
    );
  });

  test("names the child each edge feeds into", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const layout = layoutPlan(tree.root);

    expect(
      layout.edges.map((edge) => [edge.target.id, edge.target.rows])
    ).toEqual([
      ["0.0", 3],
      ["0.0.0", 50000],
      ["0.0.0.0", 50000],
      ["0.0.0.1", 5000],
      ["0.0.0.1.0", 5000],
    ]);
  });

  test("does not overlap siblings horizontally", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const layout = layoutPlan(tree.root);
    const byId = new Map(
      layout.nodes.map((placed) => [placed.node.id, placed])
    );

    const left = byId.get("0.0.0.0");
    const right = byId.get("0.0.0.1");
    if (!left || !right) throw new Error("missing laid-out node");
    expect(right.x - left.x).toBeGreaterThanOrEqual(PLAN_NODE_WIDTH);
  });
});

describe("layoutPlan with collapsed subtrees", () => {
  test("leaves out a collapsed node's descendants and their edges", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const layout = layoutPlan(tree.root, new Set(["0.0.0"]));

    // The join keeps its card; the two scans and the hash below it go.
    expect(layout.nodes.map((placed) => placed.node.id)).toEqual([
      "0",
      "0.0",
      "0.0.0",
    ]);
    expect(layout.edges.map((edge) => edge.id)).toEqual([
      "0->0.0",
      "0.0->0.0.0",
    ]);
  });

  test("closes the content box up around what is left", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const open = layoutPlan(tree.root);
    const folded = layoutPlan(tree.root, new Set(["0.0.0"]));

    expect(folded.height).toBeLessThan(open.height);
    expect(folded.width).toBeLessThan(open.width);
    expect(folded.width).toBe(PLAN_NODE_WIDTH + 64);
  });

  test("collapsing a leaf changes nothing", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    expect(layoutPlan(tree.root, new Set(["0.0.0.0"]))).toEqual(
      layoutPlan(tree.root)
    );
  });

  test("collapsing the root leaves the root alone on the canvas", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const layout = layoutPlan(tree.root, new Set(["0"]));

    expect(layout.nodes.map((placed) => placed.node.id)).toEqual(["0"]);
    expect(layout.edges).toHaveLength(0);
  });
});

describe("planIndentPixels", () => {
  test("steps with depth and then stops, so deep rows stay readable", () => {
    expect([0, 1, 2, 6, 7, 30].map(planIndentPixels)).toEqual([
      0, 12, 24, 72, 72, 72,
    ]);
  });
});

describe("planGuideLevels", () => {
  test("draws one level per indent step the row actually has", () => {
    expect(planGuideLevels([])).toEqual([]);
    expect(planGuideLevels([true, false])).toEqual([true, false]);
  });

  test("keeps the levels nearest the row once the indent stops growing", () => {
    const deep = [true, true, false, true, false, true, false, true];
    // Eight levels of nesting, six columns of indent: the six closest to the
    // row survive, so the connector into its own parent is always drawn.
    expect(planGuideLevels(deep)).toEqual(deep.slice(2));
    expect(planGuideLevels(deep)).toHaveLength(planIndentPixels(8) / 12);
  });
});

const VIEW = { scale: 1, x: 0, y: 0 };

describe("planNodeCenter", () => {
  test("is the middle of the card, not its corner", () => {
    expect(planNodeCenter({ node: {} as PlanNode, x: 100, y: 40 })).toEqual({
      x: 100 + PLAN_NODE_WIDTH / 2,
      y: 40 + PLAN_NODE_HEIGHT / 2,
    });
  });
});

describe("planViewportCenteredOn", () => {
  test("puts the point in the middle of the screen at the current zoom", () => {
    const size = { width: 800, height: 600 };

    expect(planViewportCenteredOn({ x: 500, y: 300 }, size, VIEW)).toEqual({
      scale: 1,
      x: 400 - 500,
      y: 300 - 300,
    });
    expect(
      planViewportCenteredOn({ x: 500, y: 300 }, size, { ...VIEW, scale: 0.5 })
    ).toEqual({ scale: 0.5, x: 400 - 250, y: 300 - 150 });
  });
});

describe("planFitsViewport", () => {
  test("is true only while the whole plan is on screen", () => {
    const layout = layoutPlan(parseFixture(hashJoinAggregateSort).root);
    const big = { width: layout.width, height: layout.height };

    expect(planFitsViewport(layout, big, VIEW)).toBe(true);
    expect(
      planFitsViewport(layout, { ...big, width: big.width - 1 }, VIEW)
    ).toBe(false);
    // Zooming out brings the overflow back inside.
    expect(
      planFitsViewport(
        layout,
        { ...big, width: big.width / 2 },
        {
          ...VIEW,
          scale: 0.5,
        }
      )
    ).toBe(true);
  });

  test("treats an unmeasured viewport as having nothing to navigate", () => {
    const layout = layoutPlan(parseFixture(hashJoinAggregateSort).root);
    expect(planFitsViewport(layout, { width: 0, height: 0 }, VIEW)).toBe(true);
  });
});

describe("planNodeOnScreen", () => {
  const placed = { node: {} as PlanNode, x: 300, y: 200 };
  const size = { width: 400, height: 300 };

  test("is true only when the whole card is inside the viewport", () => {
    expect(planNodeOnScreen(placed, size, { scale: 1, x: -250, y: -180 })).toBe(
      true
    );
    // Panned so the card starts past the right edge.
    expect(planNodeOnScreen(placed, size, VIEW)).toBe(false);
    // Panned so the card starts above the top edge.
    expect(planNodeOnScreen(placed, size, { scale: 1, x: -250, y: -250 })).toBe(
      false
    );
  });

  test("moves nothing before the viewport has been measured", () => {
    expect(planNodeOnScreen(placed, { width: 0, height: 0 }, VIEW)).toBe(true);
  });
});

describe("planMiniMap", () => {
  const layout = { nodes: [], edges: [], width: 800, height: 400 };
  const box = { width: 160, height: 112 };

  test("draws the whole plan to fit the box, keeping its proportions", () => {
    const map = planMiniMap(layout, { width: 400, height: 300 }, VIEW, box);

    // 160/800 is tighter than 112/400, so the width is what binds.
    expect(map.scale).toBe(0.2);
    expect(map.width).toBe(160);
    expect(map.height).toBe(80);
  });

  test("never blows a small plan up past its own size", () => {
    const small = { ...layout, width: 80, height: 40 };
    expect(
      planMiniMap(small, { width: 400, height: 300 }, VIEW, box).scale
    ).toBe(1);
  });

  test("marks the screen's share of the plan", () => {
    // A 400x300 window over an 800x400 plan, panned to the middle of it.
    const map = planMiniMap(
      layout,
      { width: 400, height: 300 },
      { scale: 1, x: -200, y: -50 },
      box
    );

    expect(map.viewport).toEqual({ x: 40, y: 10, width: 80, height: 60 });
  });

  test("shrinks the marker as the reader zooms in", () => {
    const zoomed = planMiniMap(
      layout,
      { width: 400, height: 300 },
      { scale: 2, x: 0, y: 0 },
      box
    );

    // At 2x the same window covers half as much plan.
    expect(zoomed.viewport).toEqual({ x: 0, y: 0, width: 40, height: 30 });
  });

  test("trims the marker to the plan rather than past its edges", () => {
    // Panned so the plan sits well to the right of the window, and so far the
    // other way that the window has left the plan entirely.
    const before = planMiniMap(layout, { width: 400, height: 300 }, VIEW, box);
    expect(before.viewport).toMatchObject({ x: 0, width: 80 });

    const past = planMiniMap(
      layout,
      { width: 400, height: 300 },
      { scale: 1, x: -1000, y: 0 },
      box
    );
    expect(past.viewport).toMatchObject({ x: 160, width: 0 });
  });

  test("reports only the part of an over-panned window that is on the plan", () => {
    // Panned 300px past the plan's left edge, so only 100 of the window's 400
    // pixels have any plan under them.
    const map = planMiniMap(
      { ...layout, width: 1000, height: 500 },
      { width: 400, height: 300 },
      { scale: 1, x: 300, y: 0 },
      box
    );

    expect(map.viewport).toMatchObject({ x: 0, width: 16 });
  });

  test("has nothing to draw for an empty or unscaled plan", () => {
    expect(planMiniMap({ ...layout, width: 0 }, box, VIEW, box).scale).toBe(0);
    expect(planMiniMap(layout, box, { scale: 0, x: 0, y: 0 }, box).scale).toBe(
      0
    );
  });
});
