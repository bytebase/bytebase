// Long-history fold: past the first FOLD_HEAD and before the last FOLD_TAIL
// entries, each stretch of FOLD_MIN_HIDDEN or more unpinned entries collapses
// into a single fold marker carrying the exact hidden count. Pinned entries
// stay in place and split the stretch around them. One click ("Show all")
// expands the whole list.
export interface FoldableEntry {
  id: string;
  pinned?: boolean;
}

export type FoldedItem<T extends FoldableEntry> =
  | { type: "entry"; entry: T }
  | { type: "fold"; count: number };

export const FOLD_HEAD = 3;
export const FOLD_TAIL = 3;
// Fold only when collapsing would actually hide something. ">10" in the
// design: 3 + 3 visible + >=5 hidden.
export const FOLD_MIN_HIDDEN = 5;

export function foldTimeline<T extends FoldableEntry>(
  entries: T[],
  expanded: boolean
): FoldedItem<T>[] {
  if (expanded) {
    return entries.map((entry) => ({ type: "entry", entry }));
  }
  const items: FoldedItem<T>[] = [];
  let stretch: T[] = [];
  const flush = () => {
    if (stretch.length >= FOLD_MIN_HIDDEN) {
      items.push({ type: "fold", count: stretch.length });
    } else {
      for (const entry of stretch) items.push({ type: "entry", entry });
    }
    stretch = [];
  };
  entries.forEach((entry, i) => {
    const inMiddle = i >= FOLD_HEAD && i < entries.length - FOLD_TAIL;
    if (inMiddle && !entry.pinned) {
      stretch.push(entry);
      return;
    }
    flush();
    items.push({ type: "entry", entry });
  });
  flush();
  return items;
}
