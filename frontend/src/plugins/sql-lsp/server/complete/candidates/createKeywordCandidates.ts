import { uniqBy } from "lodash-es";
import type {
  CompletionItem,
  CompletionItemKind,
} from "vscode-languageserver-types";
import { SQLDialect } from "@/plugins/sql-lsp/types";
import { ICONS, SortText } from "../utils";
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

export const createKeywordCandidates = async (
  dialect: SQLDialect
): Promise<CompletionItem[]> => {
  const existed = cache.get(dialect);
  if (existed) return existed;

  const { keywords, operators, builtinFunctions } =
    await keywordGroupsOfDialect(dialect);
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
