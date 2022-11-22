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

export const diffTableList = (
  originTableList: Table[],
  targetTableList: Table[]
) => {
  const targetTableIdList = targetTableList.map((table) => table.id);

  const createdTableList = targetTableList.filter(
    (table) => table.id === UNKNOWN_ID
  );
  const alteredTableList: Table[] = [];
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
      alteredTableList.push(table);
    }
  }
  const dropedTableList: Table[] = [];
  for (const table of originTableList) {
    if (!targetTableIdList.includes(table.id)) {
      dropedTableList.push(table);
    }
  }

  const createTableContextList: CreateTableContext[] = [];
  for (const table of createdTableList) {
    const createTableContext = transformTableToCreateTableContext(table);
    const diffColumnListResult = diffColumnList([], table.columnList);
    createTableContext.addColumnList = diffColumnListResult.addColumnList;
    createTableContextList.push(createTableContext);
  }

  const alterTableContextList: AlterTableContext[] = [];
  for (const table of alteredTableList) {
    const originTable = originTableList.find(
      (originTable) => originTable.id === table.id
    );
    if (isUndefined(originTable)) {
      continue;
    }

    const columnListDiffResult = diffColumnList(
      originTable.columnList,
      table.columnList
    );
    alterTableContextList.push({
      name: table.name,
      ...columnListDiffResult,
    });
  }

  const dropTableContextList: DropTableContext[] = [];
  for (const table of dropedTableList) {
    dropTableContextList.push({
      name: table.name,
    });
  }

  return {
    createTableList: createTableContextList,
    alterTableList: alterTableContextList,
    dropTableList: dropTableContextList,
  };
};
