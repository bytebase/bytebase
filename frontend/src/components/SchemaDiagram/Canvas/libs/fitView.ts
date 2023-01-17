import { calcBBox, minmax } from "../../common";
import { Geometry, Rect } from "../../types";

export const fitView = (
  canvas: Element,
  geometries: Geometry[],
  paddings: number[] = [0, 0, 0, 0], // [T,R,B,L]
  zoomRange: number[] = [0.5, 2]
) => {
  const contentBBox = calcBBox(geometries);
  if (
    contentBBox.width < Number.EPSILON ||
    contentBBox.height < Number.EPSILON
  ) {
    return { zoom: 1, rect: contentBBox };
  }

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

  const contentWHRatio = content.width / content.height;
  const canvasWHRatio = view.width / view.height;

  if (contentWHRatio > canvasWHRatio) {
    // content is wider than canvas, fit horizontally
    const targetWidth = view.width;
    const targetZoom = targetWidth / content.width;
    const zoom = minmax(targetZoom, zoomRange[0], zoomRange[1]);
    const targetSize = {
      width: content.width * zoom,
      height: content.height * zoom,
    };
    const targetPos = {
      x: (view.width - targetSize.width) / 2 - contentMarginLeft * zoom,
      y: (view.height - targetSize.height) / 2 - contentMarginTop * zoom,
    };

    const rect = { ...targetSize, ...targetPos };
    return { zoom, rect };
  } else {
    // content is taller than canvas, fit vertically
    const targetHeight = view.height;
    const targetZoom = targetHeight / content.height;
    const zoom = minmax(targetZoom, zoomRange[0], zoomRange[1]);
    const targetSize = {
      width: content.width * zoom,
      height: content.height * zoom,
    };
    const targetPos = {
      x: (view.width - targetSize.width) / 2 - contentMarginLeft * zoom,
      y: (view.height - targetSize.height) / 2 - contentMarginTop * zoom,
    };
    const rect = { ...targetSize, ...targetPos };
    return { zoom, rect };
  }
};
