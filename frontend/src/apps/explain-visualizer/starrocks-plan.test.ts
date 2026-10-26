import { describe, expect, test } from "vitest";
import {
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import {
  parseStarrocksPlan,
  STARROCKS_PLAN_EMPTY_MESSAGE,
  STARROCKS_PLAN_NO_PLAN_MESSAGE,
} from "./starrocks-plan";
import dorisAnalyzedJoin from "./test-data/doris/analyzed-join.txt?raw";
import dorisJoinExchange from "./test-data/doris/join-exchange.txt?raw";
import dorisScanAggregate from "./test-data/doris/scan-aggregate.txt?raw";
import dorisSingleScan from "./test-data/doris/single-scan.txt?raw";
import joinExchange from "./test-data/starrocks/join-exchange.txt?raw";
import nestedJoinBranch from "./test-data/starrocks/nested-join-branch.txt?raw";
import scanAggregate from "./test-data/starrocks/scan-aggregate.txt?raw";
import singleScan from "./test-data/starrocks/single-scan.txt?raw";

/** A chain of `depth` operators in one fragment, which drives the depth limit. */
const nestedPlan = (depth: number): string =>
  [
    "PLAN FRAGMENT 0",
    ...Array.from({ length: depth }, (_, level) => `  ${level}:Project`),
  ].join("\n");

const parse = (source: string): PlanTree => {
  const result = parseStarrocksPlan(source);
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

describe("parseStarrocksPlan", () => {
  test("joins the fragments where one sends to the other's exchange", () => {
    const tree = parse(singleScan);

    // Two fragments: the scan sends to exchange 1, which fragment 0 reads.
    expect(outline(tree)).toEqual(["0 EXCHANGE", "0.0 OlapScanNode"]);
    expect(byId(tree, "0.0").relation).toEqual({ table: "orders" });
  });

  test("reads what a scan states about the rows it expects", () => {
    const scan = byId(parse(singleScan), "0.0");

    expect(scan).toMatchObject({ rows: 2500, width: 4 });
    expect(propertyValue(scan, "Predicates")).toBe("3: total > 880");
    expect(propertyValue(scan, "Preaggregation")).toBe("ON");
    // StarRocks states no cost in the plan the editor asks for.
    expect(parse(singleScan).estimates.cost).toBe(false);
  });

  test("follows the chain through every fragment of an aggregation", () => {
    expect(outline(parse(scanAggregate))).toEqual([
      "0 EXCHANGE",
      "0.0 AGGREGATE (merge finalize)",
      "0.0.0 EXCHANGE",
      "0.0.0.0 AGGREGATE (update serialize)",
      "0.0.0.0.0 OlapScanNode",
    ]);
  });

  test("hangs both inputs of a join off it", () => {
    const tree = parse(joinExchange);
    const join = byId(tree, "0.0.0.0.0.0");

    expect(join.nodeType).toBe("HASH JOIN");
    expect(propertyValue(join, "Join op")).toBe("INNER JOIN (PARTITIONED)");
    expect(join.children).toHaveLength(2);
    // Each side reaches its own table through an exchange.
    expect(byId(tree, "0.0.0.0.0.0.0.0").relation).toEqual({
      table: "customers",
    });
    expect(byId(tree, "0.0.0.0.0.0.1.0").relation).toEqual({ table: "orders" });
  });

  test("reads a branch drawn inside another, bars and all", () => {
    const tree = parse(nestedJoinBranch);
    const inner = byId(tree, "0.0.0.0.0.0.0.0.0");

    // StarRocks draws a second input under `|----` and continues it behind a
    // bar, so which operator a line belongs to is the column it starts at.
    expect(inner.nodeType).toBe("HASH JOIN");
    expect(inner.children.map((child) => child.nodeType)).toEqual([
      "Project",
      "OlapScanNode",
    ]);
    expect(byId(tree, "0.0.0.0.0.0.0.0.0.0.0").relation).toEqual({
      table: "customers",
    });
    expect(
      propertyValue(byId(tree, "0.0.0.0.0.0.0.0.0.0.0"), "Predicates")
    ).toContain("country = 'JP'");
  });

  test("does not take a line inside a value for an operator", () => {
    // A value holding a newline puts its own lines into the plan. An operator
    // is its number and name at the head of a line, which a predicate's
    // continuation is not.
    const tree = parse(
      [
        "PLAN FRAGMENT 0",
        "  0:OlapScanNode",
        "     TABLE: venue",
        "     PREDICATES: 1: name = 'line one",
        "  1:OlapScanNode",
        "     TABLE: secrets'",
      ].join("\n")
    );

    expect(outline(tree)).toEqual(["0 OlapScanNode", "0.0 OlapScanNode"]);
    // The faked line is still read as an operator; what it cannot do is claim
    // a table the statement never named, because the relation comes from the
    // line after it.
    expect(tree.root.relation).toEqual({ table: "venue" });
  });

  test("reports a plan it cannot read", () => {
    expect(parseStarrocksPlan("  ")).toEqual({
      ok: false,
      message: STARROCKS_PLAN_EMPTY_MESSAGE,
    });
    expect(parseStarrocksPlan("PLAN FRAGMENT 0\n  RESULT SINK")).toEqual({
      ok: false,
      message: STARROCKS_PLAN_NO_PLAN_MESSAGE,
    });
    expect(parseStarrocksPlan('{"Plan": {}}')).toEqual({
      ok: false,
      message: STARROCKS_PLAN_NO_PLAN_MESSAGE,
    });
  });

  test("reads Doris, which spells the same plan its own way", () => {
    const tree = parse(dorisSingleScan);
    const scan = byId(tree, "0.0");

    // Doris prefixes the vectorized operators with V, and prints the
    // planner's own id after the name.
    expect(outline(tree)).toEqual(["0 VEXCHANGE", "0.0 VOlapScanNode"]);
    expect(propertyValue(scan, "Plan node id")).toBe("85");
    // It names the table qualified, and then what the statement called it.
    expect(scan.relation).toEqual({ database: "qp", table: "orders" });
  });

  test("splits the attributes Doris packs onto one line", () => {
    const scan = byId(parse(dorisScanAggregate), "0.0.0.0.0");

    expect(propertyValue(scan, "Cardinality")).toBe("1");
    expect(propertyValue(scan, "Num nodes")).toBe("1");
    expect(propertyValue(scan, "Preaggregation")).toBe("ON");
    // A value with a space in it is one value, not several attributes: the
    // tablets stay on their own line and the predicate stays whole.
    expect(propertyValue(scan, "Tablet list")).toBe(
      "1793041799804,1793041799806"
    );
    expect(propertyValue(scan, "Predicates")).toBe("(total[#2] > 880)");
    expect(propertyValue(scan, "Final projections")).toBe(
      "customer_id[#1], total[#2]"
    );
  });

  test("reads the rows and width Doris states, once it has statistics", () => {
    const analyzed = parse(dorisAnalyzedJoin);
    const scan = byId(analyzed, "0.0.0.0.0.1");
    const join = byId(analyzed, "0.0.0.0.0");
    const unanalyzed = byId(parse(dorisSingleScan), "0.0");

    expect(scan).toMatchObject({ rows: 5000, width: 8.398 });
    // Doris groups a standalone cardinality by thousands, and prints the same
    // number plain where it packs it onto a line beside the width.
    expect(propertyValue(join, "Cardinality")).toBe("4,499");
    expect(join.rows).toBe(4499);
    // The same scan before Doris had analyzed the table: avgRowSize=0.0 is
    // what it prints with no column statistics, and no row is zero bytes
    // wide, so that is not a width it stated.
    expect(propertyValue(unanalyzed, "Avg row size")).toBe("0.0");
    expect(unanalyzed.width).toBeUndefined();
    expect(parse(dorisSingleScan).estimates.width).toBe(false);
  });

  test("leaves the statistics Doris prints after the plan out of it", () => {
    const tree = parse(dorisJoinExchange);

    expect(outline(tree)).toEqual([
      "0 VEXCHANGE",
      "0.0 VAGGREGATE (merge finalize)",
      "0.0.0 VEXCHANGE",
      "0.0.0.0 VAGGREGATE (update serialize)",
      "0.0.0.0.0 VHASH JOIN",
      "0.0.0.0.0.0 VEXCHANGE",
      "0.0.0.0.0.0.0 VOlapScanNode",
      "0.0.0.0.0.1 VOlapScanNode",
    ]);
    // The statistics follow the plan under a rule of equals signs, so they
    // would otherwise describe whichever operator Doris printed last — the
    // scan of the join's build side, which comes in the final fragment.
    const printedLast = byId(tree, "0.0.0.0.0.0.0");
    expect(printedLast.relation).toEqual({
      database: "qp",
      table: "customers",
    });
    expect(propertyValue(printedLast, "Detail")).toBeUndefined();
  });

  test("gives up past the depth it can draw", () => {
    expect(parseStarrocksPlan(nestedPlan(400)).ok).toBe(true);
    expect(parseStarrocksPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
