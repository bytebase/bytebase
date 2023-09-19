import { minmax } from "@/utils";
import type { Geometry, Path, Point, Rect, Size } from "../types";

export const pointsOfRect = (rect: Rect): Point[] => {
  const { x, y, width, height } = rect;
  const nw = { x, y };
  const ne = { x: x + width, y };
  const sw = { x, y: y + height };
  const se = { x: x + width, y: y + height };
  return [nw, ne, sw, se];
};

export const calcBBox = (geometries: Geometry[]): Rect => {
  const points = geometries.flatMap(pointsOfGeometry);
  if (points.length === 0) {
    return { x: 0, y: 0, width: 0, height: 0 };
  }

  const min: Point = { x: Number.MAX_VALUE, y: Number.MAX_VALUE };
  const max: Point = { x: Number.MIN_VALUE, y: Number.MIN_VALUE };
  points.forEach(({ x, y }) => {
    if (x > max.x) max.x = x;
    if (y > max.y) max.y = y;
    if (x < min.x) min.x = x;
    if (y < min.y) min.y = y;
  });
  return {
    x: min.x,
    y: min.y,
    width: max.x - min.x,
    height: max.y - min.y,
  };
};

export type SegmentOverlap1D =
  | "BEFORE"
  | "OVERLAPS"
  | "CONTAINS"
  | "OVERLAPPED"
  | "CONTAINED"
  | "AFTER";

export const segmentOverlap1D = (
  a: number,
  b: number,
  c: number,
  d: number
): SegmentOverlap1D => {
  /**
   * Giving two segments `AB` and `CD` on a line,
   * find the intersection relationship of them.
   * 1. A-B C-D -> AB before CD
   * 2. A-C-B-D -> AB overlaps CD
   * 3. A-C-D-B -> AB contains CD
   * 4. C-A-D-B -> AB overlapped by CD
   * 5. C-A-B-D -> AB contained by CD
   * 6. C-D A-B -> AB after CD
   */
  console.assert(a <= b, `expected a=${a} to < b=${b}`);
  console.assert(c <= d, `expected c=${c} to < d=${d}`);

  if (b < c) return "BEFORE";
  if (a < c && b >= c && b < d) return "OVERLAPS";
  if (a < c && d < b) return "CONTAINS";
  if (a >= c && a < d && b >= d) return "OVERLAPPED";
  if (a >= c && b < d) return "CONTAINED";
  if (a >= d) return "AFTER";

  throw new Error(`should never be here a=${a}, b=${b}, c=${c}, d=${d}`);
};
export const isPoint = (geometry: Geometry): geometry is Point => {
  if (Array.isArray(geometry)) {
    return false;
  }
  return (
    typeof geometry["x"] === "number" &&
    typeof geometry["y"] === "number" &&
    typeof (geometry as Rect)["width"] === "undefined" &&
    typeof (geometry as Rect)["height"] === "undefined"
  );
};

export const isPath = (geometry: Geometry): geometry is Path => {
  return Array.isArray(geometry);
};

export const isRect = (geometry: Geometry): geometry is Rect => {
  if (Array.isArray(geometry)) {
    return false;
  }
  return (
    typeof geometry["x"] === "number" &&
    typeof geometry["y"] === "number" &&
    typeof (geometry as Rect)["width"] === "number" &&
    typeof (geometry as Rect)["height"] === "number"
  );
};

export const pointsOfGeometry = (g: Geometry): Point[] => {
  if (isPoint(g)) return [g];
  if (isPath(g)) return g;
  if (isRect(g)) return pointsOfRect(g);

  throw new Error(`'${String(g)}' is not a geometry.`);
};

export const fitBBox = (
  content: Size,
  boundary: Size,
  zoomRange: number[] = [0, Infinity]
) => {
  const contentWHRatio = content.width / content.height;
  const boundaryWHRatio = boundary.width / boundary.height;
  let zoom = 1;

  if (contentWHRatio > boundaryWHRatio) {
    // content is wider than boundary, fit horizontally
    const targetWidth = boundary.width;
    const targetZoom = targetWidth / content.width;
    zoom = minmax(targetZoom, zoomRange[0], zoomRange[1]);
  } else {
    // content is taller than boundary, fit vertically
    const targetHeight = boundary.height;
    const targetZoom = targetHeight / content.height;
    zoom = minmax(targetZoom, zoomRange[0], zoomRange[1]);
  }

  const targetSize = {
    width: content.width * zoom,
    height: content.height * zoom,
  };
  const targetPos = {
    x: (boundary.width - targetSize.width) / 2,
    y: (boundary.height - targetSize.height) / 2,
  };

  const rect = { ...targetSize, ...targetPos };
  return { zoom, rect };
};
