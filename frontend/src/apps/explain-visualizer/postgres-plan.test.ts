import { describe, expect, test } from "vitest";
import {
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import {
  POSTGRES_PLAN_EMPTY_MESSAGE,
  POSTGRES_PLAN_INVALID_JSON_MESSAGE,
  POSTGRES_PLAN_NO_PLAN_MESSAGE,
  parsePostgresPlan,
} from "./postgres-plan";
import bitmapIndexScan from "./test-data/postgres/bitmap-index-scan.json";
import cteNestedLoopInitplan from "./test-data/postgres/cte-nested-loop-initplan.json";
import hashJoinAggregateSort from "./test-data/postgres/hash-join-aggregate-sort.json";
import seqScanFilter from "./test-data/postgres/seq-scan-filter.json";

/** A chain of `depth` nodes, the shape that drives every recursive walk. */
const nestedPlan = (depth: number): string => {
  let plan: Record<string, unknown> = { "Node Type": "Seq Scan" };
  for (let level = 1; level < depth; level += 1) {
    plan = { "Node Type": "Nested Loop", Plans: [plan] };
  }
  return JSON.stringify([{ Plan: plan }]);
};

const parseFixture = (fixture: unknown): PlanTree => {
  const result = parsePostgresPlan(JSON.stringify(fixture));
  if (!result.ok)
    throw new Error(`expected a parsable plan: ${result.message}`);
  return result.tree;
};

const propertyValue = (node: PlanNode, label: string): string | undefined =>
  node.properties.find((property) => property.label === label)?.value;

describe("parsePostgresPlan", () => {
  test("maps a single seq scan with a filter", () => {
    const { root, nodes } = parseFixture(seqScanFilter);

    expect(nodes).toHaveLength(1);
    expect(root).toMatchObject({
      id: "0",
      nodeType: "Seq Scan",
      subject: "customers",
      startupCost: 0,
      totalCost: 90.5,
      selfCost: 90.5,
      rows: 1667,
      width: 12,
      children: [],
    });
    expect(root.relationship).toBeUndefined();
    expect(propertyValue(root, "Filter")).toBe("(region = 'eu'::text)");
    expect(propertyValue(root, "Alias")).toBe("customers");
  });

  test("keeps typed fields out of the property list", () => {
    const { root } = parseFixture(seqScanFilter);
    const labels = root.properties.map((property) => property.label);

    expect(labels).not.toContain("Node Type");
    expect(labels).not.toContain("Total Cost");
    expect(labels).not.toContain("Plan Rows");
    expect(labels).not.toContain("Plan Width");
    expect(labels).not.toContain("Plans");
  });

  test("names an index scan by its index and charges cost to the node itself", () => {
    const { root, nodes } = parseFixture(bitmapIndexScan);
    const [, indexScan] = nodes;

    expect(nodes).toHaveLength(2);
    expect(root.nodeType).toBe("Bitmap Heap Scan");
    expect(root.subject).toBe("orders");
    expect(root.selfCost).toBeCloseTo(39.18 - 4.37, 5);
    expect(indexScan).toMatchObject({
      id: "0.0",
      nodeType: "Bitmap Index Scan",
      subject: "orders_customer_id_idx",
      selfCost: 4.37,
    });
    expect(propertyValue(indexScan, "Index Cond")).toBe("(customer_id = 42)");
  });

  test("leaves out the Outer relationship every child has unless it says otherwise", () => {
    const { nodes } = parseFixture(hashJoinAggregateSort);
    const byId = new Map(nodes.map((node) => [node.id, node]));

    // The join's probe side is PostgreSQL's default "Outer"; its hash is "Inner".
    expect(byId.get("0.0.0.0")?.relationship).toBeUndefined();
    expect(byId.get("0.0.0.1")?.relationship).toBe("Inner");
  });

  test("walks a hash join plan depth-first and joins list-valued properties", () => {
    const { nodes } = parseFixture(hashJoinAggregateSort);

    expect(nodes.map((node) => `${node.id} ${node.nodeType}`)).toEqual([
      "0 Sort",
      // PostgreSQL's JSON says Aggregate + Strategy: Hashed; its text format
      // calls the same node HashAggregate, and so do we.
      "0.0 HashAggregate",
      "0.0.0 Hash Join",
      "0.0.0.0 Seq Scan",
      "0.0.0.1 Hash",
      "0.0.0.1.0 Seq Scan",
    ]);
    expect(propertyValue(nodes[0], "Sort Key")).toBe("(count(*)) DESC");
    expect(propertyValue(nodes[1], "Group Key")).toBe("c.region");
    expect(propertyValue(nodes[2], "Hash Cond")).toBe("(o.customer_id = c.id)");
    expect(propertyValue(nodes[2], "Join Type")).toBe("Inner");
    expect(propertyValue(nodes[2], "Inner Unique")).toBe("true");
  });

  test("names nodes the way PostgreSQL's own text format does", () => {
    const node = (raw: Record<string, unknown>) => {
      const result = parsePostgresPlan(JSON.stringify([{ Plan: raw }]));
      if (!result.ok) throw new Error(result.message);
      return result.tree.root;
    };

    expect(
      node({ "Node Type": "Aggregate", Strategy: "Hashed" }).nodeType
    ).toBe("HashAggregate");
    expect(
      node({ "Node Type": "Aggregate", Strategy: "Sorted" }).nodeType
    ).toBe("GroupAggregate");
    expect(node({ "Node Type": "Aggregate", Strategy: "Plain" }).nodeType).toBe(
      "Aggregate"
    );
    expect(
      node({
        "Node Type": "Aggregate",
        Strategy: "Hashed",
        "Partial Mode": "Partial",
      }).nodeType
    ).toBe("Partial HashAggregate");
    expect(
      node({ "Node Type": "Seq Scan", "Parallel Aware": true }).nodeType
    ).toBe("Parallel Seq Scan");
    expect(
      node({ "Node Type": "Seq Scan", "Parallel Aware": false }).nodeType
    ).toBe("Seq Scan");
    expect(
      node({ "Node Type": "ModifyTable", Operation: "Update" }).nodeType
    ).toBe("Update");
    expect(
      node({ "Node Type": "SetOp", Strategy: "Hashed", Command: "Intersect" })
        .nodeType
    ).toBe("HashSetOp Intersect");
    expect(
      node({ "Node Type": "SetOp", Strategy: "Sorted", Command: "Except All" })
        .nodeType
    ).toBe("SetOp Except All");
  });

  test("falls back to the node's condition when it names no relation", () => {
    const { nodes } = parseFixture(hashJoinAggregateSort);
    const [sort, aggregate, hashJoin] = nodes;

    // None of these three name a table, so the card would otherwise be blank.
    expect(sort.subject).toBe("(count(*)) DESC");
    expect(aggregate.subject).toBe("c.region");
    expect(hashJoin.subject).toBe("(o.customer_id = c.id)");
  });

  test("charges each node only the cost its children do not explain", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const byId = new Map(tree.nodes.map((node) => [node.id, node]));

    // The Hash Join totals 1090.86 over a 819 scan and a 78 hash.
    expect(byId.get("0.0.0")?.selfCost).toBeCloseTo(1090.86 - 819 - 78, 5);
    // The leaf scan of `orders` therefore dominates the plan's own work.
    expect(tree.maxSelfCost).toBe(819);
    expect(tree.maxRows).toBe(50000);
  });

  test("labels an InitPlan subquery and a nested loop's inner side", () => {
    const { root, nodes } = parseFixture(cteNestedLoopInitplan);
    const byId = new Map(nodes.map((node) => [node.id, node]));

    expect(root.nodeType).toBe("Limit");
    expect(root.children).toHaveLength(2);
    expect(byId.get("0.0")?.relationship).toBe("InitPlan 1");
    expect(byId.get("0.1")?.nodeType).toBe("Nested Loop");
    expect(byId.get("0.1.1")).toMatchObject({
      nodeType: "Index Scan",
      subject: "customers",
      relationship: "Inner",
    });
    expect(propertyValue(byId.get("0.1.0") as PlanNode, "Filter")).toContain(
      "(InitPlan 1).col1"
    );
  });

  test("accepts a bare plan object as well as PostgreSQL's array wrapper", () => {
    const result = parsePostgresPlan(
      JSON.stringify({ Plan: { "Node Type": "Result", "Total Cost": 1 } })
    );

    expect(result.ok).toBe(true);
    expect(result.ok && result.tree.root.nodeType).toBe("Result");
  });

  test("defaults missing numbers and an unnamed node type", () => {
    const result = parsePostgresPlan(JSON.stringify([{ Plan: {} }]));

    expect(result.ok && result.tree.root).toMatchObject({
      nodeType: "Unknown",
      startupCost: 0,
      totalCost: 0,
      selfCost: 0,
      rows: 0,
      width: 0,
    });
  });

  test("parses a plan that arrives wrapped in other output", () => {
    const plan = JSON.stringify([{ Plan: { "Node Type": "Seq Scan" } }]);

    for (const wrapped of [
      `SET\nSET\n${plan}`,
      `${plan}\n(1 row)`,
      `QUERY PLAN\n----------\n${plan}\n`,
    ]) {
      const result = parsePostgresPlan(wrapped);
      expect(result.ok && result.tree.root.nodeType).toBe("Seq Scan");
    }
  });

  test.each([
    ["", POSTGRES_PLAN_EMPTY_MESSAGE],
    ["   ", POSTGRES_PLAN_EMPTY_MESSAGE],
    ["Seq Scan on customers", POSTGRES_PLAN_INVALID_JSON_MESSAGE],
    ["[{", POSTGRES_PLAN_INVALID_JSON_MESSAGE],
    ["[]", POSTGRES_PLAN_NO_PLAN_MESSAGE],
    ["{}", POSTGRES_PLAN_NO_PLAN_MESSAGE],
    ['[{"Plan": "not an object"}]', POSTGRES_PLAN_NO_PLAN_MESSAGE],
  ])("rejects %j with a specific message", (source, message) => {
    expect(parsePostgresPlan(source)).toEqual({ ok: false, message });
  });

  test("reads a plan nested deeper than anything the planner produces", () => {
    const result = parsePostgresPlan(nestedPlan(100));
    expect(result.ok && result.tree.nodes).toHaveLength(100);
  });

  test("rejects a plan nested deeper than the walks over it can go", () => {
    // Past the cap, a plan is refused here rather than overflowing the stack
    // later, in a render with no error boundary over it.
    expect(parsePostgresPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });

  test("warns on a sequential scan of a table large enough to matter", () => {
    const { nodes } = parseFixture(hashJoinAggregateSort);
    const flagged = nodes.filter((node) => node.warnings.length > 0);

    // `orders` at 819 is flagged; `customers` at 78 is a tiny table.
    expect(flagged.map((node) => node.subject)).toEqual(["orders"]);
    expect(flagged[0].warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
  });

  test("leaves a scan of a tiny table alone", () => {
    const { root } = parseFixture(seqScanFilter);

    expect(root.totalCost).toBe(90.5);
    expect(root.warnings).toEqual([]);
  });

  test("flags the parallel sequential scan and no other scan type", () => {
    const warningsOf = (raw: Record<string, unknown>) => {
      const result = parsePostgresPlan(
        JSON.stringify([{ Plan: { "Total Cost": 5000, ...raw } }])
      );
      if (!result.ok) throw new Error(result.message);
      return result.tree.root.warnings;
    };

    expect(warningsOf({ "Node Type": "Seq Scan" })).toEqual([
      PLAN_FULL_TABLE_SCAN,
    ]);
    expect(
      warningsOf({ "Node Type": "Seq Scan", "Parallel Aware": true })
    ).toEqual([PLAN_FULL_TABLE_SCAN]);
    expect(warningsOf({ "Node Type": "Index Scan" })).toEqual([]);
    expect(warningsOf({ "Node Type": "Bitmap Heap Scan" })).toEqual([]);
  });
});
