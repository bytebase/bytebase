import normalizeWheel from "normalize-wheel";
import { useCallback, useEffect, useRef } from "react";
import { minmax } from "@/utils/math";
import { ZOOM_RANGE } from "../../common/const";
import { useSchemaDiagramContext } from "../../common/context";
import { useDraggable } from "../../common/useDraggable";
import type { Point } from "../../types";

/**
 * React port of `Canvas/composables/useDragCanvas.ts`.
 *
 * Wires the canvas element for drag-to-pan and wheel-to-zoom. Returns
 * `handleZoom(delta, center?)` which the zoom button group calls
 * imperatively.
 */
export const useDragCanvas = (canvas: Element | null) => {
  const ctx = useSchemaDiagramContext();
  const { zoom, position, setZoom, setPosition, setPanning } = ctx;

  // Capture latest values without re-running the wheel listener every render.
  const stateRef = useRef({ zoom, position });
  stateRef.current = { zoom, position };

  const handleZoom = useCallback(
    (delta: number, center: Point = { x: 0.5, y: 0.5 }) => {
      if (!canvas) return;
      const { zoom: prevZoom, position: prevPos } = stateRef.current;
      const z = minmax(prevZoom + delta, ZOOM_RANGE.min, ZOOM_RANGE.max);
      const factor = z / prevZoom;
      setZoom(z);

      const { width, height } = canvas.getBoundingClientRect();
      const cx = center.x * width;
      const cy = center.y * height;
      setPosition({
        x: cx - factor * (cx - prevPos.x),
        y: cy - factor * (cy - prevPos.y),
      });
    },
    [canvas, setZoom, setPosition]
  );

  const handleZoomRef = useRef(handleZoom);
  handleZoomRef.current = handleZoom;

  // Drag-to-pan via the shared useDraggable hook.
  useDraggable(canvas, {
    exact: false,
    onPan: (dx, dy) => {
      const { position: prev } = stateRef.current;
      setPosition({ x: prev.x + dx, y: prev.y + dy });
      setPanning(true);
    },
    onEnd: () => {
      requestAnimationFrame(() => setPanning(false));
    },
  });

  // Wheel-to-zoom. Capture phase mirrors Vue's `useEventListener(..., true)`.
  useEffect(() => {
    if (!canvas) return;
    const handler = (e: WheelEvent) => {
      const ne = normalizeWheel(e);
      const scrollDelta = -ne.pixelY;
      const delta = scrollDelta / 250; // tuned for trackpad / mouse parity
      const rect = canvas.getBoundingClientRect();
      const center = {
        x: (e.pageX - rect.left) / rect.width,
        y: (e.pageY - rect.top) / rect.height,
      };
      handleZoomRef.current(delta, center);
      e.preventDefault();
      e.stopPropagation();
    };
    canvas.addEventListener("mousewheel", handler as EventListener, true);
    canvas.addEventListener("wheel", handler as EventListener, true);
    return () => {
      canvas.removeEventListener("mousewheel", handler as EventListener, true);
      canvas.removeEventListener("wheel", handler as EventListener, true);
    };
  }, [canvas]);

  return { handleZoom };
};
