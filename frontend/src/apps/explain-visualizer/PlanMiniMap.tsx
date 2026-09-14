import { useCallback, useMemo, useRef } from "react";
import { cn } from "@/lib/utils";
import {
  PLAN_NODE_HEIGHT,
  PLAN_NODE_WIDTH,
  type PlanLayout,
  type PlanViewport,
  type PlanViewportSize,
  planMiniMap,
} from "./plan-layout";

/** The box the overview is drawn into, in CSS pixels. */
export const PLAN_MINI_MAP_BOX: PlanViewportSize = { width: 160, height: 112 };

interface Props {
  readonly layout: PlanLayout;
  /** Size of the diagram's viewport, which marks the screen's share of the plan. */
  readonly size: PlanViewportSize;
  readonly view: PlanViewport;
  readonly selectedId: string | undefined;
  /** Asks for a content-space point to be brought to the middle of the screen. */
  readonly onPanTo: (point: { x: number; y: number }) => void;
}

/**
 * An overview of the whole plan for when it no longer fits on screen: every
 * node as a dot, the visible window as a frame, and a click or drag to move it.
 *
 * It is decoration in the accessibility tree. Everything it navigates to is
 * reachable without it — the cards are tab stops that bring themselves into
 * view, the grid lists every node, and Fit to view shows the whole plan — so a
 * second, pointer-only tree of controls would be noise to announce.
 */
export function PlanMiniMap({
  layout,
  size,
  view,
  selectedId,
  onPanTo,
}: Props) {
  const surfaceRef = useRef<HTMLDivElement>(null);
  const map = planMiniMap(layout, size, view, PLAN_MINI_MAP_BOX);

  // A drag reads the scale live rather than through the listeners it installed.
  const scaleRef = useRef(map.scale);
  scaleRef.current = map.scale;
  const panToRef = useRef(onPanTo);
  panToRef.current = onPanTo;

  const panToEvent = useCallback(
    (event: { clientX: number; clientY: number }) => {
      const rect = surfaceRef.current?.getBoundingClientRect();
      const scale = scaleRef.current;
      if (!rect || scale <= 0) return;
      panToRef.current({
        x: (event.clientX - rect.left) / scale,
        y: (event.clientY - rect.top) / scale,
      });
    },
    []
  );

  const startDrag = useCallback(
    (event: React.PointerEvent) => {
      if (event.button !== 0) return;
      // The diagram pans on a pointer press anywhere on its canvas; a drag that
      // starts here is aimed at the mini-map instead.
      event.stopPropagation();
      panToEvent(event);

      const onMove = (moveEvent: PointerEvent) => panToEvent(moveEvent);
      const stop = () => {
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", stop);
        window.removeEventListener("pointercancel", stop);
      };
      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", stop);
      window.addEventListener("pointercancel", stop);
    },
    [panToEvent]
  );

  // Only the frame moves as the reader pans, and a pan re-renders on every
  // frame, so the dots are built once per plan rather than once per frame.
  const dots = useMemo(
    () =>
      layout.nodes.map(({ node, x, y }) => (
        <span
          key={node.id}
          data-testid="plan-mini-map-node"
          className={cn(
            "absolute rounded-xs",
            node.id === selectedId ? "bg-accent" : "bg-control-light"
          )}
          style={{
            left: x * map.scale,
            top: y * map.scale,
            width: PLAN_NODE_WIDTH * map.scale,
            height: PLAN_NODE_HEIGHT * map.scale,
          }}
        />
      )),
    [layout, map.scale, selectedId]
  );

  if (map.scale <= 0) return null;

  return (
    <div
      ref={surfaceRef}
      aria-hidden="true"
      data-testid="plan-mini-map"
      onPointerDown={startDrag}
      style={{ width: map.width, height: map.height }}
      className="absolute bottom-2 left-2 cursor-pointer overflow-hidden rounded-xs border border-control-border bg-background/90"
    >
      {dots}
      <span
        data-testid="plan-mini-map-viewport"
        className="absolute border border-accent bg-accent/15"
        style={{
          left: map.viewport.x,
          top: map.viewport.y,
          width: map.viewport.width,
          height: map.viewport.height,
          // A view panned right off the plan leaves no overlap to draw, and an
          // invisible frame would read as no frame at all.
          minWidth: 1,
          minHeight: 1,
        }}
      />
    </div>
  );
}
