import {
  checkPlanDepth,
  isJsonRecord,
  type JsonRecord,
  PLAN_FULL_TABLE_SCAN,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  type PlanWarning,
  parseWithDepthLimit,
  planSelfCost,
} from "./plan-model";

/**
 * Maps `EXPLAIN (FORMAT JSON)` output onto the engine-neutral plan model.
 *
 * PostgreSQL emits a one-element array whose element holds a `Plan` object;
 * children are nested under `Plans`. Every other key on a node is carried
 * through verbatim as a property so the detail pane stays useful when a future
 * server version reports something this file has never heard of.
 */

const NODE_TYPE_KEY = "Node Type";
const CHILDREN_KEY = "Plans";

/**
 * Keys the model promotes to typed fields. They are dropped from the property
 * list because the diagram and detail pane already render them.
 */
const PROMOTED_KEYS = new Set([
  NODE_TYPE_KEY,
  CHILDREN_KEY,
  "Startup Cost",
  "Total Cost",
  "Plan Rows",
  "Plan Width",
  "Parent Relationship",
  "Subplan Name",
]);

export const POSTGRES_PLAN_EMPTY_MESSAGE =
  "PostgreSQL returned no query plan for this statement.";
export const POSTGRES_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. EXPLAIN (FORMAT JSON) output is expected.";
export const POSTGRES_PLAN_NO_PLAN_MESSAGE =
  'The query plan JSON does not contain a "Plan" object.';

/** Node types that read a whole table, spelled as `toNodeType` names them. */
const FULL_TABLE_SCAN_TYPES = new Set(["Seq Scan", "Parallel Seq Scan"]);

/**
 * Total cost below which reading a whole table is not worth flagging.
 *
 * Under PostgreSQL's default settings a sequential scan costs `seq_page_cost`
 * (1.0) per page plus `cpu_tuple_cost` (0.01) per row, so 100 cost units is
 * roughly a 50-page, 5,000-row table — about 400 KB. Reading all of that is a
 * handful of page fetches that no index lookup would beat, and flagging it
 * would bury the scans a reader can act on.
 */
const FULL_SCAN_COST_THRESHOLD = 100;

function toWarnings(nodeType: string, totalCost: number): PlanWarning[] {
  return FULL_TABLE_SCAN_TYPES.has(nodeType) &&
    totalCost >= FULL_SCAN_COST_THRESHOLD
    ? [PLAN_FULL_TABLE_SCAN]
    : [];
}

function toNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return 0;
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

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const [label, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(label)) continue;
    properties.push({ label, value: formatValue(value) });
  }
  return properties;
}

/**
 * Conditions worth putting on the card when the node names no relation, in the
 * order PostgreSQL itself prints them under a node. A join or sort says far more
 * through its condition than through its node type alone.
 */
const SUBJECT_CONDITION_KEYS = [
  "Hash Cond",
  "Merge Cond",
  "Join Filter",
  "Index Cond",
  "Recheck Cond",
  "TID Cond",
  "One-Time Filter",
  "Filter",
  "Sort Key",
  "Group Key",
  "Presorted Key",
] as const;

function toSubject(raw: JsonRecord): string | undefined {
  const relation = raw["Relation Name"];
  if (typeof relation === "string" && relation) return relation;
  const index = raw["Index Name"];
  if (typeof index === "string" && index) return index;
  const cte = raw["CTE Name"];
  if (typeof cte === "string" && cte) return cte;
  const fn = raw["Function Name"];
  if (typeof fn === "string" && fn) return fn;
  for (const key of SUBJECT_CONDITION_KEYS) {
    const value = raw[key];
    if (value === undefined || value === null) continue;
    const formatted = formatValue(value);
    if (formatted) return formatted;
  }
  return undefined;
}

/**
 * Node type as PostgreSQL itself would name it.
 *
 * The JSON format reports the bare node type plus modifier fields, while the
 * text format folds them into the name — an Aggregate with `Strategy: Hashed`
 * prints as "HashAggregate". Following the text format keeps the diagram's
 * vocabulary the same as every other tool and doc a reader has seen, and
 * distinguishes nodes that would otherwise all read "Aggregate".
 */
