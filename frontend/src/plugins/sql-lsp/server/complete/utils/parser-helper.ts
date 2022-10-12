import {
  IncompleteSubqueryNode,
  parseFromClause,
  SubqueryNode,
  TableNode,
} from "@joe-re/sql-parser";
import { SQLDialect, Table } from "@/plugins/sql-lsp/types";

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
  } catch (ex) {
    // No valid from clauses found. Give up.
  }
  return { subQueries, incompleteSubQueries, fromTables };
};

export const getTableNameFromTableNode = (
  clause: TableNode,
  dialect: SQLDialect = "mysql"
): Pick<Table, "database" | "name"> => {
  const { table, db, catalog } = clause;
  if (db && catalog) {
    // format: "x.y.z"
    // The parser recognizes x and store it to the `catalog` field.
    if (dialect === "mysql") {
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
    if (dialect === "mysql") {
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
