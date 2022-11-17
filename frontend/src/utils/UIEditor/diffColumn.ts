import { detailedDiff } from "deep-object-diff";
import { cloneDeep } from "lodash-es";
import { AddColumnContext, Column } from "@/types";
import { transformColumnToAddColumnContext } from "./transform";

export const diffColumnList = (
  originColumnList: Column[],
  updatedColumnList: Column[]
) => {
  const diffResult = detailedDiff(
    cloneDeep(originColumnList),
    cloneDeep(updatedColumnList)
  );

  const addColumnList: AddColumnContext[] = [];
  const addedColumnList = Object.values(diffResult.added) as Column[];
  for (const column of addedColumnList) {
    addColumnList.push(transformColumnToAddColumnContext(column));
  }
  // TODO(Steven): Do the deleted/updated column list checks later.

  return {
    addColumnList,
  };
};
