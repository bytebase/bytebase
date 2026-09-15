import {
  addPlanProperty,
  checkPlanDepth,
  formatPlanCost,
  formatPlanShare,
  PLAN_FULL_TABLE_SCAN,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  type PlanWarning,
  parseWithDepthLimit,
  planSelfCost,
} from "./plan-model";

/**
 * Maps SQL Server's `SHOWPLAN_XML` output onto the engine-neutral plan model.
 *
 * A showplan holds statements, and a statement with a query plan holds a tree
 * of `RelOp` operators. Both become nodes: a statement is the parent of its
 * plan's root operator, and a statement that contains others — an IF, a
 * procedure call — is the parent of those. A batch of several statements gets
 * a "Batch" root so the plan stays one tree.
 *
 * Names follow SQL Server Management Studio, and attributes the model has no
 * field for are carried through under SQL Server's own names.
 */

export const MSSQL_PLAN_EMPTY_MESSAGE =
  "SQL Server returned no query plan for this statement.";
export const MSSQL_PLAN_INVALID_XML_MESSAGE =
  "The query plan is not valid XML. SHOWPLAN_XML output is expected.";
export const MSSQL_PLAN_NO_PLAN_MESSAGE =
  "The query plan XML is not a SQL Server showplan with at least one statement.";

const STATEMENT_ELEMENTS = new Set([
  "StmtSimple",
  "StmtCond",
  "StmtCursor",
  "StmtReceive",
  "StmtUseDb",
]);

/** The branches of an IF, which label the statements and plans under them. */
const BRANCH_ELEMENTS = new Set(["Condition", "Then", "Else"]);

/** Children of a `RelOp` that describe it rather than being its operator. */
const RELOP_DETAIL_ELEMENTS = new Set([
  "OutputList",
  "Warnings",
  "MemoryFractions",
  "RunTimeInformation",
  "RunTimePartitionSummary",
  "InternalInfo",
]);

/** Attributes promoted to typed fields, so the property list omits them. */
const PROMOTED_STATEMENT_ATTRIBUTES = new Set([
  "StatementText",
  "StatementType",
  "StatementSubTreeCost",
  "StatementEstRows",
]);
const PROMOTED_RELOP_ATTRIBUTES = new Set([
  "PhysicalOp",
  "LogicalOp",
  "EstimateRows",
  "AvgRowSize",
  "EstimatedTotalSubtreeCost",
]);

/** Operator children the property list puts first, in this order. */
const LEADING_OPERATOR_ELEMENTS = ["Object", "SeekPredicates", "Predicate"];

/** Operators that read a whole table: a heap, or the clustered index that is one. */
const FULL_TABLE_SCAN_OPERATORS = new Set([
  "Table Scan",
  "Clustered Index Scan",
]);

/**
 * Rows read at or below which a full table scan is not worth flagging, the
 * same size of table the PostgreSQL parser leaves alone.
 */
const FULL_SCAN_ROWS_THRESHOLD = 5000;

/** Longest name SQL Server accepts for an index, as for any identifier. */
const MAX_IDENTIFIER_LENGTH = 128;

const SEEK_OPERATORS: Record<string, string> = {
  EQ: "=",
  NE: "<>",
  GT: ">",
  GE: ">=",
  LT: "<",
  LE: "<=",
};

function childElements(element: Element | undefined, name?: string): Element[] {
  if (!element) return [];
  return Array.from(element.children).filter(
    (child) => name === undefined || child.localName === name
  );
}

function firstChild(
  element: Element | undefined,
  name: string
): Element | undefined {
  return childElements(element, name)[0];
}

