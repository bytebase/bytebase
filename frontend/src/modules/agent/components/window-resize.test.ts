import { describe, expect, test } from "vitest";
import {
  type ResizeBounds,
  type ResizeConstraints,
  type ResizeDirection,
  resizeWindowBounds,
} from "./window-resize";

const constraints: ResizeConstraints = {
  minWidth: 460,
  minHeight: 400,
  viewportWidth: 1000,
  viewportHeight: 800,
  margin: 16,
};

const startBounds: ResizeBounds = {
  x: 100,
  y: 120,
  width: 500,
  height: 450,
};

describe("resizeWindowBounds", () => {
  test.each<[ResizeDirection, { x: number; y: number }, ResizeBounds]>([
    ["n", { x: 0, y: 30 }, { x: 100, y: 150, width: 500, height: 420 }],
    ["s", { x: 0, y: 70 }, { x: 100, y: 120, width: 500, height: 520 }],
    ["e", { x: 80, y: 0 }, { x: 100, y: 120, width: 580, height: 450 }],
    ["w", { x: 20, y: 0 }, { x: 120, y: 120, width: 480, height: 450 }],
    ["ne", { x: 50, y: 20 }, { x: 100, y: 140, width: 550, height: 430 }],
    ["nw", { x: 40, y: 20 }, { x: 140, y: 140, width: 460, height: 430 }],
    ["se", { x: 50, y: 50 }, { x: 100, y: 120, width: 550, height: 500 }],
    ["sw", { x: -40, y: 50 }, { x: 60, y: 120, width: 540, height: 500 }],
  ])("resizes %s while anchoring the opposite edges", (direction, delta, expected) => {
    expect(
      resizeWindowBounds({
        direction,
        startBounds,
        deltaX: delta.x,
        deltaY: delta.y,
        constraints,
      })
    ).toEqual(expected);
  });

  test("stops west resizing at the minimum width while keeping the right edge fixed", () => {
    expect(
      resizeWindowBounds({
        direction: "w",
        startBounds: {
          x: 40,
          y: 120,
          width: 500,
          height: 450,
        },
        deltaX: 200,
        deltaY: 0,
        constraints,
      })
    ).toEqual({
      x: 80,
      y: 120,
      width: 460,
      height: 450,
    });
  });

  test("stops north resizing at the viewport margin while keeping the bottom edge fixed", () => {
    expect(
      resizeWindowBounds({
        direction: "n",
        startBounds: {
          x: 100,
          y: 40,
          width: 500,
          height: 420,
        },
        deltaX: 0,
        deltaY: -100,
        constraints,
      })
    ).toEqual({
      x: 100,
      y: 16,
      width: 500,
      height: 444,
    });
  });

  test("reduces the effective minimum width when the viewport is narrower than the preferred minimum", () => {
    expect(
      resizeWindowBounds({
        direction: "e",
        startBounds: {
          x: 16,
          y: 80,
          width: 380,
          height: 420,
        },
        deltaX: 100,
        deltaY: 0,
        constraints: {
          ...constraints,
          viewportWidth: 430,
        },
      })
    ).toEqual({
      x: 16,
      y: 80,
      width: 398,
      height: 420,
    });
  });
});
