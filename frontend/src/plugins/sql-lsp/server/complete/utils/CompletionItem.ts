import { CompletionItemKind } from "vscode-languageserver-types";

export const ICONS = {
  DATABASE: CompletionItemKind.Class,
  TABLE: CompletionItemKind.Field,
  SUBQUERY: CompletionItemKind.Field, // Same as TABLE
  COLUMN: CompletionItemKind.Interface,
  ALIAS: CompletionItemKind.Interface, // Same as COLUMN
  KEYWORD: CompletionItemKind.Keyword,
  FUNCTION: CompletionItemKind.Function,
  OPERATOR: CompletionItemKind.Operator,
  UTILITY: CompletionItemKind.Event, // Not used yet
};
