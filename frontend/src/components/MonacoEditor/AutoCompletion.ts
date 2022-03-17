import * as monaco from "monaco-editor";

import {
  Instance,
  Database,
  Table,
  EditorModel,
  EditorPosition,
  SortText,
  CompletionItems,
} from "../../types";
import { keywords } from "./keywords";

export default class AutoCompletion {
  model: EditorModel;
  position: EditorPosition;
  instanceList: Instance[];
  databaseList: Database[];
  tableList: Table[];

  constructor(
    model: EditorModel,
    position: EditorPosition,
    instanceList: Instance[],
    databaseList: Database[],
    tableList: Table[]
  ) {
    this.model = model;
    this.position = position;
    this.instanceList = instanceList;
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

  getCompletionItemsForKeywords() {
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

    return suggestions;
  }

  getCompletionItemsForDatabaseList(): CompletionItems {
    let suggestions: CompletionItems = [];

    const range = this.getWordRange();

    this.databaseList.forEach((database) => {
      suggestions.push({
        label: database.name,
        kind: monaco.languages.CompletionItemKind.Struct,
        detail: "<Database>",
        documentation: database.name,
        sortText: SortText.DATABASE,
        insertText: database.name,
        range,
      });

      suggestions = suggestions.concat(
        this.getCompletionItemsForTableList(database)
      );
    });

    return suggestions;
  }

  getCompletionItemsForTableList(
    db?: Database,
    withDatabasePrefix = true
  ): CompletionItems {
    let suggestions: CompletionItems = [];

    const range = this.getWordRange();

    const filterTableListByDB = this.tableList.filter((table: Table) => {
      return table.database.name === db?.name;
    });

    const tableList = db ? filterTableListByDB : this.tableList;

    tableList.forEach((table) => {
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
        suggestions = suggestions.concat(
          this.getCompletionItemsForTableColumnList(table)
        );
      }
    });

    return suggestions;
  }

  getCompletionItemsForTableColumnList(
    table: Table,
    withTablePrefix = true
  ): CompletionItems {
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

    return suggestions;
  }
}
