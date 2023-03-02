import type {
  CompletionItem,
  CompletionItemKind,
} from "vscode-languageserver-types";
import { uniqBy } from "lodash-es";
import { ICONS, SortText } from "../utils";
import { SQLDialect } from "@/plugins/sql-lsp/types";
import { keywordGroupsOfDialect } from "./keywords";

const createCandidate = (
  label: string,
  detail: string,
  kind: CompletionItemKind
): CompletionItem => {
  return {
    label,
    kind,
    detail,
    documentation: label,
    sortText: SortText.KEYWORD,
    insertText: label,
  };
};

const cache = new Map<SQLDialect, CompletionItem[]>();

export const createKeywordCandidates = (
  dialect: SQLDialect
): CompletionItem[] => {
  const existed = cache.get(dialect);
  if (existed) return existed;

  const { keywords, operators, builtinFunctions } =
    keywordGroupsOfDialect(dialect);
  const suggestions = uniqBy(
    [
      ...keywords.map((keyword) =>
        createCandidate(keyword, "<Keyword>", ICONS.KEYWORD)
      ),
      ...operators.map((operator) =>
        createCandidate(operator, "<Operator>", ICONS.OPERATOR)
      ),
      ...builtinFunctions.map((fn) =>
        createCandidate(fn, "<Function>", ICONS.FUNCTION)
      ),
    ],
    (item) => item.label
  );
  cache.set(dialect, suggestions);
  return suggestions;
};
