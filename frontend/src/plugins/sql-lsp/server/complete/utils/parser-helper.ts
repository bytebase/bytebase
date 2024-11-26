import type {
  ColumnListItemNode,
  IncompleteSubqueryNode,
  SelectStatement,
  SubqueryNode,
  TableNode,
} from "@joe-re/sql-parser";
import { parseFromClause } from "@joe-re/sql-parser";
import type { Column, Table } from "@/plugins/sql-lsp/types";
import type { SQLDialect } from "@/types";
import { isDialectWithSchema } from "./common";

export const getFromClauses = (sql: string) => {
  const fromTables: TableNode[] = [];
  const subQueries: SubqueryNode[] = [];
  const incompleteSubQueries: IncompleteSubqueryNode[] = [];
  try {
    const parseResult = parseFromClause(sql);
    const clauses = parseResult.from?.tables ?? [];
    clauses.forEach((clause) => {
      switch (clause.type) {
        case "table":
          fromTables.push(clause);
          break;
        case "subquery":
          subQueries.push(clause); // not used yet
          break;
        case "incomplete_subquery":
          incompleteSubQueries.push(clause); // not used yet
          break;
      }
    });
  } catch {
    // No valid from clauses found. Give up.
  }
  return { subQueries, incompleteSubQueries, fromTables };
};

export const getTableNameFromTableNode = (
  clause: TableNode,
  dialect: SQLDialect = "MYSQL"
): Pick<Table, "database" | "name"> => {
  const { table, db, catalog } = clause;
  const withSchema = isDialectWithSchema(dialect);
  if (db && catalog) {
    // format: "x.y.z"
    // The parser recognizes x and store it to the `catalog` field.
    if (!withSchema) {
      // should be "{catalog}.{database}.{table}"
      // but we don't support mysql catalog by now, so just ignore it.
      return { name: table, database: db };
    } else {
      // should be "{database}.{schema}.{table}"
      const schema = db;
      const name = `${schema}.${table}`;
      const database = catalog;
      return { name, database };
    }
  } else if (db) {
    // format: "x.y"
    if (!withSchema) {
      // should be "{database}.{table}"
      return { name: table, database: db };
    } else {
      // should be "{schema}.{table}"
      const schema = db;
      const name = `${schema}.${table}`;
      return { name, database: undefined };
    }
  }
  // format: "x"
  // should be "{table}"
  return { name: table, database: undefined };
};

export const getColumnsFromSelectStatement = (
  select: SelectStatement
): Column[] => {
  const selectedColumns = select.columns;
  if (Array.isArray(selectedColumns)) {
    // e.g. "SELECT a, b, c FROM xxx"
    // parse the selected column nodes one-by-one
    return selectedColumns
      .map((columnNode) => getColumnNameFromColumnListItemNode(columnNode))
      .filter((columnName) => columnName !== "")
      .map((name) => ({ name }));
  } else if (selectedColumns.type === "star") {
    // e.g. "SELECT * FROM xxx"
    // --
    // TODO: "xxx" might be complex, we need to parse the clause recursively,
    // and expand "SELECT *" as if SELECT every column one-by-one
    return [];
  }
  // And the parser cannot handle mixed select star and columns yet.
  // e.g. SELECT a.*, b.col FROM a, b;
  // We will get nothing here.

  return [];
};

export const getColumnNameFromColumnListItemNode = (
  node: ColumnListItemNode
): string => {
  const { as, expr } = node;
  if (expr.type === "column_ref") {
    // If the node is a column_ref, use the alias (if available) or the column name
    return as || expr.column || "";
  } else if (expr.type === "aggr_func") {
    // e.g. "MAX(hire_date) as latest_date"
    // We cannot reference a aggr_func directly without alias
    return as || "";
  }
  return "";
};
