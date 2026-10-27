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
  planSizeBudget,
} from "./plan-model";

/**
 * Maps `EXPLAIN FORMATTED` output onto the engine-neutral plan model.
 *
 * Hive answers with one JSON object whose `CBOPlan` is itself a JSON string:
 * Calcite's own plan, a flat list of operators in the order they were built.
 * An operator that names no `inputs` reads the one declared before it, which
 * is how Calcite writes a chain; the ones that name them are the joins and
 * unions, and the last operator in the list is the root.
 *
 * Its row counts and average row sizes are read. Its costs are not: only a
 * join reports one, and on a table nothing has analyzed it reports the string
 * "not available", so a ranking would rest on the one number the plan is least
 * sure of.
 *
 * Without CBO there is no `CBOPlan` at all — `STAGE DEPENDENCIES` and
 * `STAGE PLANS` are left, which describe the Tez or MapReduce stages rather
 * than an operator tree — and the viewer shows the plan as text instead.
 */

const PLAN_KEY = "CBOPlan";
const INPUTS_KEY = "inputs";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set([
  "id",
  "relOp",
  "table",
  "rowCount",
  "avgRowSize",
  INPUTS_KEY,
]);

/** Calcite names an operator by its class: `…reloperators.HiveTableScan`. */
const OPERATOR_CLASS = /([^.]+)$/;

export const HIVE_PLAN_EMPTY_MESSAGE =
  "Hive returned no query plan for this statement.";
export const HIVE_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. Hive's EXPLAIN FORMATTED output is expected.";
export const HIVE_PLAN_NO_PLAN_MESSAGE =
  "The query plan has no CBO plan to draw. Hive states one only when its cost-based optimizer planned the statement.";

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

function isScalar(value: unknown): boolean {
  return (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  );
}

/**
 * A value worth putting on a card. Calcite writes a condition, an expression
 * and an aggregate call as trees of operators and operands, which are the
 * query's own syntax turned inside out; the scalars beside them are what reads
 * as a fact about the operator.
 */
function scalarValue(value: unknown): string | undefined {
  if (isScalar(value)) return String(value);
  if (Array.isArray(value) && value.length > 0 && value.every(isScalar)) {
    return value.map((entry) => String(entry)).join(", ");
  }
  return undefined;
}

/** The name of the operator or function an expression applies. */
function operatorName(value: unknown): string | undefined {
  if (!isJsonRecord(value)) return undefined;
  const named = isJsonRecord(value.op) ? value.op : value.agg;
  return isJsonRecord(named) ? stringValue(named.name) : undefined;
}

/**
 * The table this operator reads. Hive writes it as the parts of its qualified
 * name, which for a table in a database is the database and the table.
 */
function toRelation(raw: JsonRecord): PlanRelation | undefined {
  if (!Array.isArray(raw.table)) return undefined;
  const parts = raw.table
    .map((part) => stringValue(part))
    .filter((part): part is string => part !== undefined);
  if (parts.length === 0) return undefined;
  const table = parts[parts.length - 1];
  return parts.length > 1 ? { database: parts[0], table } : { table };
}

/** The table the operator reads, with the alias the statement gave it. */
function tableSubject(raw: JsonRecord): string | undefined {
  const relation = toRelation(raw);
  if (!relation) return undefined;
  const qualified = relation.database
    ? `${relation.database}.${relation.table}`
    : relation.table;
  const alias = stringValue(raw["table:alias"]);
  return alias && alias !== relation.table
    ? `${qualified} ${alias}`
    : qualified;
}

/**
 * What the operator does, for its card: the table it reads, or else the
 * operator its condition applies, the columns it projects, or the aggregate it
 * computes.
 */
