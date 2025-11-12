import { escape } from "lodash-es";
import type { TreeOption } from "naive-ui";
import { getHighlightHTMLByRegExp } from "@/utils";

export const titleHTML = (title: string, keyword: string) => {
  const kw = keyword.toLowerCase().trim();

  if (!kw) {
    return escape(title);
  }

  return getHighlightHTMLByRegExp(
    escape(title),
    escape(kw),
    false /* !caseSensitive */
  );
};

// TODO(ed):
// Find a better UX for filter.
// 1. We do need to expand all nodes for filter results?
// 2. We do still need a tree for filter results? How about just plain list?
// 3. If we don't show empty folder in filter results, how to support "create/rename folder", "move/rename worksheet" actions?
export const filterNode =
  (rootPath: string) => (pattern: string, option: TreeOption) => {
    const keyword = pattern.trim().toLowerCase();
    if (option.key === rootPath || !keyword) {
      // always show the root node
      return true;
    }
    return option.label?.toLowerCase().includes(keyword) ?? false;
  };
