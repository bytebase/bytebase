import { detailedDiff } from "deep-object-diff";
import { cloneDeep } from "lodash-es";
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
  const diffResult = detailedDiff(
    cloneDeep(originColumnList),
    cloneDeep(targetColumnList)
  );

  const

  const addColumnList: AddColumnContext[] = [];
  const addedColumnList = targetColumnList.filter(
    (column) => column.id === UNKNOWN_ID
  );
  for (const column of addedColumnList) {
    addColumnList.push(transformColumnToAddColumnContext(column));
  }

  const updatedColumnIndexList = (
    Object.keys(diffResult.updated) as string[]
  ).map((indexStr) => Number(indexStr));
  const updatedColumnList = updatedColumnIndexList.map((index) => {
    return {
      ...originColumnList[index],
      ...(diffResult.updated as any)[`${index}`],
    };
  });
  const modifyColumnList: ModifyColumnContext[] = [];
  for (const column of updatedColumnList) {
    modifyColumnList.push(transformColumnToAddColumnContext(column));
  }

  const deletedColumnIndexList = (
    Object.keys(diffResult.deleted) as string[]
  ).map((indexStr) => Number(indexStr));
  const deletedColumnList = deletedColumnIndexList.map((index) => {
    return originColumnList[index];
  });
  const dropColumnList: DropColumnContext[] = [];
  for (const column of deletedColumnList) {
    dropColumnList.push({
      name: column.name,
    });
  }

  return {
    addColumnList,
    modifyColumnList,
    dropColumnList,
  };
};
