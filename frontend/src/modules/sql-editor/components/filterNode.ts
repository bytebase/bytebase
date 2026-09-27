import type { SavedQueryFolderNode } from "@/modules/sql-editor/model/Sheet";

/**
 * Returns a filter predicate for the saved query tree.
 *
 * The predicate always shows the root node (matched by `rootPath`), load-more
 * rows, and any node whose label contains the search keyword (case-insensitive).
 */
export const filterNode =
  (rootPath: string) =>
  (pattern: string, option: SavedQueryFolderNode): boolean => {
    const keyword = pattern.trim().toLowerCase();
    if (option.key === rootPath || option.loadMore || !keyword) {
      return true;
    }
    return option.label?.toLowerCase().includes(keyword) ?? false;
  };
