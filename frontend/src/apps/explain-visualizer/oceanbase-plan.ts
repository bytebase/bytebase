import {
  addPlanProperty,
  checkPlanDepth,
  isJsonRecord,
  type JsonRecord,
  PLAN_FULL_INDEX_SCAN,
  PLAN_FULL_TABLE_SCAN,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  type PlanRelation,
  type PlanWarning,
  parseWithDepthLimit,
  planLabel,
  planSelfCost,
} from "./plan-model";

/**
 * Maps OceanBase's `EXPLAIN FORMAT=JSON` output onto the engine-neutral plan
 * model.
 *
 * OceanBase returns the operator tree directly, each operator naming its
 * children under `CHILD_1`, `CHILD_2` and so on rather than listing them. Its
 * estimate is a time, `EST.TIME(us)`, and it covers an operator and everything
 * below it, so it reads as the plan's cost: the model's self cost is the
 * difference, and what the viewer ranks is microseconds the optimizer expects
 * rather than units of its own.
 */

const OPERATOR_KEY = "OPERATOR";
const CHILD_KEY = /^CHILD_(\d+)$/;
const COST_KEY = "EST.TIME(us)";
const ROWS_KEY = "EST.ROWS";
const NAME_KEY = "NAME";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set([OPERATOR_KEY, NAME_KEY, COST_KEY, ROWS_KEY]);

/**
 * Rows a read is estimated to return before reading all of a table or index is
 * worth flagging, as in the other parsers.
 */
const FULL_SCAN_ROWS_THRESHOLD = 1000;

export const OCEANBASE_PLAN_EMPTY_MESSAGE =
  "OceanBase returned no query plan for this statement.";
export const OCEANBASE_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. OceanBase's EXPLAIN FORMAT=JSON output is expected.";
export const OCEANBASE_PLAN_NO_PLAN_MESSAGE =
  "The query plan JSON names no operator, so there is nothing to draw.";

function optionalNumber(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return undefined;
}

function stringValue(value: unknown): string | undefined {
  // OceanBase pads an operator name to a fixed width.
  const text = typeof value === "string" ? value.trim() : "";
  return text || undefined;
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return JSON.stringify(value);
}

/** The children of an operator, in the order OceanBase numbered them. */
function childrenOf(raw: JsonRecord): JsonRecord[] {
  return Object.entries(raw)
    .map(([key, value]) => [CHILD_KEY.exec(key), value] as const)
    .filter(
      (entry): entry is readonly [RegExpExecArray, JsonRecord] =>
        entry[0] !== null && isJsonRecord(entry[1])
    )
    .sort(([a], [b]) => Number(a[1]) - Number(b[1]))
    .map(([, child]) => child);
}

/**
 * The table this operator reads.
 *
 * `NAME` is the name the statement used, which is the alias where it gave one,
 * and OceanBase does not qualify it — so a lookup answers only where that name
 * is a table Bytebase synced, as it does for MySQL. An operator that reads no
 * object leaves it empty.
 */
function toRelation(raw: JsonRecord): PlanRelation | undefined {
  const table = stringValue(raw[NAME_KEY]);
  return table ? { table } : undefined;
}

function toWarnings(operator: string, raw: JsonRecord): PlanWarning[] {
  const rows = optionalNumber(raw[ROWS_KEY]);
  if (rows !== undefined && rows < FULL_SCAN_ROWS_THRESHOLD) return [];
  if (operator.startsWith("TABLE FULL SCAN")) return [PLAN_FULL_TABLE_SCAN];
  if (operator.startsWith("INDEX FULL SCAN")) return [PLAN_FULL_INDEX_SCAN];
  return [];
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key) || CHILD_KEY.test(key)) continue;
    addPlanProperty(properties, planLabel(key), formatValue(value));
  }
  return properties;
}

function toPlanNode(raw: JsonRecord, id: string, depth: number): PlanNode {
  checkPlanDepth(depth);
  const children = childrenOf(raw).map((child, index) =>
    toPlanNode(child, `${id}.${index}`, depth + 1)
  );
  const operator = stringValue(raw[OPERATOR_KEY]) ?? "Unknown";
  const totalCost = optionalNumber(raw[COST_KEY]);
  return {
    id,
    nodeType: operator,
    subject: stringValue(raw[NAME_KEY]),
    relation: toRelation(raw),
    totalCost,
    selfCost: planSelfCost(totalCost, children),
    rows: optionalNumber(raw[ROWS_KEY]),
    properties: toProperties(raw),
    warnings: toWarnings(operator, raw),
    children,
  };
}

export function parseOceanbasePlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: OCEANBASE_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: OCEANBASE_PLAN_INVALID_JSON_MESSAGE };
  }

  if (!isJsonRecord(parsed) || !stringValue(parsed[OPERATOR_KEY])) {
    return { ok: false, message: OCEANBASE_PLAN_NO_PLAN_MESSAGE };
  }

  return parseWithDepthLimit(() => toPlanNode(parsed as JsonRecord, "0", 0));
}
