import { isEqual, isUndefined } from "lodash-es";
import {
  AlterTableContext,
  CreateTableContext,
  DropTableContext,
  Table,
} from "@/types";
import { UNKNOWN_ID } from "@/types/const";
import { transformTableToCreateTableContext } from "./transform";
import { diffColumnList } from "./diffColumn";

// diffTableList gets the difference between table object list.
// Including createTableList, alterTableList and dropTableList.
// * createTableList: the table with UNKNOWN_ID in targetTableList will be considered as new table;
// * alterTableList: table with the same id but not equal in originTableList and targetTableList;
// * dropTableList: table is in originTableList instead of targetTableList.
export const diffTableList = (
  originTableList: Table[],
  targetTableList: Table[]
) => {
  const targetTableIdList = targetTableList.map((table) => table.id);

  const createTableContextList: CreateTableContext[] = [];
  for (const table of targetTableList) {
    if (table.id === UNKNOWN_ID) {
      const createTableContext = transformTableToCreateTableContext(table);
      const diffColumnListResult = diffColumnList([], table.columnList);
      createTableContext.addColumnList = diffColumnListResult.addColumnList;
      createTableContextList.push(createTableContext);
    }
  }

  const alterTableContextList: AlterTableContext[] = [];
  for (const table of targetTableList) {
    if (table.id === UNKNOWN_ID) {
      continue;
    }
    const originTable = originTableList.find(
      (originTable) => originTable.id === table.id
    );
    if (isUndefined(originTable)) {
      continue;
    }
    if (!isEqual(originTable, table)) {
      const columnListDiffResult = diffColumnList(
        originTable.columnList,
        table.columnList
      );
      alterTableContextList.push({
        name: table.name,
        ...columnListDiffResult,
      });
    }
  }

  const dropTableContextList: DropTableContext[] = [];
  for (const table of originTableList) {
    if (!targetTableIdList.includes(table.id)) {
      dropTableContextList.push({
        name: table.name,
      });
    }
  }

  return {
    createTableList: createTableContextList,
    alterTableList: alterTableContextList,
    dropTableList: dropTableContextList,
  };
};
