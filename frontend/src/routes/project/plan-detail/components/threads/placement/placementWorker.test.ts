import { describe, expect, test } from "vitest";
import { current, UNAVAILABLE } from "./place";
import { runPlacementBatch } from "./placementWorker";

const limits = {
  maxLinesPerSheet: 1000,
  maxWorkPerPair: 10_000,
  maxWorkPerRun: 100_000,
};

const changedScript = (prefix: string) =>
  Array.from({ length: 50 }, (_, i) => `${prefix}${i}`).join("\n");

describe("runPlacementBatch", () => {
  test("places every anchor of every pair and echoes the generation", () => {
    const response = runPlacementBatch({
      generation: 7,
      limits,
      sheets: { s1: "A\nB\nC\n", c1: "X\nA\nB\nC\n", s2: "A\nB", c2: "A\nZ" },
      pairs: [
        {
          key: "p1",
          saved: "s1",
          current: "c1",
          anchors: [
            { id: "1-1", startLine: 1, endLine: 1 },
            { id: "2-3", startLine: 2, endLine: 3 },
          ],
        },
        {
          key: "p2",
          saved: "s2",
          current: "c2",
          anchors: [{ id: "2-2", startLine: 2, endLine: 2 }],
        },
      ],
    });
    expect(response.generation).toBe(7);
    expect(response.results).toEqual({
      p1: { "1-1": current(2, 2), "2-3": current(3, 4) },
      p2: { "2-2": { state: "OUTDATED" } },
    });
    expect(response.work).toBeGreaterThan(0);
  });

  test("shares the per-run budget in order and leaves the rest UNAVAILABLE", () => {
    const sheets = { a: changedScript("a"), b: changedScript("b") };
    const pair = (key: string) => ({
      key,
      saved: "a",
      current: "b",
      anchors: [{ id: "1-1", startLine: 1, endLine: 1 }],
    });
    const single = runPlacementBatch({
      generation: 1,
      limits: { ...limits, maxWorkPerRun: 1_000_000 },
      sheets,
      pairs: [pair("only")],
    });
    expect(single.results.only["1-1"]).toEqual({ state: "OUTDATED" });
    const perPair = single.work;

    // Room for one full pair plus a little: the second pair runs out.
    const response = runPlacementBatch({
      generation: 2,
      limits: { ...limits, maxWorkPerRun: perPair + 10 },
      sheets,
      pairs: [pair("first"), pair("second"), pair("third")],
    });
    expect(response.results.first["1-1"]).toEqual({ state: "OUTDATED" });
    expect(response.results.second["1-1"]).toEqual(UNAVAILABLE);
    expect(response.results.third["1-1"]).toEqual(UNAVAILABLE);
    // The second pair stops one step past its remaining budget; the third
    // has none left and does not run at all.
    expect(response.work).toBe(perPair + 10 + 1);
  });

  test("applies the per-pair budget independently of the run budget", () => {
    const response = runPlacementBatch({
      generation: 3,
      limits: { ...limits, maxWorkPerPair: 20 },
      sheets: { a: changedScript("a"), b: changedScript("b") },
      pairs: [
        {
          key: "p",
          saved: "a",
          current: "b",
          anchors: [{ id: "1-1", startLine: 1, endLine: 1 }],
        },
      ],
    });
    expect(response.results.p["1-1"]).toEqual(UNAVAILABLE);
    expect(response.work).toBe(21);
  });

  test("settles a pair whose sheet is missing from the request", () => {
    const response = runPlacementBatch({
      generation: 5,
      limits,
      sheets: { a: "A" },
      pairs: [
        {
          key: "p",
          saved: "a",
          current: "missing",
          anchors: [{ id: "1-1", startLine: 1, endLine: 1 }],
        },
      ],
    });
    expect(response.results.p["1-1"]).toEqual(UNAVAILABLE);
    expect(response.incomplete).toEqual([]);
  });

  test("returns an empty result set for no pairs", () => {
    expect(
      runPlacementBatch({ generation: 4, limits, sheets: {}, pairs: [] })
    ).toEqual({
      generation: 4,
      results: {},
      incomplete: [],
      work: 0,
    });
  });
});
