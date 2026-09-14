/**
 * Engine-neutral query-plan shape rendered by the visualizer.
 *
 * Engine adapters map their own EXPLAIN output onto this model and the viewer
 * components read nothing else, so adding an engine means adding a parser.
 *
 * Bytebase runs EXPLAIN without ANALYZE, so every number here is a planner
 * estimate: there is no actual timing, loop, buffer, or worker data to model.
 */

/** One free-form attribute the engine reported for a node. */
export interface PlanProperty {
  readonly label: string;
  readonly value: string;
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
  readonly startupCost: number;
  readonly totalCost: number;
  /** Total cost minus the children's total cost, clamped at zero. */
  readonly selfCost: number;
  readonly rows: number;
  readonly width: number;
  readonly properties: readonly PlanProperty[];
  readonly children: readonly PlanNode[];
}

export interface PlanTree {
  readonly root: PlanNode;
  /** Every node in depth-first order, root first. */
  readonly nodes: readonly PlanNode[];
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
  for (const node of nodes) {
    if (node.selfCost > maxSelfCost) maxSelfCost = node.selfCost;
    if (node.rows > maxRows) maxRows = node.rows;
    selfCostTotal += node.selfCost;
  }
  return {
    root,
    nodes,
    maxSelfCost,
    maxRows,
    costBasis: Math.max(root.totalCost, selfCostTotal),
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
const countFormat = new Intl.NumberFormat("en-US", {
  maximumFractionDigits: 0,
});

export function formatPlanCost(value: number): string {
  return costFormat.format(value);
}

export function formatPlanCount(value: number): string {
  return countFormat.format(value);
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
  return Math.min(1, Math.max(0, node.selfCost / tree.costBasis));
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
export function planEdgeWidth(rows: number, tree: PlanTree): number {
  if (tree.maxRows <= 0 || rows <= 0) return PLAN_EDGE_MIN_WIDTH;
  const ratio =
    Math.log1p(Math.min(rows, tree.maxRows)) / Math.log1p(tree.maxRows);
  const width =
    PLAN_EDGE_MIN_WIDTH + ratio * (PLAN_EDGE_MAX_WIDTH - PLAN_EDGE_MIN_WIDTH);
  return Math.round(width * 100) / 100;
}

/**
 * Node types that read a whole relation, spelled as `toNodeType` names them.
 */
const FULL_RELATION_SCAN_TYPES = new Set(["Seq Scan", "Parallel Seq Scan"]);

/**
 * Total cost below which reading a whole relation is not worth flagging.
 *
 * Under PostgreSQL's default settings a sequential scan costs `seq_page_cost`
 * (1.0) per page plus `cpu_tuple_cost` (0.01) per row, so 100 cost units is
 * roughly a 50-page, 5,000-row relation — about 400 KB. Reading all of that is
 * a handful of page fetches that no index lookup would beat, and flagging it
 * would bury the scans a reader can act on.
 */
export const PLAN_FULL_SCAN_COST_THRESHOLD = 100;

/** What a flagged full scan means, shared by every surface that reports one. */
export const PLAN_FULL_SCAN_HINT =
  "Reads every row of this relation. If the query needs only a few of them, an index on the filtered columns could avoid the full read.";

/** Whether the node reads a whole relation large enough to be worth saying so. */
export function isFlaggedFullScan(node: PlanNode): boolean {
  return (
    FULL_RELATION_SCAN_TYPES.has(node.nodeType) &&
    node.totalCost >= PLAN_FULL_SCAN_COST_THRESHOLD
  );
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
  const fraction = (value: number) =>
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
    .filter((node) => node.selfCost > 0)
    .sort((a, b) => b.selfCost - a.selfCost)
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
    entry.selfCost += node.selfCost;
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
  const value = mode === "cost" ? node.selfCost : node.rows;
  const max = mode === "cost" ? tree.maxSelfCost : tree.maxRows;
  if (max <= 0 || value <= 0) return 0;
  return Math.sqrt(Math.min(value, max) / max);
}
