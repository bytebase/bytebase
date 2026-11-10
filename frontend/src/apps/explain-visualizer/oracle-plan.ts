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
  planSubtreeCost,
} from "./plan-model";

/**
 * Maps Oracle's plan table onto the engine-neutral plan model.
 *
 * Oracle states a plan as rows rather than as a document: one row per operator
 * in `PLAN_TABLE`, each naming its parent and its position among its parent's
 * children. No EXPLAIN option renders that as a tree, so the driver sends the
 * rows as a JSON array — lower-cased column names, columns the optimizer said
 * nothing about left out — and the tree is rebuilt here.
 *
 * `COST` covers an operator and everything below it, as PostgreSQL's does, so
 * the model's self cost is the difference and the ranking is of operators
 * rather than of the statement.
 */

const ROOT_ID = "0";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set([
  "id",
  "parent_id",
  "position",
  "operation",
  "options",
  "object_name",
  "cost",
  "cardinality",
]);

/** `OBJECT_TYPE` values that name a table a lookup can be keyed on. */
const TABLE_OBJECT_TYPES = new Set(["TABLE", "TABLE (TEMP)", "MAT_VIEW"]);

/**
 * Rows a read is estimated to return before reading all of a table or index is
 * worth flagging, as in the other parsers: a full scan of a few hundred rows is
 * not one a reader can improve on.
 */
const FULL_SCAN_ROWS_THRESHOLD = 1000;

export const ORACLE_PLAN_EMPTY_MESSAGE =
  "Oracle returned no query plan for this statement.";
export const ORACLE_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. Oracle's plan table is expected, as the server renders it.";
export const ORACLE_PLAN_NO_PLAN_MESSAGE =
  "The query plan has no step without a parent, so it has no operator to start from.";

function optionalNumber(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return undefined;
}

function stringValue(value: unknown): string | undefined {
  if (typeof value === "string" && value.trim()) return value;
  if (typeof value === "number") return String(value);
  return undefined;
}

/**
 * The operator's name as `DBMS_XPLAN` prints it: the operation and the option
 * that qualifies it, such as "TABLE ACCESS FULL" or "INDEX RANGE SCAN".
 */
function toNodeType(step: JsonRecord): string {
  const operation = stringValue(step.operation);
  if (!operation) return "Unknown";
  const options = stringValue(step.options);
  return options ? `${operation} ${options}` : operation;
}

/**
 * The table this operator reads.
 *
 * Oracle names the owning schema on every object it reads, so a lookup is
 * answered with the schema the plan states rather than with whichever synced
 * table happens to share the name. An index is left out: its name is not a
 * table, and the table it belongs to is the step above it.
 */
function toRelation(step: JsonRecord): PlanRelation | undefined {
  const table = stringValue(step.object_name);
  const type = stringValue(step.object_type);
  if (!table || !type || !TABLE_OBJECT_TYPES.has(type.toUpperCase())) {
    return undefined;
  }
  return { table, schema: stringValue(step.object_owner) };
}

function toWarnings(step: JsonRecord): PlanWarning[] {
  const rows = optionalNumber(step.cardinality);
  if (rows !== undefined && rows < FULL_SCAN_ROWS_THRESHOLD) return [];
  const operation = stringValue(step.operation)?.toUpperCase();
  const options = stringValue(step.options)?.toUpperCase();
  if (operation === "TABLE ACCESS" && options === "FULL") {
    return [PLAN_FULL_TABLE_SCAN];
  }
  if (operation === "INDEX" && options?.startsWith("FULL SCAN")) {
    return [PLAN_FULL_INDEX_SCAN];
  }
  return [];
}

function toProperties(step: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const [key, value] of Object.entries(step)) {
    if (PROMOTED_KEYS.has(key)) continue;
    const text = stringValue(value);
    if (text) addPlanProperty(properties, planLabel(key), text);
  }
  return properties;
}

/**
 * The children of one step, in the order Oracle runs them.
 *
 * `POSITION` orders an operator's children; on the root it holds the
 * statement's total cost instead, which is why the root's own children are
 * ordered by id.
 */
function childrenOf(
  byParent: Map<string, JsonRecord[]>,
  id: string
): JsonRecord[] {
  const children = byParent.get(id) ?? [];
  return [...children].sort((a, b) => {
    const order =
      (optionalNumber(a.position) ?? 0) - (optionalNumber(b.position) ?? 0);
    return order !== 0
      ? order
      : (optionalNumber(a.id) ?? 0) - (optionalNumber(b.id) ?? 0);
  });
}

/**
 * @param seen - The ids on the path from the root. A plan table whose parent
 * links form a cycle would otherwise never finish; the rest of it still draws.
 */
function toPlanNode(
  step: JsonRecord,
  byParent: Map<string, JsonRecord[]>,
  id: string,
  seen: Set<string>,
  depth: number
): PlanNode {
  checkPlanDepth(depth);
  const stepID = stringValue(step.id) ?? "";
  seen.add(stepID);
  const children = childrenOf(byParent, stepID)
    .filter((child) => !seen.has(stringValue(child.id) ?? ""))
    .map((child, index) =>
      toPlanNode(child, byParent, `${id}.${index}`, seen, depth + 1)
    );
  seen.delete(stepID);

  // Oracle leaves COST off a step that adds none, and off both halves of an
  // adaptive plan's unchosen path.
  const totalCost = optionalNumber(step.cost) ?? planSubtreeCost(children);
  return {
    id,
    nodeType: toNodeType(step),
    subject:
      stringValue(step.object_name) ??
      stringValue(step.access_predicates) ??
      stringValue(step.filter_predicates),
    relation: toRelation(step),
    totalCost,
    selfCost: planSelfCost(totalCost, children),
    rows: optionalNumber(step.cardinality),
    properties: toProperties(step),
    warnings: toWarnings(step),
    children,
  };
}

export function parseOraclePlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: ORACLE_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: ORACLE_PLAN_INVALID_JSON_MESSAGE };
  }

  const steps = (Array.isArray(parsed) ? parsed : []).filter(isJsonRecord);
  const byParent = new Map<string, JsonRecord[]>();
  for (const step of steps) {
    const parent = stringValue(step.parent_id);
    if (parent === undefined) continue;
    const siblings = byParent.get(parent);
    if (siblings) siblings.push(step);
    else byParent.set(parent, [step]);
  }

  // The statement itself is the step no other step is the parent of, which
  // Oracle numbers 0. A plan whose root is missing is one we cannot walk.
  const root =
    steps.find((step) => stringValue(step.parent_id) === undefined) ??
    steps.find((step) => stringValue(step.id) === ROOT_ID);
  if (!root) {
    return {
      ok: false,
      message: steps.length
        ? ORACLE_PLAN_NO_PLAN_MESSAGE
        : ORACLE_PLAN_EMPTY_MESSAGE,
    };
  }

  return parseWithDepthLimit(() =>
    toPlanNode(root, byParent, "0", new Set(), 0)
  );
}
