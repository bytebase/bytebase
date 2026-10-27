import { describe, expect, test } from "vitest";
import {
  addPlanProperty,
  buildPlanTree,
  checkPlanDepth,
  findPlanNode,
  flattenPlan,
  formatPlanCost,
  formatPlanCount,
  formatPlanShare,
  isJsonRecord,
  PLAN_EDGE_MAX_WIDTH,
  PLAN_EDGE_MIN_WIDTH,
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  PLAN_TOO_MANY_NODES_MESSAGE,
  PLAN_WARNING_RELEVANT_SHARE,
  type PlanNode,
  type PlanProperty,
  type PlanTree,
  type PlanWarning,
  parseWithDepthLimit,
  planCollapsedAncestor,
  planCostliestNodes,
  planDescendantCount,
  planEdgeWidth,
  planHasSelfMetric,
  planHighlightIntensity,
  planRankMetric,
  planRevealNode,
  planRowEstimateFactor,
  planRows,
  planSelfCost,
  planSelfCostShare,
  planSelfShare,
  planSelfTime,
  planWarningGroups,
  planWarningIdentity,
} from "./plan-model";
import { parsePostgresPlan } from "./postgres-plan";
import cteNestedLoopInitplan from "./test-data/postgres/cte-nested-loop-initplan.json";
import hashJoinAggregateSort from "./test-data/postgres/hash-join-aggregate-sort.json";
import seqScanFilter from "./test-data/postgres/seq-scan-filter.json";

const parseFixture = (fixture: unknown): PlanTree => {
  const result = parsePostgresPlan(JSON.stringify(fixture));
  if (!result.ok) throw new Error(result.message);
  return result.tree;
};

const node = (
  id: string,
  overrides: Partial<PlanNode> = {},
  children: PlanNode[] = []
): PlanNode => ({
  id,
  nodeType: "Seq Scan",
  startupCost: 0,
  totalCost: 0,
  selfCost: 0,
  rows: 0,
  width: 0,
  properties: [],
  warnings: [],
  children,
  ...overrides,
});

/** A node the way an engine that reports no estimates describes one. */
const bareNode = (id: string, children: PlanNode[] = []): PlanNode => ({
  id,
  nodeType: "Scan",
  properties: [],
  warnings: [],
  children,
});

const sampleTree = () =>
  buildPlanTree(
    node("0", { selfCost: 10, rows: 100 }, [
      node("0.0", { selfCost: 40, rows: 400 }),
      node("0.1", { selfCost: 0, rows: 0 }, [
        node("0.1.0", { selfCost: 25, rows: 900 }),
      ]),
    ])
  );

describe("flattenPlan", () => {
  test("returns nodes depth-first with the root first", () => {
    const tree = sampleTree();
    expect(flattenPlan(tree.root).map((entry) => entry.id)).toEqual([
      "0",
      "0.0",
      "0.1",
      "0.1.0",
    ]);
  });
});

describe("buildPlanTree", () => {
  test("records the largest self cost and row estimate", () => {
    const tree = sampleTree();
    expect(tree.nodes).toHaveLength(4);
    expect(tree.maxSelfCost).toBe(40);
    expect(tree.maxRows).toBe(900);
  });

  test("says which estimates the plan carries", () => {
    expect(sampleTree().estimates).toEqual({
      cost: true,
      startupCost: true,
      rows: true,
      width: true,
    });
    expect(buildPlanTree(bareNode("0", [bareNode("0.0")])).estimates).toEqual({
      cost: false,
      startupCost: false,
      rows: false,
      width: false,
    });
    // One node reporting an estimate is enough for the plan to carry it.
    const mixed = buildPlanTree(
      bareNode("0", [{ ...bareNode("0.0"), rows: 5 }])
    );
    expect(mixed.estimates).toEqual({
      cost: false,
      startupCost: false,
      rows: true,
      width: false,
    });
    expect(mixed.maxRows).toBe(5);
  });

  test("ranks and bases nothing on a plan without estimates", () => {
    const tree = buildPlanTree(bareNode("0", [bareNode("0.0")]));

    expect(tree.maxSelfCost).toBe(0);
    expect(tree.maxRows).toBe(0);
    expect(tree.costBasis).toBe(0);
    for (const entry of tree.nodes) {
      expect(planSelfCostShare(entry, tree)).toBe(0);
      expect(planHighlightIntensity(entry, tree, "cost")).toBe(0);
      expect(planHighlightIntensity(entry, tree, "rows")).toBe(0);
    }
    expect(planCostliestNodes(tree)).toEqual([]);
  });
});