function numberAttribute(element: Element, name: string): number | undefined {
  const value = element.getAttribute(name);
  if (value === null || value.trim() === "") return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function isTrue(value: string | null): boolean {
  return value === "1" || value === "true";
}

/** "EstimatedRowsRead" as "Estimated Rows Read": SQL Server's name, spaced. */
function humanize(name: string): string {
  return name
    .replace(/([a-z\d])([A-Z])/g, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2");
}

function unbracket(identifier: string): string {
  const match = /^\[(.*)\]$/.exec(identifier);
  return match ? match[1].replaceAll("]]", "]") : identifier;
}

function bracket(identifier: string): string {
  return `[${identifier.replaceAll("]", "]]")}]`;
}

/** `o.customer_id`: the alias or table a column comes from, and the column. */
function columnName(reference: Element): string {
  const column = unbracket(reference.getAttribute("Column") ?? "");
  const owner =
    reference.getAttribute("Alias") ?? reference.getAttribute("Table");
  return owner ? `${unbracket(owner)}.${column}` : column;
}

function columnList(element: Element | undefined): string[] {
  return childElements(element, "ColumnReference").map(columnName);
}

function scalarStrings(element: Element | undefined): string[] {
  return childElements(element, "ScalarOperator")
    .map((scalar) => scalar.getAttribute("ScalarString") ?? "")
    .filter(Boolean);
}

/**
 * Attributes the showplan schema types as booleans, among those on the
 * elements read here. SQL Server writes most of them as 1 or 0.
 */
const BOOLEAN_ATTRIBUTES = new Set([
  "BatchModeOnRowStoreUsed",
  "BitmapCreator",
  "ComputeSequence",
  "ContainsInlineScalarTsqlUdfs",
  "ContainsInterleavedExecutionCandidates",
  "ContainsLedgerTables",
  "DMLRequestSort",
  "Distinct",
  "DynamicSeek",
  "ExclusiveProfileTimeActive",
  "ForceScan",
  "ForceSeek",
  "ForcedIndex",
  "ForwardOnly",
  "GroupExecuted",
  "InRow",
  "IsAdaptive",
  "IsDistributed",
  "IsExternal",
  "IsExternallyComputed",
  "IsFull",
  "IsGraphDBTransitiveClosure",
  "IsHashDistributed",
  "IsNoOp",
  "IsPercent",
  "IsReplicated",
  "IsRoundRobin",
  "IsScalar",
  "LocalParallelism",
  "Lookup",
  "ManyToMany",
  "NoExpandHint",
  "Optimized",
  "Ordered",
  "Parallel",
  "Partitioned",
  "RemoteDataAccess",
  "Remoting",
  "RowCount",
  "SecurityPolicyApplied",
  "Stack",
  "StartupExpression",
  "UsePlan",
  "WithOrderedPrefetch",
  "WithTies",
  "WithUnorderedPrefetch",
]);

/** Units of the attributes that measure memory or time, per the showplan schema. */
const ATTRIBUTE_UNITS: Record<string, string> = {
  CachedPlanSize: "KB",
  CompileTime: "ms",
  CompileCPU: "ms",
  CompileMemory: "KB",
  MemoryGrant: "KB",
  SerialRequiredMemory: "KB",
  SerialDesiredMemory: "KB",
  RequiredMemory: "KB",
  DesiredMemory: "KB",
  RequestedMemory: "KB",
  GrantWaitTime: "s",
  GrantedMemory: "KB",
  MaxUsedMemory: "KB",
  MaxQueryMemory: "KB",
  LastRequestedMemory: "KB",
};

/** SQL Server writes numbers far from one in exponent form, as in `1.157e-06`. */
const EXPONENT_NUMBER = /^[-+]?\d+(\.\d+)?e[-+]?\d+$/i;
const plainNumberFormat = new Intl.NumberFormat("en-US", {
  maximumSignificantDigits: 15,
  useGrouping: false,
});

function formatAttribute(name: string, value: string): string {
  if (BOOLEAN_ATTRIBUTES.has(name) || value === "true" || value === "false") {
    return isTrue(value) ? "True" : "False";
  }
  const number = EXPONENT_NUMBER.test(value)
    ? plainNumberFormat.format(Number(value))
    : value;
  const unit = ATTRIBUTE_UNITS[name];
  return unit ? `${number} ${unit}` : number;
}

function addAttributes(
  properties: PlanProperty[],
  element: Element | undefined,
  skip: ReadonlySet<string> = new Set()
) {
  for (const attribute of Array.from(element?.attributes ?? [])) {
    if (skip.has(attribute.name)) continue;
    addPlanProperty(
      properties,
      humanize(attribute.name),
      formatAttribute(attribute.name, attribute.value)
    );
  }
}

/** `[plandb].[dbo].[orders].[orders_customer_id_idx] [o]`, as SSMS prints it. */
function objectFullName(object: Element): string {
  const name = ["Database", "Schema", "Table", "Index"]
    .map((part) => object.getAttribute(part))
    .filter(Boolean)
    .join(".");
  const alias = object.getAttribute("Alias");
  return alias ? `${name} ${alias}` : name;
}

function descendantsNamed(
  element: Element,
  names: ReadonlySet<string>
): Element[] {
  return Array.from(element.getElementsByTagName("*")).filter((descendant) =>
    names.has(descendant.localName)
  );
}

const SEEK_ELEMENTS = new Set(["SeekPredicate", "SeekPredicateNew"]);
const SEEK_RANGE_ELEMENTS = new Set(["Prefix", "StartRange", "EndRange"]);

/**
 * The ranges within one seek all hold, but each seek is a range of its own —
 * one per value of an IN list — so separate seeks are alternatives.
 */
function formatSeekPredicates(seekPredicates: Element): string {
  return descendantsNamed(seekPredicates, SEEK_ELEMENTS)
    .map((seek) =>
      descendantsNamed(seek, SEEK_RANGE_ELEMENTS)
        .flatMap((range) => {
          const scanType = range.getAttribute("ScanType") ?? "";
          const operator = SEEK_OPERATORS[scanType] ?? scanType;
          const columns = columnList(firstChild(range, "RangeColumns"));
          const values = scalarStrings(firstChild(range, "RangeExpressions"));
          return columns.map((column, index) =>
            [column, operator, values[index]].filter(Boolean).join(" ")
          );
        })
        .join(" AND ")
    )
    .filter(Boolean)
    .join(" OR ");
}

function formatDefinedValues(definedValues: Element): string {
  const entries = childElements(definedValues, "DefinedValue").map((entry) => {
    const columns = columnList(entry).join(", ");
    const [expression] = scalarStrings(entry);
    return expression ? `${columns} = ${expression}` : columns;
  });
  const separator = entries.some((entry) => entry.includes(" = "))
    ? "\n"
    : ", ";
  return entries.join(separator);
}

function formatOrderBy(orderBy: Element): string {
  return childElements(orderBy, "OrderByColumn")
    .map((entry) => {
      const [column] = columnList(entry);
      const direction = isTrue(entry.getAttribute("Ascending"))
        ? "ASC"
        : "DESC";
      return `${column} ${direction}`;
    })
    .join(", ");
}

/** An operator's child element as text, or empty when it holds no expression. */
function formatOperatorElement(element: Element): string {
  switch (element.localName) {
    case "Object":
      return objectFullName(element);
    case "SeekPredicates":
      return formatSeekPredicates(element);
    case "DefinedValues":
      return formatDefinedValues(element);
    case "OrderBy":
      return formatOrderBy(element);
  }
  const expressions = scalarStrings(element);
  if (expressions.length > 0) return expressions.join("\n");
  return columnList(element).join(", ");
}

function pairColumns(left: string[], right: string[]): string {
  return left
    .map((column, index) => `${column} = ${right[index] ?? "?"}`)
    .join(", ");
}

/** Warnings SQL Server raises as a flag on `<Warnings>`, by attribute name. */
const FLAG_WARNINGS: Record<string, PlanWarning> = {
  NoJoinPredicate: {
    title: "No join predicate",
    detail:
      "The join has no condition, so every row of one input is paired with every row of the other.",
  },
  UnmatchedIndexes: {
    title: "Unmatched indexes",
    detail:
      "A filtered index could not be used because the statement is parameterized.",
  },
};

/**
 * `unmatchedIndexes` names the filtered indexes the flag is about, which SQL
 * Server lists beside `<Warnings>` in the query plan rather than inside it.
 */
function toServerWarnings(
  warnings: Element | undefined,
  unmatchedIndexes: readonly string[] = []
): PlanWarning[] {
  if (!warnings) return [];
  const result: PlanWarning[] = [];
  for (const attribute of Array.from(warnings.attributes)) {
    if (!isTrue(attribute.value)) continue;
    if (attribute.name === "UnmatchedIndexes" && unmatchedIndexes.length > 0) {
      result.push({
        title: FLAG_WARNINGS.UnmatchedIndexes.title,
        detail: `The statement is parameterized, so SQL Server could not use the filtered ${unmatchedIndexes.length === 1 ? "index" : "indexes"} ${unmatchedIndexes.join(", ")}.`,
      });
      continue;
    }
    result.push(
      FLAG_WARNINGS[attribute.name] ?? {
        title: humanize(attribute.name),
        detail: "SQL Server flagged this while compiling the plan.",
      }
    );
  }
  for (const child of childElements(warnings)) {
    switch (child.localName) {
      case "PlanAffectingConvert":
        result.push({
          title: "Type conversion affects the plan",
          detail: `Type conversion in expression (${child.getAttribute("Expression") ?? ""}) may affect "${child.getAttribute("ConvertIssue") ?? ""}" in query plan choice.`,
        });
        break;
      case "ColumnsWithNoStatistics":
        result.push({
          title: "Columns with no statistics",
          detail: `Estimates over ${columnList(child).join(", ")} were made without statistics, so they may be far off.`,
        });
        break;
      default: {
        const details = [
          ...Array.from(child.attributes).map(
            (attribute) =>
              `${humanize(attribute.name)}: ${formatAttribute(attribute.name, attribute.value)}`
          ),
          ...columnList(child),
        ];
        result.push({
          title: humanize(child.localName),
          detail:
            details.length > 0
              ? details.join(", ")
              : "SQL Server flagged this while compiling the plan.",
        });
      }
    }
  }
  return result;
}

/**
 * An index SQL Server suggests for the statement, written out as the statement
 * that creates it, and named in Bytebase's default index naming convention,
 * `idx_<table>_<columns>`.
 */
function toMissingIndexWarnings(queryPlan: Element): PlanWarning[] {
  const groups = childElements(
    firstChild(queryPlan, "MissingIndexes"),
    "MissingIndexGroup"
  );
  return groups.flatMap((group) =>
    childElements(group, "MissingIndex").map((index) => {
      const columns = (usages: string[]) =>
        childElements(index, "ColumnGroup")
          .filter((columnGroup) =>
            usages.includes(columnGroup.getAttribute("Usage") ?? "")
          )
          .flatMap((columnGroup) => childElements(columnGroup, "Column"))
          .map((column) => column.getAttribute("Name") ?? "");
      const keys = columns(["EQUALITY", "INEQUALITY"]);
      const includes = columns(["INCLUDE"]);
      const table = index.getAttribute("Table") ?? "";
      const name = ["idx", unbracket(table), ...keys.map(unbracket)]
        .join("_")
        .slice(0, MAX_IDENTIFIER_LENGTH);
      const target = [index.getAttribute("Schema"), table]
        .filter(Boolean)
        .join(".");
      const include =
        includes.length > 0 ? ` INCLUDE (${includes.join(", ")})` : "";
      const statement = `CREATE NONCLUSTERED INDEX ${bracket(name)} ON ${target} (${keys.join(", ")})${include};`;
      const impact = numberAttribute(group, "Impact");
      return {
        title: "Missing index",
        detail:
          impact === undefined
            ? `SQL Server suggests this index: ${statement}`
            : `SQL Server estimates this index would lower the statement's cost by ${formatPlanShare(impact / 100)}: ${statement}`,
      };
    })
  );
}

interface PlanChild {
  readonly element: Element;
  readonly relationship?: string;
}

/**
 * The label an IF's branch, a cursor's operation or a called function puts on
 * the plans under it. A function goes by the name a query calls it with,
 * `dbo.order_count` for `[plandb].[dbo].[order_count]`.
 */
function branchLabel(element: Element): string | undefined {
  if (BRANCH_ELEMENTS.has(element.localName)) return element.localName;
  if (element.localName === "Operation") {
    return humanize(element.getAttribute("OperationType") ?? "") || undefined;
  }
  if (element.localName === "UDF") {
    const name = element.getAttribute("ProcName") ?? "";
    const parts = Array.from(name.matchAll(/\[((?:[^\]]|\]\])*)\]/g), (match) =>
      match[1].replaceAll("]]", "]")
    );
    return (parts.length > 0 ? parts.slice(-2).join(".") : name) || undefined;
  }
  return undefined;
}

