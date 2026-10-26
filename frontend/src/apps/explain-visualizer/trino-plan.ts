import {
  addPlanProperty,
  checkPlanDepth,
  isJsonRecord,
  type JsonRecord,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  type PlanRelation,
  parseWithDepthLimit,
  planLabel,
} from "./plan-model";

/**
 * Maps `EXPLAIN (FORMAT JSON)` output onto the engine-neutral plan model.
 *
 * Trino does not return one tree. It returns the fragments it would distribute
 * across the cluster, keyed by number, and each `RemoteSource` names the
 * fragments feeding it — so the tree is spliced back together here, which is
 * what Trino's own text plan shows a reader anyway.
 *
 * Its row estimates are read and its cost estimates are not. `cpuCost` is each
 * operation's own rather than its subtree's, an operator Trino fused reports
 * one per fused step, and most operators report `NaN` outright on a query
 * whose statistics are incomplete. Ranking a plan where some operators state a
 * cost and the rest state NaN would point at whichever happened to be known,
 * so the numbers stay on the node and nothing is ranked by them.
 */

const CHILDREN_KEY = "children";
const ROOT_FRAGMENT = "0";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set([
  "id",
  "name",
  "descriptor",
  "estimates",
  CHILDREN_KEY,
]);

/** The operator whose children are the fragments it names, not its own. */
const REMOTE_SOURCE = "RemoteSource";

/** Trino writes a table as `catalog:schema:table`. */
const QUALIFIED_TABLE = /^([^:]+):([^:]+):([^:]+)$/;

export const TRINO_PLAN_EMPTY_MESSAGE =
  "Trino returned no query plan for this statement.";
export const TRINO_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. Trino's EXPLAIN (FORMAT JSON) output is expected.";
export const TRINO_PLAN_NO_PLAN_MESSAGE =
  "The query plan JSON contains no plan fragment to start from.";

function optionalNumber(value: unknown): number | undefined {
  // Trino writes an estimate it could not make as the string "NaN".
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

function descriptorOf(raw: JsonRecord): JsonRecord {
  return isJsonRecord(raw.descriptor) ? raw.descriptor : {};
}

/** The estimates Trino made, one per step of an operator it fused. */
function estimatesOf(raw: JsonRecord): JsonRecord[] {
  return Array.isArray(raw.estimates) ? raw.estimates.filter(isJsonRecord) : [];
}

/**
 * The fragment numbers a `RemoteSource` reads, which Trino writes as a string
 * holding a bracketed list: `[1]`, or `[1, 2]` where it unions them.
 */
function sourceFragments(raw: JsonRecord): string[] {
  const ids = stringValue(descriptorOf(raw).sourceFragmentIds);
  if (!ids) return [];
  return ids
    .replace(/^\[|\]$/g, "")
    .split(",")
    .map((id) => id.trim())
    .filter(Boolean);
}

function toRelation(raw: JsonRecord): PlanRelation | undefined {
  const table = stringValue(descriptorOf(raw).table);
  const qualified = table ? QUALIFIED_TABLE.exec(table) : null;
  if (!qualified) return undefined;
  // Trino's catalog is the database Bytebase connects to; its schema is the
  // schema, so a node answers a lookup without guessing either.
  return { database: qualified[1], schema: qualified[2], table: qualified[3] };
}

/**
 * What the operator does, for the node's card: the table it reads, or else the
 * descriptor Trino prints beside the operator in its own text plan.
 */
function toSubject(raw: JsonRecord): string | undefined {
  const descriptor = descriptorOf(raw);
  const table = stringValue(descriptor.table);
  if (table) return table;
  const described = Object.entries(descriptor)
    .map(([key, value]) => [key, formatValue(value)] as const)
    .filter(([, value]) => value)
    .map(([key, value]) => `${key}: ${value}`);
  return described.length ? described.join(", ") : undefined;
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const [key, value] of Object.entries(descriptorOf(raw))) {
    addPlanProperty(properties, planLabel(key), formatValue(value));
  }
  const details = Array.isArray(raw.details) ? raw.details : [];
  for (const detail of details) {
    addPlanProperty(properties, "Details", formatValue(detail));
  }
  // Every estimate, in the engine's own words, since none of them is ranked.
  for (const estimate of estimatesOf(raw)) {
    const described = Object.entries(estimate)
      .map(([key, value]) => `${planLabel(key)}: ${formatValue(value)}`)
      .join(", ");
    addPlanProperty(properties, "Estimate", described);
  }
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key) || key === "details") continue;
    addPlanProperty(properties, planLabel(key), formatValue(value));
  }
  return properties;
}

/**
 * The rows the operator produces. An operator Trino fused reports one estimate
 * per fused step, and the last of them is what leaves the operator.
 */
function toRows(raw: JsonRecord): number | undefined {
  const estimates = estimatesOf(raw);
  const last = estimates[estimates.length - 1];
  return last ? optionalNumber(last.outputRowCount) : undefined;
}

/**
 * @param seen - The fragments on the path to this node. A fragment that reads
 * itself, directly or through another, would never finish; the rest of the
 * plan still draws.
 */
function toPlanNode(
  raw: JsonRecord,
  fragments: Map<string, JsonRecord>,
  id: string,
  seen: Set<string>,
  depth: number
): PlanNode {
  checkPlanDepth(depth);
  const name = stringValue(raw.name) ?? "Unknown";
  const children: PlanNode[] = [];
  const add = (child: JsonRecord) =>
    children.push(
      toPlanNode(child, fragments, `${id}.${children.length}`, seen, depth + 1)
    );

  const own = Array.isArray(raw[CHILDREN_KEY]) ? raw[CHILDREN_KEY] : [];
  for (const child of own.filter(isJsonRecord)) add(child);

  if (name === REMOTE_SOURCE) {
    for (const fragment of sourceFragments(raw)) {
      const source = fragments.get(fragment);
      if (!source || seen.has(fragment)) continue;
      seen.add(fragment);
      add(source);
      seen.delete(fragment);
    }
  }

  return {
    id,
    nodeType: name,
    subject: toSubject(raw),
    relation: toRelation(raw),
    rows: toRows(raw),
    properties: toProperties(raw),
    warnings: [],
    children,
  };
}

export function parseTrinoPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: TRINO_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: TRINO_PLAN_INVALID_JSON_MESSAGE };
  }
  if (!isJsonRecord(parsed)) {
    return { ok: false, message: TRINO_PLAN_INVALID_JSON_MESSAGE };
  }

  const fragments = new Map<string, JsonRecord>();
  for (const [id, fragment] of Object.entries(parsed)) {
    if (isJsonRecord(fragment)) fragments.set(id, fragment);
  }
  // Fragment 0 is the one that returns the query's rows; the rest feed it.
  const root = fragments.get(ROOT_FRAGMENT) ?? [...fragments.values()][0];
  if (!root) {
    return { ok: false, message: TRINO_PLAN_NO_PLAN_MESSAGE };
  }

  return parseWithDepthLimit(() =>
    toPlanNode(root, fragments, "0", new Set([ROOT_FRAGMENT]), 0)
  );
}
