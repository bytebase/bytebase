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
 * Maps the stages of a BigQuery job onto the engine-neutral plan model.
 *
 * BigQuery has no EXPLAIN. What a result carries is the plan of the job that
 * returned it: the stages the query ran as, each naming the stages that fed
 * it. So this plan is measured rather than estimated — the records are what
 * each stage really read and wrote, and there are no estimates beside them.
 *
 * Nothing is ranked by time. A stage states what its average and slowest shard
 * spent waiting, reading, computing and writing, which is a shard's time rather
 * than the stage's, and adding the phases of one shard would not make the
 * stage's own duration. They stay on the node, and the graph shades by rows.
 */

const INPUTS_KEY = "inputStages";

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set(["id", "name", "steps", INPUTS_KEY]);

/** A stage's name carries its number and what it does: `S00: Input`. */
const STAGE_NAME = /^\s*S\d+\s*:\s*(.+?)\s*$/;

/** The step that reads a table, and the substep that names it. */
const READ_KIND = "READ";
const READ_FROM = /^FROM\s+(\S+)$/;

/**
 * BigQuery writes a table as `dataset.table`. A name carrying a project as well
 * names a table outside the connection, which stays on the card as the subject
 * and resolves to no relation: Bytebase's metadata is this connection's, and
 * answering from it would describe a different table of the same name.
 */
const DATASET_TABLE = /^([^.]+)\.([^.]+)$/;

export const BIGQUERY_PLAN_EMPTY_MESSAGE =
  "BigQuery returned no query plan for this statement.";
export const BIGQUERY_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. The stages of a BigQuery job are expected.";
export const BIGQUERY_PLAN_NO_PLAN_MESSAGE =
  "The query plan contains no stage to start from.";

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

function stageID(raw: JsonRecord): string | undefined {
  const id = optionalNumber(raw.id);
  return id === undefined ? stringValue(raw.id) : String(id);
}

/** The steps of a stage, in the order BigQuery ran them. */
function stepsOf(raw: JsonRecord): JsonRecord[] {
  return Array.isArray(raw.steps) ? raw.steps.filter(isJsonRecord) : [];
}

/** The substeps of a step, which are the engine's own words for what it did. */
function substepsOf(step: JsonRecord): string[] {
  return Array.isArray(step.substeps)
    ? step.substeps
        .map((substep) => stringValue(substep))
        .filter((substep): substep is string => substep !== undefined)
    : [];
}

/**
 * The table the stage reads, which BigQuery names in the `FROM` substep of its
 * read. A stage reading another stage's shuffle names that instead, which is
 * not a table and resolves to none.
 */
function readTable(raw: JsonRecord): string | undefined {
  for (const step of stepsOf(raw)) {
    if (stringValue(step.kind) !== READ_KIND) continue;
    for (const substep of substepsOf(step)) {
      const named = READ_FROM.exec(substep);
      if (named) return named[1];
    }
  }
  return undefined;
}

function toRelation(raw: JsonRecord): PlanRelation | undefined {
  const table = readTable(raw);
  const named = table ? DATASET_TABLE.exec(table) : null;
  // A dataset is the database the editor connects to.
  return named ? { database: named[1], table: named[2] } : undefined;
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  // The stage's own name, which is how BigQuery's job statistics refer to it.
  const name = stringValue(raw.name);
  if (name) addPlanProperty(properties, "Stage", name);
  for (const step of stepsOf(raw)) {
    const kind = stringValue(step.kind);
    if (!kind) continue;
    const substeps = substepsOf(step);
    addPlanProperty(
      properties,
      "Steps",
      substeps.length > 0 ? `${kind}: ${substeps.join(", ")}` : kind
    );
  }
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key)) continue;
    const scalar =
      typeof value === "string" || typeof value === "number"
        ? String(value)
        : undefined;
    if (scalar !== undefined)
      addPlanProperty(properties, planLabel(key), scalar);
  }
  return properties;
}

/**
 * @param seen - The stages on the path to this node. A stage that reads itself,
 * directly or through another, would never finish; the rest of the plan still
 * draws.
 * @param count - The size budget, since a stage two others read is drawn under
 * each of them.
 */
function toPlanNode(
  raw: JsonRecord,
  stages: Map<string, JsonRecord>,
  id: string,
  seen: Set<string>,
  depth: number,
  count: () => void
): PlanNode {
  checkPlanDepth(depth);
  count();
  const children: PlanNode[] = [];
  const inputs = Array.isArray(raw[INPUTS_KEY]) ? raw[INPUTS_KEY] : [];
  for (const input of inputs) {
    const inputID = String(input);
    const stage = stages.get(inputID);
    if (!stage || seen.has(inputID)) continue;
    seen.add(inputID);
    children.push(
      toPlanNode(
        stage,
        stages,
        `${id}.${children.length}`,
        seen,
        depth + 1,
        count
      )
    );
    seen.delete(inputID);
  }

  const name = stringValue(raw.name);
  return {
    id,
    // What the stage does, which its name states after the stage number.
    nodeType:
      (name ? STAGE_NAME.exec(name)?.[1] : undefined) ?? name ?? "Stage",
    subject: readTable(raw),
    relation: toRelation(raw),
    actualRows: optionalNumber(raw.recordsWritten),
    properties: toProperties(raw),
    warnings: [],
    children,
  };
}

export function parseBigqueryPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: BIGQUERY_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: BIGQUERY_PLAN_INVALID_JSON_MESSAGE };
  }
  if (!Array.isArray(parsed)) {
    return { ok: false, message: BIGQUERY_PLAN_INVALID_JSON_MESSAGE };
  }

  const stages = new Map<string, JsonRecord>();
  for (const stage of parsed.filter(isJsonRecord)) {
    const id = stageID(stage);
    if (id !== undefined && !stages.has(id)) stages.set(id, stage);
  }
  if (stages.size === 0) {
    return { ok: false, message: BIGQUERY_PLAN_NO_PLAN_MESSAGE };
  }

  // The stage no other stage reads is the one that returned the query's rows.
  const read = new Set<string>();
  for (const stage of stages.values()) {
    const inputs = Array.isArray(stage[INPUTS_KEY]) ? stage[INPUTS_KEY] : [];
    for (const input of inputs) read.add(String(input));
  }
  const roots = [...stages.entries()].filter(([id]) => !read.has(id));
  // Where every stage feeds another, which would be a cycle, the last stage is
  // still the output one; BigQuery lists them in the order it ran them.
  const [rootID, root] =
    roots[roots.length - 1] ?? [...stages.entries()][stages.size - 1];

  const count = planSizeBudget();
  return parseWithDepthLimit(() =>
    toPlanNode(root, stages, "0", new Set([rootID]), 0, count)
  );
}
