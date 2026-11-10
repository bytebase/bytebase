import {
  addPlanProperty,
  checkPlanDepth,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  type PlanRelation,
  parseWithDepthLimit,
  planLabel,
} from "./plan-model";

/**
 * Maps Spark's `EXPLAIN FORMATTED` output onto the engine-neutral plan model.
 *
 * Databricks SQL is Spark SQL, and FORMATTED answers in two parts: a tree of
 * operators, each tagged with a number, and then a block of detail per number.
 * The tree gives the shape, and the blocks give what each operator does — the
 * filters pushed into a read, the partitions it skipped, the keys a join
 * matched on. This parser puts the two back together.
 *
 * It states neither cost nor rows. Spark reports those from `EXPLAIN COST`,
 * and only for the logical plan, which is not the plan that ran; so nothing
 * here is ranked and what a reader gets is the shape and the detail. That is
 * what a Spark plan is read for anyway: whether a filter reached the scan,
 * whether a join broadcast or shuffled, and where the exchanges are.
 */

const PLAN_HEADER = "== Physical Plan ==";

/**
 * An operator in the tree, ending in the number its detail block carries:
 * `+- * ColumnarToRow (2)`. The `*` marks an operator Spark fused into a
 * whole-stage codegen block.
 */
const TREE_OPERATOR = /^([\s:+|-]*)(\*\s)?\s*(\S.*?)\s+\((\d+)\)\s*$/;

/** Where a tree line branches, which is how deep its operator sits. */
const BRANCH = /[+:]-/;

/** A detail block's header: `(3) Filter [codegen id : 1]`. */
const DETAIL_HEADER = /^\((\d+)\)\s+(\S.*?)(?:\s+\[([^\]]*)\])?\s*$/;

/**
 * A line of detail: `PushedFilters: [IsNotNull(city)]`. Spark counts what it
 * lists in the label — `Output [2]:` — and the count is already in the value it
 * introduces, so the label keeps the name alone.
 */
const DETAIL_LINE = /^([A-Z][A-Za-z ]*?)(?:\s*\[[^\]]*\])?\s*:\s*(.*)$/;

/** A read, which names its format and then its table: `Scan parquet db.t`. */
const SCAN = /^Scan\s+(\S+)\s+(\S+)$/;

/** Spark writes a table as `catalog.schema.table`. */
const QUALIFIED_TABLE = /^([^.]+)\.([^.]+)\.([^.]+)$/;

export const DATABRICKS_PLAN_EMPTY_MESSAGE =
  "Databricks returned no query plan for this statement.";
export const DATABRICKS_PLAN_NO_PLAN_MESSAGE =
  "The query plan has no physical plan in it. EXPLAIN FORMATTED output is expected.";

interface Detail {
  readonly name: string;
  readonly codegen?: string;
  readonly properties: PlanProperty[];
}

interface Parsed {
  readonly indent: number;
  readonly id: string;
  readonly nodeType: string;
  readonly subject?: string;
  readonly relation?: PlanRelation;
  readonly children: Parsed[];
}

/**
 * How deep a tree line sits: where its branch marker is, or above everything
 * when it has none, which is the root. Only the markers before the operator are
 * read, so an operator whose own text held one could not be taken for a node
 * deeper than it is.
 */
function indentOf(prefix: string): number {
  const branch = prefix.search(BRANCH);
  return branch >= 0 ? branch : -1;
}

/** The table a read names, where Spark qualifies it fully. */
function toRelation(table: string): PlanRelation | undefined {
  const qualified = QUALIFIED_TABLE.exec(table);
  if (!qualified) return undefined;
  // Spark's catalog holds schemas, which hold tables; Databricks calls the
  // first two a catalog and a schema as well.
  return { database: qualified[1], schema: qualified[2], table: qualified[3] };
}

