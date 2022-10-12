import { TextDocument } from "vscode-languageserver-textdocument";
import type {
  CompletionItem,
  CompletionParams,
} from "vscode-languageserver/browser";
import type { SQLDialect, Schema } from "@sql-lsp/types";
import {
  createColumnCandidates,
  createDatabaseCandidates,
  createKeywordCandidates,
  createTableCandidates,
} from "./candidates";
import { getFromClauses } from "./utils";
import { simpleTokenize } from "./tokenizer";
import { AliasMapping } from "./alias";

export const complete = (
  params: CompletionParams,
  document: TextDocument,
  schema: Schema,
  dialect: SQLDialect
): CompletionItem[] => {
  const sql = document.getText();
  const textBeforeCursor = document.getText({
    start: { line: 0, character: 0 },
    end: params.position,
  });

  const tokens = simpleTokenize(textBeforeCursor);
  const lastToken = tokens[tokens.length - 1].toLowerCase();
  const tableList = schema.databases.flatMap((db) => db.tables);

  const { fromTables } = getFromClauses(sql);
  const aliasMapping = new AliasMapping(tableList, fromTables, dialect);

  let suggestions: CompletionItem[] = [];

  // The auto-completion trigger is "."
  if (lastToken.endsWith(".") && lastToken !== ".") {
    const tokenListBeforeDot = lastToken
      .slice(0, -1)
      .split(".")
      .map((word) => word.replace(/[`'"]/g, "")); // remove quotes

    const provideTableAutoCompletion = (databaseName: string) => {
      // provide auto completion items for its tables
      const tableListOfDatabase = createTableCandidates(
        tableList.filter((table) => table.database === databaseName),
        false //  // without database prefix since it's already inputted
      );
      suggestions.push(...tableListOfDatabase);
    };

    const provideColumnAutoCompletionByAlias = (alias: string) => {
      const tables = aliasMapping.getTablesByAlias(alias);
      // provide auto completion items for table columns with table alias
      for (const table of tables) {
        const columnListOfTable = createColumnCandidates(table, false);
        suggestions.push(...columnListOfTable);
      }
    };

    const provideColumnAutoCompletion = (
      tableName: string,
      databaseName?: string
    ) => {
      const tables = tableList.filter((table) => {
        if (databaseName && table.database !== databaseName) {
          return false;
        }
        return table.name === tableName;
      });
      // provide auto completion items for table columns
      for (const table of tables) {
        const columnListOfTable = createColumnCandidates(table, false);
        suggestions.push(...columnListOfTable);
      }
    };

    if (tokenListBeforeDot.length === 1) {
      // if the input is "x." x might be a

      // - "{alias}." (mysql/postgresql)
      const maybeAlias = tokenListBeforeDot[0];
      provideColumnAutoCompletionByAlias(maybeAlias);

      // - "{database_name}." (mysql)
      if (dialect === "mysql") {
        const maybeDatabaseName = tokenListBeforeDot[0];
        provideTableAutoCompletion(maybeDatabaseName);
      }
      // - "{table_name}." (mysql)
      const maybeTableName = tokenListBeforeDot[0];
      if (dialect === "mysql") {
        provideColumnAutoCompletion(maybeTableName);
      }
      if (dialect === "postgresql") {
        // for postgresql, we also try "public.{table_name}."
        // since "public" schema can be omitted by default
        provideColumnAutoCompletion(`public.${maybeTableName}`);
      }
      // TODO: "{schema_name}." (postgresql)
    }

    if (tokenListBeforeDot.length === 2) {
      // if the input is "x.y." it might be
      // - "{database_name}.{table_name}." (mysql)
      // - "{schema_name}.{table_name}." (postgresql)
      const [maybeDatabaseName, maybeTableName] = tokenListBeforeDot;
      if (dialect === "mysql") {
        provideColumnAutoCompletion(maybeTableName, maybeDatabaseName);
      }
      if (dialect === "postgresql") {
        const maybeTableNameWithSchema = tokenListBeforeDot.join(".");
        provideColumnAutoCompletion(maybeTableNameWithSchema);
      }
      // TODO: "{database_name}.{schema_name}." (postgresql)
    }

    if (dialect === "postgresql" && tokenListBeforeDot.length === 3) {
      // if the input is "x.y.z." it might be
      // - "{database_name}.{schema_name}.{table_name}." (postgresql only)
      //   and bytebase save {schema_name}.{table_name} as the table name
      const [maybeDatabaseName, maybeSchemaName, maybeTableName] =
        tokenListBeforeDot;
      const maybeTableNameWithSchema = `${maybeSchemaName}.${maybeTableName}`;
      provideColumnAutoCompletion(maybeTableNameWithSchema, maybeDatabaseName);
    }
  } else {
    // The auto-completion trigger is SPACE
    // We didn't walk the AST, so still we don't know which type of
    // clause we are in. So we provide some naive suggestions.

    const suggestionsForAliases = aliasMapping.createAllAliasCandidates();

    // MySQL allows to query different databases, so we provide the database name suggestion for MySQL.
    const suggestionsForDatabase =
      dialect === "mysql" ? createDatabaseCandidates(schema.databases) : [];
    const suggestionsForTable = createTableCandidates(
      tableList,
      dialect === "mysql" // Add database prefix to table candidates only for MySQL.
    );
    const suggestionsForKeyword = createKeywordCandidates();

    suggestions = [
      ...suggestionsForAliases,
      ...suggestionsForKeyword,
      ...suggestionsForTable,
      ...suggestionsForDatabase,
    ];
  }

  return suggestions;
};
