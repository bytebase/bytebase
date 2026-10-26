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
  type PlanWarning,
  parseWithDepthLimit,
  planLabel,
  planSelfCost,
  planSubtreeCost,
} from "./plan-model";

/**
 * Maps `EXPLAIN FORMAT=JSON` output onto the engine-neutral plan model, for
 * MySQL and for MariaDB.
 *
 * Neither engine emits a list of operators. The plan is a tree of named
 * objects under `query_block`, where the name says what the level does —
 * `grouping_operation`, `filesort`, `materialized_from_subquery`, `table` —
 * and the leaves are the tables read. So the shape itself is the rule here:
 * every nested object is an operator named after its key, and everything else
 * is one of that operator's properties. The two dialects then differ only in
 * which keys they use, which is why one parser reads both, and a key neither
 * server emits yet still draws rather than disappearing.
 */

const QUERY_BLOCK_KEY = "query_block";
const TABLE_KEY = "table";

/**
 * Objects that describe the operator they sit on rather than one below it.
 * Their own entries are read as that operator's properties.
 */
const METADATA_OBJECTS = new Set(["cost_info", "r_engine_stats"]);

/**
 * Keys the model promotes to typed fields, dropped from the property list
 * because the node card and the table already show them. The rest of what the
 * engine reports about rows — `rows_examined_per_scan`, `filtered` — stays, as
 * those are different facts and a MySQL reader knows them by these names.
 */
const PROMOTED_KEYS = new Set([
  "table_name",
  "access_type",
  "cost",
  "rows",
  "rows_produced_per_join",
]);

/** Operators whose key alone does not read as a name for them. */
const NODE_LABELS: Record<string, string> = {
  [QUERY_BLOCK_KEY]: "Query block",
  ordering_operation: "Order",
  grouping_operation: "Group",
  duplicates_removal: "Distinct",
  union_result: "Union",
  materialized_from_subquery: "Materialize",
  materialized: "Materialize",
  filesort: "Sort",
  "block-nl-join": "Block nested loop join",
  windowing: "Window functions",
};

/**
 * How a table is read, in the words MySQL Workbench's own visual explain uses:
 * a reader who has seen one plan drawn has seen these names.
 */
const ACCESS_TYPES: Record<string, string> = {
  ALL: "Full Table Scan",
  index: "Full Index Scan",
  range: "Index Range Scan",
  index_merge: "Index Merge",
  ref: "Non-Unique Key Lookup",
  eq_ref: "Unique Key Lookup",
  ref_or_null: "Key Lookup or Null",
  const: "Single Row (constant)",
  system: "Single Row (system)",
  fulltext: "Fulltext Index Search",
  unique_subquery: "Unique Key Subquery",
  index_subquery: "Index Subquery",
};

/** A result MySQL materialized for itself: `<derived2>`, `<union1,2>`. */
const INTERNAL_TABLE = /^<.*>$/;

/**
 * Rows a read examines before reading all of them is worth flagging.
 *
 * A full scan of a few hundred rows is a page or two that no index lookup
 * would beat, and warning about it would bury the scans a reader can act on.
 */
const FULL_SCAN_ROWS_THRESHOLD = 1000;

export const MYSQL_PLAN_EMPTY_MESSAGE =
  "No query plan came back for this statement.";
export const MYSQL_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. EXPLAIN FORMAT=JSON output is expected.";
export const MYSQL_PLAN_NO_PLAN_MESSAGE =
  'The query plan JSON does not contain a "query_block" object.';

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

