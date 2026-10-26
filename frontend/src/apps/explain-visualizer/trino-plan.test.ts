import { describe, expect, test } from "vitest";
import {
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import joinAggregateFragments from "./test-data/trino/join-aggregate-fragments.json";
import scanFilterProject from "./test-data/trino/scan-filter-project.json";
import sortLimit from "./test-data/trino/sort-limit.json";
import {
  parseTrinoPlan,
  TRINO_PLAN_EMPTY_MESSAGE,
  TRINO_PLAN_INVALID_JSON_MESSAGE,
  TRINO_PLAN_NO_PLAN_MESSAGE,
} from "./trino-plan";

/** A chain of `depth` operators, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  let plan: Record<string, unknown> = { name: "TableScan" };
  for (let level = 1; level < depth; level += 1) {
    plan = { name: "Project", children: [plan] };
  }
  return JSON.stringify({ "0": plan });
};

const parse = (source: string): PlanTree => {
  const result = parseTrinoPlan(source);
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

describe("parseTrinoPlan", () => {
  test("reads a single-fragment plan", () => {
    const tree = parseFixture(scanFilterProject);
    const scan = byId(tree, "0.0");

    expect(outline(tree)).toEqual(["0 Output", "0.0 ScanFilterProject"]);
    // Trino fuses a scan, a filter and a projection into one operator and
    // estimates each step; the last of them is what leaves the operator.
    expect(scan.rows).toBe(6000);
    expect(tree.root.rows).toBe(6000);
  });

  test("splices the fragments a RemoteSource names into one tree", () => {
    const tree = parseFixture(joinAggregateFragments);

    // Three fragments, read as the one plan Trino would run across them.
    expect(outline(tree)).toEqual([
      "0 Output",
      "0.0 Aggregate",
      "0.0.0 LocalExchange",
      "0.0.0.0 RemoteSource",
      "0.0.0.0.0 Aggregate",
      "0.0.0.0.0.0 InnerJoin",
      "0.0.0.0.0.0.0 ScanFilter",
      "0.0.0.0.0.0.1 LocalExchange",
      "0.0.0.0.0.0.1.0 RemoteSource",
      "0.0.0.0.0.0.1.0.0 TableScan",
    ]);
    expect(propertyValue(byId(tree, "0.0.0.0"), "Source fragment ids")).toBe(
      "[1]"
    );
  });

  test("answers a scan with the catalog and schema Trino qualified it by", () => {
    const tree = parseFixture(joinAggregateFragments);

    expect(byId(tree, "0.0.0.0.0.0.0").relation).toEqual({
      database: "tpch",
      schema: "sf1",
      table: "orders",
    });
    expect(byId(tree, "0.0.0.0.0.0.1.0.0").relation).toEqual({
      database: "tpch",
      schema: "sf1",
      table: "customer",
    });
    // An operator that reads no table names none.
    expect(byId(tree, "0.0").relation).toBeUndefined();
  });

  test("reads rows and leaves the cost estimates unranked", () => {
    const tree = parseFixture(sortLimit);
    const scan = byId(tree, "0.0.0.0.0.0.0.0.0.0");

    expect(tree.estimates.rows).toBe(true);
    expect(tree.estimates.cost).toBe(false);
    expect(tree.costBasis).toBe(0);
    expect(scan.rows).toBe(1500000);
    for (const node of tree.nodes) {
      expect(node.totalCost).toBeUndefined();
      expect(node.selfCost).toBeUndefined();
    }
    // The numbers are still on the node, in Trino's own words.
    expect(propertyValue(scan, "Estimate")).toContain("Cpu cost:");
  });

  test("keeps an estimate Trino could not make off the row count", () => {
    const tree = parse(
      JSON.stringify({
        "0": {
          name: "Aggregate",
          estimates: [
            { outputRowCount: "NaN", cpuCost: "NaN", memoryCost: "NaN" },
          ],
        },
      })
    );

    expect(tree.root.rows).toBeUndefined();
    expect(propertyValue(tree.root, "Estimate")).toContain("NaN");
  });

  test("draws what it can of a plan whose fragments read each other", () => {
    // A fragment that reads itself would never finish.
    const tree = parse(
      JSON.stringify({
        "0": {
          name: "Output",
          children: [
            { name: "RemoteSource", descriptor: { sourceFragmentIds: "[1]" } },
          ],
        },
        "1": {
          name: "Aggregate",
          children: [
            { name: "RemoteSource", descriptor: { sourceFragmentIds: "[0]" } },
          ],
        },
      })
    );

    expect(outline(tree)).toEqual([
      "0 Output",
      "0.0 RemoteSource",
      "0.0.0 Aggregate",
      "0.0.0.0 RemoteSource",
    ]);
  });

  test("reports a plan it cannot read", () => {
    expect(parseTrinoPlan("  ")).toEqual({
      ok: false,
      message: TRINO_PLAN_EMPTY_MESSAGE,
    });
    expect(parseTrinoPlan("not json")).toEqual({
      ok: false,
      message: TRINO_PLAN_INVALID_JSON_MESSAGE,
    });
    // Trino's text plan, which is what every other EXPLAIN of its returns.
    expect(parseTrinoPlan("Fragment 0 [SINGLE]")).toEqual({
      ok: false,
      message: TRINO_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseTrinoPlan("{}")).toEqual({
      ok: false,
      message: TRINO_PLAN_NO_PLAN_MESSAGE,
    });
  });

  test("gives up past the depth it can draw", () => {
    expect(parseTrinoPlan(nestedPlan(400)).ok).toBe(true);
    expect(parseTrinoPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
