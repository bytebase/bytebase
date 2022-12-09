import { isEqual, isUndefined, omit } from "lodash-es";
import type {
  AlterTableContext,
  CreateTableContext,
  DropTableContext,
  RenameTableContext,
} from "@/types";
import { Table } from "@/types/UIEditor";
import { transformTableToCreateTableContext } from "./transform";
import { diffColumnList } from "./diffColumn";

// diffTableList gets the difference between table object list.
// Including createTableList, alterTableList and dropTableList.
// * createTableList: the table with UNKNOWN_ID in tableList will be considered as new table;
// * alterTableList: table with the same id but not equal in originTableList and tableList;
// * dropTableList: table is in originTableList instead of tableList.
export const diffTableList = (originTableList: Table[], tableList: Table[]) => {
  const createTableContextList: CreateTableContext[] = [];
  const createdTableList = tableList.filter(
    (table) => table.status === "created"
  );
  for (const table of createdTableList) {
    const createTableContext = transformTableToCreateTableContext(table);
    const diffColumnListResult = diffColumnList([], table.columnList);
    createTableContext.addColumnList = diffColumnListResult.addColumnList;
    createTableContextList.push(createTableContext);
  }

  const alterTableContextList: AlterTableContext[] = [];
  const renameTableContextList: RenameTableContext[] = [];
  const changedTableList = tableList.filter(
    (table) => table.status === undefined
  );
  for (const table of changedTableList) {
    const originTable = originTableList.find(
      (originTable) =>
        originTable.databaseId === table.databaseId &&
        originTable.oldName === table.oldName
    );
    if (isUndefined(originTable)) {
      continue;
    }

    if (!isEqual(omit(originTable, "status"), omit(table, "status"))) {
      if (table.newName !== table.oldName) {
        renameTableContextList.push({
          oldName: table.oldName,
          newName: table.newName,
        });
      }

      const columnListDiffResult = diffColumnList(
        originTable.columnList,
        table.columnList
      );
      if (
        columnListDiffResult.addColumnList.length > 0 ||
        columnListDiffResult.changeColumnList.length > 0 ||
        columnListDiffResult.dropColumnList.length > 0
      ) {
        alterTableContextList.push({
          name: originTable.newName,
          ...columnListDiffResult,
        });
      }
    }
  }

  const dropTableContextList: DropTableContext[] = [];
  const droppedTableList = tableList.filter(
    (table) => table.status === "dropped"
  );
  for (const table of droppedTableList) {
    dropTableContextList.push({
      name: table.oldName,
    });
  }

  return {
    createTableList: createTableContextList,
    alterTableList: alterTableContextList,
    renameTableList: renameTableContextList,
    dropTableList: dropTableContextList,
  };
};
