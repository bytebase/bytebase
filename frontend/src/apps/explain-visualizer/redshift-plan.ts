import {
  addPlanProperty,
  checkPlanDepth,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  type PlanRelation,
  type PlanWarning,
  parseWithDepthLimit,
  planSelfCost,
} from "./plan-model";

/**
 * Maps Redshift's `EXPLAIN` text onto the engine-neutral plan model.
 *
 * Redshift publishes no plan as a document: its EXPLAIN takes no format, and
 * the one structured copy it keeps is in the `STL_EXPLAIN` system table, which
 * nothing outside a running cluster can read. So this is the one parser here
 * that reads text, and §4 of the query-plan design is why that is not the
 * default: a text plan has no escaping, so a string holding a newline can put
 * a line into the output that looks like an operator.
 *
 * Two things make it safe enough to do here, where the alternative is no
 * picture at all. Redshift returns the plan one row per line, so the row
 * boundaries are the engine's own and a newline inside a literal stays inside
 * its row. And an operator is recognized only by a whole cost tuple —
 * `(cost=0.00..87.98 rows=8798 width=35)` — which a value would have to
 * reproduce exactly to be mistaken for one.
 *
 * The grammar is PostgreSQL's own text format, which Redshift inherited and
 * prefixed with `XN`, so this reads a PostgreSQL text plan too. That is what
 * lets the parser be checked against output captured from a real server.
 */

/**
 * An operator: its depth in the tree, its name, and the estimate that makes it
 * one. The `XN` prefix is Redshift's; PostgreSQL's own text plans omit it.
 */
const OPERATOR =
  /^(\s*)(?:->\s+)?(?:XN\s+)?(\S.*?)\s+\(cost=([\d.]+)\.\.([\d.]+)\s+rows=(\d+)\s+width=(\d+)\)\s*$/;

/** An attribute of the operator above it, such as `Hash Cond: (...)`. */
const ATTRIBUTE = /^\s*([A-Z][A-Za-z ]*?):\s*(.*)$/;

/** How Redshift says it moved data to make a join work. */
const DISTRIBUTION = /\s+(DS_[A-Z_]+)$/;

/** The table a scan reads, which Redshift names before any alias. */
const SCANNED_TABLE = /\son\s+(\S+)/;

/**
 * The strategies that move data between slices to join it. They are the first
 * thing a reader of a Redshift plan looks for, because they are what a
 * distribution key is chosen to avoid.
 */
const REDISTRIBUTING = new Set([
  "DS_BCAST_INNER",
  "DS_DIST_INNER",
  "DS_DIST_BOTH",
]);

const DATA_REDISTRIBUTION: PlanWarning = {
  title: "Data redistribution",
  detail:
    "This join moves rows between slices before it can match them. Distributing both tables on their join column, or making the smaller one ALL, lets the join run where the rows already are.",
  source: "heuristic",
};

export const REDSHIFT_PLAN_EMPTY_MESSAGE =
  "Redshift returned no query plan for this statement.";
export const REDSHIFT_PLAN_NO_PLAN_MESSAGE =
  "The query plan has no operator in it: a Redshift plan states a cost for every one.";

interface Parsed {
  readonly indent: number;
  readonly nodeType: string;
  readonly startupCost: number;
  readonly totalCost: number;
  readonly rows: number;
  readonly width: number;
  readonly relation?: PlanRelation;
  readonly subject?: string;
  readonly properties: PlanProperty[];
  readonly children: Parsed[];
}

/** Where the operator sits: at its arrow, or at its name when it has none. */
function indentOf(line: string): number {
  const arrow = line.indexOf("->");
  return arrow >= 0 ? arrow : line.length - line.trimStart().length;
}

function parseOperator(line: string): Parsed | undefined {
  const matched = OPERATOR.exec(line);
  if (!matched) return undefined;
  const [, , name, startup, total, rows, width] = matched;

  const properties: PlanProperty[] = [];
  let nodeType = name;
  const distribution = DISTRIBUTION.exec(nodeType);
  if (distribution) {
    nodeType = nodeType.slice(0, distribution.index);
    addPlanProperty(properties, "Distribution", distribution[1]);
  }

  // A scan names its table, and then whatever the statement called it.
  const scanned = SCANNED_TABLE.exec(nodeType);
  if (scanned) {
    nodeType = nodeType.slice(0, scanned.index);
  }
  const table = scanned?.[1];

  return {
    indent: indentOf(line),
    nodeType: nodeType.trim() || "Unknown",
    startupCost: Number(startup),
    totalCost: Number(total),
    rows: Number(rows),
    width: Number(width),
    relation: table ? { table } : undefined,
    subject: table,
    properties,
    children: [],
  };
}

function toPlanNode(parsed: Parsed, id: string, depth: number): PlanNode {
  checkPlanDepth(depth);
  const children = parsed.children.map((child, index) =>
    toPlanNode(child, `${id}.${index}`, depth + 1)
  );
  const warnings = parsed.properties.some(
    (property) =>
      property.label === "Distribution" && REDISTRIBUTING.has(property.value)
  )
    ? [DATA_REDISTRIBUTION]
    : [];
  return {
    id,
    nodeType: parsed.nodeType,
    subject:
      parsed.subject ??
      parsed.properties.find((property) => property.label.endsWith("Cond"))
        ?.value,
    relation: parsed.relation,
    startupCost: parsed.startupCost,
    totalCost: parsed.totalCost,
    selfCost: planSelfCost(parsed.totalCost, children),
    rows: parsed.rows,
    width: parsed.width,
    properties: parsed.properties,
    warnings,
    children,
  };
}

export function parseRedshiftPlan(source: string): PlanParseResult {
  if (!source.trim()) {
    return { ok: false, message: REDSHIFT_PLAN_EMPTY_MESSAGE };
  }

  let root: Parsed | undefined;
  // The operators from the root down to the one being read, so a line's indent
  // says which of them it belongs under.
  const open: Parsed[] = [];
  for (const line of source.split("\n")) {
    if (!line.trim()) continue;
    const operator = parseOperator(line);
    if (!operator) {
      // Anything that is not an operator describes the one above it.
      const current = open[open.length - 1];
      if (!current) continue;
      const attribute = ATTRIBUTE.exec(line);
      if (attribute) {
        addPlanProperty(current.properties, attribute[1], attribute[2]);
      } else {
        addPlanProperty(current.properties, "Detail", line.trim());
      }
      continue;
    }

    while (open.length > 0 && open[open.length - 1].indent >= operator.indent) {
      open.pop();
    }
    const parent = open[open.length - 1];
    if (parent) {
      parent.children.push(operator);
    } else if (!root) {
      root = operator;
    } else {
      // A second operator with nothing above it is not something Redshift
      // prints; keeping it under the first draws all of what arrived.
      root.children.push(operator);
    }
    open.push(operator);
  }

  if (!root) {
    return { ok: false, message: REDSHIFT_PLAN_NO_PLAN_MESSAGE };
  }
  return parseWithDepthLimit(() => toPlanNode(root, "0", 0));
}