function toNodeType(raw: JsonRecord): string {
  const nodeType = raw[NODE_TYPE_KEY];
  if (typeof nodeType !== "string" || !nodeType) return "Unknown";

  let name = nodeType;
  const strategy = raw["Strategy"];
  if (nodeType === "Aggregate" || nodeType === "SetOp") {
    if (strategy === "Hashed") {
      name = nodeType === "Aggregate" ? "HashAggregate" : "HashSetOp";
    } else if (strategy === "Sorted" && nodeType === "Aggregate") {
      name = "GroupAggregate";
    } else if (strategy === "Mixed") {
      name = "MixedAggregate";
    }
  }

  // A ModifyTable is named after what it modifies: "Update", not "ModifyTable".
  if (nodeType === "ModifyTable") {
    const operation = raw["Operation"];
    if (typeof operation === "string" && operation) name = operation;
  }
  // A set operation is named with the set it computes: "HashSetOp Intersect".
  if (nodeType === "SetOp") {
    const command = raw["Command"];
    if (typeof command === "string" && command) name = `${name} ${command}`;
  }

  const partialMode = raw["Partial Mode"];
  if (partialMode === "Partial" || partialMode === "Finalize") {
    name = `${partialMode} ${name}`;
  }
  if (raw["Parallel Aware"] === true) {
    name = `Parallel ${name}`;
  }
  if (raw["Async Capable"] === true) {
    name = `Async ${name}`;
  }
  return name;
}

function toRelationship(raw: JsonRecord): string | undefined {
  const subplan = raw["Subplan Name"];
  if (typeof subplan === "string" && subplan) return subplan;
  const parent = raw["Parent Relationship"];
  // "Outer" is the default relationship and carries no information.
  if (typeof parent === "string" && parent && parent !== "Outer") return parent;
  return undefined;
}

function toChildren(raw: JsonRecord, id: string, depth: number): PlanNode[] {
  const rawChildren = raw[CHILDREN_KEY];
  if (!Array.isArray(rawChildren)) return [];
  return rawChildren
    .filter(isJsonRecord)
    .map((child, index) => toPlanNode(child, `${id}.${index}`, depth + 1));
}

function toPlanNode(raw: JsonRecord, id: string, depth: number): PlanNode {
  checkPlanDepth(depth);
  const children = toChildren(raw, id, depth);
  const nodeType = toNodeType(raw);
  const totalCost = toNumber(raw["Total Cost"]);
  return {
    id,
    nodeType,
    subject: toSubject(raw),
    relationship: toRelationship(raw),
    startupCost: toNumber(raw["Startup Cost"]),
    totalCost,
    selfCost: planSelfCost(totalCost, children),
    rows: toNumber(raw["Plan Rows"]),
    width: toNumber(raw["Plan Width"]),
    properties: toProperties(raw),
    warnings: toWarnings(nodeType, totalCost),
    children,
  };
}

/** The text between the first `[`/`{` and the last `]`/`}`, when both exist. */
function extractBracketedSpan(source: string): string | undefined {
  const starts = [source.indexOf("["), source.indexOf("{")].filter(
    (i) => i >= 0
  );
  const ends = [source.lastIndexOf("]"), source.lastIndexOf("}")];
  const start = Math.min(...starts);
  const end = Math.max(...ends);
  if (starts.length === 0 || end <= start) return undefined;
  return source.slice(start, end + 1);
}

export function parsePostgresPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: POSTGRES_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    // A plan can arrive with something wrapped around it — a psql command tag,
    // a copied prompt line, a trailing "(1 row)". Retry on the span between the
    // outermost brackets before giving up, so noise around the plan does not
    // cost the user the diagram.
    const embedded = extractBracketedSpan(trimmed);
    if (embedded === undefined) {
      return { ok: false, message: POSTGRES_PLAN_INVALID_JSON_MESSAGE };
    }
    try {
      parsed = JSON.parse(embedded);
    } catch {
      return { ok: false, message: POSTGRES_PLAN_INVALID_JSON_MESSAGE };
    }
  }

  const entry = Array.isArray(parsed) ? parsed[0] : parsed;
  const plan = isJsonRecord(entry) ? entry["Plan"] : undefined;
  if (!isJsonRecord(plan)) {
    return { ok: false, message: POSTGRES_PLAN_NO_PLAN_MESSAGE };
  }

  return parseWithDepthLimit(() => toPlanNode(plan, "0", 0));
}
