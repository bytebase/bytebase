// @vitest-environment node
import { describe, expect, test } from "vitest";
import {
  FOLD_HEAD,
  FOLD_TAIL,
  type FoldedItem,
  foldTimeline,
} from "./foldTimeline";

interface E {
  id: string;
  pinned?: boolean;
}
const e = (id: string, pinned?: boolean): E => ({ id, pinned });
const idsOf = (items: FoldedItem<E>[]) =>
  items.map((i) => (i.type === "entry" ? i.entry.id : `FOLD:${i.count}`));

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

  test("pinned entries stay in place and split the fold", () => {
    const entries = Array.from({ length: 30 }, (_, i) =>
      e(`e${i}`, i >= 15 && i < 18)
    );
    expect(idsOf(foldTimeline(entries, false))).toEqual([
      "e0",
      "e1",
      "e2",
      "FOLD:12",
      "e15",
      "e16",
      "e17",
      "FOLD:9",
      "e27",
      "e28",
      "e29",
    ]);
  });

  test("a stretch too short to fold renders as is", () => {
    const entries = Array.from({ length: 20 }, (_, i) => e(`e${i}`, i >= 7));
    expect(idsOf(foldTimeline(entries, false))).toEqual(
      entries.map((entry) => entry.id)
    );
  });

  test("expanded shows everything", () => {
    const entries = Array.from({ length: 30 }, (_, i) => e(`e${i}`));
    expect(foldTimeline(entries, true)).toHaveLength(30);
  });

  test("a review's group is one entry, kept whole whether it folds into a stretch or stays pinned", () => {
    const entries = Array.from({ length: 14 }, (_, i) => ({
      id: `e${i}`,
      pinned: i === 3,
      children:
        i === 3 || i === 7
          ? [{ id: `card-${i}-a` }, { id: `card-${i}-b` }]
          : undefined,
    }));
    const items = foldTimeline(entries, false);
    const shown = items.flatMap((item) =>
      item.type === "entry" ? [item.entry] : []
    );
    const open = shown.find((entry) => entry.id === "e3");
    expect(open?.children?.map((child) => child.id)).toEqual([
      "card-3-a",
      "card-3-b",
    ]);
    expect(items.some((item) => item.type === "fold" && item.count === 7)).toBe(
      true
    );
    // A hidden group folds with the stretch around it, as one entry.
    expect(shown.some((entry) => entry.id === "e7")).toBe(false);
  });
});
