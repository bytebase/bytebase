import { describe, expect, it } from "vitest";
import {
  calcBBox,
  isPath,
  isPoint,
  isRect,
  pointsOfRect,
  segmentOverlap1D,
} from "./geometry";

describe("SchemaDiagram geometry helpers", () => {
  it("classifies geometries", () => {
    expect(isPoint({ x: 1, y: 2 })).toBe(true);
    expect(isPoint({ x: 1, y: 2, width: 3, height: 4 })).toBe(false);
    expect(isPath([{ x: 1, y: 2 }])).toBe(true);
    expect(isRect({ x: 1, y: 2, width: 3, height: 4 })).toBe(true);
  });

  it("returns the four corners of a rect", () => {
    const rect = { x: 10, y: 20, width: 30, height: 40 };
    expect(pointsOfRect(rect)).toEqual([
      { x: 10, y: 20 },
      { x: 40, y: 20 },
      { x: 10, y: 60 },
      { x: 40, y: 60 },
    ]);
  });

  it("computes the bbox covering mixed geometries", () => {
    const bbox = calcBBox([
      { x: 0, y: 0 },
      { x: 100, y: 80, width: 20, height: 20 },
      [
        { x: -5, y: 5 },
        { x: 50, y: 50 },
      ],
    ]);
    expect(bbox).toEqual({ x: -5, y: 0, width: 125, height: 100 });
  });

  it("returns a zero rect when no geometries are provided", () => {
    expect(calcBBox([])).toEqual({ x: 0, y: 0, width: 0, height: 0 });
  });

  describe("segmentOverlap1D", () => {
    it("classifies all six relations between AB and CD", () => {
      // 1. AB before CD (A-B C-D)
      expect(segmentOverlap1D(0, 1, 2, 3)).toBe("BEFORE");
      // 2. AB overlaps CD (A-C-B-D)
      expect(segmentOverlap1D(0, 5, 3, 8)).toBe("OVERLAPS");
      // 3. AB contains CD (A-C-D-B)
      expect(segmentOverlap1D(0, 10, 3, 7)).toBe("CONTAINS");
      // 4. AB overlapped by CD (C-A-D-B)
      expect(segmentOverlap1D(3, 8, 0, 5)).toBe("OVERLAPPED");
      // 5. AB contained by CD (C-A-B-D)
      expect(segmentOverlap1D(3, 7, 0, 10)).toBe("CONTAINED");
      // 6. AB after CD (C-D A-B)
      expect(segmentOverlap1D(5, 6, 0, 4)).toBe("AFTER");
    });
  });
});
