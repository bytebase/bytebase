import { describe, expect, test } from "vitest";
import {
  BIGQUERY_PLAN_EMPTY_MESSAGE,
  BIGQUERY_PLAN_INVALID_JSON_MESSAGE,
  BIGQUERY_PLAN_NO_PLAN_MESSAGE,
  parseBigqueryPlan,
} from "./bigquery-plan";
import {
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import filteredAggregate from "./test-data/bigquery/filtered-aggregate.json";

/** A chain of `depth` stages, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  const stages: Record<string, unknown>[] = [{ id: 0, name: "S00: Input" }];
  for (let level = 1; level < depth; level += 1) {
    stages.push({
      id: level,
      name: `S0${level}: Aggregate+`,
      inputStages: [level - 1],
    });
  }
  return JSON.stringify(stages);
};

const parse = (source: string): PlanTree => {
  const result = parseBigqueryPlan(source);
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

describe("parseBigqueryPlan", () => {
  test("reads the stage that no other stage feeds as the root", () => {
    const tree = parseFixture(filteredAggregate);
    // A stage says what it read: the input stage a table, the output stage the
    // shuffle the one below it wrote, which is the engine's own name for it.
    expect(outline(tree)).toEqual([
      "0 Output __stage00_output",
      "0.0 Input analytics.customers",
    ]);
  });

  test("is measured, since the plan is the run that returned the rows", () => {
    const tree = parseFixture(filteredAggregate);
    // What each stage really wrote, with no estimate beside it.
    expect(tree.nodes[1].actualRows).toBe(2);
    expect(tree.root.actualRows).toBe(1);
    expect(tree.nodes[1].rows).toBeUndefined();
    expect(tree.measured.rows).toBe(true);
    expect(tree.estimates.rows).toBe(false);
    expect(tree.estimates.cost).toBe(false);
  });

  test("resolves the table a stage read from its own READ step", () => {
    const tree = parseFixture(filteredAggregate);
    expect(tree.nodes[1].relation).toEqual({
      database: "analytics",
      table: "customers",
    });
    // The output stage reads a shuffle, which is not a table.
    expect(tree.root.relation).toBeUndefined();
  });

  test("leaves a table in another project unresolved", () => {
    const tree = parse(
      JSON.stringify([
        {
          id: 0,
          name: "S00: Input",
          steps: [
            {
              kind: "READ",
              substeps: ["$1:id", "FROM other-project.analytics.customers"],
            },
          ],
        },
      ])
    );
    // Named outside the connection, so the metadata here cannot answer for it.
    expect(tree.root.relation).toBeUndefined();
    expect(tree.root.subject).toBe("other-project.analytics.customers");
  });

  test("keeps the steps and the shard timings on the node", () => {
    const tree = parseFixture(filteredAggregate);
    const input = tree.nodes[1];
    expect(propertyValue(input, "Stage")).toBe("S00: Input");
    expect(propertyValue(input, "Steps")).toContain(
      "READ: $1:city, FROM analytics.customers, WHERE equal($1, 'berlin')"
    );
    expect(propertyValue(input, "Steps")).toContain("AGGREGATE:");
    expect(propertyValue(input, "Read ms avg")).toBe("11");
    expect(propertyValue(input, "Compute ms max")).toBe("9");
    expect(propertyValue(input, "Shuffle output bytes")).toBe("72");
    expect(propertyValue(input, "Records read")).toBe("4");
  });

  test("ranks nothing by time, since a stage states its shards' time", () => {
    const tree = parseFixture(filteredAggregate);
    expect(tree.root.actualTime).toBeUndefined();
    expect(tree.measured.time).toBe(false);
  });

  test("says so when there is no plan to read", () => {
    expect(parseBigqueryPlan("")).toEqual({
      ok: false,
      message: BIGQUERY_PLAN_EMPTY_MESSAGE,
    });
    expect(parseBigqueryPlan("not json")).toEqual({
      ok: false,
      message: BIGQUERY_PLAN_INVALID_JSON_MESSAGE,
    });
    // The stages are a list; an object is some other document.
    expect(parseBigqueryPlan("{}")).toEqual({
      ok: false,
      message: BIGQUERY_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseBigqueryPlan("[]")).toEqual({
      ok: false,
      message: BIGQUERY_PLAN_NO_PLAN_MESSAGE,
    });
  });

  test("stops at a stage that reads itself", () => {
    const tree = parse(
      JSON.stringify([
        { id: 0, name: "S00: Input", inputStages: [1] },
        { id: 1, name: "S01: Output", inputStages: [0] },
      ])
    );
    // Every stage feeds another, so the last is taken as the output one and the
    // cycle below it stops.
    expect(outline(tree)).toEqual(["0 Output -", "0.0 Input -"]);
  });

  test("draws a stage two others read, under each of them", () => {
    const tree = parse(
      JSON.stringify([
        { id: 0, name: "S00: Input" },
        { id: 1, name: "S01: Join+", inputStages: [0] },
        { id: 2, name: "S02: Aggregate+", inputStages: [0] },
        { id: 3, name: "S03: Output", inputStages: [1, 2] },
      ])
    );
    expect(outline(tree)).toEqual([
      "0 Output -",
      "0.0 Join+ -",
      "0.0.0 Input -",
      "0.1 Aggregate+ -",
      "0.1.0 Input -",
    ]);
  });

  test("refuses a plan deeper than the viewer draws", () => {
    expect(parseBigqueryPlan(nestedPlan(600))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
