import type { SubqueryNode } from "@joe-re/sql-parser";
import type { Table } from "@sql-lsp/types";
import type { SQLDialect } from "@/types";
import { getColumnsFromSelectStatement } from "./utils";

export class SubQueryMapping {
  tableList: Table[]; // not used yet
  subQueries: SubqueryNode[];
  dialect: SQLDialect; // not used yet
  virtualTableList: Table[];

  constructor(
    tableList: Table[],
    subQueries: SubqueryNode[],
    dialect: SQLDialect
  ) {
    this.tableList = tableList;
    this.subQueries = subQueries;
    this.dialect = dialect;

    this.virtualTableList = [];
    subQueries.forEach((clause) => {
      const { as } = clause;
      if (!as) {
        // If we met a sub query clause without alias,
        // we cannot map it to a virtual table.
        // So just skip it here.
        return;
      }
      const columns = getColumnsFromSelectStatement(clause.subquery);
      const virtualTable: Table = {
        database: undefined,
        name: as,
        columns,
      };
      this.virtualTableList.push(virtualTable);
    });
  }
}
