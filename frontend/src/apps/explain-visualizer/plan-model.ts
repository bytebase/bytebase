/**
 * Engine-neutral query-plan shape rendered by the visualizer.
 *
 * Engine adapters map their own EXPLAIN output onto this model and the viewer
 * components read nothing else, so adding an engine means adding a parser.
 *
 * Bytebase plans without executing, so every number here is an optimizer
 * estimate: there is no actual timing, loop, buffer, or worker data to model.
 * Engines differ in which estimates they report at all, so each one is
 * optional and a surface shows only what the plan carries.
 */

/** One free-form attribute the engine reported for a node. */
export interface PlanProperty {
  readonly label: string;
  readonly value: string;
}

/** Something about a node worth a reader's attention, and what it means. */
export interface PlanWarning {
  readonly title: string;
  readonly detail: string;
}

export interface PlanNode {
  /** Unique within one tree; derived from the node's path from the root. */
  readonly id: string;
  /** Operation name, such as "Seq Scan" or "Hash Join". */
  readonly nodeType: string;
  /** Table or index the operation reads, when the engine names one. */
  readonly subject?: string;
  /** How the parent consumes this node, such as "Inner" or "InitPlan 1". */
  readonly relationship?: string;
  /** Cost before the node returns its first row, including its subtree. */
  readonly startupCost?: number;
  /** Cost of returning every row, including the node's subtree. */
  readonly totalCost?: number;
  /** What the node adds on top of its children; see `planSelfCost`. */
  readonly selfCost?: number;
  /** Rows the node is estimated to return each time it runs. */
  readonly rows?: number;
  /** Estimated size of one returned row, in bytes. */
  readonly width?: number;
  readonly properties: readonly PlanProperty[];
  readonly warnings: readonly PlanWarning[];
  readonly children: readonly PlanNode[];
}

/** Which estimates appear anywhere in a plan. */
export interface PlanEstimates {
  readonly cost: boolean;
  readonly startupCost: boolean;
  readonly rows: boolean;
  readonly width: boolean;
}

export interface PlanTree {
  readonly root: PlanNode;
  /** Every node in depth-first order, root first. */
  readonly nodes: readonly PlanNode[];
  readonly estimates: PlanEstimates;
  readonly maxSelfCost: number;
  readonly maxRows: number;
  /**
   * The whole plan's cost: the denominator of every cost share and the maximum
   * of the summary timeline's axis.
   *
   * Normally this is the root's total cost, which by construction equals the
   * sum of every node's self cost. A node that stops its children early —
   * `Limit` is the common one — reports a total below its children's, and
   * `selfCost` clamps that difference at zero, so the self costs can add up to
   * more than the root reports. Taking the larger of the two keeps every share
   * at or below 100% and every timeline bar inside the axis.
   */
  readonly costBasis: number;
}

export type PlanParseResult =
  | { readonly ok: true; readonly tree: PlanTree }
  | { readonly ok: false; readonly message: string };

/**
 * Deepest plan a parser will build.
 *
 * The parsers and the walks over their result (`flattenPlan`, `planRows`,
 * `planPath`) recurse once per level, so a pathological nesting depth
 * overflows the stack. Without a cap that happens during render, where there
 * is no error boundary to catch it and React unmounts the page instead of
 * showing the parse error. The cap is an order of magnitude below the depth
 * any of those walks fails at, and far above any plan an optimizer produces.
 */
const PLAN_MAX_DEPTH = 500;

export const PLAN_TOO_DEEP_MESSAGE = `The query plan nests more than ${PLAN_MAX_DEPTH} levels deep, which is deeper than this visualizer can draw.`;

/** Raised by `checkPlanDepth` and caught by `parseWithDepthLimit` alone. */
const PLAN_TOO_DEEP = new Error(PLAN_TOO_DEEP_MESSAGE);

/** Called by a parser for each level it builds; past the cap it stops the build. */
export function checkPlanDepth(depth: number) {
  if (depth > PLAN_MAX_DEPTH) throw PLAN_TOO_DEEP;
}

/** A parser's tree from `build`, or the depth message when it went too deep. */
export function parseWithDepthLimit(build: () => PlanNode): PlanParseResult {
  try {
    return { ok: true, tree: buildPlanTree(build()) };
  } catch (error) {
    if (error !== PLAN_TOO_DEEP) throw error;
    return { ok: false, message: PLAN_TOO_DEEP_MESSAGE };
  }
}

export type JsonRecord = Record<string, unknown>;

export function isJsonRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/**
 * Appends a property, folding a value under a label already present into that
 * property on a line of its own, since an engine can report one label twice.
 */
