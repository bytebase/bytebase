import type { Table, LanguageState } from "@sql-lsp/types";
import { TextDocument } from "vscode-languageserver-textdocument";
import type {
  CompletionItem,
  CompletionParams,
} from "vscode-languageserver/browser";
import { AliasMapping } from "./alias";
import {
  createColumnCandidates,
  createDatabaseCandidates,
  createKeywordCandidates,
  createSubQueryCandidates,
  createTableCandidates,
} from "./candidates";
import { SubQueryMapping } from "./sub-query";
import { simpleTokenize } from "./tokenizer";
import { getFromClauses, isDialectWithSchema } from "./utils";

export const complete = async (
  params: CompletionParams,
  document: TextDocument,
  state: LanguageState
): Promise<CompletionItem[]> => {
  const sql = document.getText();
  const textBeforeCursor = document.getText({
    start: { line: 0, character: 0 },
    end: params.position,
  });
  const { schema, dialect, connectionScope } = state;

  const tokens = simpleTokenize(textBeforeCursor);
  const lastToken = tokens[tokens.length - 1].toLowerCase();
  const tableList = schema.databases.flatMap((db) => db.tables);

  const { fromTables, subQueries } = getFromClauses(sql);
  const subQueryMapping = new SubQueryMapping(tableList, subQueries, dialect);
  const aliasMapping = new AliasMapping(tableList, fromTables, dialect);
  const withSchema = isDialectWithSchema(dialect);

  let suggestions: CompletionItem[] = [];

  // The auto-completion trigger is "."
  if (lastToken.endsWith(".") && lastToken !== ".") {
    const tokenListBeforeDot = lastToken
      .slice(0, -1)
      .split(".")
      .map((word) => word.replace(/[`'"]/g, "")); // remove quotes

    const provideTableAutoCompletion = (
      databaseName: string,
      tableList: Table[]
    ) => {
      // provide auto completion items for its tables
      const tableListOfDatabase = createTableCandidates(
        tableList.filter((table) => table.database === databaseName),
        false // without database prefix since it's already inputted
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
      tableList: Table[],
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
      if (!withSchema) {
        const maybeDatabaseName = tokenListBeforeDot[0];
        provideTableAutoCompletion(maybeDatabaseName, tableList);
      }
      // - "{table_name}." (mysql)
      const maybeTableName = tokenListBeforeDot[0];
      if (!withSchema) {
        provideColumnAutoCompletion(maybeTableName, tableList);
        provideColumnAutoCompletion(
          maybeTableName,
          subQueryMapping.virtualTableList
        );
      }
      if (withSchema) {
        // for postgresql, we also try "public.{table_name}."
        // since "public" schema can be omitted by default
        provideColumnAutoCompletion(`public.${maybeTableName}`, tableList);
      }
      // TODO: "{schema_name}." (postgresql)
    }

    if (tokenListBeforeDot.length === 2) {
      // if the input is "x.y." it might be
      // - "{database_name}.{table_name}." (mysql)
      // - "{schema_name}.{table_name}." (postgresql)
      const [maybeDatabaseName, maybeTableName] = tokenListBeforeDot;
      if (!withSchema) {
        provideColumnAutoCompletion(
          maybeTableName,
          tableList,
          maybeDatabaseName
        );
      }
      if (withSchema) {
        const maybeTableNameWithSchema = tokenListBeforeDot.join(".");
        provideColumnAutoCompletion(maybeTableNameWithSchema, tableList);
      }
      // TODO: "{database_name}.{schema_name}." (postgresql)
    }

    if (withSchema && tokenListBeforeDot.length === 3) {
      // if the input is "x.y.z." it might be
      // - "{database_name}.{schema_name}.{table_name}." (postgresql only)
      //   and bytebase save {schema_name}.{table_name} as the table name
      const [maybeDatabaseName, maybeSchemaName, maybeTableName] =
        tokenListBeforeDot;
      const maybeTableNameWithSchema = `${maybeSchemaName}.${maybeTableName}`;
      provideColumnAutoCompletion(
        maybeTableNameWithSchema,
        tableList,
        maybeDatabaseName
      );
    }
  } else {
    // The auto-completion trigger is SPACE
    // We didn't walk the AST, so still we don't know which type of
    // clause we are in. So we provide some naive suggestions.

    const suggestionsForAliases = aliasMapping.createAllAliasCandidates();

    const suggestionsForTable = createTableCandidates(
      tableList,
      // Add database prefix to table candidates when connection scope is 'instance'
      connectionScope === "instance"
    );
    const suggestionsForSubQueryVirtualTable = createSubQueryCandidates(
      subQueryMapping.virtualTableList
    );
    const suggestionsForKeyword = await createKeywordCandidates(dialect);

    suggestions = [
      ...suggestionsForAliases,
      ...suggestionsForKeyword,
      ...suggestionsForTable,
      ...suggestionsForSubQueryVirtualTable,
    ];
    if (connectionScope === "instance") {
      // Provide database suggestions only when we are connection to instance scope
      // MySQL allows to query different databases, so we provide the database name suggestion for MySQL.
      const suggestionsForDatabase = createDatabaseCandidates(schema.databases);
      suggestions.push(...suggestionsForDatabase);
    }
  }

  return suggestions;
};
