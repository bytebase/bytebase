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
 * Maps StarRocks' and Doris's `EXPLAIN` text onto the engine-neutral plan
 * model. StarRocks forked from Doris, and the plan is what did not diverge:
 * the same fragments, the same tree, the same attributes. What differs is how
 * an operator spells itself, which is why one parser reads both.
 *
 * Neither publishes a plan as a document — asked for one they reject the
 * statement — so this reads text, for the reason the Redshift parser does and
 * with the same two things holding off the hazard §4 of the design names: the
 * plan arrives one row per line, so a newline inside a literal stays inside
 * its row, and an operator is recognized only by its own numbered heading.
 *
 * What it reads is two nested structures. A plan is a list of fragments, each
 * the piece of work one set of nodes does, joined where an `EXCHANGE` operator
 * in one fragment names the exchange another fragment sends to. Within a
 * fragment the operators are drawn as a tree: each is printed under the one it
 * feeds, and a second input is introduced by `|----` and drawn in a branch of
 * its own. Which operator a line belongs to is therefore the column its number
 * starts at, counting through the branch bars.
 *
 * Neither states a cost, so these plans rank nothing. What a scan does state
 * is the table, the predicates pushed into it, and the rows it expects.
 */

/** `PLAN FRAGMENT 2`, optionally followed by the planner's own id. */
const FRAGMENT = /^\s*PLAN FRAGMENT\s+(\d+)/;

/** An operator: `  4:HASH JOIN`, or `  |----3:Project` to start a branch. */
const OPERATOR = /^([\s|-]*?)(\d+):(\S.*)$/;

/** The exchange a fragment sends its rows to, which names where it plugs in. */
const SINK_EXCHANGE = /^\s*(?:EXCHANGE ID|OutPut Exchange Id):\s*0*(\d+)/;

/** An operator's own line, `key: value` or `key=value`. */
const ATTRIBUTE = /^([A-Za-z][A-Za-z0-9 _]*?)\s*[:=]\s*(.*)$/;

/**
 * The planner's own id for an operator, which Doris prints after its name.
 * Keeping it as a property leaves the name alone, which is what a reader
 * recognizes an operator by.
 */
const PLAN_NODE_ID = /\((\d+)\)$/;

/** A scanned table: qualified, then whatever the statement called it. */
const SCANNED_TABLE = /^([^\s(]+)(?:\([^)]*\))?$/;

/**
 * The rule Doris prints its statistics under, once the plan is done. The whole
 * line is that rule, which is what keeps a value from ending the plan early.
 */
const SEPARATOR = /^\s*={5,}[A-Z\s]*={5,}\s*$/;

export const STARROCKS_PLAN_EMPTY_MESSAGE =
  "The statement produced no query plan to draw.";
export const STARROCKS_PLAN_NO_PLAN_MESSAGE =
  "The query plan has no numbered operator in any of its fragments.";

interface Parsed {
  readonly column: number;
  readonly id: string;
  readonly nodeType: string;
  readonly properties: PlanProperty[];
  readonly children: Parsed[];
}

interface Fragment {
  root?: Parsed;
  /** The exchange this fragment feeds, by its operator number. */
  sink?: string;
}

/**
 * What one line says about its operator. Doris packs several onto a line —
 * `cardinality=1, avgRowSize=0.0` — where StarRocks gives each its own, so a
 * line is split only where every piece is a key and a value with no space in
 * it. A predicate, a join condition or a list of ids has a space, and stays
 * whole.
 */
function attributesIn(text: string): [string, string][] {
  const pieces = text.split(", ");
  const packed: [string, string][] = [];
  for (const piece of pieces) {
    const attribute = ATTRIBUTE.exec(piece);
    if (!attribute || attribute[2].includes(" ")) break;
    packed.push([planLabel(attribute[1]), attribute[2]]);
  }
  if (pieces.length > 1 && packed.length === pieces.length) return packed;

  const whole = ATTRIBUTE.exec(text);
  return whole ? [[planLabel(whole[1]), whole[2]]] : [["Detail", text]];
}

/**
 * The table a scan reads. Doris qualifies it with the database and adds what
 * the statement called it; StarRocks writes the stored name alone.
 */
function toRelation(
  properties: readonly PlanProperty[]
): PlanRelation | undefined {
  const named = properties.find((property) => property.label === "Table");
  const scanned = named ? SCANNED_TABLE.exec(named.value) : null;
  if (!scanned) return undefined;
  const qualified = scanned[1];
  const separator = qualified.lastIndexOf(".");
  const table = qualified.slice(separator + 1);
  if (!table) return undefined;
  return separator > 0
    ? { database: qualified.slice(0, separator), table }
    : { table };
}

