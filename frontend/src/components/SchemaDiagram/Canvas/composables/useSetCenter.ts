import { type Ref } from "vue";
import { DEFAULT_PADDINGS, useSchemaDiagramContext } from "../../common";
import type { CenterTarget, CenterTargetType, Rect } from "../../types";
import { fitView } from "../libs/fitView";

export const useSetCenter = (canvas: Ref<Element | undefined>) => {
  const { position, zoom, events, rectOfTable } = useSchemaDiagramContext();

  const setCenter = (rect: Rect, padding: number[], zooms: number[]) => {
    if (!canvas.value) {
      return;
    }
    // Fit view according to the center rect with limited zoom range.
    const layout = fitView(canvas.value, [rect], padding, zooms);
    zoom.value = layout.zoom;
    position.value = {
      x: layout.rect.x,
      y: layout.rect.y,
    };
  };

  events.on("set-center", (e) => {
    const { padding = DEFAULT_PADDINGS, zooms = [zoom.value, zoom.value] } = e;
    if (isTypedCenterTarget(e, "point")) {
      const center = e.target;
      setCenter({ ...center, width: 0, height: 0 }, padding, zooms);
    } else if (isTypedCenterTarget(e, "rect")) {
      setCenter(e.target, padding, zooms);
    } else if (isTypedCenterTarget(e, "table")) {
      const rect = rectOfTable(e.target);
      setCenter(rect, padding, zooms);
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
