import { describe, expect, it } from "vitest";
import { fitView } from "./fitView";

const fakeCanvas = (width: number, height: number): Element => {
  return {
    getBoundingClientRect: () => ({
      width,
      height,
      top: 0,
      left: 0,
      right: width,
      bottom: height,
      x: 0,
      y: 0,
      toJSON() {
        return { width, height };
      },
    }),
  } as Element;
};

describe("fitView", () => {
  it("fits content into canvas with no padding at zoom 1 when content equals view", () => {
    const canvas = fakeCanvas(1000, 600);
    const layout = fitView(
      canvas,
      [{ x: 0, y: 0, width: 1000, height: 600 }],
      [0, 0, 0, 0],
      [0, 2]
    );
    expect(layout.zoom).toBeCloseTo(1, 5);
    expect(layout.rect.x).toBeCloseTo(0, 5);
    expect(layout.rect.y).toBeCloseTo(0, 5);
  });

  it("scales down content larger than the viewport", () => {
    const canvas = fakeCanvas(1000, 600);
    const layout = fitView(
      canvas,
      [{ x: 0, y: 0, width: 2000, height: 1200 }],
      [0, 0, 0, 0],
      [0, 2]
    );
    // 1000/2000 = 0.5 (width is the limiting dimension since both are 2x).
    expect(layout.zoom).toBeCloseTo(0.5, 5);
  });

  it("respects symmetrical padding by shrinking the view box", () => {
    const canvas = fakeCanvas(1000, 600);
    const padded = fitView(
      canvas,
      [{ x: 0, y: 0, width: 1000, height: 600 }],
      [50, 50, 50, 50],
      [0, 2]
    );
    // The view shrinks from 1000x600 to 900x500. Limiting dimension is height
    // (500/600 ≈ 0.833 < 900/1000 = 0.9), so zoom is 500/600.
    expect(padded.zoom).toBeCloseTo(500 / 600, 5);
  });

  it("zero-content collapses without crashing (epsilon clamp)", () => {
    const canvas = fakeCanvas(1000, 600);
    const layout = fitView(canvas, [], [0, 0, 0, 0], [0.1, 2]);
    // With no geometries, calcBBox returns a zero rect. fitView clamps to
    // EPSILON to avoid divide-by-zero, so the result is well-defined.
    expect(Number.isFinite(layout.zoom)).toBe(true);
    expect(Number.isFinite(layout.rect.x)).toBe(true);
    expect(Number.isFinite(layout.rect.y)).toBe(true);
  });
});