/**
 * The statements and operators directly below `element`: the nearest ones at
 * any depth, since SQL Server nests them inside elements that are neither.
 */
function planChildren(element: Element): PlanChild[] {
  const found: PlanChild[] = [];
  const visit = (parent: Element, relationship: string | undefined) => {
    for (const child of childElements(parent)) {
      if (
        child.localName === "RelOp" ||
        STATEMENT_ELEMENTS.has(child.localName)
      ) {
        found.push({ element: child, relationship });
      } else {
        visit(child, branchLabel(child) ?? relationship);
      }
    }
  };
  visit(element, undefined);
  return found;
}

function toChildren(element: Element, id: string, depth: number): PlanNode[] {
  return planChildren(element).map((child, index) =>
    toNode(child, `${id}.${index}`, depth + 1)
  );
}

function toNode(child: PlanChild, id: string, depth: number): PlanNode {
  checkPlanDepth(depth);
  return child.element.localName === "RelOp"
    ? toOperatorNode(child, id, depth)
    : toStatementNode(child, id, depth);
}

/**
 * The query plans a statement owns, as opposed to ones nested statements do:
 * its own, an IF's condition, and each operation of a cursor.
 */
function ownQueryPlans(statement: Element): Element[] {
  return [
    ...childElements(statement, "QueryPlan"),
    ...childElements(firstChild(statement, "Condition"), "QueryPlan"),
    ...childElements(firstChild(statement, "CursorPlan"), "Operation").flatMap(
      (operation) => childElements(operation, "QueryPlan")
    ),
  ];
}

