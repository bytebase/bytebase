import { computed } from "vue";
import { useStore } from "vuex";
import { useNamespacedGetters } from "vuex-composition-helpers";
import * as monaco from "monaco-editor";
import type { editor as Editor } from "monaco-editor";

import AutoCompletion from "./AutoCompletion";
import {
  ConnectionAtom,
  Table,
  CompletionItems,
  InstanceGetters,
  SqlDialect,
} from "../../types";
import sqlFormatter from "./sqlFormatter";

const useMonaco = async (lang: string) => {
  const store = useStore();

  const { instanceList } = useNamespacedGetters<InstanceGetters>("instance", [
    "instanceList",
  ]);

  const instances = computed(() => instanceList.value());

  const databases = computed(() => {
    const currentInstanceId =
      store.state.sqlEditor.connectionContext.instanceId;
    return store.getters["database/databaseListByInstanceId"](
      currentInstanceId
    );
  });

  const tables = computed(() => {
    return databases.value
      .map((item: ConnectionAtom) =>
        store.getters["table/tableListByDatabaseId"](item.id)
      )
      .flat();
  });

  monaco.languages.typescript.typescriptDefaults.setCompilerOptions({
    ...monaco.languages.typescript.typescriptDefaults.getCompilerOptions(),
    noUnusedLocals: false,
    noUnusedParameters: false,
    allowUnreachableCode: true,
    allowUnusedLabels: true,
    strict: false,
    allowJs: true,
  });

  const completionItemProvider =
    monaco.languages.registerCompletionItemProvider(lang, {
      triggerCharacters: [" ", "."],
      provideCompletionItems: (model, position) => {
        let suggestions: CompletionItems = [];

        const { lineNumber, column } = position;
        // The text before the cursor pointer
        const textBeforePointer = model.getValueInRange({
          startLineNumber: lineNumber,
          startColumn: 0,
          endLineNumber: lineNumber,
          endColumn: column,
        });
        // The multi-text before the cursor pointer
        const textBeforePointerMulti = model.getValueInRange({
          startLineNumber: 1,
          startColumn: 0,
          endLineNumber: lineNumber,
          endColumn: column,
        });
        // The text after the cursor pointer
        const textAfterPointerMulti = model.getValueInRange({
          startLineNumber: lineNumber,
          startColumn: column,
          endLineNumber: model.getLineCount(),
          endColumn: model.getLineMaxColumn(model.getLineCount()),
        });
        const tokens = textBeforePointer.trim().split(/\s+/);
        const lastToken = tokens[tokens.length - 1].toLowerCase();

        // console.group("completionItemProvider");
        // console.log(textBeforePointer);
        // console.log(textBeforePointerMulti);
        // console.log(textAfterPointerMulti);
        // console.log(tokens);
        // console.log(lastToken);
        // console.groupEnd();

        const autoCompletion = new AutoCompletion(
          model,
          position,
          instances.value,
          databases.value,
          tables.value
        );

        const suggestionsForKeywords =
          autoCompletion.getCompletionItemsForKeywords();

        const suggestionsForTables =
          autoCompletion.getCompletionItemsForTables();

        // if enter a dot, show table columns
        if (lastToken.endsWith(".")) {
          const tableName = lastToken.replace(".", "");
          const idx = tables.value.findIndex(
            (item: Table) => item.name === tableName
          );
          if (idx !== -1) {
            suggestions = autoCompletion.getCompletionItemsForTableColumns(
              tables.value[idx],
              false
            );
          }
        } else {
          suggestions = [...suggestionsForKeywords, ...suggestionsForTables];
        }

        return { suggestions };
      },
    });

  await Promise.all([
    // load workers
    (async () => {
      const [{ default: EditorWorker }] = await Promise.all([
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-expect-error
        import("monaco-editor/esm/vs/editor/editor.worker.js?worker"),
      ]);

      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      window.MonacoEnvironment = {
        getWorker(_: any, label: string) {
          return new EditorWorker();
        },
      };
    })(),
  ]);

  const setContent = (
    editorInstance: Editor.IStandaloneCodeEditor,
    content: string
  ) => {
    if (editorInstance) editorInstance.setValue(content);
  };

  const formatContent = (
    editorInstance: Editor.IStandaloneCodeEditor,
    language: SqlDialect
  ) => {
    if (editorInstance) {
      const sql = editorInstance.getValue();
      const { data } = sqlFormatter(sql, language);
      setContent(editorInstance, data);
    }
  };

  const setPositionAtEndOfLine = (
    editorInstance: Editor.IStandaloneCodeEditor
  ) => {
    if (editorInstance) {
      // eslint-disable-next-line @typescript-eslint/ban-ts-comment
      // @ts-expect-error
      const range = editorInstance.getModel().getFullModelRange();
      editorInstance.setPosition({
        lineNumber: range?.endLineNumber,
        column: range?.endColumn,
      });
    }
  };

  return {
    monaco,
    completionItemProvider,
    formatContent,
    setContent,
    setPositionAtEndOfLine,
  };
};

export { useMonaco };
