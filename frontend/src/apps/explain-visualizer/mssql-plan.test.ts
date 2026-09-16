import { describe, expect, test } from "vitest";
import {
  MSSQL_PLAN_EMPTY_MESSAGE,
  MSSQL_PLAN_INVALID_XML_MESSAGE,
  MSSQL_PLAN_NO_PLAN_MESSAGE,
  parseMssqlPlan,
} from "./mssql-plan";
import {
  PLAN_FULL_TABLE_SCAN,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanTree,
} from "./plan-model";
import hashJoinAggregateSort from "./test-data/mssql/hash-join-aggregate-sort.xml?raw";
import ifElse from "./test-data/mssql/if-else.xml?raw";
import implicitConversionWarning from "./test-data/mssql/implicit-conversion-warning.xml?raw";
import indexSeekKeyLookup from "./test-data/mssql/index-seek-key-lookup.xml?raw";
import missingIndex from "./test-data/mssql/missing-index.xml?raw";
import noJoinPredicate from "./test-data/mssql/no-join-predicate.xml?raw";
import procedureTwoStatements from "./test-data/mssql/procedure-two-statements.xml?raw";
import scalarUdf from "./test-data/mssql/scalar-udf.xml?raw";
import splitStatementsBatch from "./test-data/mssql/split-statements-batch.xml?raw";
import staticCursor from "./test-data/mssql/static-cursor.xml?raw";
import twoStatementBatch from "./test-data/mssql/two-statement-batch.xml?raw";
import unmatchedFilteredIndex from "./test-data/mssql/unmatched-filtered-index.xml?raw";

const SHOWPLAN_NS = "http://schemas.microsoft.com/sqlserver/2004/07/showplan";

const showplan = (statements: string) =>
  `<ShowPlanXML xmlns="${SHOWPLAN_NS}" Version="1.564"><BatchSequence><Batch><Statements>${statements}</Statements></Batch></BatchSequence></ShowPlanXML>`;

/** A statement whose plan is a chain of `depth` operators. */
const nestedPlan = (depth: number): string => {
  let relop =
    '<RelOp NodeId="0" PhysicalOp="Table Scan" LogicalOp="Table Scan"><TableScan/></RelOp>';
  for (let level = 1; level < depth; level += 1) {
    relop = `<RelOp NodeId="${level}" PhysicalOp="Filter" LogicalOp="Filter"><Filter>${relop}</Filter></RelOp>`;
  }
  return showplan(
    `<StmtSimple StatementType="SELECT"><QueryPlan>${relop}</QueryPlan></StmtSimple>`
  );
};