export function addPlanProperty(
  properties: PlanProperty[],
  label: string,
  value: string
) {
  if (!value) return;
  const index = properties.findIndex((property) => property.label === label);
  if (index < 0) {
    properties.push({ label, value });
  } else if (!properties[index].value.split("\n").includes(value)) {
    properties[index] = {
      label,
      value: `${properties[index].value}\n${value}`,
    };
  }
}

/**
 * A node's total cost minus its children's, clamped at zero, or undefined when
 * the node has no total cost. `PlanTree.costBasis` says when the clamp bites.
 */
export function planSelfCost(
  totalCost: number | undefined,
  children: readonly PlanNode[]
): number | undefined {
  if (totalCost === undefined) return undefined;
  const childCost = children.reduce(
    (sum, child) => sum + (child.totalCost ?? 0),
    0
  );
  return Math.max(0, totalCost - childCost);
}

/** Warning for a read of a whole table, attached by each engine's parser. */
export const PLAN_FULL_TABLE_SCAN: PlanWarning = {
  title: "Full table scan",
  detail:
    "Reads every row of this table. If the query needs only a few of them, an index on the filtered columns could avoid the full read.",
};

/** Warning for a read of a whole index, attached by each engine's parser. */
export const PLAN_FULL_INDEX_SCAN: PlanWarning = {
  title: "Full index scan",
  detail:
    "Reads every entry of this index. If the query needs only a few of them, a condition on the index's leading columns could avoid the full read.",
};

/** Which estimate, if any, shades the diagram's node cards. */
export type PlanHighlightMode = "off" | "cost" | "rows";

export function flattenPlan(root: PlanNode): PlanNode[] {
  const nodes: PlanNode[] = [];
  const visit = (node: PlanNode) => {
    nodes.push(node);
    for (const child of node.children) visit(child);
  };
  visit(root);
  return nodes;
}

export function buildPlanTree(root: PlanNode): PlanTree {
  const nodes = flattenPlan(root);
  let maxSelfCost = 0;
  let maxRows = 0;
  let selfCostTotal = 0;
  const estimates = {
    cost: false,
    startupCost: false,
    rows: false,
    width: false,
  };
  for (const node of nodes) {
    const selfCost = node.selfCost ?? 0;
    const rows = node.rows ?? 0;
    if (selfCost > maxSelfCost) maxSelfCost = selfCost;
    if (rows > maxRows) maxRows = rows;
    selfCostTotal += selfCost;
    estimates.cost ||= node.totalCost !== undefined;
    estimates.startupCost ||= node.startupCost !== undefined;
    estimates.rows ||= node.rows !== undefined;
    estimates.width ||= node.width !== undefined;
  }
  return {
    root,
    nodes,
    estimates,
    maxSelfCost,
    maxRows,
    costBasis: Math.max(root.totalCost ?? 0, selfCostTotal),
  };
}

export interface PlanRow {
  readonly node: PlanNode;
  /** Distance from the root; 0 for the root itself. */
  readonly depth: number;
  /**
   * One entry per level between the root and this node, so `depth` of them:
   * whether the node on that level still has a sibling after it. The last
   * entry describes this node, the earlier ones its ancestors. A surface that
   * draws tree guides needs exactly this to know which connector lines carry
   * on past the row and which one turns into it.
   */
  readonly branchContinues: readonly boolean[];
}

/** Every node in plan order, each carrying how deep the planner nested it. */
export function planRows(root: PlanNode): PlanRow[] {
  const rows: PlanRow[] = [];
  const branch: boolean[] = [];
  const visit = (node: PlanNode) => {
    rows.push({ node, depth: branch.length, branchContinues: [...branch] });
    const last = node.children.length - 1;
    node.children.forEach((child, index) => {
      branch.push(index < last);
      visit(child);
      branch.pop();
    });
  };
  visit(root);
  return rows;
}

export function findPlanNode(
  tree: PlanTree,
  id: string | undefined
): PlanNode | undefined {
  if (id === undefined) return undefined;
  return tree.nodes.find((node) => node.id === id);
}

/** How many nodes sit below this one, at any depth. */
export function planDescendantCount(node: PlanNode): number {
  let count = 0;
  for (const child of node.children) count += 1 + planDescendantCount(child);
  return count;
}

/** The chain from the root down to `id`, or undefined when it is not in it. */
function planPath(root: PlanNode, id: string): PlanNode[] | undefined {
  const path: PlanNode[] = [];
  const visit = (node: PlanNode): boolean => {
    path.push(node);
    if (node.id === id) return true;
    for (const child of node.children) {
      if (visit(child)) return true;
    }
    path.pop();
    return false;
  };
  return visit(root) ? path : undefined;
}

