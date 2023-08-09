import { calcBBox, fitBBox } from "../../common";
import { ZOOM_RANGE } from "../../common/const";
import type { Geometry, Rect } from "../../types";

export const fitView = (
  canvas: Element,
  geometries: Geometry[],
  paddings: number[] = [0, 0, 0, 0], // [T,R,B,L]
  zoomRange: number[] = [ZOOM_RANGE.min, ZOOM_RANGE.max]
) => {
  const contentBBox = calcBBox(geometries);
  if (contentBBox.width < Number.EPSILON) contentBBox.width = Number.EPSILON;
  if (contentBBox.height < Number.EPSILON) contentBBox.height = Number.EPSILON;

  const canvasBBox = canvas.getBoundingClientRect();
  const [paddingTop, paddingRight, paddingBottom, paddingLeft] = paddings;
  const viewBBox = {
    x: paddingLeft,
    y: paddingTop,
    width: canvasBBox.width - paddingLeft - paddingRight,
    height: canvasBBox.height - paddingTop - paddingBottom,
  };

  const layout = calcFitRect(contentBBox, viewBBox, zoomRange);
  layout.rect.x += paddingLeft;
  layout.rect.y += paddingTop;
  return layout;
};

const calcFitRect = (content: Rect, view: Rect, zoomRange: number[]) => {
  const contentMarginTop = content.y;
  const contentMarginLeft = content.x;

  const { zoom, rect } = fitBBox(content, view, zoomRange);
  rect.x -= contentMarginLeft * zoom;
  rect.y -= contentMarginTop * zoom;

  return { zoom, rect };
};