function toStatementNode(
  { element, relationship }: PlanChild,
  id: string,
  depth: number
): PlanNode {
  const children = toChildren(element, id, depth);
  const queryPlans = ownQueryPlans(element);

  // A statement costs what the operators and statements under it do, a
  // called function's body included once, though SQL Server leaves that out
  // of its own estimate; only a statement with neither keeps the cost SQL
  // Server gave it.
  const totalCost = children.some((child) => child.totalCost !== undefined)
    ? children.reduce((sum, child) => sum + (child.totalCost ?? 0), 0)
    : numberAttribute(element, "StatementSubTreeCost");

  const text = element.getAttribute("StatementText") ?? "";
  const properties: PlanProperty[] = [];
  addPlanProperty(properties, "Statement", text.trim());
  addPlanProperty(
    properties,
    "Procedure Name",
    firstChild(element, "StoredProc")?.getAttribute("ProcName") ?? ""
  );
  addAttributes(properties, element, PROMOTED_STATEMENT_ATTRIBUTES);
  addAttributes(properties, firstChild(element, "CursorPlan"));
  for (const queryPlan of queryPlans) {
    addAttributes(properties, queryPlan);
    addAttributes(properties, firstChild(queryPlan, "MemoryGrantInfo"));
    const parameters = childElements(
      firstChild(queryPlan, "ParameterList"),
      "ColumnReference"
    ).map((parameter) =>
      [
        parameter.getAttribute("Column"),
        parameter.getAttribute("ParameterCompiledValue"),
      ]
        .filter(Boolean)
        .join(" = ")
    );
    addPlanProperty(properties, "Parameter List", parameters.join(", "));
  }

  return {
    id,
    nodeType: element.getAttribute("StatementType") ?? "Statement",
    subject: text.replace(/\s+/g, " ").trim() || undefined,
    relationship,
    totalCost,
    selfCost: planSelfCost(totalCost, children),
    rows: numberAttribute(element, "StatementEstRows"),
    properties,
    warnings: queryPlans.flatMap((queryPlan) => [
      ...toServerWarnings(
        firstChild(queryPlan, "Warnings"),
        childElements(
          firstChild(
            firstChild(queryPlan, "UnmatchedIndexes"),
            "Parameterization"
          ),
          "Object"
        ).map(objectFullName)
      ),
      ...toMissingIndexWarnings(queryPlan),
    ]),
    children,
  };
}

