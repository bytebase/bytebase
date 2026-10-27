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
} from "./plan-model";

/**
 * Maps MongoDB's `explain()` output onto the engine-neutral plan model.
 *
 * MongoDB returns one of two shapes. A query explains to `queryPlanner`, whose
 * `winningPlan` is a tree nested through `inputStage`; an aggregation explains
 * to a `stages` pipeline, which runs first to last, so the list is read
 * backwards to put the stage producing the result at the root and the `$cursor`
 * that feeds it at the bottom — where it expands into a `queryPlanner` tree of
 * its own.
 *
 * Nothing here is estimated. MongoDB's planner publishes no row or cost
 * estimate at all, so the plan is shape and index usage, and the viewer says it
 * ranks nothing. A measured plan — `explain("executionStats")`, which runs the
 * query — adds the rows each stage returned. Its `executionTimeMillisEstimate`
 * is left a property: MongoDB calls it an estimate itself, and on a query that
 * finishes quickly every stage reports 0, so a ranking built on it would be
 * arbitrary.
 *
 * `rejectedPlans` is not drawn. It holds the plans the optimizer discarded,
 * which is a different question from what this query will do.
 */

const CHILD_KEY = "inputStage";
const CHILDREN_KEY = "inputStages";
const STAGE_KEY = "stage";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set([
  STAGE_KEY,
  CHILD_KEY,
  CHILDREN_KEY,
  "nReturned",
]);

/** The operator that reads a whole collection, having no index to use. */
const COLLECTION_SCAN = "COLLSCAN";

/**
 * Documents a read examines before reading all of a collection is worth
 * flagging. A scan MongoDB gave no count for is flagged anyway: a collection
 * scan is the one thing a reader opens a MongoDB plan to find, and without a
 * count there is nothing to rule it out with.
 */
const FULL_SCAN_DOCS_THRESHOLD = 1000;

export const MONGODB_PLAN_EMPTY_MESSAGE =
  "MongoDB returned no query plan for this statement.";
export const MONGODB_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. MongoDB's explain() output is expected.";
export const MONGODB_PLAN_NO_PLAN_MESSAGE =
  'The query plan JSON has neither a "queryPlanner" nor a "stages" pipeline to draw.';

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

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return JSON.stringify(value);
}

/**
 * The collection a scan reads, from the namespace of the plan it belongs to.
 *
 * MongoDB states `database.collection` once, on the plan rather than on each
 * stage, so the stages that read it carry it down. Only the stages that reach
 * the collection get one: a sort or a projection reads its input, not a
 * collection.
 */
function toRelation(
  raw: JsonRecord,
  namespace: string | undefined
): PlanRelation | undefined {
  if (!namespace || isJsonRecord(raw[CHILD_KEY]) || raw[CHILDREN_KEY]) {
    return undefined;
  }
  const separator = namespace.indexOf(".");
  if (separator <= 0 || separator === namespace.length - 1) return undefined;
  return {
    database: namespace.slice(0, separator),
    table: namespace.slice(separator + 1),
  };
}

function toWarnings(stage: string, raw: JsonRecord): PlanWarning[] {
  if (stage !== COLLECTION_SCAN) return [];
  const examined =
    optionalNumber(raw.docsExamined) ?? optionalNumber(raw.nReturned);
  return examined !== undefined && examined < FULL_SCAN_DOCS_THRESHOLD
    ? []
    : [PLAN_FULL_TABLE_SCAN];
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key)) continue;
    addPlanProperty(properties, planLabel(key), formatValue(value));
  }
  return properties;
}

/**
 * What a stage is doing, for the node's card: the index it reads, or else the
 * condition or key that makes this stage different from the next one like it.
 */
function toSubject(raw: JsonRecord): string | undefined {
  const index = stringValue(raw.indexName);
  if (index) return index;
  for (const key of ["filter", "sortPattern", "transformBy", "keyPattern"]) {
    const value = raw[key];
    if (value !== undefined && value !== null) {
      const formatted = formatValue(value);
      if (formatted && formatted !== "{}") return formatted;
    }
  }
  return undefined;
}

