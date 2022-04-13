import { computed } from "vue";
import * as monaco from "monaco-editor";
import type { editor as Editor } from "monaco-editor";

import AutoCompletion from "./AutoCompletion";
import { Database, Table, CompletionItems, SqlDialect } from "../../types";
import sqlFormatter from "./sqlFormatter";
import {
  useDatabaseStore,
  useTableStore,
  useSQLEditorStore,
  useInstanceList,
} from "@/store";

const useMonaco = async (lang: string) => {
  const dataSourceStore = useDatabaseStore();
  const tableStore = useTableStore();
  const sqlEditorStore = useSQLEditorStore();
  const instanceList = useInstanceList();

  const databaseList = computed(() => {
    const currentInstanceId = sqlEditorStore.connectionContext.instanceId;
    return dataSourceStore.getDatabaseListByInstanceId(currentInstanceId);
  });

  const tableList = computed(() => {
    return databaseList.value
      .map((item) => tableStore.getTableListByDatabaseId(item.id))
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

        const autoCompletion = new AutoCompletion(
          model,
          position,
          instanceList.value,
          databaseList.value,
          tableList.value
        );

        // MySQL allows to query different databases, so we provide the database name suggestion for MySQL.
        const suggestionsForDatabase =
          lang === "mysql"
            ? autoCompletion.getCompletionItemsForDatabaseList()
            : [];

        const suggestionsForTable =
          autoCompletion.getCompletionItemsForTableList();

        const suggestionsForKeyword =
          autoCompletion.getCompletionItemsForKeywords();

        // if enter a dot
        if (lastToken.endsWith(".")) {
          /**
           * tokenLevel = 1 stands for the database.table or table.column
           * tokenLevel = 2 stands for the database.table.column
           */
          const tokenLevel = lastToken.split(".").length - 1;
          const lastTokenBeforeDot = lastToken.slice(0, -1);
          let [databaseName, tableName] = ["", ""];
          if (tokenLevel === 1) {
            databaseName = lastTokenBeforeDot;
            tableName = lastTokenBeforeDot;
          }
          if (tokenLevel === 2) {
            databaseName = lastTokenBeforeDot.split(".").shift() as string;
            tableName = lastTokenBeforeDot.split(".").pop() as string;
          }
          const dbIdx = databaseList.value.findIndex(
            (item: Database) => item.name === databaseName
          );
          const tableIdx = tableList.value.findIndex(
            (item: Table) => item.name === tableName
          );

          // if the last token is a database name
          if (lang === "mysql" && dbIdx !== -1 && tokenLevel === 1) {
            suggestions = autoCompletion.getCompletionItemsForTableList(
              databaseList.value[dbIdx],
              true
            );
          }
          // if the last token is a table name
          if (tableIdx !== -1 || tokenLevel === 2) {
            const table = tableList.value[tableIdx];
            if (table.columnList && table.columnList.length > 0) {
              suggestions = autoCompletion.getCompletionItemsForTableColumnList(
                tableList.value[tableIdx],
                false
              );
            }
          }
        } else {
          suggestions = [
            ...suggestionsForKeyword,
            ...suggestionsForTable,
            ...suggestionsForDatabase,
          ];
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

  /**
   * set new content in monaco editor
   * use executeEdits API can preserve undo stack, allow user to undo/redo
   * @param editorInstance Editor.IStandaloneCodeEditor
   * @param content string
   */
  const setContent = (
    editorInstance: Editor.IStandaloneCodeEditor,
    content: string
  ) => {
    if (editorInstance) {
      const range = editorInstance?.getModel()?.getFullModelRange();
      // get the current endLineNumber, or use 100000 as the default
      const endLineNumber =
        range && range?.endLineNumber > 0 ? range.endLineNumber + 1 : 100000;
      editorInstance.executeEdits("delete-content", [
        {
          range: new monaco.Range(1, 1, endLineNumber, 1),
          text: "",
          forceMoveMarkers: true,
        },
      ]);
      // set the new content
      editorInstance.executeEdits("insert-content", [
        {
          range: new monaco.Range(1, 1, 1, 1),
          text: content,
          forceMoveMarkers: true,
        },
      ]);
      // reset the selection
      editorInstance.setSelection(new monaco.Range(0, 0, 0, 0));
    }
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
