import { describe, expect, it } from "vitest";
import {
  current,
  diffPair,
  OUTDATED,
  placeOnPair,
  placeWithHunks,
  tokenizeSheet,
  UNAVAILABLE,
} from "./place";

const limits = { maxLinesPerSheet: 10_000, maxWork: 1_000_000 };
const at = (saved: string, currentText: string, first: number, last = first) =>
  placeOnPair(
    diffPair(tokenizeSheet(saved), tokenizeSheet(currentText), limits),
    { startLine: first, endLine: last }
  );

// A longer script with a comment on the middle statement.
const script = [
  "CREATE TABLE a (id INT);",
  "",
  "ALTER TABLE a",
  "  ADD COLUMN name TEXT;",
  "",
  "DROP TABLE b;",
].join("\n");

describe("place: the design's example table", () => {
  it("shifts by the net line count for edits above", () => {
    const inserted = `-- header\n-- more\n${script}`;
    expect(at(script, inserted, 3, 4)).toEqual(current(5, 6));
    const deleted = script.split("\n").slice(1).join("\n");
    expect(at(script, deleted, 3, 4)).toEqual(current(2, 3));
  });

  it("keeps the same lines for edits below", () => {
    expect(at(script, `${script}\nDROP TABLE c;`, 3, 4)).toEqual(current(3, 4));
    const deletedBelow = script.split("\n").slice(0, 5).join("\n");
    expect(at(script, deletedBelow, 3, 4)).toEqual(current(3, 4));
  });

  it("outdates when any character inside changes", () => {
    const changed = script.replace("name TEXT", "name  TEXT");
    expect(at(script, changed, 3, 4)).toEqual(OUTDATED);
    expect(at(script, changed, 4)).toEqual(OUTDATED);
    // The untouched line of the selection alone is still current.
    expect(at(script, changed, 3)).toEqual(current(3, 3));
  });

  it("comes back after an intermediate edit is reverted", () => {
    const inserted = `-- header\n${script}`;
    expect(at(script, inserted, 3, 4)).toEqual(current(4, 5));
    expect(at(script, script, 3, 4)).toEqual(current(3, 4));
  });

  it("outdates when its lines are deleted", () => {
    const removed = script.split("\n").filter((_, i) => i !== 2 && i !== 3);
    expect(at(script, removed.join("\n"), 3, 4)).toEqual(OUTDATED);
  });

  it("follows the pinned alignment for duplicate lines", () => {
    // [A, X, B, X, C] -> [A, X, C] keeps the first X.
    expect(at("A\nX\nB\nX\nC", "A\nX\nC", 2)).toEqual(current(2, 2));
    expect(at("A\nX\nB\nX\nC", "A\nX\nC", 4)).toEqual(OUTDATED);
  });
});

describe("place: hunk boundaries", () => {
  it("shifts for an insertion immediately above and ignores one below", () => {
    expect(at("A\nB\nC", "A\nX\nB\nC", 2)).toEqual(current(3, 3));
    expect(at("A\nB\nC", "A\nB\nX\nC", 2)).toEqual(current(2, 2));
    // Appending after the last line gives that line a terminator it did not
    // have, which is a change to it.
    expect(at("A\nB\nC", "A\nB\nC\nX", 2, 3)).toEqual(OUTDATED);
    expect(at("A\nB\nC\n", "A\nB\nC\nX\n", 2, 3)).toEqual(current(2, 3));
  });

  it("outdates a hunk that starts above and ends inside the selection", () => {
    expect(at("A\nB\nC\nD", "A\nX\nY\nD", 3, 4)).toEqual(OUTDATED);
  });

  it("outdates an insertion between selected lines", () => {
    expect(at("A\nB\nC", "A\nX\nB\nC", 1, 2)).toEqual(OUTDATED);
    expect(at("A\nB\nC", "A\nB\nX\nC", 2, 3)).toEqual(OUTDATED);
  });

  it("handles the first and last lines", () => {
    expect(at("A\nB", "X\nA\nB", 1)).toEqual(current(2, 2));
    expect(at("A\nB\n", "A\nB\nX\n", 2)).toEqual(current(2, 2));
    expect(at("A\nB", "A\nB\nX", 2)).toEqual(OUTDATED);
    expect(at("A\nB", "B", 1)).toEqual(OUTDATED);
    expect(at("A\nB", "A", 2)).toEqual(OUTDATED);
  });

  it("handles a multi-line anchor spanning the whole document", () => {
    expect(at("A\nB", "A\nB", 1, 2)).toEqual(current(1, 2));
    expect(at("A\nB\n", "X\nA\nB\nY\n", 1, 2)).toEqual(current(2, 3));
    expect(at("A\nB", "X\nA\nB\nY", 1, 2)).toEqual(OUTDATED);
    expect(at("A\nB", "A\nX\nB", 1, 2)).toEqual(OUTDATED);
  });
});