function toPlanNode(
  raw: JsonRecord,
  namespace: string | undefined,
  id: string,
  depth: number
): PlanNode {
  checkPlanDepth(depth);
  const children: PlanNode[] = [];
  const child = raw[CHILD_KEY];
  if (isJsonRecord(child)) {
    children.push(toPlanNode(child, namespace, `${id}.0`, depth + 1));
  }
  const siblings = raw[CHILDREN_KEY];
  if (Array.isArray(siblings)) {
    for (const each of siblings.filter(isJsonRecord)) {
      children.push(
        toPlanNode(each, namespace, `${id}.${children.length}`, depth + 1)
      );
    }
  }

  const stage = stringValue(raw[STAGE_KEY]) ?? "Unknown";
  return {
    id,
    nodeType: stage,
    subject: toSubject(raw),
    relation: toRelation(raw, namespace),
    actualRows: optionalNumber(raw.nReturned),
    properties: toProperties(raw),
    warnings: toWarnings(stage, raw),
    children,
  };
}

/** The tree a `queryPlanner` holds: the measured one where the query ran. */
function planTreeOf(
  explained: JsonRecord
): { root: JsonRecord; namespace?: string } | undefined {
  const stats = isJsonRecord(explained.executionStats)
    ? explained.executionStats
    : undefined;
  const planner = isJsonRecord(explained.queryPlanner)
    ? explained.queryPlanner
    : undefined;
  const namespace = stringValue(planner?.namespace);
  const measured =
    stats && isJsonRecord(stats.executionStages)
      ? stats.executionStages
      : undefined;
  if (measured) return { root: measured, namespace };
  const winning =
    planner && isJsonRecord(planner.winningPlan)
      ? planner.winningPlan
      : undefined;
  if (!winning) return undefined;
  // The slot-based engine reports the operator tree under `queryPlan`, beside
  // the plan it compiled for its own machine. The classic engine puts the tree
  // itself here, and which engine ran is not the reader's question.
  const root = isJsonRecord(winning.queryPlan) ? winning.queryPlan : winning;
  return { root, namespace };
}

/**
 * An aggregation's stages, read backwards so the stage that produces the
 * result is the root. The `$cursor` that feeds the pipeline is not a stage of
 * its own: it holds the query plan for the read, which takes its place.
 */
function aggregateNode(
  stages: readonly unknown[],
  index: number,
  id: string,
  depth: number
): PlanNode | undefined {
  checkPlanDepth(depth);
  if (index < 0) return undefined;
  const stage = stages[index];
  if (!isJsonRecord(stage)) {
    return aggregateNode(stages, index - 1, id, depth);
  }
  const [name, body] = Object.entries(stage)[0] ?? [];
  if (!name) return aggregateNode(stages, index - 1, id, depth);

  if (name === "$cursor" && isJsonRecord(body)) {
    const plan = planTreeOf(body);
    if (plan) return toPlanNode(plan.root, plan.namespace, id, depth);
  }

  const child = aggregateNode(stages, index - 1, `${id}.0`, depth + 1);
  return {
    id,
    nodeType: name,
    subject: body === undefined ? undefined : formatValue(body) || undefined,
    properties: isJsonRecord(body) ? toProperties(body) : [],
    warnings: [],
    children: child ? [child] : [],
  };
}

export function parseMongodbPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: MONGODB_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: MONGODB_PLAN_INVALID_JSON_MESSAGE };
  }
  if (!isJsonRecord(parsed)) {
    return { ok: false, message: MONGODB_PLAN_INVALID_JSON_MESSAGE };
  }

  const stages = parsed.stages;
  if (Array.isArray(stages) && stages.length > 0) {
    return parseWithDepthLimit(() => {
      const root = aggregateNode(stages, stages.length - 1, "0", 0);
      if (!root) throw new Error(MONGODB_PLAN_NO_PLAN_MESSAGE);
      return root;
    });
  }

  const plan = planTreeOf(parsed);
  if (!plan) {
    return { ok: false, message: MONGODB_PLAN_NO_PLAN_MESSAGE };
  }
  return parseWithDepthLimit(() =>
    toPlanNode(plan.root, plan.namespace, "0", 0)
  );
}
