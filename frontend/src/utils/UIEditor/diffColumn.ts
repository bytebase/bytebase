import { isEqual, isUndefined } from "lodash-es";
import {
  AddColumnContext,
  Column,
  DropColumnContext,
  ModifyColumnContext,
} from "@/types";
import { UNKNOWN_ID } from "@/types/const";
import { transformColumnToAddColumnContext } from "./transform";

// diffColumnList gets the differences between column object list.
// Including addColumnList, modifyColumnList and dropColumnList.
export const diffColumnList = (
  originColumnList: Column[],
  targetColumnList: Column[]
) => {
  const targetColumnIdList = targetColumnList.map((column) => column.id);

  const addColumnContextList: AddColumnContext[] = [];
  for (const column of targetColumnList) {
    if (column.id === UNKNOWN_ID) {
      addColumnContextList.push(transformColumnToAddColumnContext(column));
    }
  }

  const modifyColumnContextList: ModifyColumnContext[] = [];
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
      modifyColumnContextList.push(transformColumnToAddColumnContext(column));
    }
  }

  const dropColumnContextList: DropColumnContext[] = [];
  for (const column of originColumnList) {
    if (!targetColumnIdList.includes(column.id)) {
      dropColumnContextList.push({
        name: column.name,
      });
    }
  }

  return {
    addColumnList: addColumnContextList,
    modifyColumnList: modifyColumnContextList,
    dropColumnList: dropColumnContextList,
  };
};