function isOperatorList(value: unknown): boolean {
  return Array.isArray(value) && value.some(isJsonRecord);
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

function toNodeType(key: string, raw: JsonRecord): string {
  if (key !== TABLE_KEY) return NODE_LABELS[key] ?? planLabel(key);
  const access = stringValue(raw.access_type);
  if (!access) return "Table";
  return ACCESS_TYPES[access] ?? planLabel(access);
}

function toSubject(raw: JsonRecord): string | undefined {
  return (
    stringValue(raw.table_name) ??
    stringValue(raw.sort_key) ??
    stringValue(raw.attached_condition) ??
    stringValue(raw.having_condition)
  );
}

/**
 * The stored table this operator reads, where it reads one.
 *
 * A `table` entry can stand in for a result the engine materialized for
 * itself. Those are named after the derived result — `<derived2>` — or after
 * the alias the statement gave the subquery, and they carry the subquery's own
 * operators, so they are neither a table Bytebase can describe nor a read an
 * index would change.
 *
 * What is left is the name the statement used for a real table, which is the
 * alias where it gave one, and neither engine qualifies it. So a lookup
 * answers only where that name is a table Bytebase synced, and an aliased read
 * goes unanswered rather than being answered with another table that happens
 * to share the alias.
 */
function baseTable(
  key: string,
  raw: JsonRecord,
  children: readonly PlanNode[]
): string | undefined {
  if (key !== TABLE_KEY || children.length > 0) return undefined;
  const table = stringValue(raw.table_name);
  return table && !INTERNAL_TABLE.test(table) ? table : undefined;
}

function toWarnings(table: string | undefined, raw: JsonRecord): PlanWarning[] {
  if (!table) return [];
  const examined =
    optionalNumber(raw.rows_examined_per_scan) ?? optionalNumber(raw.rows);
  if (examined !== undefined && examined < FULL_SCAN_ROWS_THRESHOLD) return [];
  switch (stringValue(raw.access_type)) {
    case "ALL":
      return [PLAN_FULL_TABLE_SCAN];
    case "index":
      return [PLAN_FULL_INDEX_SCAN];
    default:
      return [];
  }
}

function toProperties(raw: JsonRecord, inherited: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  const collect = (record: JsonRecord) => {
    for (const [key, value] of Object.entries(record)) {
      if (PROMOTED_KEYS.has(key)) continue;
      if (isJsonRecord(value)) {
        if (METADATA_OBJECTS.has(key)) collect(value);
        continue;
      }
      if (isOperatorList(value)) continue;
      addPlanProperty(properties, planLabel(key), formatValue(value));
    }
  };
  collect(raw);
  collect(inherited);
  return properties;
}

/** The cost this operator states for itself, in either dialect's spelling. */
function ownCost(raw: JsonRecord): number | undefined {
  const info = isJsonRecord(raw.cost_info) ? raw.cost_info : undefined;
  if (info) {
    const read = optionalNumber(info.read_cost);
    const evaluate = optionalNumber(info.eval_cost);
    // `prefix_cost` covers the whole join prefix, so it is not this operator's.
    if (read !== undefined || evaluate !== undefined) {
      return (read ?? 0) + (evaluate ?? 0);
    }
    const query = optionalNumber(info.query_cost);
    if (query !== undefined) return query;
  }
  return optionalNumber(raw.cost);
}

/**
 * @param inherited - Keys of the object that wrapped this operator in a list,
 * such as `dependent` on a union branch: they describe this operator, and the
 * wrapper is not a level of the plan.
 */
function toPlanNode(
  key: string,
  raw: JsonRecord,
  id: string,
  depth: number,
  inherited: JsonRecord
): PlanNode {
  checkPlanDepth(depth);
  const children = toChildren(raw, id, depth);
  const totalCost = ownCost(raw) ?? planSubtreeCost(children);
  const table = baseTable(key, raw, children);
  return {
    id,
    nodeType: toNodeType(key, raw),
    subject: toSubject(raw),
    relation: table ? { table } : undefined,
    totalCost,
    selfCost: planSelfCost(totalCost, children),
    rows:
      optionalNumber(raw.rows_produced_per_join) ??
      optionalNumber(raw.rows) ??
      optionalNumber(raw.rows_examined_per_scan),
    properties: toProperties(raw, inherited),
    warnings: toWarnings(table, raw),
    children,
  };
}

function toChildren(
  raw: JsonRecord,
  id: string,
  depth: number
): readonly PlanNode[] {
  const children: PlanNode[] = [];
  const add = (key: string, value: JsonRecord, inherited: JsonRecord) =>
    children.push(
      toPlanNode(key, value, `${id}.${children.length}`, depth + 1, inherited)
    );

  for (const [key, value] of Object.entries(raw)) {
    if (isJsonRecord(value)) {
      if (!METADATA_OBJECTS.has(key)) add(key, value, {});
      continue;
    }
    if (!Array.isArray(value)) continue;
    // A list of operators: `nested_loop`'s joined tables, `query_specifications`'
    // union branches. The list is not a level of the plan, its entries are.
    for (const entry of value) {
      if (!isJsonRecord(entry)) continue;
      const operators = Object.entries(entry).filter(
        ([entryKey, entryValue]) =>
          isJsonRecord(entryValue) && !METADATA_OBJECTS.has(entryKey)
      );
      if (operators.length === 0) {
        add(key, entry, {});
        continue;
      }
      const named = new Set(operators.map(([operatorKey]) => operatorKey));
      const inherited = Object.fromEntries(
        Object.entries(entry).filter(([entryKey]) => !named.has(entryKey))
      );
      for (const [operatorKey, operator] of operators) {
        add(operatorKey, operator as JsonRecord, inherited);
      }
    }
  }
  return children;
}

export function parseMysqlPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: MYSQL_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: MYSQL_PLAN_INVALID_JSON_MESSAGE };
  }

  const block = isJsonRecord(parsed) ? parsed[QUERY_BLOCK_KEY] : undefined;
  if (!isJsonRecord(block)) {
    return { ok: false, message: MYSQL_PLAN_NO_PLAN_MESSAGE };
  }

  return parseWithDepthLimit(() =>
    toPlanNode(QUERY_BLOCK_KEY, block, "0", 0, {})
  );
}