describe("parseWithDepthLimit", () => {
  test("builds the tree a parser returns", () => {
    const result = parseWithDepthLimit(() => bareNode("0", [bareNode("0.0")]));
    expect(result.ok && result.tree.nodes.map((entry) => entry.id)).toEqual([
      "0",
      "0.0",
    ]);
  });

  test("turns a tree with more operators than the cap into the node-cap message", () => {
    const children = Array.from({ length: 1001 }, (_, i) => bareNode(`0.${i}`));
    const result = parseWithDepthLimit(() => bareNode("0", children));
    expect(result).toEqual({ ok: false, message: PLAN_TOO_MANY_NODES_MESSAGE });
  });

  test("turns a build that went too deep into the depth message", () => {
    const chain = (depth: number): PlanNode => {
      checkPlanDepth(depth);
      return bareNode(String(depth), [chain(depth + 1)]);
    };
    expect(parseWithDepthLimit(() => chain(0))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });

  test("lets any other error through", () => {
    expect(() =>
      parseWithDepthLimit(() => {
        throw new Error("boom");
      })
    ).toThrow("boom");
  });
});

describe("isJsonRecord", () => {
  test("is true only for a plain object", () => {
    expect(isJsonRecord({ Plan: {} })).toBe(true);
    expect(isJsonRecord([])).toBe(false);
    expect(isJsonRecord(null)).toBe(false);
    expect(isJsonRecord("Plan")).toBe(false);
  });
});

describe("addPlanProperty", () => {
  test("appends a new label and folds a repeated one onto its own line", () => {
    const properties: PlanProperty[] = [];
    addPlanProperty(properties, "Agg", "COUNT() AS $v1");
    addPlanProperty(properties, "Condition", "($a = $b)");
    addPlanProperty(properties, "Agg", "ANY() AS $v2");

    expect(properties).toEqual([
      { label: "Agg", value: "COUNT() AS $v1\nANY() AS $v2" },
      { label: "Condition", value: "($a = $b)" },
    ]);
  });

  test("skips an empty value and a value the label already holds", () => {
    const properties: PlanProperty[] = [];
    addPlanProperty(properties, "Storage", "RowStore");
    addPlanProperty(properties, "Storage", "RowStore");
    addPlanProperty(properties, "Predicate", "");

    expect(properties).toEqual([{ label: "Storage", value: "RowStore" }]);
  });
});

describe("planSelfCost", () => {
  test("subtracts the children's total cost from the node's", () => {
    expect(
      planSelfCost(100, [
        node("0.0", { totalCost: 30 }),
        node("0.1", { totalCost: 45 }),
      ])
    ).toBe(25);
  });

  test("clamps at zero when the children report more than the node", () => {
    expect(planSelfCost(10, [node("0.0", { totalCost: 40 })])).toBe(0);
  });

  test("counts a child without a cost as costing nothing", () => {
    expect(planSelfCost(10, [bareNode("0.0")])).toBe(10);
  });

  test("has no self cost for a node without a total cost", () => {
    expect(planSelfCost(undefined, [node("0.0", { totalCost: 5 })])).toBe(
      undefined
    );
  });
});

describe("findPlanNode", () => {
  test("finds a node by id and tolerates no selection", () => {
    const tree = sampleTree();
    expect(findPlanNode(tree, "0.1.0")?.selfCost).toBe(25);
    expect(findPlanNode(tree, "nope")).toBeUndefined();
    expect(findPlanNode(tree, undefined)).toBeUndefined();
  });
});

describe("planHighlightIntensity", () => {
  test("is zero for every node when highlighting is off", () => {
    const tree = sampleTree();
    for (const entry of tree.nodes) {
      expect(planHighlightIntensity(entry, tree, "off")).toBe(0);
    }
  });

  test("saturates the costliest node and ranks the rest below it", () => {
    const tree = sampleTree();
    const [root, first, , deepest] = tree.nodes;

    expect(planHighlightIntensity(first, tree, "cost")).toBe(1);
    expect(planHighlightIntensity(deepest, tree, "cost")).toBeCloseTo(
      Math.sqrt(25 / 40),
      5
    );
    expect(planHighlightIntensity(root, tree, "cost")).toBeCloseTo(
      Math.sqrt(10 / 40),
      5
    );
  });

  test("ranks by rows independently of cost", () => {
    const tree = sampleTree();
    const [, first, , deepest] = tree.nodes;

    expect(planHighlightIntensity(deepest, tree, "rows")).toBe(1);
    expect(planHighlightIntensity(first, tree, "rows")).toBeCloseTo(
      Math.sqrt(400 / 900),
      5
    );
  });

  test("stays at zero when the plan has nothing to rank", () => {
    const flat = buildPlanTree(node("0"));
    expect(planHighlightIntensity(flat.root, flat, "cost")).toBe(0);
    expect(planHighlightIntensity(flat.root, flat, "rows")).toBe(0);
  });
});

describe("number formatting", () => {
  test("keeps planner precision on costs and groups large row counts", () => {
    expect(formatPlanCost(0)).toBe("0");
    expect(formatPlanCost(1465.93)).toBe("1,465.93");
    expect(formatPlanCount(50000)).toBe("50,000");
  });

  test("keeps a fractional row estimate from rounding to a whole row", () => {
    expect(formatPlanCount(1.56767)).toBe("1.57");
    expect(formatPlanCount(16660.5)).toBe("16,660.5");
  });

  test("keeps a fractional cost's leading digits rather than rounding it away", () => {
    expect(formatPlanCost(0.68)).toBe("0.68");
    expect(formatPlanCost(0.0325135)).toBe("0.0325");
    expect(formatPlanCost(0.0000418)).toBe("0.0000418");
    expect(formatPlanCost(1.60801)).toBe("1.61");
  });

  test("prints a dash for an estimate the engine did not report", () => {
    expect(formatPlanCost(undefined)).toBe("—");
    expect(formatPlanCount(undefined)).toBe("—");
  });

  test("keeps a share that rounds to nothing from reading as nothing", () => {
    expect(formatPlanShare(0)).toBe("0%");
    expect(formatPlanShare(1)).toBe("100%");
    expect(formatPlanShare(819 / 1465.93)).toBe("55.9%");
    expect(formatPlanShare(0.001)).toBe("0.1%");
    expect(formatPlanShare(0.0004)).toBe("<0.1%");
  });
});

describe("planRows", () => {
  test("pairs every node with how deep the planner nested it", () => {
    const tree = parseFixture(cteNestedLoopInitplan);
    expect(planRows(tree.root).map((row) => [row.node.id, row.depth])).toEqual([
      ["0", 0],
      ["0.0", 1],
      ["0.0.0", 2],
      ["0.1", 1],
      ["0.1.0", 2],
      ["0.1.1", 2],
    ]);
  });

  test("says which branches carry on past each row", () => {
    // The Limit has two children, so the first one's branch continues while
    // the second one's ends; the leaf under the ending branch inherits that.
    const tree = parseFixture(cteNestedLoopInitplan);
    expect(
      planRows(tree.root).map((row) => [row.node.id, row.branchContinues])
    ).toEqual([
      ["0", []],
      ["0.0", [true]],
      ["0.0.0", [true, false]],
      ["0.1", [false]],
      ["0.1.0", [false, true]],
      ["0.1.1", [false, false]],
    ]);
  });

  test("gives every row one entry per level of its depth", () => {
    for (const row of planRows(parseFixture(hashJoinAggregateSort).root)) {
      expect(row.branchContinues).toHaveLength(row.depth);
    }
  });
});

describe("planDescendantCount", () => {
  test("counts everything below a node, at any depth", () => {
    const tree = sampleTree();
    const [root, leaf, branch] = tree.nodes;

    expect(planDescendantCount(root)).toBe(3);
    expect(planDescendantCount(branch)).toBe(1);
    expect(planDescendantCount(leaf)).toBe(0);
  });
});

describe("planCollapsedAncestor", () => {
  const tree = () => parseFixture(cteNestedLoopInitplan);

  test("names the collapse a hidden node disappeared into", () => {
    const collapsed = new Set(["0.1"]);
    const { root } = tree();

    expect(planCollapsedAncestor(root, collapsed, "0.1.1")?.id).toBe("0.1");
    // A collapsed node keeps its own card, so it is not hidden by itself.
    expect(planCollapsedAncestor(root, collapsed, "0.1")).toBeUndefined();
    expect(planCollapsedAncestor(root, collapsed, "0.0")).toBeUndefined();
  });

  test("reports the outermost collapse, which is the one on screen", () => {
    const { root } = tree();
    expect(
      planCollapsedAncestor(root, new Set(["0", "0.1"]), "0.1.1")?.id
    ).toBe("0");
  });

  test("hides nothing without a collapse or a node to look for", () => {
    const { root } = tree();
    expect(planCollapsedAncestor(root, new Set(), "0.1.1")).toBeUndefined();
    expect(
      planCollapsedAncestor(root, new Set(["0.1"]), undefined)
    ).toBeUndefined();
    expect(
      planCollapsedAncestor(root, new Set(["0.1"]), "nope")
    ).toBeUndefined();
  });
});

describe("planRevealNode", () => {
  test("opens every collapse standing between the root and the node", () => {
    const { root } = parseFixture(cteNestedLoopInitplan);
    const revealed = planRevealNode(
      root,
      new Set(["0", "0.1", "0.0"]),
      "0.1.1"
    );

    expect([...revealed].sort()).toEqual(["0.0"]);
  });

  test("leaves a collapsed node folded when it is the node asked for", () => {
    const { root } = parseFixture(cteNestedLoopInitplan);
    const collapsed = new Set(["0.1"]);

    expect(planRevealNode(root, collapsed, "0.1")).toBe(collapsed);
  });

  test("returns the same set when nothing hid the node", () => {
    const { root } = parseFixture(cteNestedLoopInitplan);
    const collapsed = new Set(["0.0"]);

    expect(planRevealNode(root, collapsed, "0.1.1")).toBe(collapsed);
    expect(planRevealNode(root, collapsed, "nope")).toBe(collapsed);
    expect(planRevealNode(root, collapsed, undefined)).toBe(collapsed);
  });
});

describe("costBasis", () => {
  test("is the root's total cost, which the self costs add back up to", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const selfTotal = tree.nodes.reduce(
      (sum, node) => sum + (node.selfCost ?? 0),
      0
    );

    expect(tree.costBasis).toBeCloseTo(1465.93, 5);
    expect(selfTotal).toBeCloseTo(tree.root.totalCost ?? Number.NaN, 5);
  });

  test("covers the self costs a Limit truncates out of the root's total", () => {
    const tree = parseFixture(cteNestedLoopInitplan);

    // The Limit stops its children early, so it reports a total below theirs.
    expect(tree.root.totalCost).toBe(979.51);
    expect(tree.costBasis).toBeCloseTo(2578.21, 5);
    for (const node of tree.nodes) {
      expect(planSelfCostShare(node, tree)).toBeLessThanOrEqual(1);
    }
  });
});

