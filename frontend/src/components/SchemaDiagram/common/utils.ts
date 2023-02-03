import { ZOOM_RANGE } from "./const";

export const expectedZoomRange = (target: number, min: number, max: number) => {
  const range = {
    min: Math.max(ZOOM_RANGE.min, min),
    max: Math.min(ZOOM_RANGE.max, max),
  };
  // If the target is in the range, don't zoom.
  if (target >= range.min && target <= range.max) {
    return [target, target];
  }
  // Zoom to limited range otherwise
  return [Math.min(target, range.min), Math.max(target, range.max)];
};