/**
 * A number an operator stated. Doris groups a standalone cardinality by
 * thousands — `cardinality=4,499` — and prints the same number plain where it
 * packs it onto a line, so the separators come off before it is read.
 */
function numberOf(
  properties: readonly PlanProperty[],
  label: string
): number | undefined {
  const value = properties.find((property) => property.label === label)?.value;
  const parsed =
    value === undefined ? Number.NaN : Number(value.replaceAll(",", ""));
  return Number.isFinite(parsed) ? parsed : undefined;
}

/**
 * @param exchanges - The fragment root each `EXCHANGE` operator reads from, so
 * the fragments are drawn as the one tree they describe.
 * @param seen - The exchanges already spliced on this path, since a plan whose
 * fragments fed each other would never finish.
 */
function toPlanNode(
  parsed: Parsed,
  exchanges: ReadonlyMap<string, Parsed>,
  id: string,
  seen: Set<string>,
  depth: number
): PlanNode {
  checkPlanDepth(depth);
  const children = [...parsed.children];
  const source = exchanges.get(parsed.id);
  if (source && !seen.has(parsed.id)) {
    seen.add(parsed.id);
    children.push(source);
  }
  const drawn = children.map((child, index) =>
    toPlanNode(child, exchanges, `${id}.${index}`, seen, depth + 1)
  );
  if (source) seen.delete(parsed.id);

  const relation = toRelation(parsed.properties);
  const width = numberOf(parsed.properties, "Avg row size");
  return {
    id,
    nodeType: parsed.nodeType,
    subject:
      relation?.table ??
      parsed.properties.find((property) =>
        ["Predicates", "Equal join conjunct", "Group by"].includes(
          property.label
        )
      )?.value,
    relation,
    rows: numberOf(parsed.properties, "Cardinality"),
    // Doris prints `avgRowSize=0.0` where it has no column statistics, and no
    // row is zero bytes wide, so that is not a width it stated.
    width: width === 0 ? undefined : width,
    properties: parsed.properties,
    warnings: [],
    children: drawn,
  };
}

export function parseStarrocksPlan(source: string): PlanParseResult {
  if (!source.trim()) {
    return { ok: false, message: STARROCKS_PLAN_EMPTY_MESSAGE };
  }

  const fragments: Fragment[] = [];
  let fragment: Fragment | undefined;
  // The operators from the fragment's root down to the one being read; an
  // operator's column says which of them it hangs under.
  let open: Parsed[] = [];
  let current: Parsed | undefined;

  for (const line of source.split("\n")) {
    if (FRAGMENT.test(line)) {
      fragment = {};
      fragments.push(fragment);
      open = [];
      current = undefined;
      continue;
    }
    if (SEPARATOR.test(line)) {
      // Nothing past the statistics describes an operator.
      fragment = undefined;
      current = undefined;
      continue;
    }
    if (!fragment) continue;

    const sink = SINK_EXCHANGE.exec(line);
    if (sink && !current) {
      fragment.sink = sink[1];
      continue;
    }

    const operator = OPERATOR.exec(line);
    if (operator) {
      const [, lead, number, name] = operator;
      const properties: PlanProperty[] = [];
      let nodeType = name.trim();
      const planNode = PLAN_NODE_ID.exec(nodeType);
      if (planNode) {
        nodeType = nodeType.slice(0, planNode.index).trim();
        addPlanProperty(properties, "Plan node id", planNode[1]);
      }
      const node: Parsed = {
        column: lead.length,
        id: number,
        nodeType,
        properties,
        children: [],
      };
      while (open.length > 0 && open[open.length - 1].column > node.column) {
        open.pop();
      }
      const parent = open[open.length - 1];
      if (parent) parent.children.push(node);
      else if (!fragment.root) fragment.root = node;
      else fragment.root.children.push(node);
      open.push(node);
      current = node;
      continue;
    }

    // Anything else describes the operator above it, once the bars drawing the
    // branch it sits in are taken off.
    if (!current) continue;
    const text = line.replace(/^[\s|]+/, "").trim();
    if (!text) continue;
    for (const [label, value] of attributesIn(text)) {
      addPlanProperty(current.properties, label, value);
    }
  }

  const root = fragments.find((each) => each.root)?.root;
  if (!root) {
    return { ok: false, message: STARROCKS_PLAN_NO_PLAN_MESSAGE };
  }
  const exchanges = new Map<string, Parsed>();
  for (const each of fragments) {
    if (each.sink && each.root) exchanges.set(each.sink, each.root);
  }

  return parseWithDepthLimit(() =>
    toPlanNode(root, exchanges, "0", new Set(), 0)
  );
}
