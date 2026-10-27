import {
  addPlanProperty,
  checkPlanDepth,
  isJsonRecord,
  type JsonRecord,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  parseWithDepthLimit,
  planLabel,
  planSelfTime,
} from "./plan-model";

/**
 * Maps an Elasticsearch profile onto the engine-neutral plan model.
 *
 * Elasticsearch states no plan for a search it has not run: the profile is a
 * record of one. Every node carries `time_in_nanos` covering itself and its
 * children, so this is a measured plan the viewer can rank by time — the first
 * here that is not an estimate.
 *
 * A profile is stated per shard, with nothing above them, so the shards hang
 * under a node for the search itself: the one node here the engine did not
 * state, carrying the `took` that no shard reports. Under each shard the
 * phases keep the engine's own words and say which phase they are — the
 * queries Lucene ran, the collectors that gathered their hits, the
 * aggregations, and the fetch.
 */

const NANOS_IN_MILLISECOND = 1e6;

/** Keys the model promotes to typed fields, so the property list drops them. */
const PROMOTED_KEYS = new Set([
  "type",
  "name",
  "description",
  "reason",
  "time_in_nanos",
  "children",
  "breakdown",
]);

/** The phases a shard profiles, as the response names them. */
const QUERY = "Query";
const COLLECTOR = "Collector";
const AGGREGATION = "Aggregation";
const FETCH = "Fetch";

export const ELASTICSEARCH_PLAN_EMPTY_MESSAGE =
  "Elasticsearch returned no query plan for this statement.";
export const ELASTICSEARCH_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. An Elasticsearch profile is expected.";
export const ELASTICSEARCH_PLAN_NO_PLAN_MESSAGE =
  "The profile contains no shard to start from. Elasticsearch profiles a search it ran.";

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

/** Elasticsearch times every node in nanoseconds; the model reads milliseconds. */
function milliseconds(value: unknown): number | undefined {
  const nanos = optionalNumber(value);
  return nanos === undefined ? undefined : nanos / NANOS_IN_MILLISECOND;
}

function recordsOf(value: unknown): JsonRecord[] {
  return Array.isArray(value) ? value.filter(isJsonRecord) : [];
}

function toProperties(raw: JsonRecord): PlanProperty[] {
  const properties: PlanProperty[] = [];
  for (const [key, value] of Object.entries(raw)) {
    if (PROMOTED_KEYS.has(key)) continue;
    if (typeof value === "string" || typeof value === "number") {
      addPlanProperty(properties, planLabel(key), String(value));
    }
  }
  // What the node spent its time on, which is what a profile is read for.
  if (isJsonRecord(raw.breakdown)) {
    for (const [key, value] of Object.entries(raw.breakdown)) {
      if (typeof value === "number") {
        addPlanProperty(properties, planLabel(key), String(value));
      }
    }
  }
  return properties;
}

/**
 * One node of a profile: a Lucene query, a collector, an aggregation or the
 * fetch. A collector names itself and says why it ran, where the rest state a
 * type and describe themselves.
 *
 * @param relationship - The phase this node belongs to, on the node that
 * begins it.
 */
function toPlanNode(
  raw: JsonRecord,
  id: string,
  depth: number,
  relationship?: string
): PlanNode {
  checkPlanDepth(depth);
  const children = recordsOf(raw.children).map((child, index) =>
    toPlanNode(child, `${id}.${index}`, depth + 1)
  );
  const actualTime = milliseconds(raw.time_in_nanos);
  return {
    id,
    nodeType: stringValue(raw.type) ?? stringValue(raw.name) ?? "Unknown",
    subject: stringValue(raw.description) ?? stringValue(raw.reason),
    relationship,
    actualTime,
    selfTime: planSelfTime(actualTime, children),
    properties: toProperties(raw),
    warnings: [],
    children,
  };
}

/** The phases of one shard, each headed by the node that begins it. */
function shardPhases(shard: JsonRecord, id: string, depth: number): PlanNode[] {
  const phases: PlanNode[] = [];
  const next = (raw: JsonRecord, relationship: string) =>
    phases.push(
      toPlanNode(raw, `${id}.${phases.length}`, depth + 1, relationship)
    );

  for (const search of recordsOf(shard.searches)) {
    for (const query of recordsOf(search.query)) next(query, QUERY);
    for (const collector of recordsOf(search.collector)) {
      next(collector, COLLECTOR);
    }
  }
  for (const aggregation of recordsOf(shard.aggregations)) {
    next(aggregation, AGGREGATION);
  }
  if (isJsonRecord(shard.fetch)) next(shard.fetch, FETCH);
  return phases;
}

function toShardNode(shard: JsonRecord, id: string, depth: number): PlanNode {
  checkPlanDepth(depth);
  const children = shardPhases(shard, id, depth);
  const index = stringValue(shard.index);
  const properties: PlanProperty[] = [];
  for (const key of ["node_id", "shard_id", "cluster"]) {
    const value = shard[key];
    if (typeof value === "string" || typeof value === "number") {
      addPlanProperty(properties, planLabel(key), String(value));
    }
  }
  // Rewriting a query precedes the phases and is timed apart from them.
  for (const search of recordsOf(shard.searches)) {
    const rewrite = milliseconds(search.rewrite_time);
    if (rewrite !== undefined) {
      addPlanProperty(properties, "Rewrite ms", String(rewrite));
    }
  }
  return {
    id,
    nodeType: "Shard",
    subject: stringValue(shard.id) ?? index,
    // An Elasticsearch index is a table in Bytebase's own metadata.
    relation: index ? { table: index } : undefined,
    // A shard states no time of its own; its phases do.
    selfTime: undefined,
    actualTime: undefined,
    properties,
    warnings: [],
    children,
  };
}

export function parseElasticsearchPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: ELASTICSEARCH_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: ELASTICSEARCH_PLAN_INVALID_JSON_MESSAGE };
  }
  if (!isJsonRecord(parsed)) {
    return { ok: false, message: ELASTICSEARCH_PLAN_INVALID_JSON_MESSAGE };
  }

  const profile = isJsonRecord(parsed.profile) ? parsed.profile : parsed;
  const shards = recordsOf(profile.shards);
  if (shards.length === 0) {
    return { ok: false, message: ELASTICSEARCH_PLAN_NO_PLAN_MESSAGE };
  }

  const indices = [
    ...new Set(
      shards
        .map((shard) => stringValue(shard.index))
        .filter((index): index is string => index !== undefined)
    ),
  ];
  const took = optionalNumber(parsed.took);

  return parseWithDepthLimit(() => {
    const children = shards.map((shard, index) =>
      toShardNode(shard, `0.${index}`, 1)
    );
    const properties: PlanProperty[] = [];
    addPlanProperty(properties, "Shards", String(shards.length));
    if (took !== undefined)
      addPlanProperty(properties, "Took ms", String(took));
    return {
      id: "0",
      // The search itself, which the response states around the shards rather
      // than among them.
      nodeType: "Search",
      subject: indices.length > 0 ? indices.join(", ") : undefined,
      relation: indices.length === 1 ? { table: indices[0] } : undefined,
      actualTime: took,
      selfTime: planSelfTime(took, children),
      properties,
      warnings: [],
      children,
    };
  });
}
