import type { Table } from "@sql-lsp/types";
import { uniqBy } from "lodash-es";
import type { CompletionItem } from "vscode-languageserver-types";
import { ICONS, SortText } from "../utils";

export const createTableCandidates = (
  tableList: Table[],
  withDatabasePrefix = true
): CompletionItem[] => {
  const suggestions: CompletionItem[] = [];

  tableList.forEach((table) => {
    const label = withDatabasePrefix
      ? `${table.database}.${table.name}`
      : table.name;
    suggestions.push({
      label,
      kind: ICONS.TABLE,
      detail: "<Table>",
      sortText: SortText.TABLE,
      insertText: label,
    });
  });

  return uniqBy(suggestions, (item) => item.label);
};

export const createSubQueryCandidates = (
  virtualTableList: Table[]
): CompletionItem[] => {
  const suggestions: CompletionItem[] = [];

  virtualTableList.forEach((table) => {
    const label = table.name;
    suggestions.push({
      label,
      kind: ICONS.SUBQUERY,
      detail: "<SubQuery>",
      sortText: SortText.SUBQUERY,
      insertText: label,
    });
  });

  return uniqBy(suggestions, (item) => item.label);
};
