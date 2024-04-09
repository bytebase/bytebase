import { escape } from "lodash-es";
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