/**
 * The collapsed node standing in for `id` on screen, or undefined when `id` is
 * on screen itself.
 *
 * Only strict ancestors hide a node — a collapsed node still shows its own
 * card — and the outermost one wins, since a collapse nested inside another
 * collapse is hidden along with everything else beneath it.
 */
export function planCollapsedAncestor(
  root: PlanNode,
  collapsed: ReadonlySet<string>,
  id: string | undefined
): PlanNode | undefined {
  if (id === undefined || collapsed.size === 0) return undefined;
  const path = planPath(root, id);
  if (!path) return undefined;
  return path.slice(0, -1).find((node) => collapsed.has(node.id));
}

/**
 * `collapsed` with every collapse that hides `id` opened, so the node has a
 * card of its own again. Returns the argument unchanged when nothing hid it,
 * so a caller can use the result as its next state without a needless render.
 */
export function planRevealNode(
  root: PlanNode,
  collapsed: ReadonlySet<string>,
  id: string | undefined
): ReadonlySet<string> {
  if (id === undefined || collapsed.size === 0) return collapsed;
  const path = planPath(root, id);
  if (!path) return collapsed;
  const hiding = path.slice(0, -1).filter((node) => collapsed.has(node.id));
  if (hiding.length === 0) return collapsed;
  const next = new Set(collapsed);
  for (const node of hiding) next.delete(node.id);
  return next;
}

/**
 * The URL fragment naming a plan node, which is what makes one shareable.
 *
 * Ids come from a parser rather than from the user, but they are not promised
 * to stay bare digits and dots, so they are escaped both ways.
 */
const NODE_FRAGMENT_PREFIX = "node-";

export function planNodeFragment(id: string): string {
  return `#${NODE_FRAGMENT_PREFIX}${encodeURIComponent(id)}`;
}

/** The node id a fragment names, or undefined when it names something else. */
export function planNodeIdFromFragment(
  fragment: string | undefined
): string | undefined {
  if (!fragment) return undefined;
  const body = fragment.startsWith("#") ? fragment.slice(1) : fragment;
  if (!body.startsWith(NODE_FRAGMENT_PREFIX)) return undefined;
  const id = body.slice(NODE_FRAGMENT_PREFIX.length);
  if (!id) return undefined;
  try {
    return decodeURIComponent(id);
  } catch {
    // A hand-edited fragment can hold a stray percent sign, which is a bad
    // fragment rather than a reason to fail the page.
    return undefined;
  }
}

// The standalone visualizer entry never initializes i18n, so number formatting
// is pinned to one locale rather than following the workspace language.
const costFormat = new Intl.NumberFormat("en-US", {
  maximumFractionDigits: 2,
});
// SQL Server's cost units are small enough that a whole plan often costs less
// than one, where two decimal places would print most operators as zero.
const fractionalCostFormat = new Intl.NumberFormat("en-US", {
  maximumSignificantDigits: 3,
});
// SQL Server estimates rows as fractions, and 1.57 rows is not 2.
const countFormat = new Intl.NumberFormat("en-US", {
  maximumFractionDigits: 2,
});

/** What a surface prints for an estimate the engine did not report. */
const PLAN_NO_ESTIMATE = "—";

export function formatPlanCost(value: number | undefined): string {
  if (value === undefined) return PLAN_NO_ESTIMATE;
  return Math.abs(value) < 1
    ? fractionalCostFormat.format(value)
    : costFormat.format(value);
}

export function formatPlanCount(value: number | undefined): string {
  return value === undefined ? PLAN_NO_ESTIMATE : countFormat.format(value);
}

const shareFormat = new Intl.NumberFormat("en-US", {
  style: "percent",
  maximumFractionDigits: 1,
});

export function formatPlanShare(value: number): string {
  // A node that costs something must not read as costing nothing, however
  // small its share of the plan is.
  if (value > 0 && value < 0.001) return "<0.1%";
  return shareFormat.format(value);
}

/**
 * A node's self cost as a fraction of the whole plan's cost, from 0 to 1.
 *
 * A literal proportion, unlike `planHighlightIntensity`'s ramp: a bar drawn to
 * this length is read as a quantity, so it has to be one.
 */
export function planSelfCostShare(node: PlanNode, tree: PlanTree): number {
  if (tree.costBasis <= 0) return 0;
  return Math.min(1, Math.max(0, (node.selfCost ?? 0) / tree.costBasis));
}

