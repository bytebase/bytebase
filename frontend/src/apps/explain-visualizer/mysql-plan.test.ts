import { describe, expect, test } from "vitest";
import {
  MYSQL_PLAN_EMPTY_MESSAGE,
  MYSQL_PLAN_INVALID_JSON_MESSAGE,
  MYSQL_PLAN_NO_PLAN_MESSAGE,
  parseMysqlPlan,
} from "./mysql-plan";
import {
  PLAN_FULL_INDEX_SCAN,
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import attachedSubquery from "./test-data/mysql/attached-subquery.json";
import derivedTable from "./test-data/mysql/derived-table.json";
import fullScanFilter from "./test-data/mysql/full-scan-filter.json";
import groupSortJoin from "./test-data/mysql/group-sort-join.json";
import mariadbBlockNlJoin from "./test-data/mysql/mariadb-block-nl-join.json";
import mariadbSortTemporaryJoin from "./test-data/mysql/mariadb-sort-temporary-join.json";
import orderingFilesort from "./test-data/mysql/ordering-filesort.json";
import unionDistinct from "./test-data/mysql/union-distinct.json";

/** A chain of `depth` nested operators, which is what drives the depth limit. */
const nestedPlan = (depth: number): string => {
  let plan: Record<string, unknown> = { table: { table_name: "orders" } };
  for (let level = 1; level < depth; level += 1) {
    plan = { materialized_from_subquery: plan };
  }
  return JSON.stringify({ query_block: plan });
};

const parse = (source: string): PlanTree => {
  const result = parseMysqlPlan(source);
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

describe("parseMysqlPlan", () => {
  test("maps a query block over a single scan", () => {
    const tree = parseFixture(fullScanFilter);
    const scan = byId(tree, "0.0");

    expect(outline(tree)).toEqual([
      "0 Query block -",
      "0.0 Full Table Scan orders",
    ]);
    expect(scan).toMatchObject({
      // read_cost + eval_cost is this operator's own; prefix_cost covers the
      // whole join prefix, so it is not.
      totalCost: 3233.41 + 1596.34,
      selfCost: 3233.41 + 1596.34,
      rows: 15963,
      relation: { table: "orders" },
      children: [],
    });
    expect(scan.warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
    expect(propertyValue(scan, "Attached condition")).toBe(
      "(`qp`.`orders`.`total` > 880)"
    );
    expect(propertyValue(scan, "Rows examined per scan")).toBe("47895");
    // A list of names reads as one line.
    expect(propertyValue(scan, "Used columns")).toBe(
      "id, customer_id, total, status, placed_on"
    );
  });

  test("keeps typed fields out of the property list", () => {
    const labels = byId(parseFixture(fullScanFilter), "0.0").properties.map(
      (property) => property.label
    );

    expect(labels).not.toContain("Table name");
    expect(labels).not.toContain("Access type");
    expect(labels).not.toContain("Rows produced per join");
  });

  test("names each level of the plan after the key that holds it", () => {
    expect(outline(parseFixture(groupSortJoin))).toEqual([
      "0 Query block -",
      "0.0 Order -",
      "0.0.0 Group -",
      "0.0.0.0 Full Table Scan o",
      "0.0.0.1 Unique Key Lookup c",
    ]);
  });

  test("charges an operator between the block and its tables what its tables cost", () => {
    const tree = parseFixture(groupSortJoin);
    const tables = [byId(tree, "0.0.0.0"), byId(tree, "0.0.0.1")];
    const tableCost = tables.reduce(
      (sum, table) => sum + (table.totalCost ?? 0),
      0
    );

    // MySQL states a cost on the query block and on each table, nothing in
    // between. Were the levels between left without one, the block above would
    // keep the whole query's cost as its own and rank first in every plan.
    for (const id of ["0.0", "0.0.0"]) {
      expect(byId(tree, id).totalCost).toBeCloseTo(tableCost, 5);
      expect(byId(tree, id).selfCost).toBe(0);
    }
    expect(tree.root.selfCost).toBe(0);
    expect(tree.costBasis).toBeCloseTo(tableCost, 5);
    expect(tables.map((table) => table.selfCost)).toEqual(
      tables.map((table) => table.totalCost)
    );
  });

  test("hangs each union branch off the union, with the branch's own keys on it", () => {
    const tree = parseFixture(unionDistinct);
    const union = byId(tree, "0.0");
    const branch = byId(tree, "0.0.0");

    expect(outline(tree)).toEqual([
      "0 Query block -",
      "0.0 Union <union1,2>",
      "0.0.0 Query block -",
      "0.0.0.0 Full Table Scan orders",
      "0.0.1 Query block -",
      "0.0.1.0 Full Table Scan customers",
    ]);
    // The object listing a branch is not a level of the plan, so what it says
    // about the branch belongs to the branch.
    expect(propertyValue(branch, "Cacheable")).toBe("true");
    expect(propertyValue(branch, "Dependent")).toBe("false");
    // A result MySQL materialized for itself is no table to look up.
    expect(union.relation).toBeUndefined();
    expect(union.totalCost).toBeCloseTo(4829.75 + 503.75, 5);
  });

  test("reads the table a subquery is attached to, which is still a read", () => {
    const tree = parseFixture(attachedSubquery);
    const scan = byId(tree, "0.0");

    // The scan carries the subquery's own operators under
    // `attached_subqueries`, which is a subquery run against the table rather
    // than a result standing in for it — so it is still a read of `o`, and
    // reading every row of it is still worth flagging.
    expect(scan.nodeType).toBe("Full Table Scan");
    expect(scan.children).toHaveLength(1);
    expect(scan.relation).toEqual({ table: "o", unresolved: true });
    expect(scan.rows).toBe(50184);
    expect(scan.warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
    // What MySQL materialized for the subquery is not a table of its own.
    expect(byId(tree, "0.0.0.0").nodeType).toBe("Materialize");
    expect(byId(tree, "0.0.0.0").relation).toBeUndefined();
  });

  test("charges a sort what MySQL says sorting costs", () => {
    const tree = parseFixture(orderingFilesort);
    const order = byId(tree, "0.0");
    const scan = byId(tree, "0.0.0");

    // `sort_cost` is the sort's alone, and the block's `query_cost` is the two
    // together. Left unread the sort looks free and the block ranks first,
    // which is not an operator a reader can do anything about.
    expect(order.nodeType).toBe("Order");
    expect(order.selfCost).toBe(50184);
    expect(order.totalCost).toBeCloseTo(50184 + 5058.65, 5);
    expect(scan.selfCost).toBeCloseTo(5058.65, 5);
    expect(tree.root.totalCost).toBeCloseTo(55242.65, 5);
  });

  test("charges a table its own cost, not what its subquery cost", () => {
    const tree = parseFixture(derivedTable);
    const derived = byId(tree, "0.0");
    const inner = byId(tree, "0.0.0.0.0.0");

    // `read_cost` and `eval_cost` are what reading and filtering that table
    // costs, and cover nothing below it: the subquery materialized for it is
    // costed apart, and the block's own `query_cost` leaves it out too. Taking
    // the subquery out of the table's cost charged the outer read a tenth of
    // what MySQL said, and ranked the inner scan above it.
    expect(derived.selfCost).toBeCloseTo(601.19 + 4789.5, 5);
    expect(derived.totalCost).toBeCloseTo(601.19 + 4789.5 + 4829.75, 5);
    expect(inner.selfCost).toBeCloseTo(40.25 + 4789.5, 5);
    // The whole plan is the two reads, so neither is more than all of it.
    expect(tree.costBasis).toBeCloseTo(601.19 + 4789.5 + 4829.75, 5);
  });

  test("leaves a derived table unanswered and unflagged", () => {
    const tree = parseFixture(derivedTable);
    const derived = byId(tree, "0.0");
    const scan = byId(tree, "0.0.0.0.0.0");

    expect(outline(tree)).toEqual([
      "0 Query block -",
      "0.0 Full Table Scan s",
      "0.0.0 Materialize -",
      "0.0.0.0 Query block -",
      "0.0.0.0.0 Group -",
      "0.0.0.0.0.0 Full Index Scan orders",
    ]);
    // `s` is the statement's name for the subquery, not a table, and reading
    // all of what MySQL materialized is not a read an index would change.
    expect(derived.relation).toBeUndefined();
    expect(derived.warnings).toEqual([]);
    expect(scan.relation).toEqual({ table: "orders", unresolved: true });
    expect(scan.warnings).toEqual([PLAN_FULL_INDEX_SCAN]);
  });

  describe("MariaDB", () => {
    test("reads the same shape under its own keys", () => {
      const tree = parseFixture(mariadbSortTemporaryJoin);
      const scan = byId(tree, "0.0.0.0");

      expect(outline(tree)).toEqual([
        "0 Query block -",
        "0.0 Sort sum(o.total) desc",
        "0.0.0 Temporary table -",
        "0.0.0.0 Full Table Scan o",
        "0.0.0.1 Unique Key Lookup c",
      ]);
      // MariaDB states one `cost` per operator and `rows` where MySQL says
      // `rows_examined_per_scan`.
      expect(scan).toMatchObject({ totalCost: 8.3654308, rows: 50416 });
      expect(scan.warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
      expect(tree.root.totalCost).toBe(53.47180584);
    });

    test("draws a block nested loop join and what it buffers", () => {
      const tree = parseFixture(mariadbBlockNlJoin);
      const join = byId(tree, "0.0.0.1");

      expect(outline(tree)).toEqual([
        "0 Query block -",
        "0.0 Sort o.total desc",
        "0.0.0 Temporary table -",
        "0.0.0.0 Full Table Scan c",
        "0.0.0.1 Block nested loop join o.`status` = c.country",
        "0.0.0.1.0 Full Table Scan o",
      ]);
      expect(propertyValue(join, "Join type")).toBe("BNL");
      expect(propertyValue(join, "Buffer size")).toBe("256KiB");
      // The sort MariaDB charges to the query is the difference between the
      // query's cost and the tables', and the query block is the only operator
      // that reports it.
      expect(tree.root.selfCost).toBeCloseTo(200622.7147 - 23419.168862, 5);
    });
  });

  test("carries an operator it has never heard of rather than dropping it", () => {
    const tree = parse(
      JSON.stringify({
        query_block: {
          regrouping_operation: { kept: "yes", table: { table_name: "t" } },
        },
      })
    );

    expect(outline(tree)).toEqual([
      "0 Query block -",
      "0.0 Regrouping operation -",
      "0.0.0 Table t",
    ]);
    expect(propertyValue(byId(tree, "0.0"), "Kept")).toBe("yes");
  });

  test("leaves a small read unflagged", () => {
    const small = parse(
      JSON.stringify({
        query_block: {
          table: {
            table_name: "countries",
            access_type: "ALL",
            rows_examined_per_scan: 200,
          },
        },
      })
    );

    expect(byId(small, "0.0").warnings).toEqual([]);
  });

  test("reports a plan it cannot read", () => {
    expect(parseMysqlPlan("   ")).toEqual({
      ok: false,
      message: MYSQL_PLAN_EMPTY_MESSAGE,
    });
    expect(parseMysqlPlan("not json")).toEqual({
      ok: false,
      message: MYSQL_PLAN_INVALID_JSON_MESSAGE,
    });
    expect(parseMysqlPlan('{"rows": []}')).toEqual({
      ok: false,
      message: MYSQL_PLAN_NO_PLAN_MESSAGE,
    });
    // A tabular EXPLAIN, which is what an unknown format can turn out to be.
    expect(parseMysqlPlan("id\tselect_type\ttable\n1\tSIMPLE\torders")).toEqual(
      {
        ok: false,
        message: MYSQL_PLAN_INVALID_JSON_MESSAGE,
      }
    );
  });

  test("gives up past the depth it can draw", () => {
    expect(parseMysqlPlan(nestedPlan(400)).ok).toBe(true);
    expect(parseMysqlPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });
});
