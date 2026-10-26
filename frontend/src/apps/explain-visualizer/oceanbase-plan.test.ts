import { describe, expect, test } from "vitest";
import {
  OCEANBASE_PLAN_EMPTY_MESSAGE,
  OCEANBASE_PLAN_INVALID_JSON_MESSAGE,
  OCEANBASE_PLAN_NO_PLAN_MESSAGE,
  parseOceanbasePlan,
} from "./oceanbase-plan";
import {
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import fullScanFilter from "./test-data/oceanbase/full-scan-filter.json";
import hashJoinGroupSort from "./test-data/oceanbase/hash-join-group-sort.json";
import indexLookup from "./test-data/oceanbase/index-lookup.json";
import unionDistinct from "./test-data/oceanbase/union-distinct.json";

/** A chain of `depth` operators, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  let plan: Record<string, unknown> = { OPERATOR: "TABLE FULL SCAN" };
  for (let level = 1; level < depth; level += 1) {
    plan = { OPERATOR: "SUBPLAN SCAN", CHILD_1: plan };
  }
  return JSON.stringify(plan);
};

const parse = (source: string): PlanTree => {
  const result = parseOceanbasePlan(source);
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

describe("parseOceanbasePlan", () => {
  test("reads the operator tree, trimming the names OceanBase pads", () => {
    const tree = parseFixture(fullScanFilter);

    expect(outline(tree)).toEqual(["0 TABLE FULL SCAN orders"]);
    expect(tree.root.relation).toEqual({ table: "orders" });
    expect(tree.root.warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
    expect(propertyValue(tree.root, "Output")).toContain("orders.total");
  });

  test("hangs children off the CHILD_n keys, in the order they are numbered", () => {
    expect(outline(parseFixture(hashJoinGroupSort))).toEqual([
      "0 SORT -",
      "0.0 HASH GROUP BY -",
      "0.0.0 HASH JOIN -",
      "0.0.0.0 TABLE FULL SCAN c",
      "0.0.0.1 TABLE FULL SCAN o",
    ]);
    expect(outline(parseFixture(unionDistinct))[0]).toBe(
      "0 MERGE UNION DISTINCT -"
    );
  });

  test("ranks by the time the optimizer expects, which covers each subtree", () => {
    const tree = parseFixture(hashJoinGroupSort);
    const join = byId(tree, "0.0.0");
    const scans = [byId(tree, "0.0.0.0"), byId(tree, "0.0.0.1")];
    const scanned = scans.reduce((sum, scan) => sum + (scan.totalCost ?? 0), 0);

    // EST.TIME(us) covers an operator and everything below it, so the join's
    // own share is what it adds on top of its two reads.
    expect(join.totalCost).toBeGreaterThan(scanned);
    expect(join.selfCost).toBe((join.totalCost ?? 0) - scanned);
    expect(tree.estimates.cost).toBe(true);
    expect(tree.estimates.rows).toBe(true);
  });

  test("leaves an operator that reads no object without a table", () => {
    const tree = parseFixture(hashJoinGroupSort);

    expect(byId(tree, "0.0.0").relation).toBeUndefined();
    // The name is the one the statement used, which is the alias it gave.
    expect(byId(tree, "0.0.0.0").relation).toEqual({ table: "c" });
  });

  test("does not flag a range scan, or a small full read", () => {
    const tree = parseFixture(indexLookup);
    expect(tree.root.nodeType).toBe("TABLE RANGE SCAN");
    expect(tree.root.warnings).toEqual([]);

    const small = parse(
      JSON.stringify({
        OPERATOR: "TABLE FULL SCAN",
        NAME: "countries",
        "EST.ROWS": 200,
      })
    );
    expect(small.root.warnings).toEqual([]);
  });

  test("reports a plan it cannot read", () => {
    expect(parseOceanbasePlan("  ")).toEqual({
      ok: false,
      message: OCEANBASE_PLAN_EMPTY_MESSAGE,
    });
    expect(parseOceanbasePlan("not json")).toEqual({
      ok: false,
      message: OCEANBASE_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseOceanbasePlan("{}")).toEqual({
      ok: false,
      message: OCEANBASE_PLAN_NO_PLAN_MESSAGE,
    });
    // The tabular plan every other OceanBase EXPLAIN returns.
    expect(parseOceanbasePlan("| ID | OPERATOR | NAME |")).toEqual({
      ok: false,
      message: OCEANBASE_PLAN_INVALID_JSON_MESSAGE,
    });
  });

  test("gives up past the depth it can draw", () => {
    expect(parseOceanbasePlan(nestedPlan(400)).ok).toBe(true);
    expect(parseOceanbasePlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
