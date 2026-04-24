import type { WorksheetFolderNode } from "@/views/sql-editor/Sheet";

/**
 * Returns a filter predicate for the worksheet tree.
 *
 * The predicate always shows the root node (matched by `rootPath`) and any
 * node whose label contains the search keyword (case-insensitive).
 */
export const filterNode =
  (rootPath: string) =>
  (pattern: string, option: WorksheetFolderNode): boolean => {
    const keyword = pattern.trim().toLowerCase();
    if (option.key === rootPath || !keyword) {
      return true;
    }
    return option.label?.toLowerCase().includes(keyword) ?? false;
  };
