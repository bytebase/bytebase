import {
  addPlanProperty,
  buildPlanTree,
  PLAN_FULL_INDEX_SCAN,
  PLAN_FULL_TABLE_SCAN,
  PLAN_MAX_DEPTH,
  PLAN_TOO_DEEP_MESSAGE,
  type PlanNode,
  type PlanParseResult,
  type PlanProperty,
  type PlanWarning,
} from "./plan-model";

/**
 * Maps a Spanner query plan onto the engine-neutral plan model.
 *
 * Spanner returns a flat `planNodes` list whose entries link to their children
 * by position. Relational nodes are operators; scalar nodes are the expressions
 * those operators evaluate. The tree keeps every operator, folds a named
 * expression — a condition, a limit — into its operator's properties, and
 * keeps a scalar node only when it is a subquery, since that is where a
 * subquery's own operators hang.
 *
 * Bytebase asks Spanner for the plan without running the query, and a plan
 * from that mode carries no estimates.
 */

export const SPANNER_PLAN_EMPTY_MESSAGE =
  "Spanner returned no query plan for this statement.";
export const SPANNER_PLAN_INVALID_JSON_MESSAGE =
  "The query plan is not valid JSON. A Spanner query plan is expected.";
export const SPANNER_PLAN_NO_PLAN_MESSAGE =
  'The query plan JSON does not contain a "planNodes" list with an operator at its root.';

/** Raised by `toPlanNode` and caught by `parseSpannerPlan` alone. */
const PLAN_TOO_DEEP = new Error(PLAN_TOO_DEEP_MESSAGE);

/** The link type that attaches a subquery to the expression using it. */
const SUBQUERY_LINK = "Scalar";

/**
 * Metadata folded into the node's name and subject, or of no use to a reader:
 * `subquery_cluster_node` is a position in the node list.
 */
const FOLDED_METADATA = new Set([
  "call_type",
  "iterator_type",
  "scan_type",
  "scan_target",
  "subquery_cluster_node",
]);

type JsonRecord = Record<string, unknown>;

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function stringField(record: JsonRecord | undefined, key: string): string {
  const value = record?.[key];
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return value === undefined || value === null ? "" : JSON.stringify(value);
}

function metadataOf(raw: JsonRecord): JsonRecord | undefined {
  return isRecord(raw.metadata) ? raw.metadata : undefined;
}

/**
 * The node's name as Spanner's tooling prints it: "Local Distributed Union",
 * "Global Stream Aggregate", "Table Scan".
 */
function toNodeType(raw: JsonRecord): string {
  const metadata = metadataOf(raw);
  const name = [
    stringField(metadata, "call_type"),
    stringField(metadata, "iterator_type"),
    stringField(metadata, "scan_type").replace(/Scan$/, ""),
    stringField(raw, "displayName") || "Unknown",
  ].filter(Boolean);
  return name.join(" ");
}

function toWarnings(raw: JsonRecord): PlanWarning[] {
  const metadata = metadataOf(raw);
  if (stringField(metadata, "Full scan") !== "true") return [];
  return stringField(metadata, "scan_type") === "IndexScan"
    ? [PLAN_FULL_INDEX_SCAN]
    : [PLAN_FULL_TABLE_SCAN];
}

function toPlanNode(
  planNodes: readonly unknown[],
  index: number,
  id: string,
  relationship: string | undefined,
  path: ReadonlySet<number>,
  depth: number
): PlanNode {
  if (depth > PLAN_MAX_DEPTH) throw PLAN_TOO_DEEP;
  const raw = planNodes[index] as JsonRecord;
  const onPath = new Set(path).add(index);
  const children: PlanNode[] = [];
  const properties: PlanProperty[] = [];

  const links = Array.isArray(raw.childLinks) ? raw.childLinks : [];
  for (const link of links.filter(isRecord)) {
    const childIndex = link.childIndex;
    if (typeof childIndex !== "number") continue;
    const child = planNodes[childIndex];
    // A link back up the path would never end; a plan that has one is
    // malformed, and the rest of it is still worth drawing.
    if (!isRecord(child) || onPath.has(childIndex)) continue;
    const type = stringField(link, "type") || undefined;
    if (child.kind === "RELATIONAL" || type === SUBQUERY_LINK) {
      children.push(
        toPlanNode(
          planNodes,
          childIndex,
          `${id}.${children.length}`,
          // A subquery's node type already says how it is used.
          type === SUBQUERY_LINK ? undefined : type,
          onPath,
          depth + 1
        )
      );
    } else if (type) {
      // An expression the operator names by its role, such as "Seek
      // Condition". Unnamed ones are the columns it reads or returns.
      const description = stringField(
        isRecord(child.shortRepresentation) ? child.shortRepresentation : {},
        "description"
      );
      const variable = stringField(link, "variable");
      addPlanProperty(
        properties,
        type,
        variable ? `${description} AS $${variable}` : description
      );
    }
  }

  const metadata = metadataOf(raw);
  for (const key of Object.keys(metadata ?? {})) {
    if (FOLDED_METADATA.has(key)) continue;
    addPlanProperty(properties, key, stringField(metadata, key));
  }

  const scanTarget = stringField(metadata, "scan_target");
  const condition = properties.find((property) =>
    property.label.endsWith("Condition")
  );
  const description =
    raw.kind === "RELATIONAL"
      ? ""
      : stringField(
          isRecord(raw.shortRepresentation) ? raw.shortRepresentation : {},
          "description"
        );

  return {
    id,
    nodeType: toNodeType(raw),
    subject: scanTarget || condition?.value || description || undefined,
    relationship,
    properties,
    warnings: toWarnings(raw),
    children,
  };
}

export function parseSpannerPlan(source: string): PlanParseResult {
  const trimmed = source.trim();
  if (!trimmed) {
    return { ok: false, message: SPANNER_PLAN_EMPTY_MESSAGE };
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch {
    return { ok: false, message: SPANNER_PLAN_INVALID_JSON_MESSAGE };
  }

  const planNodes =
    isRecord(parsed) && Array.isArray(parsed.planNodes) ? parsed.planNodes : [];
  // Nodes link to each other by position, and the root is the first.
  const root = planNodes[0];
  if (!isRecord(root) || root.kind !== "RELATIONAL") {
    return { ok: false, message: SPANNER_PLAN_NO_PLAN_MESSAGE };
  }

  try {
    return {
      ok: true,
      tree: buildPlanTree(
        toPlanNode(planNodes, 0, "0", undefined, new Set(), 0)
      ),
    };
  } catch (error) {
    if (error !== PLAN_TOO_DEEP) throw error;
    return { ok: false, message: PLAN_TOO_DEEP_MESSAGE };
  }
}
