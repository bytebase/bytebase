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
  instances: Instance[];
  databases: Database[];
  tables: Table[];

  constructor(
    model: EditorModel,
    position: EditorPosition,
    instances: Instance[],
    databases: Database[],
    tables: Table[]
  ) {
    this.model = model;
    this.position = position;
    this.instances = instances;
    this.databases = databases;
    this.tables = tables;
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

  getCompletionItemsForInstances(): CompletionItems {
    const suggestions: CompletionItems = [];

    const range = this.getWordRange();

    this.instances.forEach((instance) => {
      suggestions.push({
        label: instance.name,
        kind: monaco.languages.CompletionItemKind.Method,
        detail: "<Instance>",
        documentation: instance.name,
        sortText: SortText.INSTASNCE,
        insertText: instance.name,
        range,
      });
    });

    return suggestions;
  }

  getCompletionItemsForDatabases(): CompletionItems {
    const suggestions: CompletionItems = [];

    const range = this.getWordRange();

    this.databases.forEach((database) => {
      suggestions.push({
        label: database.name,
        kind: monaco.languages.CompletionItemKind.Struct,
        detail: "<Database>",
        documentation: database.name,
        sortText: SortText.DATABASE,
        insertText: database.name,
        range,
      });
    });

    return suggestions;
  }

  getCompletionItemsForTables(): CompletionItems {
    let suggestions: CompletionItems = [];

    const range = this.getWordRange();

    this.tables.forEach((table) => {
      suggestions.push({
        label: table.name,
        kind: monaco.languages.CompletionItemKind.Function,
        detail: "<Table>",
        documentation: table.name,
        sortText: SortText.TABLE,
        insertText: table.name,
        range,
      });
      const tableColumnSuggestions =
        this.getCompletionItemsForTableColumns(table);
      suggestions = suggestions.concat(tableColumnSuggestions);
    });

    return suggestions;
  }

  getCompletionItemsForTableColumns(
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
