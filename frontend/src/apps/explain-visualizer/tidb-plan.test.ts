import { describe, expect, test } from "vitest";
import {
  PLAN_FULL_INDEX_SCAN,
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import indexHashJoin from "./test-data/tidb/index-hash-join.json";
import measuredHashJoin from "./test-data/tidb/measured-hash-join.json";
import tableReaderSelection from "./test-data/tidb/table-reader-selection.json";
import unionHashAgg from "./test-data/tidb/union-hash-agg.json";
import {
  parseTidbPlan,
  TIDB_PLAN_EMPTY_MESSAGE,
  TIDB_PLAN_INVALID_JSON_MESSAGE,
  TIDB_PLAN_NO_PLAN_MESSAGE,
} from "./tidb-plan";

/** A chain of `depth` operators, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  let plan: Record<string, unknown> = { id: "TableFullScan_1" };
  for (let level = 1; level < depth; level += 1) {
    plan = { id: `Selection_${level + 1}`, subOperators: [plan] };
  }
  return JSON.stringify([plan]);
};

const parse = (source: string): PlanTree => {
  const result = parseTidbPlan(source);
  if (!result.ok)
    throw new Error(`expected a parsable plan: ${result.message}`);
  return result.tree;
};

const parseFixture = (fixture: unknown): PlanTree =>
  parse(JSON.stringify(fixture));

const outline = (tree: PlanTree): string[] =>
  tree.nodes.map(
    (node) => `${node.id} ${node.nodeType} ${node.subject ?? "-"}`
  );

const propertyValue = (node: PlanNode, label: string): string | undefined =>
  node.properties.find((property) => property.label === label)?.value;

const byId = (tree: PlanTree, id: string): PlanNode => {
  const node = tree.nodes.find((candidate) => candidate.id === id);
  if (!node) throw new Error(`no node ${id} in ${outline(tree).join(", ")}`);
  return node;
};

describe("parseTidbPlan", () => {
  test("names each operator as TiDB's own plan does, without its sequence number", () => {
    const tree = parseFixture(tableReaderSelection);
    const scan = byId(tree, "0.0.0");

    expect(outline(tree)).toEqual([
      "0 TableReader data:Selection_6",
      "0.0 Selection gt(qp.orders.total, 880)",
      "0.0.0 TableFullScan orders",
    ]);
    expect(scan).toMatchObject({
      rows: 50000,
      relation: { table: "orders" },
      children: [],
    });
    expect(scan.warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
    // The id TiDB prints stays on the node, so a reader can match it against
    // the text plan they already have.
    expect(propertyValue(scan, "Operator")).toBe("TableFullScan_5");
    expect(propertyValue(scan, "Task type")).toBe("cop[tikv]");
  });

  test("ranks nothing, because no TiDB plan format carries a cost", () => {
    const tree = parseFixture(indexHashJoin);

    expect(tree.estimates.cost).toBe(false);
    expect(tree.costBasis).toBe(0);
    expect(tree.estimates.rows).toBe(true);
    for (const node of tree.nodes) {
      expect(node.totalCost).toBeUndefined();
      expect(node.selfCost).toBeUndefined();
    }
  });

  test("reads which side of a join an operator feeds off its id", () => {
    const tree = parseFixture(indexHashJoin);

    expect(outline(tree)).toEqual([
      "0 IndexHashJoin inner join, inner:IndexLookUp_13, outer key:qp.customers.id, inner key:qp.orders.customer_id, equal cond:eq(qp.customers.id, qp.orders.customer_id)",
      "0.0 TableReader data:TableRangeScan_43",
      "0.0.0 TableRangeScan c",
      "0.1 IndexLookUp -",
      "0.1.0 Selection lt(qp.orders.customer_id, 10), not(isnull(qp.orders.customer_id))",
      "0.1.0.0 IndexRangeScan o",
      "0.1.1 TableRowIDScan o",
    ]);
    expect(byId(tree, "0.0").relationship).toBe("Build");
    expect(byId(tree, "0.1").relationship).toBe("Probe");
    expect(byId(tree, "0.0.0").relationship).toBeUndefined();
  });

  test("splits an access object into the table it reads and the index it uses", () => {
    const indexScan = byId(parseFixture(indexHashJoin), "0.1.0.0");

    // `o` is the statement's name for the table, which TiDB does not resolve,
    // so it names the read and nothing is looked up by it.
    expect(indexScan.relation).toEqual({ table: "o", unresolved: true });
    expect(propertyValue(indexScan, "Table")).toBe("o");
    // The index names its columns in parentheses, where a comma separates
    // columns rather than parts of the access object.
    expect(propertyValue(indexScan, "Index")).toBe(
      "orders_customer_id_idx(customer_id)"
    );
    // A range read of an index is not a read of all of it.
    expect(indexScan.warnings).toEqual([]);
  });

  test("hangs both halves of a union off it", () => {
    const tree = parseFixture(unionHashAgg);

    expect(outline(tree).slice(0, 3)).toEqual([
      "0 HashAgg group by:Column#9, funcs:firstrow(Column#9)->Column#9",
      "0.0 Union -",
      "0.0.0 TableReader data:Projection_14",
    ]);
    expect(byId(tree, "0.0").children).toHaveLength(2);
    expect(byId(tree, "0.0.0.0.0.0").relation).toEqual({
      table: "orders",
      unresolved: true,
    });
    expect(byId(tree, "0.0.1.0.0.0").relation).toEqual({
      table: "customers",
      unresolved: true,
    });
  });

  test("reads a measured plan for its rows and not its timings", () => {
    const tree = parseFixture(measuredHashJoin);
    const join = byId(tree, "0.0.0");

    expect(tree.measured).toEqual({ rows: true, time: false });
    expect(join.actualRows).toBe(16666);
    expect(join.rows).toBe(16666);
    expect(join.actualTime).toBeUndefined();
    // `time:` covers an operator and its subtree, and the operators pushed
    // down to a coprocessor report none, so the string stays a property rather
    // than becoming a number anything ranks by.
    expect(propertyValue(join, "Execute info")).toContain("time:");
    expect(
      propertyValue(byId(tree, "0.0.0.0.0"), "Execute info")
    ).not.toContain("time:");
  });

  test("flags a read of every row of an index, and leaves a small one alone", () => {
    const full = parse(
      JSON.stringify([
        {
          id: "IndexFullScan_5",
          estRows: "50000",
          accessObject: "table:orders, index:i(total)",
        },
      ])
    );
    const small = parse(
      JSON.stringify([
        {
          id: "IndexFullScan_5",
          estRows: "200",
          accessObject: "table:countries, index:i(name)",
        },
      ])
    );

    expect(full.root.warnings).toEqual([PLAN_FULL_INDEX_SCAN]);
    expect(small.root.warnings).toEqual([]);
  });

  test("reports a plan it cannot read", () => {
    expect(parseTidbPlan("  ")).toEqual({
      ok: false,
      message: TIDB_PLAN_EMPTY_MESSAGE,
    });
    expect(parseTidbPlan("not json")).toEqual({
      ok: false,
      message: TIDB_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseTidbPlan("[]")).toEqual({
      ok: false,
      message: TIDB_PLAN_NO_PLAN_MESSAGE,
    });
    // A tabular EXPLAIN, which is what every other TiDB format returns.
    expect(
      parseTidbPlan("id\testRows\ttask\nTableReader_7\t5.00\troot")
    ).toEqual({ ok: false, message: TIDB_PLAN_INVALID_JSON_MESSAGE });
  });

  test("gives up past the depth it can draw", () => {
    expect(parseTidbPlan(nestedPlan(400)).ok).toBe(true);
    expect(parseTidbPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