function operatorElement(relop: Element): Element | undefined {
  return childElements(relop).find(
    (child) => !RELOP_DETAIL_ELEMENTS.has(child.localName)
  );
}

/**
 * The operator's name as SSMS prints it: the physical operator, with the
 * logical one it implements when that says something the first does not.
 */
function toOperatorName(relop: Element, operator: Element | undefined): string {
  const physical = relop.getAttribute("PhysicalOp") ?? "Unknown";
  const logical = relop.getAttribute("LogicalOp");
  if (operator?.localName === "IndexScan") {
    if (operator.getAttribute("Storage") === "ColumnStore") {
      return "Columnstore Index Scan";
    }
    // SQL Server reports a lookup into the clustered index as a seek.
    if (isTrue(operator.getAttribute("Lookup"))) {
      return physical === "Clustered Index Seek" ? "Key Lookup" : physical;
    }
  }
  if (!logical || physical.includes(logical) || logical.includes(physical)) {
    return logical && logical.length > physical.length ? logical : physical;
  }
  return `${physical} (${logical})`;
}

/** What the operator works on, for the one line a card has room for. */
function toOperatorSubject(operator: Element | undefined): string | undefined {
  const object = firstChild(operator, "Object");
  if (object) {
    return ["Table", "Index"]
      .map((part) => object.getAttribute(part))
      .filter((part): part is string => Boolean(part))
      .map(unbracket)
      .join(".");
  }
  const hashBuild = columnList(firstChild(operator, "HashKeysBuild"));
  const hashProbe = columnList(firstChild(operator, "HashKeysProbe"));
  if (hashBuild.length > 0 && hashProbe.length > 0) {
    return pairColumns(hashBuild, hashProbe);
  }
  const mergeOuter = columnList(firstChild(operator, "OuterSideJoinColumns"));
  const mergeInner = columnList(firstChild(operator, "InnerSideJoinColumns"));
  if (mergeOuter.length > 0 && mergeInner.length > 0) {
    return pairColumns(mergeOuter, mergeInner);
  }
  const groupBy = [
    ...hashBuild,
    ...columnList(firstChild(operator, "GroupBy")),
  ];
  if (groupBy.length > 0) return groupBy.join(", ");
  const orderBy = firstChild(operator, "OrderBy");
  if (orderBy) return formatOrderBy(orderBy);
  const [predicate] = scalarStrings(firstChild(operator, "Predicate"));
  return predicate;
}

