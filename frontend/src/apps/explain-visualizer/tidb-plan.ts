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
} from "./plan-model";

/**
 * Maps `EXPLAIN FORMAT=tidb_json` output onto the engine-neutral plan model.
 *
 * TiDB returns the operator tree directly: one object per operator, children
 * under `subOperators`, and the operator's own id — `HashJoin_40` — as the
 * name TiDB's text plan prints too. What it does not return is a cost, in this
 * format or any other the editor can ask for, so a TiDB plan ranks nothing and
 * the viewer says so.
 *
 * A measured plan, from a typed `EXPLAIN ANALYZE FORMAT=tidb_json`, is read
 * for its rows alone. Its timings live in a free-form `executeInfo` string
 * where `time:` covers an operator and its subtree, and the operators pushed
 * down to a coprocessor report no `time:` at all — so a ranking built on it
 * would be wrong for half of the plan. The string stays on the node, where a
 * reader sees it next to the operator it belongs to.
 */

const CHILDREN_KEY = "subOperators";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set([
  "estRows",
  "actRows",
  "accessObject",
  CHILDREN_KEY,
]);

/**
 * An operator id: a name, the plan-wide sequence number TiDB gives it, and for
 * a join input the side it feeds — `TableReader_49(Build)`.
 */
const OPERATOR_ID = /^(.*?)_(\d+)(?:\(([^)]*)\))?$/;

/** Operators that read every row of a table or an index. */
const FULL_SCANS: Record<string, PlanWarning> = {
  TableFullScan: PLAN_FULL_TABLE_SCAN,
  IndexFullScan: PLAN_FULL_INDEX_SCAN,
};

/**
 * Rows a read examines before reading all of them is worth flagging, as in the
 * other parsers: a full scan of a few hundred rows is not one a reader can
 * improve on.
 */
const FULL_SCAN_ROWS_THRESHOLD = 1000;

export const TIDB_PLAN_EMPTY_MESSAGE =
  "TiDB returned no query plan for this statement.";
export const TIDB_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. EXPLAIN FORMAT=tidb_json output is expected.";
export const TIDB_PLAN_NO_PLAN_MESSAGE =
  "The query plan JSON does not contain an operator at its root.";

function optionalNumber(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return undefined;
}

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" && value ? value : undefined;
}

/**
 * The parts of an `accessObject` such as
 * `table:orders, index:orders_customer_id_idx(customer_id)`. An index names
 * its columns in parentheses, where a comma does not separate parts.
 */
function accessObjectParts(accessObject: string): [string, string][] {
  const parts: string[] = [];
  let depth = 0;
  let start = 0;
  for (let index = 0; index < accessObject.length; index++) {
    const character = accessObject[index];
    if (character === "(") depth++;
    else if (character === ")") depth = Math.max(0, depth - 1);
    else if (character === "," && depth === 0) {
      parts.push(accessObject.slice(start, index));
      start = index + 1;
    }
  }
  parts.push(accessObject.slice(start));
  const named: [string, string][] = [];
  for (const part of parts) {
    const separator = part.indexOf(":");
    if (separator <= 0) continue;
    const label = part.slice(0, separator).trim();
    const value = part.slice(separator + 1).trim();
    if (label && value) named.push([label, value]);
  }
  return named;
}

function accessObjectOf(raw: JsonRecord): [string, string][] {
  const accessObject = stringValue(raw.accessObject);
  return accessObject ? accessObjectParts(accessObject) : [];
}

/**
 * The table this operator reads.
 *
 * TiDB prints the name the statement used, which is the alias where it gave
 * one, and never qualifies it — so a lookup answers only where that name is a
 * table Bytebase synced, as it does for MySQL.
 */
function toRelation(raw: JsonRecord): PlanRelation | undefined {
  const table = accessObjectOf(raw).find(([label]) => label === "table");
  return table ? { table: table[1] } : undefined;
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const [label, value] of accessObjectOf(raw)) {
    addPlanProperty(properties, planLabel(label), value);
  }
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key)) continue;
    addPlanProperty(
      properties,
      // The id with its sequence number is how TiDB's own text plan names this
      // operator, which is what a reader has in front of them.
      key === "id" ? "Operator" : planLabel(key),
      typeof value === "string" ? value : JSON.stringify(value)
    );
  }
  return properties;
}

function toWarnings(nodeType: string, raw: JsonRecord): PlanWarning[] {
  const warning = FULL_SCANS[nodeType];
  if (!warning) return [];
  const read = optionalNumber(raw.estRows) ?? optionalNumber(raw.actRows);
  return read !== undefined && read < FULL_SCAN_ROWS_THRESHOLD ? [] : [warning];
}

function toPlanNode(raw: JsonRecord, id: string, depth: number): PlanNode {
  checkPlanDepth(depth);
  const rawChildren = Array.isArray(raw[CHILDREN_KEY]) ? raw[CHILDREN_KEY] : [];
  const children = rawChildren
    .filter(isJsonRecord)
    .map((child, index) => toPlanNode(child, `${id}.${index}`, depth + 1));

  const operator = stringValue(raw.id) ?? "";
  const named = OPERATOR_ID.exec(operator);
  const nodeType = named?.[1] || operator || "Unknown";
  const table = accessObjectOf(raw).find(([label]) => label === "table");
  return {
    id,
    nodeType,
    // An operator TiDB names no object for says what it does instead, which is
    // the condition on a Selection and the join keys on a join.
    subject: table?.[1] ?? stringValue(raw.operatorInfo),
    relation: toRelation(raw),
    relationship: named?.[3] || undefined,
    rows: optionalNumber(raw.estRows),
    actualRows: optionalNumber(raw.actRows),
    properties: toProperties(raw),
    warnings: toWarnings(nodeType, raw),
    children,
  };
}

export function parseTidbPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: TIDB_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: TIDB_PLAN_INVALID_JSON_MESSAGE };
  }

  // TiDB wraps the plan in a one-element list, the root operator inside it.
  const root = Array.isArray(parsed) ? parsed[0] : parsed;
  if (!isJsonRecord(root) || !stringValue(root.id)) {
    return { ok: false, message: TIDB_PLAN_NO_PLAN_MESSAGE };
  }

  return parseWithDepthLimit(() => toPlanNode(root, "0", 0));
}
