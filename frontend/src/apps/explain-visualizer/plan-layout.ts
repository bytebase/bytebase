import { hierarchy, tree } from "d3-hierarchy";
// Deep import: `@/utils` (the barrel) transitively pulls in Monaco, which this
// standalone entry never loads. `minmax` itself is a leaf math helper.
import { minmax } from "@/utils/math";
import type { PlanNode } from "./plan-model";

/**
 * Top-down tidy-tree layout for a plan.
 *
 * Cards are a fixed size, so the layout is a pure function of the tree shape:
 * no DOM measuring pass, and the same input always produces the same geometry.
 * Coordinates are in content space; the diagram applies pan and zoom on top.
 */

export const PLAN_NODE_WIDTH = 216;
export const PLAN_NODE_HEIGHT = 96;

/**
 * Indent per nesting level for the surfaces that list the plan as rows, capped
 * so a ten-deep plan still leaves the node column readable at phone width.
 * Callers expose the uncapped depth through `aria-level`.
 */
export const PLAN_INDENT_STEP = 12;
const MAX_INDENT_LEVELS = 6;

export function planIndentPixels(depth: number): number {
  return Math.min(Math.max(depth, 0), MAX_INDENT_LEVELS) * PLAN_INDENT_STEP;
}

/**
 * The connector levels a row has room to draw, one per `PLAN_INDENT_STEP` of
 * its indent.
 *
 * Past the indent cap there is no column left to draw in, so the deepest levels
 * are kept: those are the ones that tie the row to its own parent, which is
 * what a reader traces. Mirrors `planIndentPixels`, and each entry says whether
 * that level's branch carries on below the row.
 */
export function planGuideLevels(
  branchContinues: readonly boolean[]
): readonly boolean[] {
  return branchContinues.slice(-MAX_INDENT_LEVELS);
}

const COLUMN_GAP = 32;
const ROW_GAP = 56;
const CONTENT_PADDING = 32;

/** Cousins get a little more room than siblings so subtrees read apart. */
const COUSIN_SEPARATION = 1.25;

export interface PlanLayoutNode {
  readonly node: PlanNode;
  /** Left edge of the card in content space. */
  readonly x: number;
  /** Top edge of the card in content space. */
  readonly y: number;
}

export interface PlanLayoutEdge {
  readonly id: string;
  /** SVG path data from the parent's bottom edge to the child's top edge. */
  readonly path: string;
  /** The child feeding the parent; its row estimate sizes the stroke. */
  readonly target: PlanNode;
}

export interface PlanLayout {
  /** Every card on the canvas, root first. */
  readonly nodes: readonly PlanLayoutNode[];
  readonly edges: readonly PlanLayoutEdge[];
  readonly width: number;
  readonly height: number;
}

const round = (value: number) => Math.round(value * 100) / 100;

const NOTHING_COLLAPSED: ReadonlySet<string> = new Set();

/**
 * @param collapsed - Ids whose subtrees are folded away. A collapsed node keeps
 * its own card; the layout closes up around the descendants it hides.
 */
export function layoutPlan(
  root: PlanNode,
  collapsed: ReadonlySet<string> = NOTHING_COLLAPSED
): PlanLayout {
  const laidOut = tree<PlanNode>()
    .nodeSize([PLAN_NODE_WIDTH + COLUMN_GAP, PLAN_NODE_HEIGHT + ROW_GAP])
    .separation((a, b) => (a.parent === b.parent ? 1 : COUSIN_SEPARATION))(
    hierarchy<PlanNode>(root, (node) =>
      collapsed.has(node.id) ? [] : (node.children as PlanNode[])
    )
  );

  const points = laidOut.descendants();
  let minCenterX = Number.POSITIVE_INFINITY;
  let maxCenterX = Number.NEGATIVE_INFINITY;
  let maxTop = 0;
  for (const point of points) {
    if (point.x < minCenterX) minCenterX = point.x;
    if (point.x > maxCenterX) maxCenterX = point.x;
    if (point.y > maxTop) maxTop = point.y;
  }

  // d3 centers each node on its own x; shift so the leftmost card's left edge
  // lands on the content padding.
  const offsetX = CONTENT_PADDING + PLAN_NODE_WIDTH / 2 - minCenterX;
  const offsetY = CONTENT_PADDING;

  const nodes = points.map((point) => ({
    node: point.data,
    x: round(point.x + offsetX - PLAN_NODE_WIDTH / 2),
    y: round(point.y + offsetY),
  }));

  const edges = laidOut.links().map(({ source, target }) => {
    const startX = round(source.x + offsetX);
    const startY = round(source.y + offsetY + PLAN_NODE_HEIGHT);
    const endX = round(target.x + offsetX);
    const endY = round(target.y + offsetY);
    const midY = round((startY + endY) / 2);
    return {
      id: `${source.data.id}->${target.data.id}`,
      path: `M${startX},${startY} C${startX},${midY} ${endX},${midY} ${endX},${endY}`,
      target: target.data,
    };
  });

  return {
    nodes,
    edges,
    width: round(
      maxCenterX - minCenterX + PLAN_NODE_WIDTH + CONTENT_PADDING * 2
    ),
    height: round(maxTop + PLAN_NODE_HEIGHT + CONTENT_PADDING * 2),
  };
}

