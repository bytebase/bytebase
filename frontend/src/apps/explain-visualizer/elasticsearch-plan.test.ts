import { describe, expect, test } from "vitest";
import {
  ELASTICSEARCH_PLAN_EMPTY_MESSAGE,
  ELASTICSEARCH_PLAN_INVALID_JSON_MESSAGE,
  ELASTICSEARCH_PLAN_NO_PLAN_MESSAGE,
  parseElasticsearchPlan,
} from "./elasticsearch-plan";
import {
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import boolAggregation from "./test-data/elasticsearch/bool-aggregation.json";
import multiShard from "./test-data/elasticsearch/multi-shard.json";

/** A profile of one shard whose query nests `depth` deep. */
const nestedPlan = (depth: number): string => {
  let query: Record<string, unknown> = {
    type: "TermQuery",
    time_in_nanos: 1000,
  };
  for (let level = 1; level < depth; level += 1) {
    query = {
      type: "BooleanQuery",
      time_in_nanos: 1000,
      children: [query],
    };
  }
  return JSON.stringify({
    profile: { shards: [{ id: "[n][i][0]", searches: [{ query: [query] }] }] },
  });
};

const parse = (source: string): PlanTree => {
  const result = parseElasticsearchPlan(source);
  if (!result.ok)
    throw new Error(`expected a parsable plan: ${result.message}`);
  return result.tree;
};

const parseFixture = (fixture: unknown): PlanTree =>
  parse(JSON.stringify(fixture));

const outline = (tree: PlanTree): string[] =>
  tree.nodes.map(
    (node) =>
      `${node.id} ${node.nodeType}${node.relationship ? ` (${node.relationship})` : ""}`
  );

const propertyValue = (node: PlanNode, label: string): string | undefined =>
  node.properties.find((property) => property.label === label)?.value;

describe("parseElasticsearchPlan", () => {
  test("puts each shard's phases under it, saying which phase they are", () => {
    const tree = parseFixture(boolAggregation);
    expect(outline(tree)).toEqual([
      "0 Search",
      "0.0 Shard",
      "0.0.0 BooleanQuery (Query)",
      "0.0.0.0 TermQuery",
      "0.0.0.1 IndexOrDocValuesQuery",
      "0.0.1 QueryPhaseCollector (Collector)",
      "0.0.1.0 SimpleTopScoreDocCollector",
      // A collector per aggregation, which is how the terms agg gathered.
      "0.0.1.1 AggregatorCollector: [by_city]",
      "0.0.2 GlobalOrdinalsStringTermsAggregator (Aggregation)",
      // The avg nested inside the terms aggregation.
      "0.0.2.0 AvgAggregator",
      "0.0.3 fetch (Fetch)",
      "0.0.3.0 FetchFieldsPhase",
      "0.0.3.1 FetchSourcePhase",
      "0.0.3.2 StoredFieldsPhase",
    ]);
  });

  test("is measured in milliseconds, with each node's own time apart", () => {
    const tree = parseFixture(boolAggregation);
    // Elasticsearch times in nanoseconds and covers the subtree, which is what
    // the model's actual time means.
    const query = tree.nodes[2];
    expect(query.actualTime).toBeCloseTo(16.279942, 5);
    expect(tree.nodes[3].actualTime).toBeCloseTo(5.147482, 5);
    // 16.28ms over children totalling 9.88ms leaves the query's own 6.4ms.
    expect(query.selfTime).toBeCloseTo(6.397715, 5);
    expect(tree.measured.time).toBe(true);
    expect(tree.estimates.cost).toBe(false);
    expect(tree.estimates.rows).toBe(false);
  });

  test("carries the search's own time on the node that holds the shards", () => {
    const tree = parseFixture(boolAggregation);
    expect(tree.root.nodeType).toBe("Search");
    expect(tree.root.actualTime).toBe(102);
    expect(propertyValue(tree.root, "Took ms")).toBe("102");
    expect(propertyValue(tree.root, "Shards")).toBe("1");
  });

  test("resolves the index a shard read, which is a table here", () => {
    const tree = parseFixture(boolAggregation);
    expect(tree.nodes[1].relation).toEqual({ table: "customers" });
    expect(tree.nodes[1].subject).toBe(
      "[xBvstVRMRUOqYLT8Rvlu8w][customers][0]"
    );
    expect(propertyValue(tree.nodes[1], "Shard id")).toBe("0");
  });

  test("keeps the breakdown that says where a node spent its time", () => {
    const tree = parseFixture(boolAggregation);
    const query = tree.nodes[2];
    expect(propertyValue(query, "Next doc")).toBe("66149");
    expect(propertyValue(query, "Score count")).toBe("16");
    // The description is the Lucene query, which is the subject of the card.
    expect(query.subject).toContain("+city:berlin");
  });

  test("draws every shard of a search that touched several", () => {
    const tree = parseFixture(multiShard);
    const shards = tree.nodes.filter((node) => node.nodeType === "Shard");
    expect(shards).toHaveLength(3);
    expect(propertyValue(tree.root, "Shards")).toBe("3");
    // Each names the same index, so the search names it once.
    expect(tree.root.subject).toBe("events3");
    expect(
      new Set(shards.map((shard) => propertyValue(shard, "Shard id")))
    ).toEqual(new Set(["0", "1", "2"]));
  });

  test("says so when there is no profile to read", () => {
    expect(parseElasticsearchPlan("")).toEqual({
      ok: false,
      message: ELASTICSEARCH_PLAN_EMPTY_MESSAGE,
    });
    expect(parseElasticsearchPlan("not json")).toEqual({
      ok: false,
      message: ELASTICSEARCH_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseElasticsearchPlan("[]")).toEqual({
      ok: false,
      message: ELASTICSEARCH_PLAN_INVALID_JSON_MESSAGE,
    });
    // A search that profiled nothing, and a response with no profile at all.
    expect(
      parseElasticsearchPlan(JSON.stringify({ profile: { shards: [] } }))
    ).toEqual({ ok: false, message: ELASTICSEARCH_PLAN_NO_PLAN_MESSAGE });
    expect(parseElasticsearchPlan(JSON.stringify({ took: 3 }))).toEqual({
      ok: false,
      message: ELASTICSEARCH_PLAN_NO_PLAN_MESSAGE,
    });
  });

  test("reads a bare profile, without the response around it", () => {
    const tree = parse(
      JSON.stringify({ shards: [{ id: "[n][i][0]", index: "i" }] })
    );
    expect(tree.root.nodeType).toBe("Search");
    expect(tree.nodes[1].relation).toEqual({ table: "i" });
  });

  test("refuses a plan deeper than the viewer draws", () => {
    expect(parseElasticsearchPlan(nestedPlan(600))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
