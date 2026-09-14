import {
  ChevronsDownUp,
  ChevronsUpDown,
  Maximize2,
  TriangleAlert,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
// Deep import: `@/utils` (the barrel) transitively pulls in Monaco, which this
// standalone entry never loads. `minmax` itself is a leaf math helper.
import { minmax } from "@/utils/math";
import { PlanMiniMap } from "./PlanMiniMap";
import {
  layoutPlan,
  PLAN_NODE_HEIGHT,
  PLAN_NODE_WIDTH,
  type PlanViewport,
  type PlanViewportSize,
  planFitsViewport,
  planNodeCenter,
  planNodeOnScreen,
  planViewportCenteredOn,
} from "./plan-layout";
import {
  formatPlanCost,
  formatPlanCount,
  formatPlanShare,
  isFlaggedFullScan,
  PLAN_FULL_SCAN_HINT,
  type PlanHighlightMode,
  type PlanNode,
  type PlanTree,
  planDescendantCount,
  planEdgeWidth,
  planHighlightIntensity,
  planSelfCostShare,
} from "./plan-model";
import { PlanCostShareBar } from "./plan-shared";

interface Props {
  readonly tree: PlanTree;
  readonly selectedId: string | undefined;
  readonly onSelect: (id: string) => void;
  readonly highlight: PlanHighlightMode;
  /** Nodes whose subtrees are folded away. */
  readonly collapsedIds: ReadonlySet<string>;
  readonly onToggleCollapse: (id: string) => void;
  /** A node the diagram should bring into view, such as a deep link's target. */
  readonly revealId?: string;
  /** Called once `revealId` has been centered, so the caller can clear it. */
  readonly onRevealed?: () => void;
}

const MIN_SCALE = 0.2;
const MAX_SCALE = 2;
const ZOOM_STEP = 1.25;
/** Pointer travel, in CSS pixels, that turns a click into a pan. */
const PAN_THRESHOLD = 4;
const IDENTITY_VIEWPORT: PlanViewport = { scale: 1, x: 0, y: 0 };
const UNMEASURED: PlanViewportSize = { width: 0, height: 0 };

/**
 * What one wheel notch means in CSS pixels when the event reports something
 * else. Firefox reports the wheel in lines and some devices in pages; read as
 * pixels, a line-mode notch would move the zoom by a fraction of a percent.
 */
const WHEEL_LINE_PIXELS = 40;
const WHEEL_PAGE_PIXELS = 800;

function wheelPixels(event: WheelEvent): number {
  if (event.deltaMode === WheelEvent.DOM_DELTA_LINE) {
    return event.deltaY * WHEEL_LINE_PIXELS;
  }
  if (event.deltaMode === WheelEvent.DOM_DELTA_PAGE) {
    return event.deltaY * WHEEL_PAGE_PIXELS;
  }
  return event.deltaY;
}

/** Semantic surface each shading scale tints a card with. */
const HIGHLIGHT_CLASS: Record<Exclude<PlanHighlightMode, "off">, string> = {
  cost: "bg-warning",
  rows: "bg-info",
};

/** Opacity of the tint on the plan's worst node; the rest scale below it. */
const MAX_TINT_OPACITY = 0.45;

/**
 * Shading rides on a translucent overlay rather than the card's own background
 * so the intensity is a plain opacity over a semantic token: no per-node color
 * arithmetic, and it follows the theme.
 */
function PlanNodeTint({
  node,
  tree,
  highlight,
}: {
  node: PlanNode;
  tree: PlanTree;
  highlight: PlanHighlightMode;
}) {
  if (highlight === "off") return null;
  const intensity = planHighlightIntensity(node, tree, highlight);
  if (intensity <= 0) return null;
  return (
    <span
      aria-hidden="true"
      data-testid="plan-node-tint"
      className={cn(
        "pointer-events-none absolute inset-0",
        HIGHLIGHT_CLASS[highlight]
      )}
      style={{ opacity: intensity * MAX_TINT_OPACITY }}
    />
  );
}

/**
 * Folds a node's subtree away and says how much it took with it.
 *
 * It sits on the card's bottom edge, where the collapsed subtree's own edges
 * would leave, and is a sibling of the card rather than a control inside it:
 * the card is itself a button, and a button cannot hold another one.
 */
function PlanCollapseToggle({
  node,
  collapsed,
  hiddenCount,
  onToggle,
}: {
  node: PlanNode;
  collapsed: boolean;
  hiddenCount: number;
  onToggle: () => void;
}) {
  const label = collapsed
    ? `Expand ${formatPlanCount(hiddenCount)} nodes under ${node.nodeType}`
    : `Collapse ${formatPlanCount(hiddenCount)} nodes under ${node.nodeType}`;
  return (
    <Tooltip content={label}>
      <Button
        appearance="outline"
        size="xs"
        aria-expanded={!collapsed}
        aria-label={label}
        data-testid="plan-node-collapse"
        data-plan-collapsed={collapsed ? "true" : "false"}
        onClick={onToggle}
        // Half of the control hangs below the card so it reads as the seam the
        // subtree folds into, rather than as part of the card's own content.
        className="absolute -bottom-3 left-1/2 -translate-x-1/2 rounded-full bg-background px-1.5"
      >
        {collapsed ? (
          <ChevronsUpDown aria-hidden="true" className="size-3" />
        ) : (
          <ChevronsDownUp aria-hidden="true" className="size-3" />
        )}
        {collapsed ? (
          <span className="tabular-nums">{formatPlanCount(hiddenCount)}</span>
        ) : null}
      </Button>
    </Tooltip>
  );
}

export function QueryPlanDiagram({
  tree,
  selectedId,
  onSelect,
  highlight,
  collapsedIds,
  onToggleCollapse,
  revealId,
  onRevealed,
}: Props) {
  const layout = useMemo(
    () => layoutPlan(tree.root, collapsedIds),
    [tree, collapsedIds]
  );
  const viewportRef = useRef<HTMLDivElement>(null);
  const [view, setView] = useState<PlanViewport>(IDENTITY_VIEWPORT);
  const [size, setSize] = useState<PlanViewportSize>(UNMEASURED);

  // Pan reads the live viewport without re-binding its window listeners.
  const viewRef = useRef(view);
  viewRef.current = view;
  const sizeRef = useRef(size);
  sizeRef.current = size;
  // A drag that moved must not also select the card it started on.
  const draggedRef = useRef(false);
  const stopPanRef = useRef<(() => void) | undefined>(undefined);
  // Until the user zooms or pans, the diagram keeps refitting itself as the
  // container resizes. Once they place the view themselves, a resize must not
  // throw it away.
  const viewIsUsersRef = useRef(false);

  const measure = useCallback(() => {
    const rect = viewportRef.current?.getBoundingClientRect();
    const next = rect ? { width: rect.width, height: rect.height } : UNMEASURED;
    // The ref takes the size now rather than on the render the state schedules,
    // so an effect running later in this same commit already has it.
    sizeRef.current = next;
    setSize((prev) =>
      prev.width === next.width && prev.height === next.height ? prev : next
    );
    return next;
  }, []);

  const zoomAt = useCallback((factor: number, cx: number, cy: number) => {
    viewIsUsersRef.current = true;
    setView((prev) => {
      const scale = minmax(prev.scale * factor, MIN_SCALE, MAX_SCALE);
      const ratio = scale / prev.scale;
      return {
        scale,
        x: cx - ratio * (cx - prev.x),
        y: cy - ratio * (cy - prev.y),
      };
    });
  }, []);

  const zoomFromCenter = useCallback(
    (factor: number) => {
      const rect = viewportRef.current?.getBoundingClientRect();
      zoomAt(factor, (rect?.width ?? 0) / 2, (rect?.height ?? 0) / 2);
    },
    [zoomAt]
  );

  const fitToView = useCallback(() => {
    viewIsUsersRef.current = false;
    const measured = measure();
    // jsdom and a not-yet-laid-out container both report a zero-size box.
    if (measured.width === 0 || measured.height === 0) {
      setView(IDENTITY_VIEWPORT);
      return;
    }
    const scale = minmax(
      Math.min(measured.width / layout.width, measured.height / layout.height),
      MIN_SCALE,
      1
    );
    setView({
      scale,
      x: (measured.width - layout.width * scale) / 2,
      y: (measured.height - layout.height * scale) / 2,
    });
  }, [layout, measure]);

  const panTo = useCallback((point: { x: number; y: number }) => {
    viewIsUsersRef.current = true;
    setView((prev) => planViewportCenteredOn(point, sizeRef.current, prev));
  }, []);

  useLayoutEffect(() => {
    // `fitToView` is rebuilt whenever the layout is, so this also runs when a
    // collapse re-lays out the plan. Folding a subtree away must not throw out
    // a view the reader placed, so only an untouched view refits.
    if (viewIsUsersRef.current) return;
    fitToView();
  }, [fitToView]);

  useEffect(() => {
    const element = viewportRef.current;
    if (!element) return;
    // The first measurement can land before the surrounding split pane has
    // sized itself, so refit until the user takes the view over.
    const observer = new ResizeObserver(() => {
      if (viewIsUsersRef.current) measure();
      else fitToView();
    });
    observer.observe(element);
    return () => observer.disconnect();
  }, [fitToView, measure]);

  useEffect(() => {
    const element = viewportRef.current;
    if (!element) return;
    // React's onWheel is passive, so preventDefault needs a manual listener;
    // without it the browser scrolls the page instead of zooming.
    const onWheel = (event: WheelEvent) => {
      event.preventDefault();
      const rect = element.getBoundingClientRect();
      zoomAt(
        Math.exp(-wheelPixels(event) / 400),
        event.clientX - rect.left,
        event.clientY - rect.top
      );
    };
    element.addEventListener("wheel", onWheel, { passive: false });
    return () => element.removeEventListener("wheel", onWheel);
  }, [zoomAt]);

  // Release a drag still in flight when the diagram unmounts.
  useEffect(() => () => stopPanRef.current?.(), []);

  // Fitting runs in a layout effect above, so this plain effect always lands
  // after it and the centering survives. Keep the two in that order.
  useEffect(() => {
    if (revealId === undefined) return;
    const placed = layout.nodes.find((entry) => entry.node.id === revealId);
    if (placed) panTo(planNodeCenter(placed));
    onRevealed?.();
  }, [revealId, layout, panTo, onRevealed]);

  const startPan = useCallback((event: React.PointerEvent) => {
    if (event.button !== 0) return;
    draggedRef.current = false;
    const startX = event.clientX;
    const startY = event.clientY;
    const origin = viewRef.current;

    const onMove = (moveEvent: PointerEvent) => {
      const dx = moveEvent.clientX - startX;
      const dy = moveEvent.clientY - startY;
      if (Math.abs(dx) + Math.abs(dy) > PAN_THRESHOLD) {
        draggedRef.current = true;
        viewIsUsersRef.current = true;
      }
      if (!draggedRef.current) return;
      setView((prev) => ({ ...prev, x: origin.x + dx, y: origin.y + dy }));
    };
    const stop = () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", stop);
      window.removeEventListener("pointercancel", stop);
      stopPanRef.current = undefined;
    };
    stopPanRef.current = stop;
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", stop);
    window.addEventListener("pointercancel", stop);
  }, []);

  const selectNode = useCallback(
    (id: string) => {
      if (draggedRef.current) return;
      onSelect(id);
    },
    [onSelect]
  );

  // Panning and zooming change `view` on every frame of a drag. The cards and
  // edges do not depend on it — the canvas they sit on carries the transform —
  // so they are built once per layout and reused across those frames.
  const edges = useMemo(
    () =>
      layout.edges.map((edge) => (
        <path
          key={edge.id}
          data-testid="plan-diagram-edge"
          data-plan-edge-rows={edge.target.rows}
          d={edge.path}
          // The enclosing svg stays transparent to pointers so a drag anywhere
          // on the canvas pans; the stroke takes them back so its title
          // surfaces on hover.
          className="pointer-events-auto fill-none stroke-control-border"
          strokeWidth={planEdgeWidth(edge.target.rows, tree)}
          strokeLinecap="round"
        >
          <title>{`${formatPlanCount(edge.target.rows)} estimated rows`}</title>
        </path>
      )),
    [layout, tree]
  );

  const cards = useMemo(
    () =>
      layout.nodes.map((placed) => {
        const { node, x, y } = placed;
        const selected = node.id === selectedId;
        const metrics = `cost ${formatPlanCost(node.totalCost)} · rows ${formatPlanCount(node.rows)}`;
        const share = planSelfCostShare(node, tree);
        const fullScan = isFlaggedFullScan(node);
        const collapsed = collapsedIds.has(node.id);
        const subtreeCount = planDescendantCount(node);
        // "Outer" is the default relationship and carries no information.
        const relationship =
          node.relationship === "Outer" ? undefined : node.relationship;
        return (
          <div
            key={node.id}
            className="absolute"
            style={{
              left: x,
              top: y,
              width: PLAN_NODE_WIDTH,
              height: PLAN_NODE_HEIGHT,
            }}
          >
            <Button
              appearance="outline"
              data-testid="plan-node-card"
              aria-pressed={selected}
              // The card's stacked lines concatenate into an unreadable name,
              // so it states its own. The full-scan warning is an icon with a
              // hover-only explanation, so the label has to carry it too.
              aria-label={[
                node.nodeType,
                relationship,
                node.subject,
                metrics,
                `${formatPlanShare(share)} of plan cost`,
                fullScan ? "full table scan" : undefined,
                collapsed
                  ? `${formatPlanCount(subtreeCount)} nodes hidden`
                  : undefined,
              ]
                .filter(Boolean)
                .join(", ")}
              onClick={() => selectNode(node.id)}
              // Tabbing through a large plan has to be able to see where it
              // has got to, so a card off screen brings itself on.
              onFocus={() => {
                if (
                  !planNodeOnScreen(placed, sizeRef.current, viewRef.current)
                ) {
                  panTo(planNodeCenter(placed));
                }
              }}
              className={cn(
                "absolute inset-0 h-auto items-stretch justify-start overflow-hidden rounded-sm bg-background p-0 text-left",
                selected
                  ? "border-accent ring-2 ring-accent"
                  : "hover:border-control"
              )}
            >
              <PlanNodeTint node={node} tree={tree} highlight={highlight} />
              <span className="relative flex min-w-0 flex-1 flex-col gap-1 px-2.5 py-2">
                <span className="flex w-full items-center gap-1">
                  <span className="min-w-0 flex-1 truncate text-sm leading-5 font-medium text-main">
                    {node.nodeType}
                  </span>
                  {fullScan ? (
                    // A native SVG title rather than a `Tooltip`: a large plan
                    // would otherwise mount one tooltip provider per flagged
                    // scan, and the detail pane states the same warning in the
                    // accessibility tree.
                    <TriangleAlert
                      aria-hidden="true"
                      data-testid="plan-node-full-scan"
                      className="size-3.5 shrink-0 text-warning"
                    >
                      <title>{PLAN_FULL_SCAN_HINT}</title>
                    </TriangleAlert>
                  ) : null}
                  {relationship ? (
                    <span className="shrink-0 rounded-xs bg-control-bg px-1 text-xs leading-4 text-control-light">
                      {relationship}
                    </span>
                  ) : null}
                </span>
                <span className="w-full truncate text-xs leading-4 text-control-light">
                  {node.subject ?? "—"}
                </span>
                <span className="w-full truncate text-xs leading-4 text-control">
                  {metrics}
                </span>
                <span className="flex w-full items-center gap-2">
                  <PlanCostShareBar share={share} className="min-w-0 flex-1" />
                  <span className="shrink-0 text-xs leading-4 text-control-light tabular-nums">
                    {formatPlanShare(share)}
                  </span>
                </span>
              </span>
            </Button>

            {node.children.length > 0 ? (
              <PlanCollapseToggle
                node={node}
                collapsed={collapsed}
                hiddenCount={subtreeCount}
                onToggle={() => onToggleCollapse(node.id)}
              />
            ) : null}
          </div>
        );
      }),
    [
      layout,
      tree,
      highlight,
      selectedId,
      collapsedIds,
      selectNode,
      panTo,
      onToggleCollapse,
    ]
  );

  return (
    <div
      ref={viewportRef}
      data-testid="plan-diagram-viewport"
      onPointerDown={startPan}
      className="relative h-full min-h-0 w-full touch-none overflow-hidden bg-control-bg"
    >
      <div
        data-testid="plan-diagram-canvas"
        className="absolute top-0 left-0 origin-top-left"
        style={{
          width: layout.width,
          height: layout.height,
          transform: `translate(${view.x}px, ${view.y}px) scale(${view.scale})`,
        }}
      >
        <svg
          aria-hidden="true"
          data-testid="plan-diagram-edges"
          className="pointer-events-none absolute top-0 left-0 overflow-visible"
          width={layout.width}
          height={layout.height}
        >
          {edges}
        </svg>

        {cards}
      </div>

      {planFitsViewport(layout, size, view) ? null : (
        <PlanMiniMap
          layout={layout}
          size={size}
          view={view}
          selectedId={selectedId}
          onPanTo={panTo}
        />
      )}

      <div className="absolute right-2 bottom-2 flex items-center gap-1">
        <Tooltip content="Zoom out">
          <Button
            appearance="outline"
            size="xs"
            aria-label="Zoom out"
            className="bg-background"
            onClick={() => zoomFromCenter(1 / ZOOM_STEP)}
          >
            <ZoomOut className="size-3.5" />
          </Button>
        </Tooltip>
        <Tooltip content="Zoom in">
          <Button
            appearance="outline"
            size="xs"
            aria-label="Zoom in"
            className="bg-background"
            onClick={() => zoomFromCenter(ZOOM_STEP)}
          >
            <ZoomIn className="size-3.5" />
          </Button>
        </Tooltip>
        <Tooltip content="Fit to view">
          <Button
            appearance="outline"
            size="xs"
            aria-label="Fit to view"
            className="bg-background"
            onClick={fitToView}
          >
            <Maximize2 className="size-3.5" />
          </Button>
        </Tooltip>
      </div>
    </div>
  );
}
