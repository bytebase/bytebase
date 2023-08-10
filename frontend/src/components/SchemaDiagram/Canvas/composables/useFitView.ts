import { type Ref } from "vue";
import { useSchemaDiagramContext } from "../../common";
import { DEFAULT_PADDINGS, ZOOM_RANGE } from "../../common/const";
import { fitView } from "../libs/fitView";

export const useFitView = (canvas: Ref<Element | undefined>) => {
  const { zoom, position, geometries, events } = useSchemaDiagramContext();

  const handleFitView = () => {
    if (!canvas.value) return;
    const layout = fitView(
      canvas.value,
      [...geometries.value],
      DEFAULT_PADDINGS,
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
