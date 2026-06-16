import { describe, expect, test } from "vitest";
import { FOLD_HEAD, FOLD_TAIL, foldTimeline } from "./foldTimeline";

interface E {
  id: string;
}
const e = (id: string): E => ({ id });

describe("foldTimeline", () => {
  test("timelines of 10 or fewer never fold", () => {
    const entries = Array.from({ length: 10 }, (_, i) => e(`e${i}`));
    const items = foldTimeline(entries, false);
    expect(items).toHaveLength(10);
    expect(items.every((i) => i.type === "entry")).toBe(true);
  });

  test("folds the middle once over 10 entries", () => {
    const entries = Array.from({ length: 11 }, (_, i) => e(`e${i}`));
    const items = foldTimeline(entries, false);
    // first 3 + fold + last 3
    expect(items).toHaveLength(FOLD_HEAD + 1 + FOLD_TAIL);
    const ids = items.map((i) => (i.type === "entry" ? i.entry.id : "FOLD"));
    expect(ids).toEqual(["e0", "e1", "e2", "FOLD", "e8", "e9", "e10"]);
    const fold = items.find((i) => i.type === "fold");
    expect(fold).toMatchObject({
      type: "fold",
      count: 11 - FOLD_HEAD - FOLD_TAIL,
    });
  });

  test("hidden count grows with the timeline length", () => {
    const entries = Array.from({ length: 30 }, (_, i) => e(`e${i}`));
    const items = foldTimeline(entries, false);
    expect(items).toHaveLength(FOLD_HEAD + 1 + FOLD_TAIL);
    expect(items.find((i) => i.type === "fold")).toMatchObject({ count: 24 });
  });

  test("expanded shows everything", () => {
    const entries = Array.from({ length: 30 }, (_, i) => e(`e${i}`));
    expect(foldTimeline(entries, true)).toHaveLength(30);
  });
});
