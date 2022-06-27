import { uniqBy } from "lodash-es";

import {
  Database,
  Table,
  EditorModel,
  EditorPosition,
  SortText,
  CompletionItems,
} from "@/types";
import { keywords } from "./keywords";

export default class AutoCompletion {
  model: EditorModel;
  position: EditorPosition;
  databaseList: Database[];
  tableList: Table[];

  constructor(
    model: EditorModel,
    position: EditorPosition,
    databaseList: Database[],
    tableList: Table[]
  ) {
    this.model = model;
    this.position = position;
    this.databaseList = databaseList;
    this.tableList = tableList;
  }

  getWordRange() {
    const position = this.position;
    const word = this.model.getWordUntilPosition(position);
    const range = {
      startLineNumber: position.lineNumber,
      endLineNumber: position.lineNumber,
      startColumn: word.startColumn,
      endColumn: word.endColumn,
    };
    return range;
  }

  async getCompletionItemsForKeywords() {
    const monaco = await import("monaco-editor");
    const suggestions: CompletionItems = [];
    const range = this.getWordRange();

    keywords.forEach((keyword) => {
      suggestions.push({
        label: keyword,
        kind: monaco.languages.CompletionItemKind.Keyword,
        detail: "<Keyword>",
        documentation: keyword,
        sortText: SortText.KEYWORD,
        insertText: keyword,
        range,
      });
    });

    return uniqBy(suggestions, "label");
  }

  async getCompletionItemsForDatabaseList(): Promise<CompletionItems> {
    const suggestions: CompletionItems = [];
    const range = this.getWordRange();
    const monaco = await import("monaco-editor");
    this.databaseList.forEach(async (database) => {
      suggestions.push({
        label: database.name,
        kind: monaco.languages.CompletionItemKind.Struct,
        detail: "<Database>",
        documentation: database.name,
        sortText: SortText.DATABASE,
        insertText: database.name,
        range,
      });

      suggestions.push(
        ...(await this.getCompletionItemsForTableList(database))
      );
    });

    return uniqBy(suggestions, "label");
  }

  async getCompletionItemsForTableList(
    db?: Database,
    withDatabasePrefix = true
  ): Promise<CompletionItems> {
    const monaco = await import("monaco-editor");
    const suggestions: CompletionItems = [];
    const range = this.getWordRange();

    const filterTableListByDB = this.tableList.filter((table: Table) => {
      return table.database.name === db?.name;
    });

    const tableList = db ? filterTableListByDB : this.tableList;

    tableList.forEach(async (table) => {
      const label =
        withDatabasePrefix && db ? `${db?.name}.${table.name}` : table.name;
      suggestions.push({
        label,
        kind: monaco.languages.CompletionItemKind.Function,
        detail: "<Table>",
        documentation: label,
        sortText: SortText.TABLE,
        insertText: label,
        range,
      });
      if (table.columnList && table.columnList.length > 0) {
        suggestions.push(
          ...(await this.getCompletionItemsForTableColumnList(table))
        );
      }
    });

    return uniqBy(suggestions, "label");
  }

  async getCompletionItemsForTableColumnList(
    table: Table,
    withTablePrefix = true
  ): Promise<CompletionItems> {
    const monaco = await import("monaco-editor");
    const suggestions: CompletionItems = [];
    const range = this.getWordRange();

    table.columnList.forEach((column) => {
      const label = withTablePrefix
        ? `${table.name}.${column.name}`
        : column.name;

      suggestions.push({
        label,
        kind: monaco.languages.CompletionItemKind.Field,
        detail: "<Field>",
        documentation: label,
        sortText: SortText.COLUMN,
        insertText: label,
        range,
      });
    });

    return uniqBy(suggestions, "label");
  }
}