const parseFixture = (source: string): PlanTree => {
  const result = parseMssqlPlan(source);
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

const propertyValue = (node: PlanNode, label: string): string | undefined =>
  node.properties.find((property) => property.label === label)?.value;

describe("parseMssqlPlan", () => {
  test("puts the statement above its operators, depth-first", () => {
    const tree = parseFixture(hashJoinAggregateSort);

    expect(nodeTypes(tree)).toEqual([
      "0 SELECT",
      "0.0 Sort",
      "0.0.0 Compute Scalar",
      "0.0.0.0 Hash Match (Aggregate)",
      "0.0.0.0.0 Merge Join (Inner Join)",
      "0.0.0.0.0.0 Stream Aggregate",
      "0.0.0.0.0.0.0 Index Scan",
      "0.0.0.0.0.1 Clustered Index Scan",
    ]);
    expect(tree.root).toMatchObject({
      subject:
        "SELECT c.region, COUNT(*) AS orders FROM orders o JOIN customers c ON c.id = o.customer_id GROUP BY c.region ORDER BY COUNT(*) DESC",
      totalCost: 0.276656,
      selfCost: 0,
      rows: 3,
    });
    expect(tree.root.width).toBeUndefined();
  });

  test("names operators the way SQL Server Management Studio does", () => {
    // The logical operator only joins the name when it says something new.
    expect(nodeTypes(parseFixture(indexSeekKeyLookup))).toEqual([
      "0 SELECT",
      "0.0 Nested Loops (Inner Join)",
      "0.0.0 Index Seek",
      // SQL Server reports the lookup as a clustered index seek.
      "0.0.1 Key Lookup",
    ]);
    const procedure = parseFixture(procedureTwoStatements);
    expect(byType(procedure, "TopN Sort").subject).toBe("o.total DESC");
    expect(byType(procedure, "Hash Match (Inner Join)").subject).toBe(
      "c.id = o.customer_id"
    );
  });

  test("reports SQL Server's estimates, with no startup cost to report", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const scan = byType(tree, "Index Scan");

    expect(tree.estimates).toEqual({
      cost: true,
      startupCost: false,
      rows: true,
      width: true,
    });
    expect(scan).toMatchObject({ totalCost: 0.121986, rows: 50000, width: 11 });
    expect(scan.startupCost).toBeUndefined();
  });

  test("charges each operator only the cost its children do not explain", () => {
    const tree = parseFixture(hashJoinAggregateSort);
    const selfTotal = tree.nodes.reduce(
      (sum, node) => sum + (node.selfCost ?? 0),
      0
    );

    expect(byType(tree, "Sort").selfCost).toBeCloseTo(0.276656 - 0.265287, 9);
    expect(byType(tree, "Compute Scalar").selfCost).toBe(0);
    expect(selfTotal).toBeCloseTo(0.276656, 9);
    expect(tree.costBasis).toBeCloseTo(0.276656, 9);
  });

  test("says what each operator works on", () => {
    const tree = parseFixture(hashJoinAggregateSort);

    expect(byType(tree, "Index Scan").subject).toBe(
      "orders.orders_customer_id_idx"
    );
    expect(byType(tree, "Merge Join (Inner Join)").subject).toBe(
      "o.customer_id = c.id"
    );
    expect(byType(tree, "Hash Match (Aggregate)").subject).toBe("c.region");
    expect(byType(tree, "Stream Aggregate").subject).toBe("o.customer_id");
    expect(byType(tree, "Sort").subject).toBe("Expr1002 DESC");
    expect(byType(tree, "Compute Scalar").subject).toBeUndefined();
  });

  test("carries the operator's expressions and attributes as properties", () => {
    const tree = parseFixture(indexSeekKeyLookup);
    const seek = byType(tree, "Index Seek");
    const lookup = byType(tree, "Key Lookup");

    expect(seek.properties.slice(0, 3)).toEqual([
      {
        label: "Object",
        value: "[plandb].[dbo].[orders].[orders_customer_id_idx]",
      },
      { label: "Index Kind", value: "NonClustered" },
      { label: "Seek Predicates", value: "orders.customer_id = (42)" },
    ]);
    expect(propertyValue(seek, "Output List")).toBe(
      "orders.id, orders.customer_id"
    );
    expect(propertyValue(seek, "Estimated Rows Read")).toBe("10");
    expect(propertyValue(seek, "Scan Direction")).toBe("FORWARD");
    // Only an operator that reruns says how many times it runs.
    expect(propertyValue(seek, "Estimated Executions")).toBeUndefined();
    expect(propertyValue(lookup, "Estimated Executions")).toBe("10");
    expect(propertyValue(tree.root, "Parameter List")).toBe("@1 = (42)");
    expect(propertyValue(tree.root, "Statement Optm Level")).toBe("FULL");
  });

  test("writes flags, sizes, times and tiny numbers out the way SSMS shows them", () => {
    const tree = parseFixture(indexSeekKeyLookup);
    const seek = byType(tree, "Index Seek");

    expect(propertyValue(seek, "Ordered")).toBe("True");
    expect(propertyValue(seek, "Forced Index")).toBe("False");
    expect(propertyValue(byType(tree, "Key Lookup"), "Lookup")).toBe("True");
    expect(propertyValue(tree.root, "Retrieved From Cache")).toBe("False");
    expect(propertyValue(tree.root, "Cached Plan Size")).toBe("32 KB");
    expect(propertyValue(tree.root, "Compile Time")).toBe("1 ms");
    expect(
      propertyValue(byType(tree, "Nested Loops (Inner Join)"), "Estimate CPU")
    ).toBe("0.0000418");
  });

  test("keeps fields the model already has out of the property list", () => {
    const tree = parseFixture(indexSeekKeyLookup);
    const labels = tree.nodes.flatMap((node) =>
      node.properties.map((property) => property.label)
    );

    for (const promoted of [
      "Physical Op",
      "Logical Op",
      "Estimate Rows",
      "Avg Row Size",
      "Estimated Total Subtree Cost",
      "Statement Type",
      "Statement Sub Tree Cost",
      "Statement Est Rows",
    ]) {
      expect(labels).not.toContain(promoted);
    }
  });

  test("labels the condition and branches of an IF", () => {
    const { root } = parseFixture(ifElse);

    expect(root.nodeType).toBe("COND WITH QUERY");
    expect(
      root.children.map((child) => [child.relationship, child.nodeType])
    ).toEqual([
      ["Condition", "Compute Scalar"],
      ["Then", "SELECT"],
      ["Else", "SELECT WITHOUT QUERY"],
    ]);
    // The ELSE branch returns a constant and has nothing to estimate.
    expect(root.children[2].totalCost).toBeUndefined();
    expect(root.totalCost).toBeCloseTo(0.00330167 + 0.151986, 9);
    expect(root.selfCost).toBe(0);
  });

  test("puts a procedure's statements under the call", () => {
    const { root } = parseFixture(procedureTwoStatements);

    expect(root.nodeType).toBe("EXECUTE PROC");
    expect(propertyValue(root, "Procedure Name")).toBe("dbo.region_report");
    expect(root.children.map((child) => child.nodeType)).toEqual([
      "SELECT",
      "SELECT",
    ]);
    expect(root.totalCost).toBeCloseTo(0.0343858 + 1.60801, 9);
  });

  test("labels a called function's statements with the function's name", () => {
    const { root } = parseFixture(scalarUdf);

    expect(
      root.children.map((child) => [child.relationship, child.nodeType])
    ).toEqual([
      [undefined, "Compute Scalar"],
      ["dbo.order_count", "SELECT"],
      ["dbo.order_count", "RETURN"],
    ]);
    // SQL Server's own estimate for the statement, 0.00345998, leaves the
    // function out.
    expect(root.totalCost).toBeCloseTo(0.00345998 + 0.0032842, 9);
  });

  test("roots a batch of statements in one tree", () => {
    const { root } = parseFixture(twoStatementBatch);

    expect(root).toMatchObject({ nodeType: "Batch", subject: "2 statements" });
    expect(root.children.map((child) => child.subject)).toEqual([
      "SELECT COUNT(*) FROM customers",
      "; SELECT COUNT(*) FROM orders",
    ]);
    expect(root.totalCost).toBeCloseTo(0.0354862 + 0.151986, 9);
  });

  test("reads every statement of a batch that gives each its own block", () => {
    const { root } = parseFixture(splitStatementsBatch);

    expect(root.children.map((child) => child.nodeType)).toEqual([
      "CREATE TABLE",
      "ASSIGN",
      "EXECUTE STRING",
      "DECLARE CURSOR",
      "OPEN CURSOR",
      "FETCH CURSOR",
      "COND",
      "CLOSE CURSOR",
      "DEALLOCATE CURSOR",
      "DROP OBJECT",
    ]);
  });

  test("turns an operator warning SQL Server reports into a node warning", () => {
    const tree = parseFixture(noJoinPredicate);

    expect(byType(tree, "Nested Loops (Inner Join)").warnings).toEqual([
      {
        title: "No join predicate",
        detail:
          "The join has no condition, so every row of one input is paired with every row of the other.",
      },
    ]);
  });

  test("puts a statement's warnings on the statement", () => {
    const { root } = parseFixture(implicitConversionWarning);

    expect(root.warnings.map((warning) => warning.detail)).toEqual([
      'Type conversion in expression (CONVERT_IMPLICIT(int,[plandb].[dbo].[customers].[code],0)) may affect "Cardinality Estimate" in query plan choice.',
      'Type conversion in expression (CONVERT_IMPLICIT(int,[plandb].[dbo].[customers].[code],0)=CONVERT_IMPLICIT(int,[@1],0)) may affect "Seek Plan" in query plan choice.',
    ]);
  });

  test("writes a missing index out as the statement that creates it", () => {
    const { root } = parseFixture(missingIndex);

    expect(root.warnings).toEqual([
      {
        title: "Missing index",
        detail:
          "SQL Server estimates this index would lower the statement's cost by 78.4%: CREATE NONCLUSTERED INDEX [idx_orders_total] ON [dbo].[orders] ([total]) INCLUDE ([customer_id]);",
      },
    ]);
  });

  test("warns on a full scan of a table large enough to matter", () => {
    const tree = parseFixture(missingIndex);
    const scans = tree.nodes.filter(
      (node) => node.nodeType === "Clustered Index Scan"
    );

    // `orders` reads 50,000 rows; `customers` reads 5,000, too few to matter.
    expect(scans.map((scan) => [scan.subject, scan.warnings])).toEqual([
      ["customers.PK__customer__3213E83FBDBA5577", []],
      ["orders.PK__orders__3213E83F52C28C59", [PLAN_FULL_TABLE_SCAN]],
    ]);
    // A nonclustered index is narrower than the table, so reading all of it
    // is not flagged.
    expect(
      byType(parseFixture(hashJoinAggregateSort), "Index Scan").warnings
    ).toEqual([]);
  });

  test("reads separate seeks as alternatives and one seek's ranges as all holding", () => {
    const range = (
      element: string,
      scanType: string,
      column: string,
      value: string
    ) =>
      `<${element} ScanType="${scanType}"><RangeColumns><ColumnReference Table="[orders]" Column="${column}"/></RangeColumns><RangeExpressions><ScalarOperator ScalarString="${value}"/></RangeExpressions></${element}>`;
    const seek = (...ranges: string[]) =>
      `<SeekPredicateNew><SeekKeys>${ranges.join("")}</SeekKeys></SeekPredicateNew>`;
    const tree = parseFixture(
      showplan(
        `<StmtSimple StatementType="SELECT"><QueryPlan><RelOp NodeId="0" PhysicalOp="Index Seek" LogicalOp="Index Seek"><IndexScan><Object Table="[orders]" Index="[orders_customer_id_idx]"/><SeekPredicates>${seek(range("Prefix", "EQ", "customer_id", "(1)"))}${seek(range("Prefix", "EQ", "customer_id", "(2)"))}${seek(range("StartRange", "GT", "total", "(10)"), range("EndRange", "LT", "total", "(20)"))}</SeekPredicates></IndexScan></RelOp></QueryPlan></StmtSimple>`
      )
    );

    // An IN list seeks once per value; a range seek bounds both ends at once.
    expect(propertyValue(byType(tree, "Index Seek"), "Seek Predicates")).toBe(
      "orders.customer_id = (1) OR orders.customer_id = (2) OR orders.total > (10) AND orders.total < (20)"
    );
  });

  test("explains a flag SQL Server raises on a statement without naming an operator", () => {
    const { root } = parseFixture(
      showplan(
        '<StmtSimple StatementType="SELECT"><QueryPlan><Warnings UnmatchedIndexes="true" SpatialGuess="1"/></QueryPlan></StmtSimple>'
      )
    );

    expect(root.warnings).toEqual([
      {
        title: "Unmatched indexes",
        detail:
          "A filtered index could not be used because the statement is parameterized.",
      },
      {
        title: "Spatial Guess",
        detail: "SQL Server flagged this while compiling the plan.",
      },
    ]);
  });

  test("names the filtered indexes a parameterized statement could not use", () => {
    const statement = byType(parseFixture(unmatchedFilteredIndex), "SELECT");

    expect(statement.warnings).toContainEqual({
      title: "Unmatched indexes",
      detail:
        "The statement is parameterized, so SQL Server could not use the filtered index [plandb].[dbo].[orders].[orders_big_total_idx].",
    });
  });

  test("says what an unfamiliar warning names, and still says something when it names nothing", () => {
    const { root } = parseFixture(
      showplan(
        '<StmtSimple StatementType="SELECT"><QueryPlan><Warnings><ColumnsWithStaleStatistics><ColumnReference Table="[customers]" Column="region"/></ColumnsWithStaleStatistics><SpillOccurred/></Warnings></QueryPlan></StmtSimple>'
      )
    );

    expect(root.warnings).toEqual([
      { title: "Columns With Stale Statistics", detail: "customers.region" },
      {
        title: "Spill Occurred",
        detail: "SQL Server flagged this while compiling the plan.",
      },
    ]);
  });

  test("keeps a suggested index's name within SQL Server's identifier limit", () => {
    const columns = [1, 2, 3, 4]
      .map(
        (n) =>
          `<Column Name="[a_column_name_long_enough_to_matter_here_${n}]"/>`
      )
      .join("");
    const { root } = parseFixture(
      showplan(
        `<StmtSimple StatementType="SELECT"><QueryPlan><MissingIndexes><MissingIndexGroup Impact="50"><MissingIndex Schema="[dbo]" Table="[customer_order_history]"><ColumnGroup Usage="EQUALITY">${columns}</ColumnGroup></MissingIndex></MissingIndexGroup></MissingIndexes></QueryPlan></StmtSimple>`
      )
    );
    const name = /INDEX \[([^\]]+)\]/.exec(root.warnings[0].detail)?.[1] ?? "";

    expect(name).toHaveLength(128);
    expect(name.startsWith("idx_customer_order_history_")).toBe(true);
  });

  test("escapes a closing bracket in a suggested index's name", () => {
    const { root } = parseFixture(
      showplan(
        '<StmtSimple StatementType="SELECT"><QueryPlan><MissingIndexes><MissingIndexGroup><MissingIndex Schema="[dbo]" Table="[sales]]2026]"><ColumnGroup Usage="EQUALITY"><Column Name="[total]"/></ColumnGroup></MissingIndex></MissingIndexGroup></MissingIndexes></QueryPlan></StmtSimple>'
      )
    );

    expect(root.warnings[0].detail).toBe(
      "SQL Server suggests this index: CREATE NONCLUSTERED INDEX [idx_sales]]2026_total] ON [dbo].[sales]]2026] ([total]);"
    );
  });

  test("reads a cursor's query plans as the cursor statement's own", () => {
    const { root } = parseFixture(
      showplan(
        '<StmtCursor StatementType="DECLARE CURSOR" StatementText="DECLARE c CURSOR FOR SELECT id FROM orders WHERE total = 500.99"><CursorPlan CursorName="c"><Operation OperationType="FetchQuery"><QueryPlan CachedPlanSize="16"><MissingIndexes><MissingIndexGroup Impact="90"><MissingIndex Schema="[dbo]" Table="[orders]"><ColumnGroup Usage="EQUALITY"><Column Name="[total]"/></ColumnGroup></MissingIndex></MissingIndexGroup></MissingIndexes><RelOp NodeId="0" PhysicalOp="Clustered Index Scan" LogicalOp="Clustered Index Scan"><IndexScan/></RelOp></QueryPlan></Operation></CursorPlan></StmtCursor>'
      )
    );

    expect(root.nodeType).toBe("DECLARE CURSOR");
    expect(root.children.map((child) => child.nodeType)).toEqual([
      "Clustered Index Scan",
    ]);
    expect(root.warnings.map((warning) => warning.title)).toEqual([
      "Missing index",
    ]);
    expect(propertyValue(root, "Cached Plan Size")).toBe("16 KB");
  });

  test("labels a cursor's population and fetch queries and says what kind of cursor it is", () => {
    const { root } = parseFixture(staticCursor);

    expect(root.nodeType).toBe("DECLARE CURSOR");
    expect(
      root.children.map((child) => [child.relationship, child.nodeType])
    ).toEqual([
      ["Populate Query", "Clustered Index Insert"],
      ["Fetch Query", "Clustered Index Seek"],
    ]);
    expect(propertyValue(root, "Cursor Actual Type")).toBe("SnapShot");
    expect(propertyValue(root, "Cursor Concurrency")).toBe("Read Only");
    expect(propertyValue(root, "Forward Only")).toBe("False");
  });

  test("reads a plan nested deeper than anything the optimizer produces", () => {
    const result = parseMssqlPlan(nestedPlan(100));
    // The statement, then its hundred operators.
    expect(result.ok && result.tree.nodes).toHaveLength(101);
  });

  test("rejects a plan nested deeper than the walks over it can go", () => {
    expect(parseMssqlPlan(nestedPlan(2000))).toEqual({
      ok: false,
      message: PLAN_TOO_DEEP_MESSAGE,
    });
  });

  test.each([
    ["", MSSQL_PLAN_EMPTY_MESSAGE],
    ["   ", MSSQL_PLAN_EMPTY_MESSAGE],
    ["Clustered Index Scan", MSSQL_PLAN_INVALID_XML_MESSAGE],
    ["<ShowPlanXML>", MSSQL_PLAN_INVALID_XML_MESSAGE],
    ["<plan/>", MSSQL_PLAN_NO_PLAN_MESSAGE],
    [showplan(""), MSSQL_PLAN_NO_PLAN_MESSAGE],
  ])("rejects %j with a specific message", (source, message) => {
    expect(parseMssqlPlan(source)).toEqual({ ok: false, message });
  });
});