function toOperatorProperties(
  relop: Element,
  operator: Element | undefined
): PlanProperty[] {
  const properties: PlanProperty[] = [];
  const operatorChildren = childElements(operator).filter(
    (child) => child.localName !== "RelOp"
  );
  const leading = LEADING_OPERATOR_ELEMENTS.flatMap((name) =>
    operatorChildren.filter((child) => child.localName === name)
  );
  const rest = operatorChildren.filter((child) => !leading.includes(child));
  for (const child of [...leading, ...rest]) {
    addPlanProperty(
      properties,
      humanize(child.localName),
      formatOperatorElement(child)
    );
    if (child.localName === "Object") {
      addPlanProperty(
        properties,
        "Index Kind",
        child.getAttribute("IndexKind") ?? ""
      );
    }
  }
  addPlanProperty(
    properties,
    "Output List",
    columnList(firstChild(relop, "OutputList")).join(", ")
  );

  // The first run plus every rerun. Row estimates are per run, so an operator
  // that reruns — the inner side of a nested loop — returns this many times
  // its estimate.
  const reruns =
    (numberAttribute(relop, "EstimateRebinds") ?? 0) +
    (numberAttribute(relop, "EstimateRewinds") ?? 0);
  if (reruns > 0) {
    addPlanProperty(
      properties,
      "Estimated Executions",
      formatPlanCost(1 + reruns)
    );
  }
  addAttributes(properties, relop, PROMOTED_RELOP_ATTRIBUTES);
  addAttributes(properties, operator);
  return properties;
}

