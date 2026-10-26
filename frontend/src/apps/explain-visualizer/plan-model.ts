/**
 * Engine-neutral query-plan shape rendered by the visualizer.
 *
 * Engine adapters map their own EXPLAIN output onto this model and the viewer
 * components read nothing else, so adding an engine means adding a parser.
 *
 * Most plans are estimates: Bytebase explains without executing, so there is
 * no timing or loop data to model. A plan the statement produced by running —
 * a typed `EXPLAIN ANALYZE`, SQL Server's actual plan — also carries what each
 * operator really did. Engines differ in which numbers they report at all, so
 * each one is optional and a surface shows only what the plan carries.
 */

/** One free-form attribute the engine reported for a node. */
export interface PlanProperty {
  readonly label: string;
  readonly value: string;
}

/** Where a warning came from, which decides whether relevance can hide it. */
export type PlanWarningSource = "engine" | "heuristic";

/** Something about a node worth a reader's attention, and what it means. */
export interface PlanWarning {
  readonly title: string;
  readonly detail: string;
  readonly source: PlanWarningSource;
}

export interface PlanNode {
  /** Unique within one tree; derived from the node's path from the root. */
  readonly id: string;
  /** Operation name, such as "Seq Scan" or "Hash Join". */
  readonly nodeType: string;
  /** Table or index the operation reads, when the engine names one. */
  readonly subject?: string;
  /**
   * The table this operator reads, named as the catalog names it, where it
   * reads one. Narrower than `subject`, which is whatever is worth printing on
   * a card: a join condition, an index, a sort key. This is what a lookup of
   * Bytebase's own metadata can be keyed on.
   */
  readonly relation?: string;
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
  /** How many times the node ran, on a measured plan. */
  readonly loops?: number;
  /** Rows the node really returned, over all its runs together. */
  readonly actualRows?: number;
  /** Milliseconds the node and its subtree really took, over all its runs. */
  readonly actualTime?: number;
  /** What the node adds on top of its children's time; see `planSelfTime`. */
  readonly selfTime?: number;
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

/**
 * What a plan measured, where it ran at all.
 *
 * Separate from `PlanEstimates` because the two answer different questions: an
 * estimate says what the planner expected, a measurement what happened, and a
 * surface that shows both has to say which is which.
 */
export interface PlanMeasurements {
  readonly time: boolean;
  readonly rows: boolean;
}

export interface PlanTree {
  readonly root: PlanNode;
  /** Every node in depth-first order, root first. */
  readonly nodes: readonly PlanNode[];
  readonly estimates: PlanEstimates;
  readonly measured: PlanMeasurements;
  readonly maxSelfCost: number;
  readonly maxSelfTime: number;
  readonly maxRows: number;
  /**
   * The whole plan's cost: the denominator of every cost share.
   *
   * Normally this is the root's total cost, which by construction equals the
   * sum of every node's self cost. A node that stops its children early —
   * `Limit` is the common one — reports a total below its children's, and
   * `selfCost` clamps that difference at zero, so the self costs can add up to
   * more than the root reports. Taking the larger of the two keeps every share
   * at or below 100%.
   */
  readonly costBasis: number;
  /** The whole plan's measured time, on the same rule as `costBasis`. */
  readonly timeBasis: number;
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

/**
 * Past this many operators the diagram is unreadable and its layout slow, so
 * the plan is shown as text instead. SSMS degrades the same way, at 2 MB.
 */
const PLAN_MAX_NODES = 1000;

export const PLAN_TOO_MANY_NODES_MESSAGE = `The query plan has more than ${PLAN_MAX_NODES} operators, which is more than this visualizer draws.`;

/**
 * A parser's tree from `build`, or the depth message when it went too deep or
 * the node-cap message when it is too large to draw.
 */
export function parseWithDepthLimit(build: () => PlanNode): PlanParseResult {
  try {
    const tree = buildPlanTree(build());
    if (tree.nodes.length > PLAN_MAX_NODES) {
      return { ok: false, message: PLAN_TOO_MANY_NODES_MESSAGE };
    }
    return { ok: true, tree };
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

/** A cumulative number of a node's own, with its children's share removed. */
function planSelfValue(
  total: number | undefined,
  children: readonly PlanNode[],
  of: (child: PlanNode) => number | undefined
): number | undefined {
  if (total === undefined) return undefined;
  const childTotal = children.reduce((sum, child) => sum + (of(child) ?? 0), 0);
  return Math.max(0, total - childTotal);
}

/**
 * A node's total cost minus its children's, clamped at zero, or undefined when
 * the node has no total cost. `PlanTree.costBasis` says when the clamp bites.
 */
export function planSelfCost(
  totalCost: number | undefined,
  children: readonly PlanNode[]
): number | undefined {
  return planSelfValue(totalCost, children, (child) => child.totalCost);
}

/**
 * A node's measured time minus its children's, on the same rule as
 * `planSelfCost`: the time the operator itself spent, which is what a reader
 * can act on. Ranking the cumulative number would only ever find the root.
 */
export function planSelfTime(
  actualTime: number | undefined,
  children: readonly PlanNode[]
): number | undefined {
  return planSelfValue(actualTime, children, (child) => child.actualTime);
}

/** Warning for a read of a whole table, attached by each engine's parser. */
export const PLAN_FULL_TABLE_SCAN: PlanWarning = {
  title: "Full table scan",
  detail:
    "Reads every row of this table. If the query needs only a few of them, an index on the filtered columns could avoid the full read.",
  source: "heuristic",
};

/** Warning for a read of a whole index, attached by each engine's parser. */
export const PLAN_FULL_INDEX_SCAN: PlanWarning = {
  title: "Full index scan",
  detail:
    "Reads every entry of this index. If the query needs only a few of them, a condition on the index's leading columns could avoid the full read.",
  source: "heuristic",
};

/** One index on a table an operator reads, as Bytebase last synced it. */
export interface PlanTableIndex {
  readonly name: string;
  /** The indexed columns or expressions, in order, as one line. */
  readonly columns: string;
  readonly unique?: boolean;
  readonly primary?: boolean;
}

/**
 * What Bytebase already knows about a table an operator reads.
 *
 * No standalone plan viewer can show this: pev2 and its kind are handed a
 * pasted file with no catalog behind it. It is what turns "this scan is
 * expensive" into a next step.
 */
export interface PlanTableFacts {
  readonly rowCount?: number;
  readonly indexes: readonly PlanTableIndex[];
}

/** Answers what Bytebase knows about a relation, where it knows anything. */
export type PlanTableLookup = (relation: string) => PlanTableFacts | undefined;

/** Which number, if any, shades the diagram's node cards. */
export type PlanHighlightMode = "off" | "cost" | "time" | "rows";

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
  let maxSelfTime = 0;
  let maxRows = 0;
  let selfCostTotal = 0;
  let selfTimeTotal = 0;
  const estimates = {
    cost: false,
    startupCost: false,
    rows: false,
    width: false,
  };
  const measured = { time: false, rows: false };
  for (const node of nodes) {
    const selfCost = node.selfCost ?? 0;
    const selfTime = node.selfTime ?? 0;
    const rows = node.rows ?? 0;
    if (selfCost > maxSelfCost) maxSelfCost = selfCost;
    if (selfTime > maxSelfTime) maxSelfTime = selfTime;
    if (rows > maxRows) maxRows = rows;
    selfCostTotal += selfCost;
    selfTimeTotal += selfTime;
    estimates.cost ||= node.totalCost !== undefined;
    estimates.startupCost ||= node.startupCost !== undefined;
    estimates.rows ||= node.rows !== undefined;
    estimates.width ||= node.width !== undefined;
    measured.time ||= node.actualTime !== undefined;
    measured.rows ||= node.actualRows !== undefined;
  }
  return {
    root,
    nodes,
    estimates,
    measured,
    maxSelfCost,
    maxSelfTime,
    maxRows,
    costBasis: Math.max(root.totalCost ?? 0, selfCostTotal),
    timeBasis: Math.max(root.actualTime ?? 0, selfTimeTotal),
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

// Number formatting is pinned to one locale: the viewer's text arrives through
// a translate function, and its numbers stay comparable across plans.
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

// Milliseconds, as PostgreSQL reports them; an operator can run in a few
// microseconds, so sub-millisecond times keep their digits. The unit is in the
// label beside the number, which comes through the viewer's translate.
const durationFormat = new Intl.NumberFormat("en-US", {
  maximumFractionDigits: 3,
});

export function formatPlanDuration(value: number | undefined): string {
  return value === undefined ? PLAN_NO_ESTIMATE : durationFormat.format(value);
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

function planShare(value: number | undefined, basis: number): number {
  if (basis <= 0) return 0;
  return Math.min(1, Math.max(0, (value ?? 0) / basis));
}

/**
 * A node's self cost as a fraction of the whole plan's cost, from 0 to 1.
 *
 * A literal proportion, unlike `planHighlightIntensity`'s ramp: a bar drawn to
 * this length is read as a quantity, so it has to be one.
 */
export function planSelfCostShare(node: PlanNode, tree: PlanTree): number {
  return planShare(node.selfCost, tree.costBasis);
}

/**
 * Which number the ranking and its shares go by.
 *
 * Measured time wins wherever the plan has it: planner costs are arbitrary
 * units, and they are wrong in exactly the cases worth investigating, so
 * ranking a measured plan by estimate would point at the wrong operator
 * precisely when it matters.
 */
export type PlanRankMetric = "time" | "cost";

export function planRankMetric(tree: PlanTree): PlanRankMetric {
  return tree.measured.time ? "time" : "cost";
}

/** A node's share of the plan by whichever number it is ranked on, 0 to 1. */
export function planSelfShare(node: PlanNode, tree: PlanTree): number {
  return planRankMetric(tree) === "time"
    ? planShare(node.selfTime, tree.timeBasis)
    : planShare(node.selfCost, tree.costBasis);
}

/**
 * Below this multiple the planner's row estimate was close enough that saying
 * so would be noise; the two row counts are on the node either way.
 */
export const PLAN_ROW_ESTIMATE_NOTABLE_FACTOR = 2;

/**
 * How far out the planner's row estimate was, as a multiple of it: above 1 the
 * node returned more rows than planned, below 1 fewer.
 *
 * The estimate is per execution and the measurement is a total, so the
 * executions are multiplied back in — the comparison a reader would otherwise
 * have to do by hand, and the one that explains a bad plan that cost alone
 * does not. Undefined when the node was not measured, or when the planner
 * expected no rows at all and there is no multiple to state.
 */
export function planRowEstimateFactor(node: PlanNode): number | undefined {
  if (node.actualRows === undefined || node.rows === undefined)
    return undefined;
  const expected = node.rows * (node.loops ?? 1);
  if (expected <= 0) return undefined;
  return node.actualRows / expected;
}

/**
 * Whether the plan reports what each operator added of its own, so a surface
 * can give it a column and a bar. Says nothing about the numbers being above
 * zero; `planShareIsMeaningful` answers that.
 */
export function planHasSelfMetric(tree: PlanTree): boolean {
  return tree.measured.time || tree.estimates.cost;
}

/** Whether the plan's total is above zero, so a share of it means something. */
function planShareIsMeaningful(tree: PlanTree): boolean {
  return planRankMetric(tree) === "time"
    ? tree.timeBasis > 0
    : tree.costBasis > 0;
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

/** How many operators the side panel ranks before it stops being a summary. */
export const PLAN_COSTLIEST_LIMIT = 3;

function planOwnValue(tree: PlanTree): (node: PlanNode) => number {
  return planRankMetric(tree) === "time"
    ? (node) => node.selfTime ?? 0
    : (node) => node.selfCost ?? 0;
}

/**
 * The nodes that own the most of the plan — measured time where it has it,
 * estimated cost otherwise — largest first. A node that adds nothing on top of
 * its children is not a cost the reader can go after, so it is left out.
 */
export function planCostliestNodes(
  tree: PlanTree,
  limit: number = PLAN_COSTLIEST_LIMIT
): PlanNode[] {
  const own = planOwnValue(tree);
  return tree.nodes
    .filter((node) => own(node) > 0)
    .sort((a, b) => own(b) - own(a))
    .slice(0, limit);
}

/**
 * Share of the plan below which one of our own scan heuristics stays off the
 * summary panel.
 *
 * A full scan of a small lookup table is the correct plan, and warning about
 * every scan teaches readers to ignore the warnings that matter. It applies to
 * Bytebase's heuristics alone: a diagnostic the engine itself reports always
 * reaches the panel, because "columns with no statistics" is the explanation
 * for a node that a cost threshold would call cheap.
 */
export const PLAN_WARNING_RELEVANT_SHARE = 0.1;

/** One warning and the nodes reporting it, worst first. */
export interface PlanWarningGroup {
  readonly warning: PlanWarning;
  readonly nodes: readonly PlanNode[];
}

/**
 * The warnings worth the summary panel, grouped by warning so one heuristic
 * firing on eight nodes is one line and not eight. Groups come in plan order;
 * node details still show every warning a node carries, relevant or not.
 */
export function planWarningGroups(tree: PlanTree): PlanWarningGroup[] {
  // With no cost and no measurement there is nothing to judge relevance by, so
  // every warning stands rather than being dropped for want of a number.
  const ranked = planShareIsMeaningful(tree);
  const own = planOwnValue(tree);
  const groups = new Map<string, { warning: PlanWarning; nodes: PlanNode[] }>();
  for (const node of tree.nodes) {
    for (const warning of node.warnings) {
      if (
        warning.source === "heuristic" &&
        ranked &&
        planSelfShare(node, tree) < PLAN_WARNING_RELEVANT_SHARE
      ) {
        continue;
      }
      const group = groups.get(warning.title);
      if (group) group.nodes.push(node);
      else groups.set(warning.title, { warning, nodes: [node] });
    }
  }
  return [...groups.values()].map(({ warning, nodes }) => ({
    warning,
    nodes: nodes.sort((a, b) => own(b) - own(a)),
  }));
}

/** What each shading mode reads off a node, and the plan's largest of it. */
const PLAN_HIGHLIGHT_VALUES: Record<
  Exclude<PlanHighlightMode, "off">,
  {
    readonly of: (node: PlanNode) => number | undefined;
    readonly max: (tree: PlanTree) => number;
  }
> = {
  cost: { of: (node) => node.selfCost, max: (tree) => tree.maxSelfCost },
  time: { of: (node) => node.selfTime, max: (tree) => tree.maxSelfTime },
  rows: { of: (node) => node.rows, max: (tree) => tree.maxRows },
};

/**
 * How strongly a node should be shaded, from 0 (no shading) to 1.
 *
 * Costs, times and row counts span orders of magnitude within one plan, so a
 * linear ramp leaves everything but the single worst node invisible. The square
 * root keeps the extremes ordered while giving mid-range nodes a readable tint.
 */
export function planHighlightIntensity(
  node: PlanNode,
  tree: PlanTree,
  mode: PlanHighlightMode
): number {
  if (mode === "off") return 0;
  const { of, max: maxOf } = PLAN_HIGHLIGHT_VALUES[mode];
  const value = of(node) ?? 0;
  const max = maxOf(tree);
  if (max <= 0 || value <= 0) return 0;
  return Math.sqrt(Math.min(value, max) / max);
}