/** Narrowest and widest stroke a diagram edge is drawn with, in pixels. */
export const PLAN_EDGE_MIN_WIDTH = 1;
export const PLAN_EDGE_MAX_WIDTH = 6;

/**
 * How thick the edge carrying `rows` estimated rows to a parent should be.
 *
 * Row estimates within one plan run from a single row to millions, so the
 * scale is logarithmic; a linear one would draw everything but the largest
 * edge as the same hairline.
 */
export function planEdgeWidth(
  rows: number | undefined,
  tree: PlanTree
): number {
  if (tree.maxRows <= 0 || rows === undefined || rows <= 0) {
    return PLAN_EDGE_MIN_WIDTH;
  }
  const ratio =
    Math.log1p(Math.min(rows, tree.maxRows)) / Math.log1p(tree.maxRows);
  const width =
    PLAN_EDGE_MIN_WIDTH + ratio * (PLAN_EDGE_MAX_WIDTH - PLAN_EDGE_MIN_WIDTH);
  return Math.round(width * 100) / 100;
}

export interface PlanTimelineRow extends PlanRow {
  /** Where the node's first row is ready, as a fraction of the plan's cost. */
  readonly start: number;
  /** Where the node's last row is ready, as a fraction of the plan's cost. */
  readonly end: number;
  /** The node's self cost as a fraction of the plan's cost. */
  readonly share: number;
}

/**
 * Every node as a span from its startup cost to its total cost.
 *
 * Both costs include the node's whole subtree, so a span says how much of the
 * plan's cost a node accounts for and how much of that is paid before its first
 * row. It is not a schedule: siblings each start from their own subtree's cost,
 * so two inputs of a join overlap on this axis while actually running one after
 * the other.
 */
export function planTimeline(tree: PlanTree): PlanTimelineRow[] {
  const fraction = (value = 0) =>
    tree.costBasis <= 0 ? 0 : Math.min(1, Math.max(0, value / tree.costBasis));
  return planRows(tree.root).map((row) => {
    const start = fraction(row.node.startupCost);
    return {
      ...row,
      start,
      end: Math.max(start, fraction(row.node.totalCost)),
      share: planSelfCostShare(row.node, tree),
    };
  });
}

/** How many operators the summary names before it stops being a summary. */
export const PLAN_COSTLIEST_LIMIT = 5;

/**
 * The nodes that own the most cost, largest first. A node that adds nothing on
 * top of its children is not a cost the reader can go after, so it is left out.
 */
export function planCostliestNodes(
  tree: PlanTree,
  limit: number = PLAN_COSTLIEST_LIMIT
): PlanNode[] {
  return tree.nodes
    .filter((node) => (node.selfCost ?? 0) > 0)
    .sort((a, b) => (b.selfCost ?? 0) - (a.selfCost ?? 0))
    .slice(0, limit);
}

export interface PlanOperationCost {
  readonly nodeType: string;
  readonly count: number;
  readonly selfCost: number;
  readonly share: number;
}

/** Each node type in the plan with what it costs in total, largest first. */
export function planCostByOperation(tree: PlanTree): PlanOperationCost[] {
  const totals = new Map<string, { count: number; selfCost: number }>();
  for (const node of tree.nodes) {
    const entry = totals.get(node.nodeType) ?? { count: 0, selfCost: 0 };
    entry.count += 1;
    entry.selfCost += node.selfCost ?? 0;
    totals.set(node.nodeType, entry);
  }
  return [...totals]
    .map(([nodeType, entry]) => ({
      nodeType,
      count: entry.count,
      selfCost: entry.selfCost,
      share:
        tree.costBasis <= 0 ? 0 : Math.min(1, entry.selfCost / tree.costBasis),
    }))
    .sort(
      (a, b) => b.selfCost - a.selfCost || a.nodeType.localeCompare(b.nodeType)
    );
}

/**
 * How strongly a node should be shaded, from 0 (no shading) to 1.
 *
 * Costs and row counts span orders of magnitude within one plan, so a linear
 * ramp leaves everything but the single worst node invisible. The square root
 * keeps the extremes ordered while giving mid-range nodes a readable tint.
 */
export function planHighlightIntensity(
  node: PlanNode,
  tree: PlanTree,
  mode: PlanHighlightMode
): number {
  if (mode === "off") return 0;
  const value = (mode === "cost" ? node.selfCost : node.rows) ?? 0;
  const max = mode === "cost" ? tree.maxSelfCost : tree.maxRows;
  if (max <= 0 || value <= 0) return 0;
  return Math.sqrt(Math.min(value, max) / max);
}
