import { keywords } from "./keywords";
import * as monaco from "monaco-editor";

type Model = monaco.editor.ITextModel;
type Position = monaco.Position;

export default class AutoCompletion {
  model: Model;
  position: Position;

  constructor(model: Model, position: Position) {
    this.model = model;
    this.position = position;
  }

  getWordRange () {
    const position = this.position
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
    const suggestions: monaco.languages.CompletionItem[] = [];

    const range = this.getWordRange()

    keywords.forEach((keyword) => {
      suggestions.push({
        label: keyword,
        kind: monaco.languages.CompletionItemKind.Keyword,
        detail: "<Keyword>",
        documentation: keyword,
        sortText: "3",
        insertText: keyword,
        range,
      });
    });

    return suggestions;
  }

  getCompletionItemsForDatabase() {
    const suggestions: monaco.languages.CompletionItem[] = [];

    const range = this.getWordRange();
  }
}