describe("place: line endings and terminal lines", () => {
  it("never keeps a comment on a line that no longer exists", () => {
    // "A\n" has an empty line 2; removing the newline removes that line.
    expect(at("A\n", "A", 2)).toEqual(OUTDATED);
    // Line 1 lost its terminator, so it changed too.
    expect(at("A\n", "A", 1)).toEqual(OUTDATED);
    expect(at("A\nB\n", "A\nB", 1)).toEqual(current(1, 1));
  });

  it("treats a final-newline change as a change to the last line", () => {
    expect(at("A\nB", "A\nB\n", 2)).toEqual(OUTDATED);
    expect(at("A\nB", "A\nB\n", 1)).toEqual(current(1, 1));
  });

  it("treats a line-ending change as a text change", () => {
    expect(at("A\r\nB\r\n", "A\nB\n", 1)).toEqual(OUTDATED);
    expect(at("A\r\nB\r\n", "A\nB\n", 2)).toEqual(OUTDATED);
    // The final empty line has no terminator on either side.
    expect(at("A\r\nB\r\n", "A\nB\n", 3)).toEqual(current(3, 3));
    expect(at("A\rB", "A\nB", 2)).toEqual(current(2, 2));
  });

  it("handles empty documents", () => {
    expect(at("", "", 1)).toEqual(current(1, 1));
    expect(at("", "A", 1)).toEqual(OUTDATED);
    expect(at("A", "", 1)).toEqual(OUTDATED);
  });
});

describe("place: invalid input and limits", () => {
  it("returns UNAVAILABLE for source bounds outside the saved sheet", () => {
    expect(at("A\nB", "A\nB", 0)).toEqual(UNAVAILABLE);
    expect(at("A\nB", "A\nB", 3)).toEqual(UNAVAILABLE);
    expect(at("A\nB", "A\nB", 2, 1)).toEqual(UNAVAILABLE);
    expect(at("A\nB", "A\nB", 1, 3)).toEqual(UNAVAILABLE);
  });

  it("returns UNAVAILABLE rather than trusting an inconsistent hunk list", () => {
    // A hunk list that claims one deleted line above but a shorter target.
    expect(
      placeWithHunks(["A", "B"], ["B"], [], { startLine: 2, endLine: 2 })
    ).toEqual(UNAVAILABLE);
    // Mapped range below line 1.
    expect(
      placeWithHunks(["A", "B"], ["B"], [{ sStart: 1, sLen: 2, cLen: 0 }], {
        startLine: 2,
        endLine: 2,
      })
    ).toEqual(OUTDATED);
    expect(
      placeWithHunks(["A", "B"], ["A"], [{ sStart: 1, sLen: 1, cLen: 0 }], {
        startLine: 2,
        endLine: 2,
      })
    ).toEqual(UNAVAILABLE);
    // Mapped lines exist but their text differs.
    expect(
      placeWithHunks(["A", "B"], ["A", "X"], [], { startLine: 2, endLine: 2 })
    ).toEqual(UNAVAILABLE);
  });

  it("skips the diff for identical content", () => {
    const pair = diffPair(tokenizeSheet("A\nB"), tokenizeSheet("A\nB"), limits);
    expect(pair.hunks).toEqual([]);
    expect(pair.work).toBe(0);
  });

  it("returns UNAVAILABLE for every anchor when the work budget is exceeded", () => {
    const saved = Array.from({ length: 200 }, (_, i) => `a${i}`).join("\n");
    const currentText = Array.from({ length: 200 }, (_, i) => `b${i}`).join(
      "\n"
    );
    const pair = diffPair(tokenizeSheet(saved), tokenizeSheet(currentText), {
      maxLinesPerSheet: 10_000,
      maxWork: 50,
    });
    expect(pair.hunks).toBeUndefined();
    expect(pair.work).toBe(51);
    expect(placeOnPair(pair, { startLine: 1, endLine: 1 })).toEqual(
      UNAVAILABLE
    );
    expect(placeOnPair(pair, { startLine: 200, endLine: 200 })).toEqual(
      UNAVAILABLE
    );
  });

  it("returns UNAVAILABLE when a sheet exceeds the line cap, even if identical", () => {
    const text = "A\nB\nC";
    const pair = diffPair(tokenizeSheet(text), tokenizeSheet(text), {
      maxLinesPerSheet: 2,
      maxWork: 1000,
    });
    expect(pair.hunks).toBeUndefined();
    expect(placeOnPair(pair, { startLine: 1, endLine: 1 })).toEqual(
      UNAVAILABLE
    );
    // The cap is inclusive.
    expect(
      diffPair(tokenizeSheet(text), tokenizeSheet(text), {
        maxLinesPerSheet: 3,
        maxWork: 1,
      }).hunks
    ).toEqual([]);
  });
});