describe("planSelfCostShare", () => {
  test("reports each node's own cost as a fraction of the plan's", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const shareOf = (id: string) => {
      const node = findPlanNode(tree, id);
      if (!node) throw new Error(`no node ${id}`);
      return planSelfCostShare(node, tree);
    };

    // The sequential scan of `orders` under the hash join.
    expect(shareOf("0.0.0.0")).toBeCloseTo(819 / 1465.93, 6);
    // A Hash node adds nothing on top of the scan it hashes.
    expect(shareOf("0.0.0.1")).toBe(0);
    expect(
      tree.nodes.reduce((sum, node) => sum + planSelfCostShare(node, tree), 0)
    ).toBeCloseTo(1, 6);
  });

  test("gives a lone node the whole plan", () => {
    const tree = parseFixture(seqScanFilter);
    expect(planSelfCostShare(tree.root, tree)).toBe(1);
  });

  test("is zero throughout a plan the planner costs at nothing", () => {
    const tree = buildPlanTree(node("0", {}, [node("0.0")]));
    expect(tree.costBasis).toBe(0);
    for (const entry of tree.nodes) {
      expect(planSelfCostShare(entry, tree)).toBe(0);
    }
  });
});

describe("planEdgeWidth", () => {
  test("spans the full range from the smallest to the largest estimate", () => {
    const tree = parseFixture(hashJoinAggregateSort);

    expect(planEdgeWidth(tree.maxRows, tree)).toBe(PLAN_EDGE_MAX_WIDTH);
    expect(planEdgeWidth(1, tree)).toBeGreaterThan(PLAN_EDGE_MIN_WIDTH);
    expect(planEdgeWidth(1, tree)).toBeLessThan(planEdgeWidth(5000, tree));
    expect(planEdgeWidth(5000, tree)).toBeLessThan(PLAN_EDGE_MAX_WIDTH);
  });

  test("keeps a rowless or unrankable edge at the visible minimum", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const rowless = buildPlanTree(node("0"));

    expect(planEdgeWidth(0, tree)).toBe(PLAN_EDGE_MIN_WIDTH);
    expect(planEdgeWidth(undefined, tree)).toBe(PLAN_EDGE_MIN_WIDTH);
    expect(planEdgeWidth(10, rowless)).toBe(PLAN_EDGE_MIN_WIDTH);
  });

  test("keeps an estimate far past the plan's largest inside the range", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    expect(planEdgeWidth(10_000_000, tree)).toBe(PLAN_EDGE_MAX_WIDTH);
  });
});

