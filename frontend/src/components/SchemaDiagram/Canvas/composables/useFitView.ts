import { type Ref } from "vue";

import { ZOOM_RANGE } from "../const";
import { useSchemaDiagramContext } from "../../common";
import { fitView } from "../libs/fitView";

export const useFitView = (canvas: Ref<Element | undefined>) => {
  const { zoom, position, geometries, events } = useSchemaDiagramContext();

  const handleFitView = () => {
    if (!canvas.value) return;
    const layout = fitView(
      canvas.value,
      [...geometries.value],
      [10, 20, 40, 20] /* paddings [T,R,B,L] */,
      [ZOOM_RANGE.min, 1.0] /* [zoomMin, zoomMax] */
    );
    zoom.value = layout.zoom;
    position.value = {
      x: layout.rect.x,
      y: layout.rect.y,
    };
  };
  events.on("fit-view", handleFitView);

  return handleFitView;
};
