import { detailedDiff } from "deep-object-diff";
import {
  AlterTableContext,
  CreateTableContext,
  DropTableContext,
  Table,
} from "@/types";
import { transformTableToCreateTableContext } from "./transform";
import { diffColumnList } from "./diffColumn";
import { cloneDeep } from "lodash-es";

export const diffTableList = (
  originTableList: Table[],
  targetTableList: Table[]
) => {
  const diffResult = detailedDiff(
    cloneDeep(originTableList).map((table) => stringifyColumnList(table)),
    cloneDeep(targetTableList).map((table) => stringifyColumnList(table))
  );

  const addedTableIndexList = (Object.keys(diffResult.added) as string[]).map(
    (indexStr) => Number(indexStr)
  );
  const createTableList: CreateTableContext[] = [];
  for (const index of addedTableIndexList) {
    const table = targetTableList[index];
    const createTableContext = transformTableToCreateTableContext(table);
    const diffColumnListResult = diffColumnList([], table.columnList);
    createTableContext.addColumnList = diffColumnListResult.addColumnList;
    createTableList.push(createTableContext);
  }

  const updatedTableIndexList = (
    Object.keys(diffResult.updated) as string[]
  ).map((indexStr) => Number(indexStr));
  const alterTableList: AlterTableContext[] = [];
  for (const index of updatedTableIndexList) {
    const columnListDiffResult = diffColumnList(
      originTableList[index].columnList,
      targetTableList[index].columnList
    );
    alterTableList.push({
      name: targetTableList[index].name,
      ...columnListDiffResult,
    });
  }

  const deletedTableIndexList = (
    Object.keys(diffResult.deleted) as string[]
  ).map((indexStr) => Number(indexStr));
  const deletedTableList = deletedTableIndexList.map((index) => {
    return originTableList[index];
  });
  const dropTableList: DropTableContext[] = [];
  for (const table of deletedTableList) {
    dropTableList.push({
      name: table.name,
    });
  }

  return {
    createTableList,
    alterTableList,
    dropTableList,
  };
};

const stringifyColumnList = (table: Table) => {
  return {
    ...table,
    columnList: JSON.stringify(table.columnList),
  };
};
