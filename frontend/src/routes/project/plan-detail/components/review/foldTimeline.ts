// Long-history fold: once the timeline has more than FOLD_THRESHOLD entries,
// only the first FOLD_HEAD and last FOLD_TAIL render; everything in between
// collapses into a single fold marker carrying the exact hidden count. One
// click ("Show all") expands the whole list.
export interface FoldableEntry {
  id: string;
}

export type FoldedItem<T extends FoldableEntry> =
  | { type: "entry"; entry: T }
  | { type: "fold"; count: number };

export const FOLD_HEAD = 3;
export const FOLD_TAIL = 3;
// Fold only when collapsing would actually hide something, i.e. there are more
// than head + tail entries. ">10" in the design (3 + 3 visible + >=5 hidden).
export const FOLD_THRESHOLD = 10;

export function foldTimeline<T extends FoldableEntry>(
  entries: T[],
  expanded: boolean
): FoldedItem<T>[] {
  const all = entries.map((entry) => ({ type: "entry" as const, entry }));
  if (expanded || entries.length <= FOLD_THRESHOLD) {
    return all;
  }
  const hiddenCount = entries.length - FOLD_HEAD - FOLD_TAIL;
  const items: FoldedItem<T>[] = entries
    .slice(0, FOLD_HEAD)
    .map((entry) => ({ type: "entry" as const, entry }));
  items.push({ type: "fold", count: hiddenCount });
  for (const entry of entries.slice(entries.length - FOLD_TAIL)) {
    items.push({ type: "entry", entry });
  }
  return items;
}
