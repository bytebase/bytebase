import { type Ref } from "vue";

import type { CenterTarget, CenterTargetType, Rect } from "../../types";
import { useSchemaDiagramContext } from "../../common";
import { fitView } from "../libs/fitView";
import { minmax } from "@/utils";
import { ZOOM_RANGE } from "../const";

export const useSetCenter = (canvas: Ref<Element | undefined>) => {
  const { position, zoom, events, rectOfTable } = useSchemaDiagramContext();

  const setCenter = (rect: Rect, padding: number[]) => {
    if (!canvas.value) {
      return;
    }
    // Fit view according to the center rect with limited zoom range.
    const zoomMin = minmax(zoom.value, ZOOM_RANGE.min, 0.5);
    const zoomMax = minmax(zoom.value, zoomMin, 1);
    const layout = fitView(canvas.value, [rect], padding, [zoomMin, zoomMax]);
    zoom.value = layout.zoom;
    position.value = {
      x: layout.rect.x,
      y: layout.rect.y,
    };
  };

  events.on("set-center", (e) => {
    const { padding = [0, 0, 0, 0] } = e;
    if (isTypedCenterTarget(e, "point")) {
      const center = e.target;
      setCenter({ ...center, width: 0, height: 0 }, padding);
    } else if (isTypedCenterTarget(e, "rect")) {
      setCenter(e.target, padding);
    } else if (isTypedCenterTarget(e, "table")) {
      const rect = rectOfTable(e.target);
      setCenter(rect, padding);
    } else {
      console.assert(false, `unknown set-center target "${e}"`);
    }
  });
};

const isTypedCenterTarget = <T extends CenterTargetType>(
  e: CenterTarget,
  type: T
): e is CenterTarget<T> => {
  return e.type === type;
};
