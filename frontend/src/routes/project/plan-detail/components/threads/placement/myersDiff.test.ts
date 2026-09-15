import { describe, expect, it } from "vitest";
import { diffLines, type Hunk } from "./myersDiff";

// Applies `hunks` to `saved`, taking replacement lines from `current`; used by
// tests to prove a hunk list is a faithful edit script.
function applyHunks(
  saved: readonly string[],
  current: readonly string[],
  hunks: readonly Hunk[]
): string[] {
  const out: string[] = [];
  let sPos = 1;
  let cPos = 1;
  for (const hunk of hunks) {
    while (sPos < hunk.sStart) {
      out.push(saved[sPos - 1]);
      sPos++;
      cPos++;
    }
    for (let i = 0; i < hunk.cLen; i++) out.push(current[cPos - 1 + i]);
    sPos += hunk.sLen;
    cPos += hunk.cLen;
  }
  while (sPos <= saved.length) {
    out.push(saved[sPos - 1]);
    sPos++;
  }
  return out;
}

const diff = (a: string[], b: string[], budget = 1_000_000): Hunk[] => {
  const result = diffLines(a, b, budget);
  if (!result.complete) throw new Error("diff did not complete");
  // Every pinned hunk list must also be a correct edit script.
  expect(applyHunks(a, b, result.hunks)).toEqual(b);
  return result.hunks;
};

describe("diffLines", () => {
  it("returns no hunks for identical or empty inputs", () => {
    expect(diff([], [])).toEqual([]);
    expect(diff(["A"], ["A"])).toEqual([]);
    expect(diff(["A", "B", "C"], ["A", "B", "C"])).toEqual([]);
  });

  it("reports a whole-document insertion or deletion", () => {
    expect(diff([], ["A", "B"])).toEqual([{ sStart: 1, sLen: 0, cLen: 2 }]);
    expect(diff(["A", "B"], [])).toEqual([{ sStart: 1, sLen: 2, cLen: 0 }]);
  });

  it("places pure insertions at the saved line they precede", () => {
    expect(diff(["B", "C"], ["A", "B", "C"])).toEqual([
      { sStart: 1, sLen: 0, cLen: 1 },
    ]);
    expect(diff(["A", "C"], ["A", "B", "C"])).toEqual([
      { sStart: 2, sLen: 0, cLen: 1 },
    ]);
    expect(diff(["A", "B"], ["A", "B", "C"])).toEqual([
      { sStart: 3, sLen: 0, cLen: 1 },
    ]);
  });

  it("reports deletions at the first removed saved line", () => {
    expect(diff(["A", "B", "C"], ["B", "C"])).toEqual([
      { sStart: 1, sLen: 1, cLen: 0 },
    ]);
    expect(diff(["A", "B", "C"], ["A", "C"])).toEqual([
      { sStart: 2, sLen: 1, cLen: 0 },
    ]);
    expect(diff(["A", "B", "C"], ["A", "B"])).toEqual([
      { sStart: 3, sLen: 1, cLen: 0 },
    ]);
  });

  it("merges an adjacent delete and insert into one replacement hunk", () => {
    expect(diff(["A", "B", "C"], ["A", "X", "C"])).toEqual([
      { sStart: 2, sLen: 1, cLen: 1 },
    ]);
    expect(diff(["A", "B", "C", "D"], ["A", "X", "Y", "Z", "D"])).toEqual([
      { sStart: 2, sLen: 2, cLen: 3 },
    ]);
  });

  it("keeps separate hunks separated by unchanged lines", () => {
    expect(diff(["A", "B", "C", "D", "E"], ["A", "X", "C", "D", "Y"])).toEqual([
      { sStart: 2, sLen: 1, cLen: 1 },
      { sStart: 5, sLen: 1, cLen: 1 },
    ]);
  });

  // Pinned alignment for repeated lines: the design accepts whichever
  // occurrence the fixed implementation keeps. Changing this expectation
  // moves comments and needs a deliberate decision.
  it("pins which duplicate survives when one of two copies is removed", () => {
    // [A, X, B, X, C] -> [A, X, C]: the first X is kept, B and the second X
    // are deleted as one hunk.
    expect(diff(["A", "X", "B", "X", "C"], ["A", "X", "C"])).toEqual([
      { sStart: 3, sLen: 2, cLen: 0 },
    ]);
    // Adjacent duplicates: removing one of [X, X] deletes the second.
    expect(diff(["A", "X", "X", "B"], ["A", "X", "B"])).toEqual([
      { sStart: 3, sLen: 1, cLen: 0 },
    ]);
    // Adding a duplicate inserts it after the existing one.
    expect(diff(["A", "X", "B"], ["A", "X", "X", "B"])).toEqual([
      { sStart: 3, sLen: 0, cLen: 1 },
    ]);
  });

  it("pins the alignment of a block reorder", () => {
    // Moving C above A/B is reported as deleting C below and inserting it
    // above, so A and B stay mapped and C does not.
    expect(diff(["A", "B", "C"], ["C", "A", "B"])).toEqual([
      { sStart: 1, sLen: 0, cLen: 1 },
      { sStart: 3, sLen: 1, cLen: 0 },
    ]);
    // Swapping two lines keeps the second one: A is deleted above B and
    // re-inserted below it.
    expect(diff(["A", "B"], ["B", "A"])).toEqual([
      { sStart: 1, sLen: 1, cLen: 0 },
      { sStart: 3, sLen: 0, cLen: 1 },
    ]);
  });

  it("compares tokens byte for byte, including terminators", () => {
    expect(diff(["A\r\n", "B"], ["A\n", "B"])).toEqual([
      { sStart: 1, sLen: 1, cLen: 1 },
    ]);
    expect(diff(["A\n", ""], ["A"])).toEqual([{ sStart: 1, sLen: 2, cLen: 1 }]);
  });

  it("counts work and stops without hunks once the budget is exceeded", () => {
    const a = Array.from({ length: 40 }, (_, i) => `a${i}`);
    const b = Array.from({ length: 40 }, (_, i) => `b${i}`);
    const full = diffLines(a, b, 1_000_000);
    expect(full.complete).toBe(true);
    expect(full.work).toBeGreaterThan(40);
    const bounded = diffLines(a, b, 100);
    expect(bounded).toEqual({ complete: false, work: 101 });
    // A budget exactly equal to the work needed still completes.
    expect(diffLines(a, b, full.work).complete).toBe(true);
    expect(diffLines(a, b, full.work - 1).complete).toBe(false);
  });

  it("produces a valid edit script for random inputs", () => {
    let seed = 7;
    const rand = () => {
      seed = (seed * 1103515245 + 12345) & 0x7fffffff;
      return seed;
    };
    for (let round = 0; round < 200; round++) {
      const alphabet = ["A", "B", "C", "D"];
      const a = Array.from({ length: rand() % 12 }, () => alphabet[rand() % 4]);
      const b = Array.from({ length: rand() % 12 }, () => alphabet[rand() % 4]);
      const result = diffLines(a, b, 1_000_000);
      expect(result.complete).toBe(true);
      if (result.complete) expect(applyHunks(a, b, result.hunks)).toEqual(b);
    }
  });
});