function toOperatorWarnings(
  relop: Element,
  operator: Element | undefined,
  physical: string
): PlanWarning[] {
  const warnings = toServerWarnings(firstChild(relop, "Warnings"));
  const rowsRead =
    numberAttribute(relop, "EstimatedRowsRead") ??
    numberAttribute(relop, "TableCardinality") ??
    0;
  if (
    FULL_TABLE_SCAN_OPERATORS.has(physical) &&
    operator?.getAttribute("Storage") !== "ColumnStore" &&
    rowsRead > FULL_SCAN_ROWS_THRESHOLD
  ) {
    warnings.push(PLAN_FULL_TABLE_SCAN);
  }
  return warnings;
}

function toOperatorNode(
  { element: relop, relationship }: PlanChild,
  id: string,
  depth: number
): PlanNode {
  const operator = operatorElement(relop);
  const children = operator ? toChildren(operator, id, depth) : [];
  const totalCost = numberAttribute(relop, "EstimatedTotalSubtreeCost");
  return {
    id,
    nodeType: toOperatorName(relop, operator),
    subject: toOperatorSubject(operator),
    relationship,
    totalCost,
    selfCost: planSelfCost(totalCost, children),
    rows: numberAttribute(relop, "EstimateRows"),
    width: numberAttribute(relop, "AvgRowSize"),
    properties: toOperatorProperties(relop, operator),
    warnings: toOperatorWarnings(
      relop,
      operator,
      relop.getAttribute("PhysicalOp") ?? ""
    ),
    children,
  };
}

function toRoot(statements: Element[]): PlanNode {
  if (statements.length === 1) {
    return toNode({ element: statements[0] }, "0", 0);
  }
  const children = statements.map((element, index) =>
    toNode({ element }, `0.${index}`, 1)
  );
  const totalCost = children.some((child) => child.totalCost !== undefined)
    ? children.reduce((sum, child) => sum + (child.totalCost ?? 0), 0)
    : undefined;
  return {
    id: "0",
    nodeType: "Batch",
    subject: `${children.length} statements`,
    totalCost,
    selfCost: planSelfCost(totalCost, children),
    properties: [],
    warnings: [],
    children,
  };
}

export function parseMssqlPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: MSSQL_PLAN_EMPTY_MESSAGE };
  }

  const document = new DOMParser().parseFromString(trimmed, "application/xml");
  if (document.getElementsByTagName("parsererror").length > 0) {
    return { ok: false, message: MSSQL_PLAN_INVALID_XML_MESSAGE };
  }

  const root = document.documentElement;
  // A batch can split its statements across several `<Statements>`, one
  // each, as SQL Server 2016 writes them.
  const statements =
    root.localName === "ShowPlanXML"
      ? childElements(firstChild(root, "BatchSequence"), "Batch")
          .flatMap((batch) => childElements(batch, "Statements"))
          .flatMap((block) =>
            childElements(block).filter((element) =>
              STATEMENT_ELEMENTS.has(element.localName)
            )
          )
      : [];
  if (statements.length === 0) {
    return { ok: false, message: MSSQL_PLAN_NO_PLAN_MESSAGE };
  }

  return parseWithDepthLimit(() => toRoot(statements));
}
