import { isEqual, isUndefined } from "lodash-es";
import {
  AddColumnContext,
  Column,
  DropColumnContext,
  ModifyColumnContext,
} from "@/types";
import { UNKNOWN_ID } from "@/types/const";
import { transformColumnToAddColumnContext } from "./transform";

export const diffColumnList = (
  originColumnList: Column[],
  targetColumnList: Column[]
) => {
  const targetColumnIdList = targetColumnList.map((column) => column.id);

  const addedColumnList = targetColumnList.filter(
    (column) => column.id === UNKNOWN_ID
  );
  const modifiedColumnList: Column[] = [];
  for (const column of targetColumnList) {
    if (column.id === UNKNOWN_ID) {
      continue;
    }
    const originColumn = originColumnList.find(
      (originColumn) => originColumn.id === column.id
    );
    if (isUndefined(originColumn)) {
      continue;
    }
    if (!isEqual(originColumn, column)) {
      modifiedColumnList.push(column);
    }
  }
  const dropedColumnList: Column[] = [];
  for (const column of originColumnList) {
    if (!targetColumnIdList.includes(column.id)) {
      dropedColumnList.push(column);
    }
  }

  const addColumnContextList: AddColumnContext[] = [];
  for (const column of addedColumnList) {
    addColumnContextList.push(transformColumnToAddColumnContext(column));
  }

  const modifyColumnContextList: ModifyColumnContext[] = [];
  for (const column of modifiedColumnList) {
    modifyColumnContextList.push(transformColumnToAddColumnContext(column));
  }

  const dropColumnContextList: DropColumnContext[] = [];
  for (const column of dropedColumnList) {
    dropColumnContextList.push({
      name: column.name,
    });
  }

  return {
    addColumnList: addColumnContextList,
    modifyColumnList: modifyColumnContextList,
    dropColumnList: dropColumnContextList,
  };
};
