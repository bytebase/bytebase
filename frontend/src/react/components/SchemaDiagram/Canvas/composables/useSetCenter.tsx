import { useEffect, useRef } from "react";
import { DEFAULT_PADDINGS } from "../../common/const";
import { useSchemaDiagramContext } from "../../common/context";
import type { CenterTarget, CenterTargetType, Rect } from "../../types";
import { fitView } from "../libs/fitView";

const isTypedCenterTarget = <T extends CenterTargetType>(
  e: CenterTarget,
  type: T
): e is CenterTarget<T> => e.type === type;

/**
 * React port of `Canvas/composables/useSetCenter.ts`. Subscribes to
 * `set-center` events and animates `(zoom, position)` onto the
 * target — a table, rect, or point.
 */
export const useSetCenter = (canvas: Element | null) => {
  const ctx = useSchemaDiagramContext();
  const { zoom, setZoom, setPosition, events, rectOfTable } = ctx;

  // Capture latest dependency values for the event handler so we don't
  // have to re-subscribe on every state update.
  const stateRef = useRef({ canvas, zoom, setZoom, setPosition, rectOfTable });
  stateRef.current = { canvas, zoom, setZoom, setPosition, rectOfTable };

  useEffect(() => {
    const off = events.on("set-center", (e) => {
      const {
        canvas: el,
        zoom: currentZoom,
        setZoom: applyZoom,
        setPosition: applyPosition,
        rectOfTable: lookupRect,
      } = stateRef.current;
      if (!el) return;

      const padding = e.padding ?? DEFAULT_PADDINGS;
      const zooms = e.zooms ?? [currentZoom, currentZoom];

      let rect: Rect | null = null;
      if (isTypedCenterTarget(e, "point")) {
        const center = e.target;
        rect = { ...center, width: 0, height: 0 };
      } else if (isTypedCenterTarget(e, "rect")) {
        rect = e.target;
      } else if (isTypedCenterTarget(e, "table")) {
        rect = lookupRect(e.target);
      } else {
        console.assert(false, `unknown set-center target "${String(e)}"`);
        return;
      }
      if (!rect) return;
      const layout = fitView(el, [rect], padding, zooms);
      applyZoom(layout.zoom);
      applyPosition({ x: layout.rect.x, y: layout.rect.y });
    });
    return () => {
      off();
    };
  }, [events]);
};