describe("planCostliestNodes", () => {
  test("ranks by the cost a node adds on top of its children", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    expect(
      planCostliestNodes(tree).map((entry) => [entry.nodeType, entry.subject])
    ).toEqual([
      ["Seq Scan", "orders"],
      ["HashAggregate", "c.region"],
      ["Hash Join", "(o.customer_id = c.id)"],
    ]);
  });

  test("leaves out nodes that add nothing and honours the limit", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    // The Hash node's total equals the scan it hashes, so it adds nothing.
    expect(planCostliestNodes(tree, 10)).toHaveLength(5);
    expect(planCostliestNodes(tree, 2).map((entry) => entry.id)).toEqual([
      "0.0.0.0",
      "0.0",
    ]);
  });

  test("is empty for a plan the planner costs at nothing", () => {
    expect(planCostliestNodes(buildPlanTree(node("0")))).toEqual([]);
  });
});

/** A node the way a plan that ran describes one. */
const timedNode = (
  id: string,
  overrides: Partial<PlanNode> = {},
  children: PlanNode[] = []
): PlanNode =>
  node(
    id,
    { actualRows: 0, actualTime: 0, selfTime: 0, loops: 1, ...overrides },
    children
  );

const engineWarning = (title: string): PlanWarning => ({
  title,
  detail: "SQL Server flagged this while compiling the plan.",
  source: "engine",
});

