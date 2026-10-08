import { describe, expect, test } from "vitest";
import {
  PLAN_FULL_INDEX_SCAN,
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import {
  parseSpannerPlan,
  SPANNER_PLAN_EMPTY_MESSAGE,
  SPANNER_PLAN_EMULATOR_MESSAGE,
  SPANNER_PLAN_INVALID_JSON_MESSAGE,
  SPANNER_PLAN_NO_PLAN_MESSAGE,
} from "./spanner-plan";
import filterLimit from "./test-data/spanner/filter-limit.json";
import hashJoin from "./test-data/spanner/hash-join.json";
import nestedArraySubqueries from "./test-data/spanner/nested-array-subqueries.json";
import scalarSubquery from "./test-data/spanner/scalar-subquery.json";

/** A plan whose operators form a chain `depth` long. */
const nestedPlan = (depth: number): string =>
  JSON.stringify({
    planNodes: Array.from({ length: depth }, (_, index) => ({
      index,
      kind: "RELATIONAL",
      displayName: "Filter",
      childLinks: index + 1 < depth ? [{ childIndex: index + 1 }] : undefined,
    })),
  });

const parseFixture = (fixture: unknown): PlanTree => {
  const result = parseSpannerPlan(JSON.stringify(fixture));
  if (!result.ok)
    throw new Error(`expected a parsable plan: ${result.message}`);
  return result.tree;
};

const nodeTypes = (tree: PlanTree) =>
  tree.nodes.map((node) => `${node.id} ${node.nodeType}`);

const byType = (tree: PlanTree, nodeType: string): PlanNode => {
  const node = tree.nodes.find((entry) => entry.nodeType === nodeType);
  if (!node) throw new Error(`no ${nodeType} node`);
  return node;
};

describe("parseSpannerPlan", () => {
  test("keeps the operators and leaves the expressions out of the tree", () => {
    const tree = parseFixture(hashJoin);

    expect(nodeTypes(tree)).toEqual([
      "0 Distributed Union",
      "0.0 Serialize Result",
      "0.0.0 Hash Join",
      "0.0.0.0 Local Distributed Union",
      "0.0.0.0.0 Table Scan",
      "0.0.0.1 Local Distributed Union",
      "0.0.0.1.0 Index Scan",
    ]);
  });

  test("names nodes with the call, iterator and scan types Spanner reports", () => {
    expect(nodeTypes(parseFixture(filterLimit))).toEqual([
      "0 Serialize Result",
      "0.0 Filter",
      "0.0.0 Global Limit",
      "0.0.0.0 Distributed Union",
      "0.0.0.0.0 Local Limit",
      "0.0.0.0.0.0 Local Distributed Union",
      "0.0.0.0.0.0.0 FilterScan",
      "0.0.0.0.0.0.0.0 Index Scan",
    ]);
    const subquery = parseFixture(scalarSubquery);
    expect(byType(subquery, "Global Stream Aggregate")).toBeDefined();
    expect(byType(subquery, "Local Stream Aggregate")).toBeDefined();
  });

  test("labels an operator's inputs with the role their link names", () => {
    const join = byType(parseFixture(hashJoin), "Hash Join");

    expect(join.children.map((child) => child.relationship)).toEqual([
      "Build",
      "Probe",
    ]);
  });

  test("folds the expressions an operator names into its properties", () => {
    const tree = parseFixture(hashJoin);

    expect(byType(tree, "Hash Join").properties).toEqual([
      {
        label: "Condition",
        value: "(($SingerId = $SingerId_1) AND ($AlbumId = $AlbumId_1))",
      },
      { label: "Build", value: "$AlbumTitle AS $AlbumTitle'" },
      { label: "join_type", value: "INNER" },
    ]);
    expect(
      byType(parseFixture(filterLimit), "Global Limit").properties
    ).toEqual([{ label: "Limit", value: "3" }]);
  });

  test("leaves metadata already in the name, or meaningless, out of the properties", () => {
    const tree = parseFixture(scalarSubquery);
    const labels = tree.nodes.flatMap((node) =>
      node.properties.map((property) => property.label)
    );

    for (const folded of [
      "call_type",
      "iterator_type",
      "scan_type",
      "scan_target",
      "subquery_cluster_node",
    ]) {
      expect(labels).not.toContain(folded);
    }
    expect(labels).toContain("scalar_aggregate");
  });

  test("leaves Spanner's internal metadata out of the properties", () => {
    const result = parseSpannerPlan(
      JSON.stringify({
        planNodes: [
          {
            index: 0,
            kind: "RELATIONAL",
            displayName: "Scan",
            metadata: {
              scan_type: "TableScan",
              scan_target: "Singers",
              "Full scan": "true",
              _internal_id: "7",
            },
          },
        ],
      })
    );

    expect(result.ok && result.tree.root.properties).toEqual([
      { label: "Full scan", value: "true" },
    ]);
  });

  test("names what a scan reads and what a filter keeps", () => {
    const tree = parseFixture(scalarSubquery);

    expect(byType(tree, "Table Scan").subject).toBe("Songs");
    expect(byType(tree, "Index Scan").subject).toBe("ConcertsBySingerId");
    expect(
      tree.nodes
        .filter((node) => node.nodeType === "FilterScan")
        .map((node) => node.subject)
    ).toEqual([
      "IF(($SongGenre = 'ROCKS'), true, $sv_1)",
      "($SingerId_1 = $SingerId)",
    ]);
  });

  test("hangs a subquery's operators under the subquery", () => {
    const tree = parseFixture(scalarSubquery);
    const filter = tree.nodes.find((node) => node.nodeType === "FilterScan");
    const subquery = byType(tree, "Scalar Subquery");

    expect(filter?.children.map((child) => child.nodeType)).toEqual([
      "Table Scan",
      "Scalar Subquery",
    ]);
    expect(subquery).toMatchObject({ subject: "$sv_1" });
    expect(subquery.relationship).toBeUndefined();
    expect(subquery.children.map((child) => child.nodeType)).toEqual([
      "Global Stream Aggregate",
    ]);
  });

  test("nests a subquery inside another one", () => {
    const tree = parseFixture(nestedArraySubqueries);
    const subqueries = tree.nodes.filter(
      (node) => node.nodeType === "Array Subquery"
    );

    expect(subqueries.map((node) => [node.id, node.subject])).toEqual([
      ["0.0.0.1", "$sv_2"],
      ["0.0.0.1.0.0.1", "$sv_1"],
    ]);
  });

  test("warns on the full scans Spanner reports, table or index", () => {
    const tree = parseFixture(hashJoin);

    expect(byType(tree, "Table Scan").warnings).toEqual([PLAN_FULL_TABLE_SCAN]);
    expect(byType(tree, "Index Scan").warnings).toEqual([PLAN_FULL_INDEX_SCAN]);
    // This one seeks with a condition, so Spanner does not call it full.
    expect(byType(parseFixture(filterLimit), "Index Scan").warnings).toEqual(
      []
    );
  });

  test("carries no estimates, because Spanner plans without any", () => {
    const tree = parseFixture(hashJoin);

    expect(tree.estimates).toEqual({
      cost: false,
      startupCost: false,
      rows: false,
      width: false,
    });
    expect(tree.costBasis).toBe(0);
  });

  test("reads a root that leaves its zero index out", () => {
    const result = parseSpannerPlan(
      JSON.stringify({
        planNodes: [
          {
            kind: "RELATIONAL",
            displayName: "Serialize Result",
            childLinks: [{ childIndex: 1 }],
          },
          {
            index: 1,
            kind: "RELATIONAL",
            displayName: "Scan",
            metadata: { scan_type: "TableScan", scan_target: "Singers" },
          },
        ],
      })
    );

    expect(result.ok && nodeTypes(result.tree)).toEqual([
      "0 Serialize Result",
      "0.0 Table Scan",
    ]);
  });

  test("skips a link back up the tree instead of following it forever", () => {
    const result = parseSpannerPlan(
      JSON.stringify({
        planNodes: [
          {
            index: 0,
            kind: "RELATIONAL",
            displayName: "Distributed Union",
            childLinks: [{ childIndex: 1 }],
          },
          {
            index: 1,
            kind: "RELATIONAL",
            displayName: "Filter",
            childLinks: [{ childIndex: 0 }],
          },
        ],
      })
    );

    expect(result.ok && nodeTypes(result.tree)).toEqual([
      "0 Distributed Union",
      "0.0 Filter",
    ]);
  });

  test("reads a plan nested deeper than anything the optimizer produces", () => {
    const result = parseSpannerPlan(nestedPlan(100));
    expect(result.ok && result.tree.nodes).toHaveLength(100);
  });

  test("rejects a plan nested deeper than the walks over it can go", () => {
    expect(parseSpannerPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });

  test.each([
    ["", SPANNER_PLAN_EMPTY_MESSAGE],
    ["   ", SPANNER_PLAN_EMPTY_MESSAGE],
    ["Distributed Union", SPANNER_PLAN_INVALID_JSON_MESSAGE],
    ["{}", SPANNER_PLAN_NO_PLAN_MESSAGE],
    ['{"planNodes": []}', SPANNER_PLAN_NO_PLAN_MESSAGE],
    // What Bytebase returns for the Spanner emulator, which does not plan queries.
    [
      '{"planNodes":[{"index":0,"kind":"KIND_UNSPECIFIED","displayName":"No query plan"}]}',
      SPANNER_PLAN_EMULATOR_MESSAGE,
    ],
  ])("rejects %j with a specific message", (source, message) => {
    expect(parseSpannerPlan(source)).toEqual({ ok: false, message });
  });
});