function parseOperator(line: string): Parsed | undefined {
  const matched = TREE_OPERATOR.exec(line);
  if (!matched) return undefined;
  const [, prefix, , text, id] = matched;

  const scan = SCAN.exec(text);
  if (scan) {
    // A read says which format it read and which table, and both belong on the
    // card: the format is how the operator differs from another read.
    return {
      indent: indentOf(prefix),
      id,
      nodeType: `Scan ${scan[1]}`,
      subject: scan[2],
      relation: toRelation(scan[2]),
      children: [],
    };
  }
  // Everything else is an operator name, and then whatever Spark prints beside
  // it: a join's type and which side it built.
  const space = text.indexOf(" ");
  return {
    indent: indentOf(prefix),
    id,
    nodeType: space > 0 ? text.slice(0, space) : text,
    subject: space > 0 ? text.slice(space + 1) : undefined,
    children: [],
  };
}

/** The detail blocks, by the number their operator carries in the tree. */
function parseDetails(lines: string[]): Map<string, Detail> {
  const details = new Map<string, Detail>();
  let current: Detail | undefined;
  for (const line of lines) {
    const header = DETAIL_HEADER.exec(line);
    if (header) {
      current = { name: header[2], codegen: header[3], properties: [] };
      details.set(header[1], current);
      continue;
    }
    if (!current || !line.trim()) continue;
    const detail = DETAIL_LINE.exec(line);
    if (detail) {
      addPlanProperty(
        current.properties,
        planLabel(detail[1].trim()),
        detail[2].trim()
      );
    } else {
      // A line the block continues onto, such as the rest of a long condition.
      addPlanProperty(current.properties, "Details", line.trim());
    }
  }
  return details;
}

function toPlanNode(
  parsed: Parsed,
  details: Map<string, Detail>,
  id: string,
  depth: number
): PlanNode {
  checkPlanDepth(depth);
  const children = parsed.children.map((child, index) =>
    toPlanNode(child, details, `${id}.${index}`, depth + 1)
  );
  const detail = details.get(parsed.id);
  const properties: PlanProperty[] = [];
  // The number Spark's own plan prints for this operator, so a reader can line
  // the node up with the output they may also be looking at.
  addPlanProperty(properties, "Operator", parsed.id);
  if (detail?.codegen) {
    addPlanProperty(properties, "Codegen", detail.codegen);
  }
  for (const property of detail?.properties ?? []) {
    addPlanProperty(properties, property.label, property.value);
  }
  return {
    id,
    nodeType: parsed.nodeType,
    subject: parsed.subject,
    relation: parsed.relation,
    properties,
    warnings: [],
    children,
  };
}

export function parseDatabricksPlan(source: string): PlanParseResult {
  if (!source.trim()) {
    return { ok: false, message: DATABRICKS_PLAN_EMPTY_MESSAGE };
  }

  const lines = source.split("\n");
  const start = lines.findIndex((line) => line.trim() === PLAN_HEADER);
  if (start < 0) {
    return { ok: false, message: DATABRICKS_PLAN_NO_PLAN_MESSAGE };
  }

  let root: Parsed | undefined;
  // The operators from the root down to the one being read, so a line's indent
  // says which of them it belongs under.
  const open: Parsed[] = [];
  let at = start + 1;
  for (; at < lines.length; at++) {
    const line = lines[at];
    if (!line.trim()) continue;
    // The tree ends where the detail blocks begin.
    if (DETAIL_HEADER.test(line)) break;
    const parsed = parseOperator(line);
    if (!parsed) continue;
    while (open.length > 0 && open[open.length - 1].indent >= parsed.indent) {
      open.pop();
    }
    const parent = open[open.length - 1];
    if (parent) parent.children.push(parsed);
    else if (!root) root = parsed;
    else break;
    open.push(parsed);
  }
  if (!root) {
    return { ok: false, message: DATABRICKS_PLAN_NO_PLAN_MESSAGE };
  }

  const details = parseDetails(lines.slice(at));
  return parseWithDepthLimit(() => toPlanNode(root, details, "0", 0));
}
