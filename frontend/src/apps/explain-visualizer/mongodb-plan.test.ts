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
import measuredAggregateGroup from "./test-data/mongodb/measured-aggregate-group.json";
import measuredCollectionScan from "./test-data/mongodb/measured-collection-scan.json";
import measuredShardedIndexScan from "./test-data/mongodb/measured-sharded-index-scan.json";
import measuredSlotBasedJoin from "./test-data/mongodb/measured-slot-based-join.json";
import shardedAggregateGroup from "./test-data/mongodb/sharded-aggregate-group.json";
import shardedIndexScan from "./test-data/mongodb/sharded-index-scan.json";
import slotBasedIndexScan from "./test-data/mongodb/slot-based-index-scan.json";

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

  test("keeps what a measured pipeline stage returned and spent", () => {
    const tree = parseFixture(measuredAggregateGroup);
    const group = byId(tree, "0.0");

    expect(outline(tree)).toEqual([
      "0 $sort",
      "0.0 $group",
      "0.0.0 PROJECTION_SIMPLE",
      "0.0.0.0 COLLSCAN",
    ]);
    // A stage states these beside itself rather than inside its body, so
    // reading the body alone loses both the rows and the spill diagnostics.
    expect(group.actualRows).toBe(500);
    expect(propertyValue(group, "Used disk")).toBe("false");
    expect(propertyValue(group, "Spilled records")).toBe("0");
    // The read under the pipeline keeps reporting its own.
    expect(byId(tree, "0.0.0.0").actualRows).toBe(5000);
  });

  test("draws each shard's own plan under the merge that reads them", () => {
    const tree = parseFixture(shardedIndexScan);

    // A sharded collection is planned a shard at a time, and the merge holds
    // each shard's plan under `shards` rather than as an input stage.
    expect(outline(tree)).toEqual([
      "0 SHARD_MERGE",
      "0.0 SHARDING_FILTER",
      "0.0.0 FETCH",
      "0.0.0.0 IXSCAN",
      "0.1 SHARDING_FILTER",
      "0.1.0 FETCH",
      "0.1.0.0 IXSCAN",
    ]);
    expect(propertyValue(byId(tree, "0.0"), "Shard")).toBe("sh1");
    expect(propertyValue(byId(tree, "0.1"), "Shard")).toBe("sh2");
    // Each shard states the namespace it read, so its scans answer a lookup.
    expect(byId(tree, "0.0.0.0").relation).toEqual({
      database: "qp",
      table: "orders",
    });
    // The merge reads its shards, not the collection.
    expect(tree.root.relation).toBeUndefined();
  });

  test("reads what each shard returned from a measured sharded plan", () => {
    const tree = parseFixture(measuredShardedIndexScan);

    expect(tree.root.actualRows).toBe(2785);
    expect(byId(tree, "0.0").actualRows).toBe(1420);
    expect(byId(tree, "0.1").actualRows).toBe(1365);
    // A shard's execution stages state no namespace of their own, so it comes
    // from the planner's shards, which name the one collection they all read.
    expect(byId(tree, "0.1.0.0").relation).toEqual({
      database: "qp",
      table: "orders",
    });
  });

  test("follows a slot-based join to both of the sides it reads", () => {
    const tree = parseFixture(measuredSlotBasedJoin);

    // The slot-based engine spells a join's two sides `outerStage` and
    // `innerStage`, so following `inputStage` alone stops at the join and
    // leaves the seek that does the reading inside a property.
    expect(outline(tree)).toEqual([
      "0 filter",
      "0.0 nlj",
      "0.0.0 ixseek",
      "0.0.1 limit",
      "0.0.1.0 seek",
    ]);
    expect(byId(tree, "0.0").actualRows).toBe(5000);
  });

  test("draws a pipeline split across shards under the stage that merges it", () => {
    const tree = parseFixture(shardedAggregateGroup);

    // A split pipeline states no `stages` of its own: the merger's part is
    // the plan, and each shard's part runs under `$mergeCursors`.
    expect(outline(tree).slice(0, 4)).toEqual([
      "0 $sort",
      "0.0 $group",
      "0.0.0 $mergeCursors",
      "0.0.0.0 $group",
    ]);
    expect(propertyValue(byId(tree, "0.0.0.0"), "Shard")).toBe("sh1");
    expect(propertyValue(byId(tree, "0.0.0.1"), "Shard")).toBe("sh2");
    // Each shard's own read is drawn, down to the collection it scanned.
    expect(byId(tree, "0.0.0.0.0.0.0").relation).toEqual({
      database: "qp",
      table: "orders",
    });
  });

  test("reads the tree the slot-based engine wraps", () => {
    // MongoDB's slot-based engine reports the operator tree under
    // `winningPlan.queryPlan`, beside the plan it compiled for itself; the
    // classic engine puts the tree directly under `winningPlan`. Taking the
    // wrapper for the tree draws one nameless node and hides the read.
    const tree = parseFixture(slotBasedIndexScan);

    expect(outline(tree)).toEqual(["0 FETCH", "0.0 IXSCAN"]);
    expect(propertyValue(byId(tree, "0.0"), "Index name")).toBe("customerId_1");
    expect(tree.root.warnings).toEqual([]);
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
