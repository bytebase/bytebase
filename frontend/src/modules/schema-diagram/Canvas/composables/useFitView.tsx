import { useCallback, useEffect, useRef } from "react";
import { DEFAULT_PADDINGS, ZOOM_RANGE } from "../../common/const";
import { useSchemaDiagramContext } from "../../common/context";
import { fitView } from "../libs/fitView";

/**
 * React port of `Canvas/composables/useFitView.ts`.
 *
 * Returns a `handleFitView()` callback the zoom buttons / external code
 * can invoke imperatively, AND subscribes to the context's `fit-view`
 * Emittery channel so other parts of the diagram can request a fit
 * without holding a ref.
 */
export const useFitView = (canvas: Element | null) => {
  const ctx = useSchemaDiagramContext();
  const { setZoom, setPosition, geometries, events } = ctx;

  const handleFitView = useCallback(() => {
    if (!canvas) return;
    const layout = fitView(canvas, [...geometries], DEFAULT_PADDINGS, [
      ZOOM_RANGE.min,
      1.0,
    ]);
    setZoom(layout.zoom);
    setPosition({ x: layout.rect.x, y: layout.rect.y });
  }, [canvas, geometries, setZoom, setPosition]);

  const handleFitViewRef = useRef(handleFitView);
  handleFitViewRef.current = handleFitView;

  useEffect(() => {
    const off = events.on("fit-view", () => {
      handleFitViewRef.current();
    });
    return () => {
      off();
    };
  }, [events]);

  return handleFitView;
};
