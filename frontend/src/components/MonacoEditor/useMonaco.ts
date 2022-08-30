import { ref } from "vue";
import { uniqBy } from "lodash-es";

import type { editor as Editor } from "monaco-editor";
import { Database, Table, CompletionItems, SQLDialect } from "@/types";
import AutoCompletion from "./AutoCompletion";
import sqlFormatter from "./sqlFormatter";
import { ExtractPromiseType } from "@/utils";

export const useMonaco = async (defaultDialect: SQLDialect) => {
  const monaco = await import("monaco-editor");

  const dialect = ref(defaultDialect);

  monaco.editor.defineTheme("bb-sql-editor-theme", {
    base: "vs",
    inherit: true,
    rules: [],
    colors: {
      "editorCursor.foreground": "#504de2",
      "editorLineNumber.foreground": "#aaaaaa",
      "editorLineNumber.activeForeground": "#111111",
    },
  });
  monaco.editor.setTheme("bb-sql-editor-theme");
  const databaseList = ref<Database[]>([]);
  const tableList = ref<Table[]>([]);

  const completionItemProvider =
    monaco.languages.registerCompletionItemProvider(
      ["sql", "mysql", "postgresql"],
      {
        triggerCharacters: [" ", "."],
        provideCompletionItems: async (model, position) => {
          let suggestions: CompletionItems = [];

          const { lineNumber, column } = position;
          // The text before the cursor pointer
          const textBeforePointer = model.getValueInRange({
            startLineNumber: lineNumber,
            startColumn: 0,
            endLineNumber: lineNumber,
            endColumn: column,
          });
          const tokens = textBeforePointer.trim().split(/\s+/);
          const lastToken = tokens[tokens.length - 1].toLowerCase();

          const autoCompletion = new AutoCompletion(
            model,
            position,
            databaseList.value,
            tableList.value
          );

          // The auto-completion trigger is "."
          if (lastToken.endsWith(".") && lastToken !== ".") {
            const tokenListBeforeDot = lastToken
              .slice(0, -1)
              .split(".")
              .map((word) => word.replace(/[`'"]/g, "")); // remove quotes

            const provideTableAutoCompletion = async (databaseName: string) => {
              const database = databaseList.value.find(
                (db) => db.name === databaseName
              );
              if (database) {
                // provide auto completion items for its tables
                const tableListOfDatabase =
                  await autoCompletion.getCompletionItemsForTableList(
                    database,
                    false // without database prefix since it's already inputted
                  );
                suggestions.push(...tableListOfDatabase);
              }
            };

            const provideColumnAutoCompletion = async (
              tableName: string,
              databaseName?: string
            ) => {
              const tables = tableList.value.filter((table) => {
                if (databaseName && table.database.name !== databaseName) {
                  return false;
                }
                return table.name === tableName;
              });
              // provide auto completion items for table columns
              for (const table of tables) {
                const columnListOfTable =
                  await autoCompletion.getCompletionItemsForTableColumnList(
                    table,
                    false // without table prefix since it's already inputted
                  );
                suggestions.push(...columnListOfTable);
              }
            };

            if (tokenListBeforeDot.length === 1) {
              // if the input is "x." x might be a
              // - "{database_name}." (mysql)
              if (dialect.value === "mysql") {
                const maybeDatabaseName = tokenListBeforeDot[0];
                await provideTableAutoCompletion(maybeDatabaseName);
              }
              // - "{table_name}." (mysql)
              const maybeTableName = tokenListBeforeDot[0];
              if (dialect.value === "mysql") {
                await provideColumnAutoCompletion(maybeTableName);
              }
              if (dialect.value === "postgresql") {
                // for postgresql, we also try "public.{table_name}."
                // since "public" schema can be omitted by default
                await provideColumnAutoCompletion(`public.${maybeTableName}`);
              }
              // "{schema_name}." (postgresql) - will implement next time
              // - alias (can not recognize yet)
            }

            if (tokenListBeforeDot.length === 2) {
              // if the input is "x.y." it might be
              // - "{database_name}.{table_name}." (mysql)
              // - "{schema_name}.{table_name}." (postgresql)
              const [maybeDatabaseName, maybeTableName] = tokenListBeforeDot;
              if (dialect.value === "mysql") {
                await provideColumnAutoCompletion(
                  maybeTableName,
                  maybeDatabaseName
                );
              }
              if (dialect.value === "postgresql") {
                const maybeTableNameWithSchema = tokenListBeforeDot.join(".");
                await provideColumnAutoCompletion(maybeTableNameWithSchema);
              }
              // "{database_name}.{schema_name}." (postgresql) - will implement next time
            }

            if (
              dialect.value === "postgresql" &&
              tokenListBeforeDot.length === 3
            ) {
              // if the input is "x.y.z." it might be
              // - "{database_name}.{schema_name}.{table_name}." (postgresql only)
              //   and bytebase save {schema_name}.{table_name} as the table name
              const [maybeDatabaseName, maybeSchemaName, maybeTableName] =
                tokenListBeforeDot;
              const maybeTableNameWithSchema = `${maybeSchemaName}.${maybeTableName}`;
              await provideColumnAutoCompletion(
                maybeTableNameWithSchema,
                maybeDatabaseName
              );
            }
          } else {
            // The auto-completion trigger is SPACE
            // We didn't walk the AST, so still we don't know which type of
            // clause we are in. So we provide some naive suggestions.

            // MySQL allows to query different databases, so we provide the database name suggestion for MySQL.
            const suggestionsForDatabase =
              dialect.value === "mysql"
                ? await autoCompletion.getCompletionItemsForDatabaseList()
                : [];
            const suggestionsForTable =
              await autoCompletion.getCompletionItemsForTableList();
            const suggestionsForKeyword =
              await autoCompletion.getCompletionItemsForKeywords();

            suggestions = [
              ...suggestionsForKeyword,
              ...suggestionsForTable,
              ...suggestionsForDatabase,
            ];
          }

          return {
            suggestions: uniqBy(suggestions, "label"),
          };
        },
      }
    );

  await Promise.all([
    // load workers
    (async () => {
      const [{ default: EditorWorker }] = await Promise.all([
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

  const dispose = () => {
    completionItemProvider.dispose();
  };

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
    const range = editorInstance.getModel()?.getFullModelRange();
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
  };

  const formatContent = (
    editorInstance: Editor.IStandaloneCodeEditor,
    dialect: SQLDialect
  ) => {
    const sql = editorInstance.getValue();
    const { data } = sqlFormatter(sql, dialect);
    setContent(editorInstance, data);
  };

  const setPositionAtEndOfLine = (
    editorInstance: Editor.IStandaloneCodeEditor
  ) => {
    const range = editorInstance.getModel()?.getFullModelRange();
    if (range) {
      editorInstance.setPosition({
        lineNumber: range?.endLineNumber,
        column: range?.endColumn,
      });
    }
  };

  const setAutoCompletionContext = (databases: Database[], tables: Table[]) => {
    databaseList.value = databases;
    tableList.value = tables;
  };

  const setDialect = (newDialect: SQLDialect) => {
    dialect.value = newDialect;
  };

  return {
    dispose,
    monaco,
    setContent,
    formatContent,
    setAutoCompletionContext,
    setDialect,
    setPositionAtEndOfLine,
  };
};

export type MonacoHelper = ExtractPromiseType<ReturnType<typeof useMonaco>>;