/** The pan and zoom the diagram applies to content space, in CSS pixels. */
export interface PlanViewport {
  readonly scale: number;
  readonly x: number;
  readonly y: number;
}

/** The size of the element content is drawn into, in CSS pixels. */
export interface PlanViewportSize {
  readonly width: number;
  readonly height: number;
}

export const PLAN_IDENTITY_VIEWPORT: PlanViewport = { scale: 1, x: 0, y: 0 };

/** The furthest out and in the diagram zooms. */
export const PLAN_MIN_SCALE = 0.2;
export const PLAN_MAX_SCALE = 2;

/**
 * Smallest scale the diagram opens a plan at, where a card's 12px secondary
 * text still renders at about ten pixels. Further out, the diagram shows the
 * plan's shape with nothing in it to read.
 */
export const PLAN_READABLE_SCALE = 0.8;

/**
 * The whole plan, centered, at the largest scale that shows all of it. A plan
 * is never enlarged past its own size nor shrunk past the diagram's furthest
 * zoom, and an unmeasured viewport — jsdom's, or one not laid out yet — gets
 * the identity, since there is nothing to fit to.
 */
export function planFitViewport(
  layout: PlanLayout,
  size: PlanViewportSize
): PlanViewport {
  if (size.width <= 0 || size.height <= 0) return PLAN_IDENTITY_VIEWPORT;
  const scale = minmax(
    Math.min(size.width / layout.width, size.height / layout.height),
    PLAN_MIN_SCALE,
    1
  );
  return {
    scale,
    x: (size.width - layout.width * scale) / 2,
    y: (size.height - layout.height * scale) / 2,
  };
}

/**
 * The view the diagram opens on: the whole plan when its cards are readable at
 * the scale that shows all of it, and otherwise the readable scale from the
 * root down, centered on the root as far as the plan's edges allow. The
 * mini-map then offers the rest.
 */
export function planOpeningViewport(
  layout: PlanLayout,
  size: PlanViewportSize
): PlanViewport {
  const fit = planFitViewport(layout, size);
  if (size.width <= 0 || size.height <= 0) return fit;
  if (fit.scale >= PLAN_READABLE_SCALE) return fit;
  const scale = PLAN_READABLE_SCALE;
  const width = layout.width * scale;
  const height = layout.height * scale;
  const [root] = layout.nodes;
  const rootX = root ? planNodeCenter(root).x * scale : width / 2;
  return {
    scale,
    x:
      width <= size.width
        ? (size.width - width) / 2
        : minmax(size.width / 2 - rootX, size.width - width, 0),
    y: height <= size.height ? (size.height - height) / 2 : 0,
  };
}

export interface PlanRect {
  readonly x: number;
  readonly y: number;
  readonly width: number;
  readonly height: number;
}

/** The middle of a card, in content space. */
export function planNodeCenter(placed: PlanLayoutNode): {
  x: number;
  y: number;
} {
  return {
    x: placed.x + PLAN_NODE_WIDTH / 2,
    y: placed.y + PLAN_NODE_HEIGHT / 2,
  };
}

/**
 * The viewport that puts a content-space point in the middle of the screen,
 * at the zoom already in use. Both the mini-map and a deep link land on a node
 * this way.
 */
