import { uniq } from "lodash-es";
import type { TableNode } from "@joe-re/sql-parser";
import type { SQLDialect, Table } from "@sql-lsp/types";
import { getTableNameFromTableNode } from "./utils";
import { createColumnCandidatesByAlias } from "./candidates";

export class AliasMapping {
  tableList: Table[];
  fromTable: TableNode[];
  dialect: SQLDialect;
  aliasMap: Map<string, Table[]>;

  constructor(
    tableList: Table[],
    fromTables: TableNode[],
    dialect: SQLDialect
  ) {
    this.tableList = tableList;
    this.fromTable = fromTables;
    this.dialect = dialect;

    this.aliasMap = new Map();
    fromTables.forEach((clause) => {
      const { as } = clause;
      if (!as) {
        // If we met a "from {table}" clause without alias, ignore it here.
        return;
      }

      // Otherwise, set up the mapping relationship between the alias and table(s).
      const { name, database } = getTableNameFromTableNode(
        clause,
        this.dialect
      );
      const list = this.aliasMap.get(as) ?? [];
      list.push(
        ...this.tableList.filter((t) => {
          if (database && t.database !== database) return false;
          return t.name === name;
        })
      );
      this.aliasMap.set(as, uniq(list));
    });
  }

  getTablesByAlias(alias: string): Table[] {
    return this.aliasMap.get(alias) ?? [];
  }

  createAllAliasCandidates() {
    return Array.from(this.aliasMap.entries()).flatMap(([alias, tables]) => {
      return createColumnCandidatesByAlias(alias, tables);
    });
  }
}
