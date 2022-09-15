import { CompletionItemKind } from "vscode-languageserver-types";

export const ICONS = {
  DATABASE: CompletionItemKind.Class,
  TABLE: CompletionItemKind.Field,
  ALIAS: CompletionItemKind.Variable, // Not used yet
  COLUMN: CompletionItemKind.Interface,
  KEYWORD: CompletionItemKind.Keyword,
  FUNCTION: CompletionItemKind.Function,
  OPERATOR: CompletionItemKind.Operator,
  UTILITY: CompletionItemKind.Event, // Not used yet
};
