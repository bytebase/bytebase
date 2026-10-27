import { describe, expect, test } from "vitest";
import {
  DATABRICKS_PLAN_EMPTY_MESSAGE,
  DATABRICKS_PLAN_NO_PLAN_MESSAGE,
  parseDatabricksPlan,
} from "./databricks-plan";
import {
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import broadcastJoin from "./test-data/databricks/broadcast-join.txt?raw";
import existsSubquery from "./test-data/databricks/exists-subquery.txt?raw";
import partitionFilter from "./test-data/databricks/partition-filter.txt?raw";
import singleScan from "./test-data/databricks/single-scan.txt?raw";
import sortLimit from "./test-data/databricks/sort-limit.txt?raw";
import unionAll from "./test-data/databricks/union-all.txt?raw";

/** A chain of `depth` operators, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  const tree = [`${" ".repeat(0)}Project (${depth})`];
  for (let level = 1; level < depth; level += 1) {
    tree.push(`${" ".repeat(level * 3)}+- Filter (${depth - level})`);
  }
  return ["== Physical Plan ==", ...tree].join("\n");
};

const parse = (source: string): PlanTree => {
  const result = parseDatabricksPlan(source);
  if (!result.ok)
    throw new Error(`expected a parsable plan: ${result.message}`);
  return result.tree;
};

const outline = (tree: PlanTree): string[] =>
  tree.nodes.map(
    (node) => `${node.id} ${node.nodeType} ${node.subject ?? "-"}`
  );

const propertyValue = (node: PlanNode, label: string): string | undefined =>
  node.properties.find((property) => property.label === label)?.value;

describe("parseDatabricksPlan", () => {
  test("reads the tree and the detail block that belongs to each operator", () => {
    const tree = parse(singleScan);
    expect(outline(tree)).toEqual([
      "0 Filter -",
      "0.0 ColumnarToRow -",
      "0.0.0 Scan parquet spark_catalog.default.customers",
    ]);
    const scan = tree.nodes[2];
    // What the read pushed down, which is the first thing a Spark plan is read
    // for.
    expect(propertyValue(scan, "Pushed filters")).toBe(
      "[IsNotNull(city), EqualTo(city,berlin)]"
    );
    expect(propertyValue(scan, "Read schema")).toBe("struct<city:string>");
    // Spark counts what it lists in the label; the count is in the value, and
    // the column carries the expression id of the session that planned it.
    expect(propertyValue(scan, "Output")).toMatch(/^\[city#\d+\]$/);
    expect(propertyValue(tree.root, "Condition")).toMatch(
      /^\(isnotnull\(city#\d+\) AND \(city#\d+ = berlin\)\)$/
    );
    // The operator number Spark's own output prints, and the stage it fused it
    // into.
    expect(propertyValue(tree.root, "Operator")).toBe("3");
    expect(propertyValue(tree.root, "Codegen")).toBe("codegen id : 1");
  });

  test("resolves a read's table from the name Spark qualifies", () => {
    const tree = parse(singleScan);
    expect(tree.nodes[2].relation).toEqual({
      database: "spark_catalog",
      schema: "default",
      table: "customers",
    });
    // An operator that reads no table has none to look up.
    expect(tree.root.relation).toBeUndefined();
  });

  test("follows both sides of a join, and the exchange under one", () => {
    const tree = parse(broadcastJoin);
    expect(outline(tree)).toEqual([
      // FORMATTED prints the operator alone on the tree line; the isFinalPlan
      // flag a plain EXPLAIN puts beside it is in the detail block instead,
      // which the node carries as a property.
      "0 AdaptiveSparkPlan -",
      "0.0 HashAggregate -",
      "0.0.0 Exchange -",
      "0.0.0.0 HashAggregate -",
      "0.0.0.0.0 Project -",
      "0.0.0.0.0.0 BroadcastHashJoin Inner BuildLeft",
      "0.0.0.0.0.0.0 BroadcastExchange -",
      "0.0.0.0.0.0.0.0 Filter -",
      "0.0.0.0.0.0.0.0.0 Scan parquet spark_catalog.default.customers",
      "0.0.0.0.0.0.1 Filter -",
      "0.0.0.0.0.0.1.0 Scan parquet spark_catalog.default.orders",
    ]);
    const join = tree.nodes[5];
    // Spark suffixes a column with the expression id of the session that
    // planned it, so the shape is what is worth pinning.
    expect(propertyValue(join, "Left keys")).toMatch(/^\[id#\d+\]$/);
    expect(propertyValue(join, "Join type")).toBe("Inner");
    expect(propertyValue(tree.root, "Arguments")).toBe("isFinalPlan=false");
  });

  test("draws both branches of a union", () => {
    const tree = parse(unionAll);
    expect(tree.nodes[1].nodeType).toBe("Union");
    expect(tree.nodes[1].children).toHaveLength(2);
    expect(
      tree.nodes.filter((node) => node.nodeType === "Scan parquet")
    ).toHaveLength(2);
  });

  test("keeps the partitions a read skipped", () => {
    const tree = parse(partitionFilter);
    const scan = tree.nodes.find((node) => node.relation?.table === "events");
    expect(propertyValue(scan ?? tree.root, "Partition filters")).toContain(
      "day"
    );
  });

  test("reads a semi join, and a sort with its fetch", () => {
    expect(
      outline(parse(existsSubquery)).some((line) =>
        line.includes("BroadcastHashJoin LeftSemi BuildRight")
      )
    ).toBe(true);
    expect(
      outline(parse(sortLimit)).some((line) =>
        line.includes("TakeOrderedAndProject")
      )
    ).toBe(true);
  });

  test("ranks nothing, since FORMATTED states neither cost nor rows", () => {
    const tree = parse(broadcastJoin);
    expect(tree.estimates.cost).toBe(false);
    expect(tree.estimates.rows).toBe(false);
    expect(tree.measured.rows).toBe(false);
    expect(tree.root.totalCost).toBeUndefined();
    expect(tree.root.rows).toBeUndefined();
  });

  test("reads the indent from the markers, not from the operator's own text", () => {
    // An operator can print an expression holding the same characters a branch
    // does, and taking that for an indent would leave the read below it out of
    // the tree.
    const tree = parse(
      [
        "== Physical Plan ==",
        "Project [cast((a+-b) as int) AS x] (2)",
        "+- Scan parquet spark_catalog.default.t (1)",
      ].join("\n")
    );
    expect(outline(tree)).toEqual([
      "0 Project [cast((a+-b) as int) AS x]",
      "0.0 Scan parquet spark_catalog.default.t",
    ]);
  });

  test("says so when there is no physical plan to read", () => {
    expect(parseDatabricksPlan("")).toEqual({
      ok: false,
      message: DATABRICKS_PLAN_EMPTY_MESSAGE,
    });
    expect(parseDatabricksPlan("not a plan")).toEqual({
      ok: false,
      message: DATABRICKS_PLAN_NO_PLAN_MESSAGE,
    });
    expect(parseDatabricksPlan("{}")).toEqual({
      ok: false,
      message: DATABRICKS_PLAN_NO_PLAN_MESSAGE,
    });
    // The header alone, with every operator line missing its number.
    expect(
      parseDatabricksPlan("== Physical Plan ==\n* Filter\n+- Scan parquet t\n")
    ).toEqual({ ok: false, message: DATABRICKS_PLAN_NO_PLAN_MESSAGE });
  });

  test("refuses a plan deeper than the viewer draws", () => {
    expect(parseDatabricksPlan(nestedPlan(600))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
