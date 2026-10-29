import { describe, expect, test } from "vitest";
import {
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import {
  parseSnowflakePlan,
  SNOWFLAKE_PLAN_EMPTY_MESSAGE,
  SNOWFLAKE_PLAN_INVALID_JSON_MESSAGE,
  SNOWFLAKE_PLAN_NO_PLAN_MESSAGE,
} from "./snowflake-plan";
import innerJoin from "./test-data/snowflake/inner-join.json";

/** One step holding `operations`, which is how Snowflake nests a plan. */
const planOf = (operations: Record<string, unknown>[]): string =>
  JSON.stringify({ Operations: [operations] });

/** A chain of `depth` operations, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  const operations: Record<string, unknown>[] = [
    { id: 0, operation: "Result" },
  ];
  for (let level = 1; level < depth; level += 1) {
    operations.push({
      id: level,
      parentOperators: [level - 1],
      operation: "Filter",
    });
  }
  return planOf(operations);
};

const parse = (source: string): PlanTree => {
  const result = parseSnowflakePlan(source);
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

describe("parseSnowflakePlan", () => {
  test("turns the parent ids around into a tree", () => {
    // The plan states each operation's parents, so the join's two inputs are
    // found by looking for the operations that name it.
    const tree = parseFixture(innerJoin);
    expect(outline(tree)).toEqual([
      "0 Result Z1.ID, Z2.ID",
      "0.0 InnerJoin joinKey: (Z2.ID = Z1.ID)",
      "0.0.0 TableScan TESTDB.TEMPORARY_DOC_TEST.Z2",
      "0.0.1 JoinFilter joinKey: (Z2.ID = Z1.ID)",
      "0.0.1.0 TableScan TESTDB.TEMPORARY_DOC_TEST.Z1",
    ]);
  });

  test("resolves a scan's table from the name Snowflake qualifies", () => {
    const tree = parseFixture(innerJoin);
    expect(tree.nodes[2].relation).toEqual({
      database: "TESTDB",
      schema: "TEMPORARY_DOC_TEST",
      table: "Z2",
    });
    // An operation that reads no object has no table to look up.
    expect(tree.root.relation).toBeUndefined();
  });

  test("keeps the pruning on the scan and the totals on the root", () => {
    const tree = parseFixture(innerJoin);
    expect(propertyValue(tree.nodes[2], "Partitions assigned")).toBe("1");
    expect(propertyValue(tree.nodes[2], "Partitions total")).toBe("1");
    expect(propertyValue(tree.nodes[2], "Bytes assigned")).toBe("512");
    // GlobalStats is what the plan says about the query as a whole.
    expect(propertyValue(tree.root, "Partitions total")).toBe("2");
    expect(propertyValue(tree.root, "Bytes assigned")).toBe("1024");
    // The operation id, so a reader can line the node up with Snowflake's own
    // tabular plan.
    expect(propertyValue(tree.root, "Operation")).toBe("0");
  });

  test("ranks nothing, since Snowflake estimates neither cost nor rows", () => {
    const tree = parseFixture(innerJoin);
    expect(tree.estimates.cost).toBe(false);
    expect(tree.estimates.rows).toBe(false);
    expect(tree.root.totalCost).toBeUndefined();
    expect(tree.root.rows).toBeUndefined();
  });

  test("flags a scan assigned every partition of its table", () => {
    const tree = parse(
      planOf([
        { id: 0, operation: "Result" },
        {
          id: 1,
          parentOperators: [0],
          operation: "TableScan",
          objects: ["DB.PUBLIC.EVENTS"],
          partitionsAssigned: 40,
          partitionsTotal: 40,
        },
      ])
    );
    expect(tree.nodes[1].warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
  });

  test("leaves a pruned scan, and a one-partition table, unflagged", () => {
    const tree = parse(
      planOf([
        { id: 0, operation: "Result" },
        {
          id: 1,
          parentOperators: [0],
          operation: "TableScan",
          objects: ["DB.PUBLIC.EVENTS"],
          partitionsAssigned: 3,
          partitionsTotal: 40,
        },
        {
          id: 2,
          parentOperators: [0],
          operation: "TableScan",
          objects: ["DB.PUBLIC.SMALL"],
          partitionsAssigned: 1,
          partitionsTotal: 1,
        },
      ])
    );
    expect(tree.nodes[1].warnings).toEqual([]);
    expect(tree.nodes[2].warnings).toEqual([]);
  });

  test("draws the last step of a plan compiled into several", () => {
    const tree = parse(
      JSON.stringify({
        Operations: [
          [{ id: 0, operation: "Result", expressions: ["first step"] }],
          [{ id: 0, operation: "Result", expressions: ["last step"] }],
        ],
      })
    );
    expect(tree.root.subject).toBe("last step");
    expect(propertyValue(tree.root, "Steps")).toBe("2");
  });

  test("says so when there is no operation to start from", () => {
    expect(parseSnowflakePlan("")).toEqual({
      ok: false,
      message: SNOWFLAKE_PLAN_EMPTY_MESSAGE,
    });
    expect(parseSnowflakePlan("not json")).toEqual({
      ok: false,
      message: SNOWFLAKE_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseSnowflakePlan("[]")).toEqual({
      ok: false,
      message: SNOWFLAKE_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseSnowflakePlan(JSON.stringify({ Operations: [] }))).toEqual({
      ok: false,
      message: SNOWFLAKE_PLAN_NO_PLAN_MESSAGE,
    });
    // Every operation naming a parent means there is no root to draw.
    expect(
      parseSnowflakePlan(
        planOf([
          { id: 0, parentOperators: [1], operation: "Result" },
          { id: 1, parentOperators: [0], operation: "InnerJoin" },
        ])
      )
    ).toEqual({ ok: false, message: SNOWFLAKE_PLAN_NO_PLAN_MESSAGE });
  });

  test("stops at an operation that reaches itself", () => {
    // A cycle below the root would never finish; the rest still draws.
    const tree = parse(
      planOf([
        { id: 0, operation: "Result" },
        { id: 1, parentOperators: [0, 2], operation: "InnerJoin" },
        { id: 2, parentOperators: [1], operation: "JoinFilter" },
      ])
    );
    expect(outline(tree)).toEqual([
      "0 Result -",
      "0.0 InnerJoin -",
      "0.0.0 JoinFilter -",
    ]);
  });

  test("refuses a plan deeper than the viewer draws", () => {
    expect(parseSnowflakePlan(nestedPlan(600))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
