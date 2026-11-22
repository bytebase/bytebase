import { describe, expect, test } from "vitest";
import {
  HIVE_PLAN_EMPTY_MESSAGE,
  HIVE_PLAN_INVALID_JSON_MESSAGE,
  HIVE_PLAN_NO_PLAN_MESSAGE,
  parseHivePlan,
} from "./hive-plan";
import {
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import analyzedJoin from "./test-data/hive/analyzed-join.json";
import joinAggregate from "./test-data/hive/join-aggregate.json";
import noCboPlan from "./test-data/hive/no-cbo-plan.json";
import orderLimit from "./test-data/hive/order-limit.json";
import singleScan from "./test-data/hive/single-scan.json";
import unionAll from "./test-data/hive/union-all.json";

/** A plan of `rels`, as Hive nests one: Calcite's JSON within the answer. */
const planOf = (rels: Record<string, unknown>[]): string =>
  JSON.stringify({ CBOPlan: JSON.stringify({ rels }) });

/** A chain of `depth` operators, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  const rels: Record<string, unknown>[] = [
    { id: "0", relOp: "HiveTableScan", table: ["default", "t"], inputs: [] },
  ];
  for (let level = 1; level < depth; level += 1) {
    rels.push({ id: String(level), relOp: "HiveFilter" });
  }
  return planOf(rels);
};

const parse = (source: string): PlanTree => {
  const result = parseHivePlan(source);
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

describe("parseHivePlan", () => {
  test("reads a chain whose operators name no input", () => {
    // Calcite leaves `inputs` out when an operator reads the one declared
    // before it, so the chain is the list read backwards from the last.
    const tree = parseFixture(singleScan);
    expect(outline(tree)).toEqual([
      "0 HiveProject id, name, city",
      "0.0 HiveFilter =",
      // The alias the statement gave the table is the table's own name here,
      // so the card does not repeat it.
      "0.0.0 HiveTableScan default.customers",
    ]);
  });

  test("follows the inputs a join names, and reads its row counts", () => {
    const tree = parseFixture(joinAggregate);
    expect(outline(tree)).toEqual([
      "0 HiveProject city, _o__c1",
      "0.0 HiveAggregate count",
      "0.0.0 HiveProject $f0",
      "0.0.0.0 HiveJoin =",
      "0.0.0.0.0 HiveProject id",
      "0.0.0.0.0.0 HiveFilter AND",
      "0.0.0.0.0.0.0 HiveTableScan default.customers c",
      "0.0.0.0.1 HiveProject customer_id",
      "0.0.0.0.1.0 HiveFilter IS NOT NULL",
      "0.0.0.0.1.0.0 HiveTableScan default.orders o",
    ]);
    const scan = tree.nodes[6];
    expect(scan.rows).toBe(1);
    expect(scan.width).toBe(104);
    expect(propertyValue(scan, "Join type")).toBeUndefined();
    expect(propertyValue(tree.nodes[3], "Join type")).toBe("inner");
  });

  test("resolves a scan's table from the name Hive qualifies", () => {
    const tree = parseFixture(joinAggregate);
    expect(tree.nodes[6].relation).toEqual({
      database: "default",
      table: "customers",
    });
    expect(tree.root.relation).toBeUndefined();
  });

  test("draws both sides of a union", () => {
    const tree = parseFixture(unionAll);
    expect(
      tree.nodes.filter((node) => node.nodeType === "HiveTableScan")
    ).toHaveLength(2);
    expect(tree.root.children).toHaveLength(2);
  });

  test("reads a sort and its fetch", () => {
    const tree = parseFixture(orderLimit);
    expect(outline(tree).some((line) => line.includes("HiveSortLimit"))).toBe(
      true
    );
  });

  test("reads the row counts of tables Hive has analyzed", () => {
    // The same query against analyzed tables: the scan states the table's rows
    // and its average row width, and the filter above it states what the
    // optimizer expects its predicate to leave.
    const tree = parseFixture(analyzedJoin);
    const scan = tree.nodes.find(
      (node) => node.relation?.table === "customers_d"
    );
    expect(scan?.rows).toBe(5);
    expect(scan?.width).toBe(9.4);
    expect(tree.nodes[5].nodeType).toBe("HiveFilter");
    expect(tree.nodes[5].rows).toBeCloseTo(1.667, 3);
    expect(tree.maxRows).toBe(6);
  });

  test("ranks nothing, since only a join states a cost", () => {
    const tree = parseFixture(joinAggregate);
    expect(tree.estimates.cost).toBe(false);
    expect(tree.estimates.rows).toBe(true);
    // Hive states the cost it could not compute as words, which is why none of
    // them is read as a number. It says the same of an analyzed table, so the
    // numbers a reader gets are rows and widths.
    expect(propertyValue(tree.nodes[3], "Cost")).toBe("not available");
    const analyzed = parseFixture(analyzedJoin);
    expect(analyzed.estimates.cost).toBe(false);
    expect(
      propertyValue(
        analyzed.nodes.find((node) => node.nodeType === "HiveJoin") ??
          analyzed.root,
        "Cost"
      )
    ).toBe("not available");
  });

  test("keeps the column count rather than the column list", () => {
    const tree = parseFixture(singleScan);
    expect(propertyValue(tree.nodes[2], "Columns")).toBe("7");
    // The condition tree is the query's own syntax; the operator it applies is
    // what reads as a fact.
    expect(propertyValue(tree.nodes[1], "Condition")).toBe("=");
  });

  test("says so when the optimizer planned nothing", () => {
    // Hive states a CBO plan only when its cost-based optimizer ran; the stages
    // it leaves instead are not an operator tree.
    expect(parseHivePlan(JSON.stringify(noCboPlan))).toEqual({
      ok: false,
      message: HIVE_PLAN_NO_PLAN_MESSAGE,
    });
    expect(parseHivePlan(planOf([]))).toEqual({
      ok: false,
      message: HIVE_PLAN_NO_PLAN_MESSAGE,
    });
  });

  test("says so when there is no plan to read", () => {
    expect(parseHivePlan("")).toEqual({
      ok: false,
      message: HIVE_PLAN_EMPTY_MESSAGE,
    });
    expect(parseHivePlan("not json")).toEqual({
      ok: false,
      message: HIVE_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseHivePlan("[]")).toEqual({
      ok: false,
      message: HIVE_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseHivePlan(JSON.stringify({ CBOPlan: "not json" }))).toEqual({
      ok: false,
      message: HIVE_PLAN_INVALID_JSON_MESSAGE,
    });
  });

  test("stops at an operator that reads itself", () => {
    const tree = parse(
      planOf([
        {
          id: "0",
          relOp: "HiveTableScan",
          table: ["default", "t"],
          inputs: [],
        },
        { id: "1", relOp: "HiveJoin", inputs: ["0", "2"] },
        { id: "2", relOp: "HiveProject", inputs: ["1"] },
      ])
    );
    expect(outline(tree)).toEqual([
      "0 HiveProject -",
      "0.0 HiveJoin -",
      "0.0.0 HiveTableScan default.t",
    ]);
  });

  test("refuses a plan deeper than the viewer draws", () => {
    expect(parseHivePlan(nestedPlan(600))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
