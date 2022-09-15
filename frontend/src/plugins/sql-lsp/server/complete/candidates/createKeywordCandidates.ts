import type {
  CompletionItem,
  CompletionItemKind,
} from "vscode-languageserver-types";
import { uniqBy } from "lodash-es";
import { keywords, operators, builtinFunctions } from "./keywords";
import { ICONS, SortText } from "../utils";

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

export const createKeywordCandidates = (): CompletionItem[] => {
  return suggestions;
};
