import {
  addPlanProperty,
  checkPlanDepth,
  isJsonRecord,
  type JsonRecord,
  PLAN_FULL_TABLE_SCAN,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  type PlanRelation,
  type PlanWarning,
  parseWithDepthLimit,
  planLabel,
  planSizeBudget,
} from "./plan-model";

/**
 * Maps `EXPLAIN USING JSON` output onto the engine-neutral plan model.
 *
 * Snowflake states each operation with the ids of its parents rather than its
 * children, so the tree is those edges turned around. The operation that names
 * no parent is the root: the `Result` that returns the query's rows.
 *
 * It estimates neither cost nor rows. What it states instead is how much of
 * each table the query would read — the micro-partitions assigned out of the
 * table's total, and the bytes they hold — which is what a Snowflake plan is
 * read for. Those numbers stay on the node and nothing is ranked by them: only
 * a scan reports them, so a ranking would weigh scans against parents that
 * report nothing.
 */

const PARENTS_KEY = "parentOperators";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set(["id", "operation", "objects", PARENTS_KEY]);

/** The operation that reads a table, and the only one that reports pruning. */
const TABLE_SCAN = "TableScan";

/** Snowflake writes an object as `database.schema.name`. */
const QUALIFIED_OBJECT = /^([^".]+)\.([^".]+)\.([^".]+)$/;

export const SNOWFLAKE_PLAN_EMPTY_MESSAGE =
  "Snowflake returned no query plan for this statement.";
export const SNOWFLAKE_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. Snowflake's EXPLAIN USING JSON output is expected.";
export const SNOWFLAKE_PLAN_NO_PLAN_MESSAGE =
  "The query plan JSON contains no operation to start from.";

function optionalNumber(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return undefined;
}

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.map((entry) => formatValue(entry)).join(", ");
  }
  return JSON.stringify(value);
}

/** The operation's id as the plan writes it, which its children refer to. */
function operationID(raw: JsonRecord): string | undefined {
  const id = optionalNumber(raw.id);
  return id === undefined ? stringValue(raw.id) : String(id);
}

function parentIDs(raw: JsonRecord): string[] {
  if (!Array.isArray(raw[PARENTS_KEY])) return [];
  return raw[PARENTS_KEY]
    .map((parent) => {
      const id = optionalNumber(parent);
      return id === undefined ? stringValue(parent) : String(id);
    })
    .filter((id): id is string => id !== undefined);
}

/** The first object the operation names, which for a scan is its table. */
function firstObject(raw: JsonRecord): string | undefined {
  if (Array.isArray(raw.objects)) {
    for (const object of raw.objects) {
      const name = stringValue(object);
      if (name) return name;
    }
    return undefined;
  }
  return stringValue(raw.objects);
}

/**
 * The table the operation reads. Snowflake qualifies an object fully, so a
 * node answers a lookup without guessing the database or the schema; a name
 * written any other way — a quoted identifier holding a dot, a stage — stays on
 * the card as the subject alone.
 */
function toRelation(raw: JsonRecord): PlanRelation | undefined {
  const object = firstObject(raw);
  const qualified = object ? QUALIFIED_OBJECT.exec(object) : null;
  if (!qualified) return undefined;
  return { database: qualified[1], schema: qualified[2], table: qualified[3] };
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  // The id Snowflake's own tabular plan prints for this operation, which is
  // what a reader comparing the two has in front of them.
  const id = operationID(raw);
  if (id !== undefined) addPlanProperty(properties, "Operation", id);
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key)) continue;
    addPlanProperty(properties, planLabel(key), formatValue(value));
  }
  return properties;
}

/**
 * A scan that was assigned every micro-partition the table has read all of it.
 * A table of one partition is not one a reader can improve on, as in the other
 * parsers.
 */
function toWarnings(raw: JsonRecord): PlanWarning[] {
  if (stringValue(raw.operation) !== TABLE_SCAN) return [];
  const total = optionalNumber(raw.partitionsTotal);
  const assigned = optionalNumber(raw.partitionsAssigned);
  if (total === undefined || assigned === undefined) return [];
  return total > 1 && assigned >= total ? [PLAN_FULL_TABLE_SCAN] : [];
}