function toSubject(raw: JsonRecord): string | undefined {
  const table = tableSubject(raw);
  if (table) return table;
  const condition = operatorName(raw.condition);
  if (condition) return condition;
  const aggregates = Array.isArray(raw.aggs)
    ? raw.aggs
        .map((aggregate) => operatorName(aggregate))
        .filter((name): name is string => name !== undefined)
    : [];
  if (aggregates.length > 0) return aggregates.join(", ");
  return scalarValue(raw.fields);
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  // The id Calcite gives the operator, which its inputs refer to.
  const id = stringValue(raw.id);
  if (id) addPlanProperty(properties, "Operator", id);
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key)) continue;
    const scalar = scalarValue(value);
    if (scalar !== undefined) {
      addPlanProperty(properties, planLabel(key), scalar);
      continue;
    }
    // The column list is worth its length alone; the names are on the columns
    // the operator projects.
    if (key === "rowType" && Array.isArray(value)) {
      addPlanProperty(properties, "Columns", String(value.length));
    }
  }
  const condition = operatorName(raw.condition);
  if (condition) addPlanProperty(properties, "Condition", condition);
  return properties;
}

/**
 * Which operators each operator reads, as positions in the list.
 *
 * Calcite leaves `inputs` out when an operator reads the one declared before
 * it, writes `[]` on a leaf, and names them where there is a choice.
 */
function inputPositions(rels: JsonRecord[]): number[][] {
  const positionOf = new Map<string, number>();
  rels.forEach((rel, index) => {
    const id = stringValue(rel.id);
    if (id !== undefined && !positionOf.has(id)) positionOf.set(id, index);
  });
  return rels.map((raw, index) => {
    if (!(INPUTS_KEY in raw)) return index > 0 ? [index - 1] : [];
    if (!Array.isArray(raw[INPUTS_KEY])) return [];
    return raw[INPUTS_KEY]
      .map((input) => positionOf.get(String(input)))
      .filter((at): at is number => at !== undefined);
  });
}

/**
 * @param seen - The operators on the path to this node. An operator that reads
 * itself, directly or through another, would never finish; the rest of the
 * plan still draws.
 * @param count - The size budget, since an operator two others read is drawn
 * under each of them.
 */
function toPlanNode(
  index: number,
  rels: JsonRecord[],
  inputs: number[][],
  id: string,
  seen: Set<number>,
  depth: number,
  count: () => void
): PlanNode {
  checkPlanDepth(depth);
  count();
  const raw = rels[index];
  const children: PlanNode[] = [];
  for (const next of inputs[index]) {
    if (seen.has(next)) continue;
    seen.add(next);
    children.push(
      toPlanNode(
        next,
        rels,
        inputs,
        `${id}.${children.length}`,
        seen,
        depth + 1,
        count
      )
    );
    seen.delete(next);
  }

  const operator = stringValue(raw.relOp) ?? "";
  return {
    id,
    nodeType: OPERATOR_CLASS.exec(operator)?.[1] ?? "Unknown",
    subject: toSubject(raw),
    relation: toRelation(raw),
    rows: optionalNumber(raw.rowCount),
    width: optionalNumber(raw.avgRowSize),
    properties: toProperties(raw),
    warnings: [],
    children,
  };
}

export function parseHivePlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: HIVE_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: HIVE_PLAN_INVALID_JSON_MESSAGE };
  }
  if (!isJsonRecord(parsed)) {
    return { ok: false, message: HIVE_PLAN_INVALID_JSON_MESSAGE };
  }

  // The plan Hive draws from is a JSON string within the JSON object.
  const encoded = stringValue(parsed[PLAN_KEY]);
  if (!encoded) {
    return { ok: false, message: HIVE_PLAN_NO_PLAN_MESSAGE };
  }
  let plan: unknown;
  try {
    plan = JSON.parse(encoded);
  } catch {
    return { ok: false, message: HIVE_PLAN_INVALID_JSON_MESSAGE };
  }
  const rels =
    isJsonRecord(plan) && Array.isArray(plan.rels)
      ? plan.rels.filter(isJsonRecord)
      : [];
  if (rels.length === 0) {
    return { ok: false, message: HIVE_PLAN_NO_PLAN_MESSAGE };
  }

  // The operator declared last returns the query's rows.
  const root = rels.length - 1;
  const inputs = inputPositions(rels);
  const count = planSizeBudget();
  return parseWithDepthLimit(() =>
    toPlanNode(root, rels, inputs, "0", new Set([root]), 0, count)
  );
}