export function planViewportCenteredOn(
  point: { x: number; y: number },
  size: PlanViewportSize,
  view: PlanViewport
): PlanViewport {
  return {
    scale: view.scale,
    x: size.width / 2 - point.x * view.scale,
    y: size.height / 2 - point.y * view.scale,
  };
}

/**
 * Whether the whole plan is already on screen, which is when the mini-map has
 * nothing to offer. A viewport that has not been measured yet counts as
 * fitting rather than flashing a mini-map over an unsized box.
 */
export function planFitsViewport(
  layout: PlanLayout,
  size: PlanViewportSize,
  view: PlanViewport
): boolean {
  if (size.width <= 0 || size.height <= 0) return true;
  // Content and viewport are both rounded to hundredths, so compare with the
  // tolerance of that rounding rather than exactly.
  const slack = 0.01;
  return (
    layout.width * view.scale <= size.width + slack &&
    layout.height * view.scale <= size.height + slack
  );
}

/**
 * How many times the mini-map's box the viewport has to be, across and down,
 * before the diagram offers it. In anything smaller — a phone's stacked pane —
 * the overview would cover the plan it is an overview of.
 */
const MINI_MAP_ROOM = 3;

/**
 * Whether the diagram offers a mini-map: only while part of the plan is off
 * screen, and only in a viewport with room for one.
 */
export function planShowsMiniMap(
  layout: PlanLayout,
  size: PlanViewportSize,
  view: PlanViewport,
  box: PlanViewportSize
): boolean {
  return (
    !planFitsViewport(layout, size, view) &&
    size.width >= box.width * MINI_MAP_ROOM &&
    size.height >= box.height * MINI_MAP_ROOM
  );
}

/**
 * Whether a card is fully on screen at the current pan and zoom. An unmeasured
 * viewport counts as showing everything, so nothing moves before the first
 * layout pass.
 */
export function planNodeOnScreen(
  placed: PlanLayoutNode,
  size: PlanViewportSize,
  view: PlanViewport
): boolean {
  if (size.width <= 0 || size.height <= 0) return true;
  const left = placed.x * view.scale + view.x;
  const top = placed.y * view.scale + view.y;
  return (
    left >= 0 &&
    top >= 0 &&
    left + PLAN_NODE_WIDTH * view.scale <= size.width &&
    top + PLAN_NODE_HEIGHT * view.scale <= size.height
  );
}

export interface PlanMiniMap {
  /** Mini-map pixels per content pixel. */
  readonly scale: number;
  readonly width: number;
  readonly height: number;
  /** The part of the plan currently on screen, in mini-map coordinates. */
  readonly viewport: PlanRect;
}

const EMPTY_MINI_MAP: PlanMiniMap = {
  scale: 0,
  width: 0,
  height: 0,
  viewport: { x: 0, y: 0, width: 0, height: 0 },
};

/**
 * The whole plan drawn to fit `box`, with the screen's share of it marked.
 *
 * The mini-map never enlarges the plan: a plan small enough to fit the box
 * outright is drawn at its own size, so the overview is always an overview.
 */
export function planMiniMap(
  layout: PlanLayout,
  size: PlanViewportSize,
  view: PlanViewport,
  box: PlanViewportSize
): PlanMiniMap {
  if (layout.width <= 0 || layout.height <= 0 || view.scale <= 0) {
    return EMPTY_MINI_MAP;
  }
  const scale = Math.min(
    box.width / layout.width,
    box.height / layout.height,
    1
  );
  const width = layout.width * scale;
  const height = layout.height * scale;
  // Undo the pan and zoom to place a screen coordinate on the mini-map. Both
  // edges of the window are trimmed to the plan's bounds and the size comes
  // from the trimmed edges: taking it from the untrimmed window instead would
  // draw a pan past the left or top edge as more of the plan than is on screen.
  const onMap = (screen: number, pan: number) =>
    ((screen - pan) / view.scale) * scale;
  const left = minmax(onMap(0, view.x), 0, width);
  const top = minmax(onMap(0, view.y), 0, height);
  const right = minmax(onMap(size.width, view.x), left, width);
  const bottom = minmax(onMap(size.height, view.y), top, height);
  return {
    scale,
    width,
    height,
    viewport: {
      x: left,
      y: top,
      width: right - left,
      height: bottom - top,
    },
  };
}
