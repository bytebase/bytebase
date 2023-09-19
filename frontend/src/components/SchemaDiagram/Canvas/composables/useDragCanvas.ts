import { useEventListener } from "@vueuse/core";
import normalizeWheel from "normalize-wheel";
import { type Ref, ref } from "vue";
import { minmax } from "@/utils";
import { useDraggable, useSchemaDiagramContext } from "../../common";
import { ZOOM_RANGE } from "../../common/const";
import type { Point } from "../../types";

export const useDragCanvas = (canvas: Ref<Element | undefined>) => {
  const { zoom, position, panning } = useSchemaDiagramContext();

  const zoomCenter = ref<Point>({ x: 0.5, y: 0.5 });

  const handleZoom = (delta: number, center: Point = { x: 0.5, y: 0.5 }) => {
    if (!canvas.value) return "";

    const z = minmax(zoom.value + delta, ZOOM_RANGE.min, ZOOM_RANGE.max);
    const factor = z / zoom.value;
    zoom.value = z;

    zoomCenter.value = center;

    // While zooming, we need to adjust the `position` according to the
    // zoom level and zoom center.
    const { width, height } = canvas.value.getBoundingClientRect();
    const cx = center.x * width;
    const cy = center.y * height;
    position.value.x = cx - factor * (cx - position.value.x);
    position.value.y = cy - factor * (cy - position.value.y);
  };
  const handlePan = (x: number, y: number) => {
    position.value.x += x;
    position.value.y += y;
    panning.value = true;
  };
  const handlePanEnd = () => {
    requestAnimationFrame(() => {
      panning.value = false;
    });
  };

  useDraggable(canvas, {
    exact: false,
    onPan: handlePan,
    onEnd: handlePanEnd,
  });

  useEventListener(
    canvas,
    "mousewheel",
    (e: WheelEvent) => {
      const ne = normalizeWheel(e);

      const scrollDelta = -ne.pixelY;
      const delta = scrollDelta / 250; // adjust the scrolling speed
      const rect = canvas.value!.getBoundingClientRect();
      const center = {
        x: (e.pageX - rect.left) / rect.width,
        y: (e.pageY - rect.top) / rect.height,
      };
      handleZoom(delta, center);

      e.preventDefault();
      e.stopPropagation();
    },
    true
  );

  return { handleZoom, handlePan, handlePanEnd };
};
