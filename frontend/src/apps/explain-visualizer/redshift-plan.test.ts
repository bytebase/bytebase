import { describe, expect, test } from "vitest";
import {
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import {
  parseRedshiftPlan,
  REDSHIFT_PLAN_EMPTY_MESSAGE,
  REDSHIFT_PLAN_NO_PLAN_MESSAGE,
} from "./redshift-plan";
import hashAggregate from "./test-data/redshift/hash-aggregate.txt?raw";
import hashJoinBroadcast from "./test-data/redshift/hash-join-broadcast.txt?raw";
import postgresSameGrammar from "./test-data/redshift/postgres-same-grammar.txt?raw";
import scanFilter from "./test-data/redshift/scan-filter.txt?raw";
import threeWayJoin from "./test-data/redshift/three-way-join.txt?raw";

/** A chain of `depth` operators, which is what drives the depth limit. */
const nestedPlan = (depth: number): string =>
  Array.from({ length: depth }, (_, level) =>
    level === 0
      ? "XN Limit  (cost=0.00..1.00 rows=1 width=1)"
      : `${" ".repeat(level * 2)}->  XN Nested Loop  (cost=0.00..1.00 rows=1 width=1)`
  ).join("\n");

const parse = (source: string): PlanTree => {
  const result = parseRedshiftPlan(source);
  if (!result.ok)
    throw new Error(`expected a parsable plan: ${result.message}`);
  return result.tree;
};

const outline = (tree: PlanTree): string[] =>
  tree.nodes.map((node) => `${node.id} ${node.nodeType}`);

const propertyValue = (node: PlanNode, label: string): string | undefined =>
  node.properties.find((property) => property.label === label)?.value;

const byId = (tree: PlanTree, id: string): PlanNode => {
  const node = tree.nodes.find((candidate) => candidate.id === id);
  if (!node) throw new Error(`no node ${id} in ${outline(tree).join(", ")}`);
  return node;
};

describe("parseRedshiftPlan", () => {
  test("builds the tree from how far each operator is indented", () => {
    const tree = parse(hashJoinBroadcast);

    expect(outline(tree)).toEqual([
      "0 Hash Join",
      "0.0 Seq Scan",
      "0.1 Hash",
      "0.1.0 Seq Scan",
    ]);
    // Redshift indents the whole plan by one space in this one, so the depths
    // are relative to each other rather than to the left margin.
    expect(byId(tree, "0.1.0").relation).toEqual({ table: "category" });
  });

  test("reads cost, rows and width, and charges each operator its own", () => {
    const tree = parse(hashJoinBroadcast);
    const join = tree.root;
    const scan = byId(tree, "0.0");

    // Cost covers an operator and everything below it, as PostgreSQL's does.
    expect(join).toMatchObject({
      startupCost: 0.14,
      totalCost: 6600286.07,
      rows: 8798,
      width: 84,
    });
    expect(scan.selfCost).toBe(87.98);
    expect(join.selfCost).toBeCloseTo(6600286.07 - 87.98 - 0.11, 2);
  });

  test("names the table a scan reads, and not the statement's alias", () => {
    const tree = parse(threeWayJoin);

    expect(byId(tree, "0.0.0").relation).toEqual({ table: "listing" });
    expect(byId(tree, "0.0.1").relation).toEqual({ table: "sales" });
    // A join reads its inputs, not a table.
    expect(tree.root.relation).toBeUndefined();
  });

  test("flags a join that has to move rows between slices", () => {
    const broadcast = parse(hashJoinBroadcast).root;
    const merge = byId(parse(threeWayJoin), "0.0");

    expect(propertyValue(broadcast, "Distribution")).toBe("DS_BCAST_INNER");
    expect(broadcast.warnings.map((warning) => warning.title)).toEqual([
      "Data redistribution",
    ]);
    // DS_DIST_NONE is the join that did not have to move anything.
    expect(propertyValue(merge, "Distribution")).toBe("DS_DIST_NONE");
    expect(merge.warnings).toEqual([]);
  });

  test("keeps what an operator says about itself", () => {
    expect(propertyValue(parse(scanFilter).root, "Filter")).toBe(
      "(venueseats IS NOT NULL)"
    );
    expect(outline(parse(hashAggregate))).toEqual([
      "0 HashAggregate",
      "0.0 Seq Scan",
    ]);
  });

  test("reads a PostgreSQL text plan, which is the grammar Redshift inherited", () => {
    // Captured from a real PostgreSQL server. Redshift cannot be run from this
    // repository's tests, so this is what pins the shape of the grammar to
    // output a server actually produced rather than to one written by hand.
    const tree = parse(postgresSameGrammar);

    expect(outline(tree)).toEqual([
      "0 Limit",
      "0.0 Sort",
      "0.0.0 HashAggregate",
      "0.0.0.0 Hash Join",
      "0.0.0.0.0 Seq Scan",
      "0.0.0.0.1 Hash",
      "0.0.0.0.1.0 Seq Scan",
    ]);
    expect(byId(tree, "0.0.0.0.0").relation).toEqual({ table: "orders" });
    expect(propertyValue(byId(tree, "0.0.0.0.0"), "Filter")).toBe(
      "(total > 880)"
    );
  });

  test("does not take a line inside a value for an operator", () => {
    // The hazard §4 of the design names: a text plan has no escaping, so a
    // literal holding a newline puts its own lines into the output. An
    // operator is recognized only by a whole cost tuple, which is what a value
    // would have to reproduce to be mistaken for one.
    const tree = parse(
      [
        "XN Seq Scan on venue  (cost=0.00..2.02 rows=187 width=45)",
        "  Filter: (name = 'line one",
        "  ->  XN Seq Scan on secrets  (cost=1.00)",
        "  ->  XN Seq Scan on secrets  rows=1',)",
      ].join("\n")
    );

    expect(outline(tree)).toEqual(["0 Seq Scan"]);
    expect(tree.root.relation).toEqual({ table: "venue" });
  });

  test("reports a plan it cannot read", () => {
    expect(parseRedshiftPlan("   ")).toEqual({
      ok: false,
      message: REDSHIFT_PLAN_EMPTY_MESSAGE,
    });
    expect(parseRedshiftPlan("Filter: (venueseats IS NOT NULL)")).toEqual({
      ok: false,
      message: REDSHIFT_PLAN_NO_PLAN_MESSAGE,
    });
    // JSON, which is what every other engine here returns.
    expect(parseRedshiftPlan('{"Plan": {}}')).toEqual({
      ok: false,
      message: REDSHIFT_PLAN_NO_PLAN_MESSAGE,
    });
  });

  test("gives up past the depth it can draw", () => {
    expect(parseRedshiftPlan(nestedPlan(400)).ok).toBe(true);
    expect(parseRedshiftPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
