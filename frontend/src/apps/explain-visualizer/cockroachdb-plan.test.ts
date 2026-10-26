import { describe, expect, test } from "vitest";
import {
  COCKROACHDB_PLAN_EMPTY_MESSAGE,
  COCKROACHDB_PLAN_INVALID_JSON_MESSAGE,
  COCKROACHDB_PLAN_NO_PLAN_MESSAGE,
  parseCockroachdbPlan,
} from "./cockroachdb-plan";
import {
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import indexJoin from "./test-data/cockroachdb/index-join.json";
import scanFilterGroup from "./test-data/cockroachdb/scan-filter-group.json";
import sortLimit from "./test-data/cockroachdb/sort-limit.json";

/** A chain of `depth` processors, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  const processors = Array.from({ length: depth }, (_, index) => ({
    core: { title: `Processor/${index}`, details: [] },
  }));
  const edges = Array.from({ length: depth - 1 }, (_, index) => ({
    sourceProc: index,
    destProc: index + 1,
  }));
  return JSON.stringify({ processors, edges });
};

const parse = (source: string): PlanTree => {
  const result = parseCockroachdbPlan(source);
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

describe("parseCockroachdbPlan", () => {
  test("roots the flow at the processor nothing reads from", () => {
    const tree = parseFixture(scanFilterGroup);

    // The streams run from the read upwards; Response is where the query's
    // rows come out, so it is the root and the read is the leaf.
    expect(outline(tree)).toEqual([
      "0 Response",
      "0.0 Aggregator",
      "0.0.0 Filterer",
      "0.0.0.0 TableReader",
    ]);
    expect(propertyValue(byId(tree, "0.0.0"), "Filter")).toBe(
      "@2 > 880:::INT8"
    );
  });

  test("ranks nothing, because the diagram carries no estimate", () => {
    const tree = parseFixture(scanFilterGroup);

    expect(tree.estimates).toEqual({
      cost: false,
      startupCost: false,
      rows: false,
      width: false,
    });
    expect(tree.measured).toEqual({ time: false, rows: false });
  });

  test("names the stored table a read covers, not the statement's alias", () => {
    const tree = parseFixture(indexJoin);

    // The statement aliased both tables; CockroachDB names neither alias.
    expect(byId(tree, "0.0.0").relation).toEqual({ table: "orders" });
    expect(byId(tree, "0.0").relation).toEqual({ table: "customers" });
    expect(byId(tree, "0.0.0").subject).toBe("orders@idx_customer");
  });

  test("flags a read whose spans cover everything, and leaves a range alone", () => {
    expect(byId(parseFixture(sortLimit), "0.0.0").warnings).toEqual([
      PLAN_FULL_TABLE_SCAN,
    ]);
    // `/!NULL-/10` is a bounded read of the index.
    const bounded = byId(parseFixture(indexJoin), "0.0.0");
    expect(propertyValue(bounded, "Spans")).toBe("/!NULL-/10");
    expect(bounded.warnings).toEqual([]);
  });

  test("keeps a detail whose colon is part of an expression", () => {
    const join = byId(parseFixture(indexJoin), "0.0");

    // `ON @3 < 10:::INT8` carries a cast, not a label, so splitting it at the
    // colon would cut the condition in half.
    expect(propertyValue(join, "Detail")).toContain("ON @3 < 10:::INT8");
    expect(propertyValue(join, "Lookup join on")).toBe("@2");
  });

  test("hangs both inputs of a processor that joins two streams", () => {
    const tree = parse(
      JSON.stringify({
        processors: [
          { core: { title: "TableReader/0", details: ["a@a_pkey"] } },
          { core: { title: "TableReader/1", details: ["b@b_pkey"] } },
          { core: { title: "HashJoiner/2", details: [] } },
        ],
        edges: [
          { sourceProc: 0, destProc: 2, destInput: 0 },
          { sourceProc: 1, destProc: 2, destInput: 1 },
        ],
      })
    );

    expect(outline(tree)).toEqual([
      "0 HashJoiner",
      "0.0 TableReader",
      "0.1 TableReader",
    ]);
  });

  test("draws what it can of a flow whose streams form a cycle", () => {
    const tree = parse(
      JSON.stringify({
        processors: [
          { core: { title: "Response/-1", details: [] } },
          { core: { title: "Sorter/1", details: [] } },
        ],
        // The sorter reads itself, which a walk would follow forever.
        edges: [
          { sourceProc: 1, destProc: 0 },
          { sourceProc: 1, destProc: 1 },
        ],
      })
    );

    expect(outline(tree)).toEqual(["0 Response", "0.0 Sorter"]);
  });

  test("reports a plan it cannot read", () => {
    expect(parseCockroachdbPlan("  ")).toEqual({
      ok: false,
      message: COCKROACHDB_PLAN_EMPTY_MESSAGE,
    });
    expect(parseCockroachdbPlan("not json")).toEqual({
      ok: false,
      message: COCKROACHDB_PLAN_INVALID_JSON_MESSAGE,
    });
    // The text plan, which every other EXPLAIN of CockroachDB's returns.
    expect(parseCockroachdbPlan("• group (hash)")).toEqual({
      ok: false,
      message: COCKROACHDB_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseCockroachdbPlan('{"edges": []}')).toEqual({
      ok: false,
      message: COCKROACHDB_PLAN_NO_PLAN_MESSAGE,
    });
  });

  test("gives up past the depth it can draw", () => {
    expect(parseCockroachdbPlan(nestedPlan(400)).ok).toBe(true);
    expect(parseCockroachdbPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
