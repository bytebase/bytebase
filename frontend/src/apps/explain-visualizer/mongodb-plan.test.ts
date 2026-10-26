import { describe, expect, test } from "vitest";
import {
  MONGODB_PLAN_EMPTY_MESSAGE,
  MONGODB_PLAN_INVALID_JSON_MESSAGE,
  MONGODB_PLAN_NO_PLAN_MESSAGE,
  parseMongodbPlan,
} from "./mongodb-plan";
import {
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import aggregateGroup from "./test-data/mongodb/aggregate-group.json";
import collectionScan from "./test-data/mongodb/collection-scan.json";
import indexScanSort from "./test-data/mongodb/index-scan-sort.json";
import measuredCollectionScan from "./test-data/mongodb/measured-collection-scan.json";

/** A chain of `depth` stages, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  let plan: Record<string, unknown> = { stage: "COLLSCAN" };
  for (let level = 1; level < depth; level += 1) {
    plan = { stage: "FETCH", inputStage: plan };
  }
  return JSON.stringify({ queryPlanner: { winningPlan: plan } });
};

const parse = (source: string): PlanTree => {
  const result = parseMongodbPlan(source);
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

describe("parseMongodbPlan", () => {
  test("draws the winning plan and flags a read of the whole collection", () => {
    const tree = parseFixture(collectionScan);

    expect(outline(tree)).toEqual(["0 COLLSCAN"]);
    expect(tree.root.subject).toBe('{"total":{"$gt":880}}');
    expect(tree.root.warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
    expect(propertyValue(tree.root, "Direction")).toBe("forward");
  });

  test("ranks nothing, because MongoDB's planner publishes no estimate", () => {
    const tree = parseFixture(indexScanSort);

    expect(tree.estimates).toEqual({
      cost: false,
      startupCost: false,
      rows: false,
      width: false,
    });
    expect(tree.measured).toEqual({ time: false, rows: false });
  });

  test("nests a scan under what consumes it, and names the index it reads", () => {
    const tree = parseFixture(indexScanSort);
    const scan = byId(tree, "0.0.0");

    expect(outline(tree)).toEqual(["0 SORT", "0.0 FETCH", "0.0.0 IXSCAN"]);
    expect(scan.subject).toBe("customer_id_1");
    expect(propertyValue(scan, "Index bounds")).toContain("[42, 42]");
    // An index read is not a read of the whole collection.
    expect(scan.warnings).toEqual([]);
  });

  test("answers only the stage that reaches the collection", () => {
    const tree = parseFixture(indexScanSort);

    // MongoDB states the namespace once, on the plan, so the stage that reads
    // it carries it; a sort reads its input rather than a collection.
    expect(byId(tree, "0.0.0").relation).toMatchObject({ table: "orders" });
    expect(tree.root.relation).toBeUndefined();
    expect(byId(tree, "0.0").relation).toBeUndefined();
  });

  test("reads an aggregation backwards, with its cursor's plan at the bottom", () => {
    const tree = parseFixture(aggregateGroup);

    // The pipeline runs first to last, so the stage producing the result is
    // the root and the $cursor feeding it becomes the query plan below.
    expect(outline(tree)).toEqual([
      "0 $group",
      "0.0 PROJECTION_SIMPLE",
      "0.0.0 COLLSCAN",
    ]);
    expect(tree.root.subject).toContain("$sum");
    expect(byId(tree, "0.0.0").warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
  });

  test("reads a measured plan for its rows and not its timings", () => {
    const tree = parseFixture(measuredCollectionScan);

    expect(tree.measured).toEqual({ rows: true, time: false });
    expect(tree.root.actualRows).toBe(145);
    expect(tree.root.actualTime).toBeUndefined();
    // MongoDB calls the per-stage time an estimate, and reports 0 on a query
    // that finishes quickly, so it stays a property.
    expect(
      propertyValue(tree.root, "Execution time millis estimate")
    ).toBeDefined();
    expect(propertyValue(tree.root, "Docs examined")).toBe("5000");
  });

  test("leaves a small collection scan unflagged", () => {
    const small = parse(
      JSON.stringify({
        executionStats: {
          executionStages: {
            stage: "COLLSCAN",
            docsExamined: 12,
            nReturned: 3,
          },
        },
      })
    );

    expect(small.root.warnings).toEqual([]);
  });

  test("hangs every branch of a stage that merges several", () => {
    const tree = parse(
      JSON.stringify({
        queryPlanner: {
          winningPlan: {
            stage: "OR",
            inputStages: [
              { stage: "IXSCAN", indexName: "a_1" },
              { stage: "IXSCAN", indexName: "b_1" },
            ],
          },
        },
      })
    );

    expect(outline(tree)).toEqual(["0 OR", "0.0 IXSCAN", "0.1 IXSCAN"]);
  });

  test("reports a plan it cannot read", () => {
    expect(parseMongodbPlan("  ")).toEqual({
      ok: false,
      message: MONGODB_PLAN_EMPTY_MESSAGE,
    });
    expect(parseMongodbPlan("not json")).toEqual({
      ok: false,
      message: MONGODB_PLAN_INVALID_JSON_MESSAGE,
    });
    // An ordinary document, which is what a query returns.
    expect(parseMongodbPlan('{"_id": 1, "total": 100}')).toEqual({
      ok: false,
      message: MONGODB_PLAN_NO_PLAN_MESSAGE,
    });
    expect(parseMongodbPlan("[]")).toEqual({
      ok: false,
      message: MONGODB_PLAN_INVALID_JSON_MESSAGE,
    });
  });

  test("gives up past the depth it can draw", () => {
    expect(parseMongodbPlan(nestedPlan(400)).ok).toBe(true);
    expect(parseMongodbPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