describe("planSelfTime", () => {
  test("takes the children's measured time off a cumulative one", () => {
    expect(
      planSelfTime(10, [
        timedNode("0.0", { actualTime: 4 }),
        timedNode("0.1", { actualTime: 3 }),
      ])
    ).toBe(3);
  });

  test("clamps a node that stopped its children early at zero", () => {
    expect(planSelfTime(1, [timedNode("0.0", { actualTime: 9 })])).toBe(0);
  });

  test("is undefined for a node the plan did not measure", () => {
    expect(planSelfTime(undefined, [])).toBeUndefined();
  });
});

describe("measured plans", () => {
  const measuredTree = () =>
    buildPlanTree(
      timedNode("0", { actualTime: 100, selfTime: 10, selfCost: 90 }, [
        timedNode("0.0", { actualTime: 70, selfTime: 70, selfCost: 5 }),
        timedNode("0.1", { actualTime: 20, selfTime: 20, selfCost: 5 }),
      ])
    );

  test("records which measurements the plan carries", () => {
    expect(measuredTree().measured).toEqual({ time: true, rows: true });
    expect(buildPlanTree(node("0")).measured).toEqual({
      time: false,
      rows: false,
    });
  });

  test("ranks by measured time, not by the planner's cost", () => {
    const tree = measuredTree();
    expect(planRankMetric(tree)).toBe("time");
    // The root is the costliest by estimate and the least of the work in fact.
    expect(planCostliestNodes(tree).map((entry) => entry.id)).toEqual([
      "0.0",
      "0.1",
      "0",
    ]);
  });

  test("takes every share from the metric the plan is ranked on", () => {
    const tree = measuredTree();
    expect(planSelfShare(tree.nodes[1], tree)).toBeCloseTo(0.7);
    // Cost shares stay available for the surfaces that print a cost.
    expect(planSelfCostShare(tree.nodes[1], tree)).toBeCloseTo(0.05);
  });

  test("keeps the time basis at or above the self times it adds up", () => {
    const limited = buildPlanTree(
      timedNode("0", { actualTime: 1, selfTime: 0 }, [
        timedNode("0.0", { actualTime: 40, selfTime: 40 }),
      ])
    );
    expect(limited.timeBasis).toBe(40);
    expect(planSelfShare(limited.nodes[1], limited)).toBe(1);
  });

  test("shades by measured time where asked", () => {
    const tree = measuredTree();
    expect(planHighlightIntensity(tree.nodes[1], tree, "time")).toBe(1);
    expect(planHighlightIntensity(tree.nodes[0], tree, "time")).toBeCloseTo(
      Math.sqrt(10 / 70)
    );
  });

  test("gives an unmeasured plan no ranking by time", () => {
    const tree = buildPlanTree(node("0", { selfCost: 5 }));
    expect(planRankMetric(tree)).toBe("cost");
    expect(planHasSelfMetric(tree)).toBe(true);
    expect(planHasSelfMetric(buildPlanTree(bareNode("0")))).toBe(false);
  });
});

