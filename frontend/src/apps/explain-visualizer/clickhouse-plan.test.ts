import { describe, expect, test } from "vitest";
import {
  CLICKHOUSE_PLAN_EMPTY_MESSAGE,
  CLICKHOUSE_PLAN_INVALID_JSON_MESSAGE,
  CLICKHOUSE_PLAN_NO_PLAN_MESSAGE,
  parseClickhousePlan,
} from "./clickhouse-plan";
import {
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import joinTwoTables from "./test-data/clickhouse/join-two-tables.json";
import primaryKeyRange from "./test-data/clickhouse/primary-key-range.json";
import readFilterAggregate from "./test-data/clickhouse/read-filter-aggregate.json";

/** A chain of `depth` operators, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  let plan: Record<string, unknown> = { "Node Type": "ReadFromMergeTree" };
  for (let level = 1; level < depth; level += 1) {
    plan = { "Node Type": "Expression", Plans: [plan] };
  }
  return JSON.stringify([{ Plan: plan }]);
};

const parse = (source: string): PlanTree => {
  const result = parseClickhousePlan(source);
  if (!result.ok)
    throw new Error(`expected a parsable plan: ${result.message}`);
  return result.tree;
};

const parseFixture = (fixture: unknown): PlanTree =>
  parse(JSON.stringify(fixture));

const outline = (tree: PlanTree): string[] =>
  tree.nodes.map((node) => `${node.id} ${node.nodeType}`);

const propertyValue = (node: PlanNode, label: string): string | undefined =>
  node.properties.find((property) => property.label === label)?.value;

const byId = (tree: PlanTree, id: string): PlanNode => {
  const node = tree.nodes.find((candidate) => candidate.id === id);
  if (!node) throw new Error(`no node ${id} in ${outline(tree).join(", ")}`);
  return node;
};

describe("parseClickhousePlan", () => {
  test("reads the pipeline from the query's result down to the read", () => {
    const tree = parseFixture(readFilterAggregate);

    expect(outline(tree)).toEqual([
      "0 Expression",
      "0.0 Limit",
      "0.0.0 Sorting",
      "0.0.0.0 Expression",
      "0.0.0.0.0 Aggregating",
      "0.0.0.0.0.0 Expression",
      "0.0.0.0.0.0.0 Expression",
      "0.0.0.0.0.0.0.0 ReadFromMergeTree",
    ]);
    // ClickHouse says what each step is for, which is all it says: the one
    // thing a node carries besides its name.
    expect(tree.root.subject).toBe("Project names");
    expect(byId(tree, "0.0.0").subject).toBe("Sorting for ORDER BY");
  });

  test("ranks nothing, because no ClickHouse EXPLAIN carries a cost", () => {
    const tree = parseFixture(readFilterAggregate);

    expect(tree.estimates).toEqual({
      cost: false,
      startupCost: false,
      rows: false,
      width: false,
    });
    expect(tree.measured).toEqual({ time: false, rows: false });
    expect(tree.costBasis).toBe(0);
  });

  test("answers a read with the database ClickHouse qualified it by", () => {
    const read = byId(parseFixture(primaryKeyRange), "0.0.0");

    expect(read.nodeType).toBe("ReadFromMergeTree");
    expect(read.relation).toEqual({ database: "default", table: "orders" });
  });

  test("reports how much each index let the read skip", () => {
    const read = byId(parseFixture(primaryKeyRange), "0.0.0");

    // Granules in against granules out is what a ClickHouse plan is read for.
    const index = propertyValue(read, "PrimaryKey index");
    expect(index).toContain("Initial granules:");
    expect(index).toContain("Selected granules:");
    expect(index).toContain("Condition:");
  });

  test("hangs both sides of a join off it, and names only the read it can", () => {
    const tree = parseFixture(joinTwoTables);
    const join = byId(tree, "0.0");

    expect(join.nodeType).toBe("Join");
    expect(join.children).toHaveLength(2);
    expect(byId(tree, "0.0.0.0").relation).toEqual({
      database: "default",
      table: "orders",
    });
    // A read of a table function names no table to look up.
    expect(byId(tree, "0.0.1.0").nodeType).toBe("ReadFromSystemNumbers");
    expect(byId(tree, "0.0.1.0").relation).toBeUndefined();
  });

  test("leaves an unqualified description unresolved", () => {
    const tree = parse(
      JSON.stringify([
        { Plan: { "Node Type": "ReadFromMergeTree", Description: "orders" } },
      ])
    );

    // A bare name is the one the statement used, which is not enough to say
    // which database's table it is.
    expect(tree.root.relation).toBeUndefined();
    expect(tree.root.subject).toBe("orders");
  });

  test("reports a plan it cannot read", () => {
    expect(parseClickhousePlan("  ")).toEqual({
      ok: false,
      message: CLICKHOUSE_PLAN_EMPTY_MESSAGE,
    });
    expect(parseClickhousePlan("not json")).toEqual({
      ok: false,
      message: CLICKHOUSE_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseClickhousePlan("[]")).toEqual({
      ok: false,
      message: CLICKHOUSE_PLAN_NO_PLAN_MESSAGE,
    });
    // The plan ClickHouse returns without `json = 1`, which is indented text.
    expect(
      parseClickhousePlan("Expression ((Project names + Projection))")
    ).toEqual({ ok: false, message: CLICKHOUSE_PLAN_INVALID_JSON_MESSAGE });
  });

  test("gives up past the depth it can draw", () => {
    expect(parseClickhousePlan(nestedPlan(400)).ok).toBe(true);
    expect(parseClickhousePlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