/**
 * @param seen - The operations on the path to this node. An operation that
 * reaches itself through its parents would never finish; the rest of the plan
 * still draws.
 * @param count - The size budget, since an operation named by two parents is
 * drawn under each of them.
 */
function toPlanNode(
  raw: JsonRecord,
  children: Map<string, JsonRecord[]>,
  id: string,
  seen: Set<string>,
  depth: number,
  count: () => void
): PlanNode {
  checkPlanDepth(depth);
  count();
  const operation = operationID(raw);
  const own = operation ? (children.get(operation) ?? []) : [];
  const built: PlanNode[] = [];
  for (const child of own) {
    const childID = operationID(child);
    if (childID !== undefined && seen.has(childID)) continue;
    if (childID !== undefined) seen.add(childID);
    built.push(
      toPlanNode(
        child,
        children,
        `${id}.${built.length}`,
        seen,
        depth + 1,
        count
      )
    );
    if (childID !== undefined) seen.delete(childID);
  }

  // An operation that names no object says what it does instead, which is the
  // join keys on a join and the projected columns on a result.
  const expressions = formatValue(raw.expressions);
  return {
    id,
    nodeType: stringValue(raw.operation) ?? "Unknown",
    subject: firstObject(raw) ?? (expressions || undefined),
    relation: toRelation(raw),
    properties: toProperties(raw),
    warnings: toWarnings(raw),
    children: built,
  };
}

/** The operations of the step this plan is drawn from, and how many there are. */
function stepOperations(parsed: JsonRecord): {
  operations: JsonRecord[];
  steps: number;
} {
  const steps = Array.isArray(parsed.Operations)
    ? parsed.Operations.filter(
        (step): step is unknown[] => Array.isArray(step) && step.length > 0
      ).map((step) => step.filter(isJsonRecord))
    : [];
  // Snowflake compiles most statements into one step. Where it makes several,
  // the last is the one that returns the query's rows, and the root's Steps
  // property says how many there were.
  return { operations: steps[steps.length - 1] ?? [], steps: steps.length };
}

export function parseSnowflakePlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: SNOWFLAKE_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: SNOWFLAKE_PLAN_INVALID_JSON_MESSAGE };
  }
  if (!isJsonRecord(parsed)) {
    return { ok: false, message: SNOWFLAKE_PLAN_INVALID_JSON_MESSAGE };
  }

  const { operations, steps } = stepOperations(parsed);
  const children = new Map<string, JsonRecord[]>();
  for (const operation of operations) {
    for (const parent of parentIDs(operation)) {
      const siblings = children.get(parent);
      if (siblings) siblings.push(operation);
      else children.set(parent, [operation]);
    }
  }
  // The operation that names no parent returns the query's rows. A plan whose
  // operations all name one — which would mean a cycle — has no root to draw.
  const root = operations.find(
    (operation) => parentIDs(operation).length === 0
  );
  if (!root) {
    return { ok: false, message: SNOWFLAKE_PLAN_NO_PLAN_MESSAGE };
  }

  const count = planSizeBudget();
  const rootID = operationID(root);
  const result = parseWithDepthLimit(() =>
    toPlanNode(
      root,
      children,
      "0",
      new Set(rootID === undefined ? [] : [rootID]),
      0,
      count
    )
  );
  if (!result.ok) return result;

  // What the plan says about the query as a whole: the partitions it would
  // read of every table together, and the step count where there was more than
  // one. Both belong on the root, which is the query's own node.
  const properties = [...result.tree.root.properties];
  if (isJsonRecord(parsed.GlobalStats)) {
    for (const [key, value] of Object.entries(parsed.GlobalStats)) {
      addPlanProperty(properties, planLabel(key), formatValue(value));
    }
  }
  if (steps > 1) addPlanProperty(properties, "Steps", String(steps));
  const withGlobals = { ...result.tree.root, properties };
  return {
    ok: true,
    tree: {
      ...result.tree,
      root: withGlobals,
      nodes: [withGlobals, ...result.tree.nodes.slice(1)],
    },
  };
}
