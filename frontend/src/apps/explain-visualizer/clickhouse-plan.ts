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
 * Maps `EXPLAIN json = 1` output onto the engine-neutral plan model.
 *
 * ClickHouse borrows PostgreSQL's spelling — a one-element array holding a
 * `Plan`, children under `Plans`, the operator under `Node Type` — but not its
 * numbers: it publishes no cost or row estimate in any EXPLAIN it offers, so a
 * ClickHouse plan is a shape and a set of descriptions, and the viewer says it
 * ranks nothing.
 *
 * What it does report, with `indexes = 1`, is how much each index let it skip,
 * which is the question a reader of a ClickHouse plan actually has.
 */

const PLAN_KEY = "Plan";
const CHILDREN_KEY = "Plans";
const NODE_TYPE_KEY = "Node Type";
const DESCRIPTION_KEY = "Description";
const INDEXES_KEY = "Indexes";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set([NODE_TYPE_KEY, DESCRIPTION_KEY, CHILDREN_KEY]);

/** The operator that reads a table, and so names one in its description. */
const TABLE_READERS = /^ReadFrom/;

export const CLICKHOUSE_PLAN_EMPTY_MESSAGE =
  "ClickHouse returned no query plan for this statement.";
export const CLICKHOUSE_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. EXPLAIN json = 1 output is expected.";
export const CLICKHOUSE_PLAN_NO_PLAN_MESSAGE =
  'The ClickHouse query plan JSON does not contain a "Plan" object.';

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" && value ? value : undefined;
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return JSON.stringify(value);
}

/**
 * The table a read names, where it names one.
 *
 * ClickHouse writes it qualified, as `default.orders`, so an unqualified
 * description is left alone rather than guessed at — and a bare name would in
 * any case be the one the statement used.
 */
function toRelation(
  nodeType: string,
  raw: JsonRecord
): PlanRelation | undefined {
  if (!TABLE_READERS.test(nodeType)) return undefined;
  const description = stringValue(raw[DESCRIPTION_KEY]);
  if (!description) return undefined;
  const separator = description.lastIndexOf(".");
  if (separator <= 0 || separator === description.length - 1) return undefined;
  return {
    database: description.slice(0, separator),
    table: description.slice(separator + 1),
  };
}

/**
 * Each index the read consulted, as one property: what it is, the condition it
 * was given, and how many of the table's parts and granules survived it. That
 * last pair is the whole point of the entry — granules in against granules out
 * is how much work the index saved.
 */
function indexProperties(raw: JsonRecord, properties: PlanProperty[]) {
  const indexes = raw[INDEXES_KEY];
  if (!Array.isArray(indexes)) return;
  for (const index of indexes.filter(isJsonRecord)) {
    const type = stringValue(index.Type) ?? "Index";
    const described = Object.entries(index)
      .filter(([key]) => key !== "Type")
      .map(([key, value]) => `${planLabel(key)}: ${formatValue(value)}`)
      .join(", ");
    addPlanProperty(properties, `${type} index`, described);
  }
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key) || key === INDEXES_KEY) continue;
    addPlanProperty(properties, planLabel(key), formatValue(value));
  }
  indexProperties(raw, properties);
  return properties;
}

function toPlanNode(raw: JsonRecord, id: string, depth: number): PlanNode {
  checkPlanDepth(depth);
  const rawChildren = raw[CHILDREN_KEY];
  const children = (Array.isArray(rawChildren) ? rawChildren : [])
    .filter(isJsonRecord)
    .map((child, index) => toPlanNode(child, `${id}.${index}`, depth + 1));

  const nodeType = stringValue(raw[NODE_TYPE_KEY]) ?? "Unknown";
  return {
    id,
    nodeType,
    subject: stringValue(raw[DESCRIPTION_KEY]),
    relation: toRelation(nodeType, raw),
    properties: toProperties(raw),
    warnings: [],
    children,
  };
}

export function parseClickhousePlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: CLICKHOUSE_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: CLICKHOUSE_PLAN_INVALID_JSON_MESSAGE };
  }

  const entry = Array.isArray(parsed) ? parsed[0] : parsed;
  const plan = isJsonRecord(entry) ? entry[PLAN_KEY] : undefined;
  if (!isJsonRecord(plan)) {
    return { ok: false, message: CLICKHOUSE_PLAN_NO_PLAN_MESSAGE };
  }

  return parseWithDepthLimit(() => toPlanNode(plan, "0", 0));
}
