import { Position, Rect } from "../types";

export const pointsOfRect = (rect: Rect): Position[] => {
  const { x, y, width, height } = rect;
  const nw = { x, y };
  const ne = { x: x + width, y };
  const sw = { x, y: y + height };
  const se = { x: x + width, y: y + height };
  return [nw, ne, sw, se];
};

export const calcBBox = (points: Position[]): Rect => {
  const min: Position = { x: Number.MAX_VALUE, y: Number.MAX_VALUE };
  const max: Position = { x: Number.MIN_VALUE, y: Number.MIN_VALUE };
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
