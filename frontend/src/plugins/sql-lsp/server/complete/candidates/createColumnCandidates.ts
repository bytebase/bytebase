import { Table } from "@sql-lsp/types";
import { uniqBy } from "lodash-es";
import { CompletionItem } from "vscode-languageserver-types";
import { ICONS, SortText } from "../utils";

export const createColumnCandidates = (
  table: Table,
  withTablePrefix = true
): CompletionItem[] => {
  const suggestions = table.columns.map<CompletionItem>((column) => {
    const label = withTablePrefix
      ? `${table.name}.${column.name}`
      : column.name;
    return {
      label,
      kind: ICONS.COLUMN,
      detail: "<Column>",
      documentation: label,
      sortText: SortText.COLUMN,
      insertText: label,
    };
  });

  return uniqBy(suggestions, (item) => item.label);
};

export const createColumnCandidatesByAlias = (
  alias: string,
  tables: Table[]
): CompletionItem[] => {
  const suggestions = tables.flatMap<CompletionItem>((table) => {
    return table.columns.map<CompletionItem>((column) => {
      const label = `${alias}.${column.name}`;
      return {
        label,
        kind: ICONS.ALIAS,
        detail: "<Column>",
        documentation: label,
        sortText: SortText.ALIAS,
        insertText: label,
      };
    });
  });
  return uniqBy(suggestions, (item) => item.label);
};
