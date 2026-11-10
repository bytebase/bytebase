import { describe, expect, test } from "vitest";
import {
  ORACLE_PLAN_EMPTY_MESSAGE,
  ORACLE_PLAN_INVALID_JSON_MESSAGE,
  ORACLE_PLAN_NO_PLAN_MESSAGE,
  parseOraclePlan,
} from "./oracle-plan";
import {
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import fullScanFilter from "./test-data/oracle/full-scan-filter.json";
import hashJoinGroupSort from "./test-data/oracle/hash-join-group-sort.json";
import indexRangeScan from "./test-data/oracle/index-range-scan.json";
import unionDistinct from "./test-data/oracle/union-distinct.json";

/** A chain of `depth` steps, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  const steps = [{ id: "0", operation: "SELECT STATEMENT" }];
  for (let level = 1; level < depth; level += 1) {
    steps.push({
      id: String(level),
      parent_id: String(level - 1),
      operation: "NESTED LOOPS",
      position: "1",
    } as never);
  }
  return JSON.stringify(steps);
};

const parse = (source: string): PlanTree => {
  const result = parseOraclePlan(source);
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

describe("parseOraclePlan", () => {
  test("rebuilds the tree from the parent links in the plan table", () => {
    const tree = parseFixture(fullScanFilter);
    const scan = byId(tree, "0.0");

    expect(outline(tree)).toEqual([
      "0 SELECT STATEMENT -",
      "0.0 TABLE ACCESS FULL ORDERS",
    ]);
    // Oracle's COST covers an operator and everything below it, so the scan
    // owns all of what the statement costs.
    expect(tree.root.totalCost).toBe(48);
    expect(tree.root.selfCost).toBe(0);
    expect(scan).toMatchObject({ totalCost: 48, selfCost: 48, rows: 1613 });
    expect(scan.warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
    expect(propertyValue(scan, "Filter predicates")).toBe('"TOTAL">880');
  });

  test("names an operator the way DBMS_XPLAN prints it", () => {
    expect(outline(parseFixture(indexRangeScan))).toEqual([
      "0 SELECT STATEMENT -",
      "0.0 TABLE ACCESS BY INDEX ROWID BATCHED ORDERS",
      "0.0.0 INDEX RANGE SCAN ORDERS_CUSTOMER_ID_IDX",
    ]);
    expect(outline(parseFixture(unionDistinct)).slice(0, 3)).toEqual([
      "0 SELECT STATEMENT -",
      "0.0 HASH UNIQUE -",
      "0.0.0 UNION-ALL -",
    ]);
  });

  test("answers a table with the schema Oracle named, and leaves an index alone", () => {
    const tree = parseFixture(indexRangeScan);

    // Oracle writes the owner on every object it reads, so the lookup is not
    // left guessing which schema's ORDERS this is.
    expect(byId(tree, "0.0").relation).toEqual({
      table: "ORDERS",
      schema: "SYSTEM",
    });
    // An index is not a table, and the table it belongs to is the step above.
    expect(byId(tree, "0.0.0").relation).toBeUndefined();
    expect(propertyValue(byId(tree, "0.0.0"), "Access predicates")).toBe(
      '"CUSTOMER_ID"=42'
    );
  });

  test("carries a step's cost down to the operator that spends it", () => {
    const tree = parseFixture(hashJoinGroupSort);

    // Oracle leaves COST off a step that adds none. Were those left without
    // one, the operator above would keep the subtree's cost as its own.
    const collector = tree.nodes.find(
      (node) => node.nodeType === "STATISTICS COLLECTOR"
    );
    expect(collector?.totalCost).toBe(7);
    expect(collector?.selfCost).toBe(0);
    const customers = tree.nodes.find(
      (node) => node.relation?.table === "CUSTOMERS"
    );
    expect(customers?.selfCost).toBe(7);
  });

  test("orders an operator's children the way Oracle runs them", () => {
    const tree = parseFixture(hashJoinGroupSort);
    const join = tree.nodes.find((node) => node.nodeType === "HASH JOIN");

    // POSITION orders siblings; the build side is 1 and the probe side 2.
    expect(join?.children.map((child) => child.nodeType)).toEqual([
      "NESTED LOOPS",
      "TABLE ACCESS FULL",
    ]);
  });

  test("leaves a small read unflagged", () => {
    const small = parse(
      JSON.stringify([
        { id: "0", operation: "SELECT STATEMENT" },
        {
          id: "1",
          parent_id: "0",
          operation: "TABLE ACCESS",
          options: "FULL",
          object_name: "COUNTRIES",
          object_type: "TABLE",
          cardinality: "200",
        },
      ])
    );

    expect(byId(small, "0.0").warnings).toEqual([]);
  });

  test("draws what it can of a plan whose parent links form a cycle", () => {
    // A step that is its own parent is its own child, which a walk would
    // follow forever. The rest of the plan is still worth drawing.
    const tree = parse(
      JSON.stringify([
        { id: "0", parent_id: "0", operation: "SELECT STATEMENT" },
        { id: "1", parent_id: "0", operation: "HASH JOIN" },
      ])
    );

    expect(outline(tree)).toEqual(["0 SELECT STATEMENT -", "0.0 HASH JOIN -"]);
  });

  test("reports a plan it cannot read", () => {
    expect(parseOraclePlan("   ")).toEqual({
      ok: false,
      message: ORACLE_PLAN_EMPTY_MESSAGE,
    });
    expect(parseOraclePlan("not json")).toEqual({
      ok: false,
      message: ORACLE_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseOraclePlan("[]")).toEqual({
      ok: false,
      message: ORACLE_PLAN_EMPTY_MESSAGE,
    });
    // Every step claiming a parent leaves no operator to start from.
    expect(
      parseOraclePlan('[{"id":"1","parent_id":"2"},{"id":"2","parent_id":"1"}]')
    ).toEqual({ ok: false, message: ORACLE_PLAN_NO_PLAN_MESSAGE });
  });

  test("gives up past the depth it can draw", () => {
    expect(parseOraclePlan(nestedPlan(400)).ok).toBe(true);
    expect(parseOraclePlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
