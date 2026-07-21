import { describe, expect, it } from "vitest";
import { anchor, generateLine, grow, pickPorts } from "./ForeignKeyLine";

describe("ForeignKeyLine math", () => {
  describe("pickPorts", () => {
    it("BEFORE → src right, dest left", () => {
      expect(pickPorts("BEFORE", [0, 1], [2, 3])).toEqual(["RIGHT", "LEFT"]);
    });
    it("AFTER → src left, dest right", () => {
      expect(pickPorts("AFTER", [5, 6], [0, 1])).toEqual(["LEFT", "RIGHT"]);
    });
    it("CONTAINS → both LEFT when dest is to the left of src center", () => {
      expect(pickPorts("CONTAINS", [0, 100], [10, 30])).toEqual([
        "LEFT",
        "LEFT",
      ]);
    });
    it("CONTAINS → both RIGHT when dest is to the right of src center", () => {
      expect(pickPorts("CONTAINS", [0, 100], [60, 80])).toEqual([
        "RIGHT",
        "RIGHT",
      ]);
    });
    it("CONTAINED → both LEFT when src is to the left of dest center", () => {
      expect(pickPorts("CONTAINED", [10, 30], [0, 100])).toEqual([
        "LEFT",
        "LEFT",
      ]);
    });
  });

  describe("anchor / grow", () => {
    it("anchors on the right edge mid-height for RIGHT", () => {
      expect(anchor({ x: 10, y: 20, width: 30, height: 40 }, "RIGHT")).toEqual({
        x: 40,
        y: 40,
      });
    });
    it("anchors on the left edge mid-height for LEFT", () => {
      expect(anchor({ x: 10, y: 20, width: 30, height: 40 }, "LEFT")).toEqual({
        x: 10,
        y: 40,
      });
    });
    it("grows away from the anchor in the named direction", () => {
      expect(grow({ x: 10, y: 5 }, "RIGHT", 16)).toEqual({ x: 26, y: 5 });
      expect(grow({ x: 10, y: 5 }, "LEFT", 16)).toEqual({ x: -6, y: 5 });
    });
  });

  describe("generateLine", () => {
    it("creates a 4-point S-curve between two rects", () => {
      const a = { x: 0, y: 0, width: 100, height: 50 };
      const b = { x: 200, y: 100, width: 100, height: 50 };
      // a is BEFORE b → src RIGHT, dest LEFT
      const line = generateLine(a, "RIGHT", b, "LEFT");
      expect(line).toHaveLength(4);
      // anchor right of a = (100, 25)
      expect(line[0]).toEqual({ x: 100, y: 25 });
      // grow right by 16 = (116, 25)
      expect(line[1]).toEqual({ x: 116, y: 25 });
      // grow left from b's anchor (200, 125) by 16 = (184, 125)
      expect(line[2]).toEqual({ x: 184, y: 125 });
      // anchor left of b = (200, 125)
      expect(line[3]).toEqual({ x: 200, y: 125 });
    });
  });
});