describe("planRowEstimateFactor", () => {
  test("multiplies the per-execution estimate out by the executions", () => {
    // 5 rows expected each time round a 10-loop nested loop, 500 returned.
    expect(
      planRowEstimateFactor(
        timedNode("0", { rows: 5, loops: 10, actualRows: 500 })
      )
    ).toBe(10);
  });

  test("reads below one when the node returned fewer rows than planned", () => {
    expect(
      planRowEstimateFactor(timedNode("0", { rows: 100, actualRows: 25 }))
    ).toBe(0.25);
  });

  test("is undefined without a measurement, or without an estimate to compare", () => {
    expect(planRowEstimateFactor(node("0", { rows: 10 }))).toBeUndefined();
    expect(
      planRowEstimateFactor(timedNode("0", { rows: 0, actualRows: 10 }))
    ).toBeUndefined();
  });
});

describe("planWarningGroups", () => {
  // One plan: a scan that is most of it, a scan that is a rounding error, and
  // a diagnostic on a node the planner thinks costs nothing.
  const mixedTree = () =>
    buildPlanTree(
      node("0", { selfCost: 10 }, [
        node("0.0", { selfCost: 85, warnings: [PLAN_FULL_TABLE_SCAN] }),
        node("0.1", { selfCost: 5, warnings: [PLAN_FULL_TABLE_SCAN] }),
        node("0.2", {
          selfCost: 0,
          warnings: [engineWarning("Columns with no statistics")],
        }),
      ])
    );

  test("keeps an engine diagnostic whatever the node costs", () => {
    expect(
      planWarningGroups(mixedTree()).map((group) => [
        planWarningIdentity(group.warning),
        group.nodes.map((entry) => entry.id),
      ])
    ).toEqual([
      ["full-table-scan", ["0.0"]],
      ["Columns with no statistics", ["0.2"]],
    ]);
  });

  test("drops one of our scan heuristics on a node that costs little", () => {
    const share = planSelfShare(mixedTree().nodes[2], mixedTree());
    expect(share).toBeLessThan(PLAN_WARNING_RELEVANT_SHARE);
    expect(
      planWarningGroups(mixedTree()).flatMap((group) =>
        group.nodes.map((entry) => entry.id)
      )
    ).not.toContain("0.1");
  });

  test("groups a warning several nodes report, the worst of them first", () => {
    const tree = buildPlanTree(
      node("0", {}, [
        node("0.0", { selfCost: 20, warnings: [PLAN_FULL_TABLE_SCAN] }),
        node("0.1", { selfCost: 80, warnings: [PLAN_FULL_TABLE_SCAN] }),
      ])
    );
    expect(planWarningGroups(tree)).toHaveLength(1);
    expect(planWarningGroups(tree)[0].nodes.map((entry) => entry.id)).toEqual([
      "0.1",
      "0.0",
    ]);
  });

  test("keeps every warning when the plan has no number to weigh them by", () => {
    const tree = buildPlanTree({
      ...bareNode("0", [
        { ...bareNode("0.0"), warnings: [PLAN_FULL_TABLE_SCAN] },
      ]),
    });
    expect(
      planWarningGroups(tree).map((group) => planWarningIdentity(group.warning))
    ).toEqual(["full-table-scan"]);
  });

  test("is empty for a plan nothing is wrong with", () => {
    expect(planWarningGroups(buildPlanTree(node("0")))).toEqual([]);
  });
});
