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
 * Maps the plan behind `EXPLAIN (DISTSQL)` onto the engine-neutral plan model.
 *
 * CockroachDB publishes no plan as a document: every option its EXPLAIN takes
 * returns text. What DISTSQL adds is a link to a diagram of the physical flow,
 * and the payload in that link is a plan a parser can read — processors and the
 * streams between them. The server decodes it, so what arrives here is the
 * document CockroachDB's own diagram page would draw.
 *
 * It is a graph rather than a tree: the processor nothing reads from is the
 * root, and an edge names the processor feeding it. Nothing in it is estimated
 * — the row counts a reader wants are in the text plan beside the link, not in
 * the link — so these plans rank nothing.
 */

const PROCESSORS_KEY = "processors";
const EDGES_KEY = "edges";

/** A processor's title, which ends in the index it has in the list. */
const PROCESSOR_TITLE = /^(.*?)(?:\/-?\d+)?$/;

/** How a reader names a table it reads: `table@index`. */
const TABLE_AND_INDEX = /^([^@\s]+)@([^@\s]+)$/;

/**
 * A detail CockroachDB labelled, such as `Columns: id, total`. The colon has
 * to be followed by a space and preceded by words alone: a detail like
 * `ON @3 < 10:::INT8` carries a cast, not a label.
 */
const DETAIL_LABEL = /^([A-Za-z][A-Za-z0-9 ]*): (.*)$/;

/** The detail naming the keys a read covers; `-` is all of them. */
const SPANS_DETAIL = /^Spans:\s*(.*)$/;
const ALL_SPANS = "-";

export const COCKROACHDB_PLAN_EMPTY_MESSAGE =
  "CockroachDB returned no query plan for this statement.";
export const COCKROACHDB_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. The diagram EXPLAIN (DISTSQL) encodes is expected.";
export const COCKROACHDB_PLAN_NO_PLAN_MESSAGE =
  'The query plan JSON contains no "processors" to draw.';

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" && value ? value : undefined;
}

function optionalNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value)
    ? value
    : undefined;
}

function detailsOf(raw: JsonRecord): string[] {
  const core = isJsonRecord(raw.core) ? raw.core : {};
  const details = Array.isArray(core.details) ? core.details : [];
  return details
    .map(stringValue)
    .filter((detail): detail is string => !!detail);
}

function titleOf(raw: JsonRecord): string {
  const core = isJsonRecord(raw.core) ? raw.core : {};
  const title = stringValue(core.title);
  if (!title) return "Unknown";
  return PROCESSOR_TITLE.exec(title)?.[1] || title;
}

/**
 * The table a processor reads.
 *
 * CockroachDB names the stored table even where the statement aliased it, so a
 * node answers a lookup without the alias ambiguity the MySQL family has. It
 * does not qualify the name, which the connected database settles.
 */
function toRelation(details: readonly string[]): PlanRelation | undefined {
  const named = details.map((detail) => TABLE_AND_INDEX.exec(detail))[0];
  return named ? { table: named[1] } : undefined;
}

function toWarnings(details: readonly string[]): PlanWarning[] {
  // A read whose spans are unbounded covers the whole index, which is the
  // thing a reader opens a CockroachDB plan to find. There is no row estimate
  // here to weigh it against, so it is always worth saying.
  const spans = details
    .map((detail) => SPANS_DETAIL.exec(detail)?.[1]?.trim())
    .find((value) => value !== undefined);
  return spans === ALL_SPANS ? [PLAN_FULL_TABLE_SCAN] : [];
}

function toProperties(
  raw: JsonRecord,
  details: readonly string[]
): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const detail of details) {
    const label = DETAIL_LABEL.exec(detail);
    if (label) {
      addPlanProperty(properties, planLabel(label[1]), label[2].trim());
    } else {
      addPlanProperty(properties, "Detail", detail);
    }
  }
  for (const [key, value] of Object.entries(raw)) {
    if (key === "core" || key === "inputs" || key === "outputs") continue;
    addPlanProperty(properties, planLabel(key), String(value));
  }
  return properties;
}

/**
 * @param seen - The processors on the path to this one. A flow whose streams
 * form a cycle would never finish; the rest of it still draws.
 */
function toPlanNode(
  processors: readonly JsonRecord[],
  feeders: ReadonlyMap<number, number[]>,
  index: number,
  id: string,
  seen: Set<number>,
  depth: number
): PlanNode {
  checkPlanDepth(depth);
  seen.add(index);
  const raw = processors[index];
  const children = (feeders.get(index) ?? [])
    .filter((source) => !seen.has(source) && processors[source])
    .map((source, position) =>
      toPlanNode(
        processors,
        feeders,
        source,
        `${id}.${position}`,
        seen,
        depth + 1
      )
    );
  seen.delete(index);

  const details = detailsOf(raw);
  return {
    id,
    nodeType: titleOf(raw),
    subject: details[0],
    relation: toRelation(details),
    properties: toProperties(raw, details),
    warnings: toWarnings(details),
    children,
  };
}

export function parseCockroachdbPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: COCKROACHDB_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: COCKROACHDB_PLAN_INVALID_JSON_MESSAGE };
  }

  const raw = isJsonRecord(parsed) ? parsed[PROCESSORS_KEY] : undefined;
  const processors = (Array.isArray(raw) ? raw : []).filter(isJsonRecord);
  if (processors.length === 0) {
    return { ok: false, message: COCKROACHDB_PLAN_NO_PLAN_MESSAGE };
  }

  // An edge names the processors either end of one stream, by their position
  // in the list. The one nothing reads from is where the query's rows come
  // out, and so the root.
  const feeders = new Map<number, number[]>();
  const feeds = new Set<number>();
  const edges = isJsonRecord(parsed) ? parsed[EDGES_KEY] : undefined;
  for (const edge of (Array.isArray(edges) ? edges : []).filter(isJsonRecord)) {
    const source = optionalNumber(edge.sourceProc);
    const destination = optionalNumber(edge.destProc);
    if (source === undefined || destination === undefined) continue;
    feeders.set(destination, [...(feeders.get(destination) ?? []), source]);
    feeds.add(source);
  }
  const root = processors.findIndex((_, index) => !feeds.has(index));
  if (root < 0) {
    return { ok: false, message: COCKROACHDB_PLAN_NO_PLAN_MESSAGE };
  }

  return parseWithDepthLimit(() =>
    toPlanNode(processors, feeders, root, "0", new Set(), 0)
  );
}
