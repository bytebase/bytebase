import { useCallback, useEffect, useRef } from "react";
import { DEFAULT_PADDINGS, ZOOM_RANGE } from "../../common/const";
import { useSchemaDiagramContext } from "../../common/context";
import type { Geometry } from "../../types";
import { fitView } from "../libs/fitView";

/**
 * Returns a `handleFitView()` callback the zoom buttons / external code
 * can invoke imperatively, AND subscribes to the context's `fit-view`
 * Emittery channel so other parts of the diagram can request a fit
 * without holding a ref.
 */
export const useFitView = (canvas: Element | null) => {
  const ctx = useSchemaDiagramContext();
  const { setZoom, setPosition, geometries, events } = ctx;

  const fitTo = useCallback(
    (targets: Geometry[]) => {
      if (!canvas) return;
      const layout = fitView(canvas, targets, DEFAULT_PADDINGS, [
        ZOOM_RANGE.min,
        1.0,
      ]);
      setZoom(layout.zoom);
      setPosition({ x: layout.rect.x, y: layout.rect.y });
    },
    [canvas, setZoom, setPosition]
  );

  const handleFitView = useCallback(() => {
    fitTo([...geometries]);
  }, [fitTo, geometries]);

  const fitToRef = useRef(fitTo);
  fitToRef.current = fitTo;

  useEffect(() => {
    const off = events.on("fit-view", ({ data }) => {
      fitToRef.current(data.geometries);
    });
    return () => {
      off();
    };
  }, [events]);

  return handleFitView;
};
