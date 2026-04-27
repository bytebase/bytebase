/**
 * Generic tree-shape contract — every tree node has a stable string key
 * and (optionally) child nodes of the same shape. Both
 * `WorksheetFolderNode` and `SQLEditorTreeNode` satisfy this.
 */
export interface VisibleRowNode {
  readonly key: string;
  // Each consumer's tree shape extends this — we don't constrain `children`
  // to the same `T` here because TypeScript struggles with self-referential
  // generic constraints. Callers pass concrete node types and the recursion
  // type-flows naturally through their own definitions.
}

/**
 * Count the number of rows the `Tree` primitive (react-arborist) will
 * actually render for a given root, so the Tree's fixed viewport can be
 * sized to its content rather than the 300px default.
 *
 * - When `keyword` is empty, count the node plus descendants of expanded
 *   folders — react-arborist hides collapsed children.
 * - When `keyword` is set, react-arborist filters visible nodes to matches
 *   plus their ancestors regardless of expand state. Mirror that here so
 *   the viewport shrinks during search; otherwise the tree reserves the
 *   full pre-filter height and leaves a large empty block under the
 *   matches.
 *
 * The `searchMatch` callback is the per-tree predicate that decides which
 * nodes match the keyword. Pass `() => false` (or omit `keyword`) when the
 * caller's filter is server-side and the rendered list is already
 * pre-filtered.
 */
export function countVisibleRows<
  T extends VisibleRowNode & { readonly children?: readonly T[] },
>(
  node: T,
  expandedKeys: ReadonlySet<string>,
  keyword: string,
  searchMatch: (node: T, term: string) => boolean
): number {
  if (!keyword) {
    let count = 1;
    if (expandedKeys.has(node.key)) {
      for (const child of node.children ?? []) {
        count += countVisibleRows(child, expandedKeys, keyword, searchMatch);
      }
    }
    return count;
  }

  let childCount = 0;
  for (const child of node.children ?? []) {
    childCount += countVisibleRows(child, expandedKeys, keyword, searchMatch);
  }
  const selfMatches = searchMatch(node, keyword);
  if (selfMatches || childCount > 0) {
    return 1 + childCount;
  }
  return 0;
}
