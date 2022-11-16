import { detailedDiff } from "deep-object-diff";
import { cloneDeep } from "lodash-es";
import { CreateTableContext, Table } from "@/types";
import {
  transformColumnToAddColumnContext,
  transformTableToCreateTableContext,
} from "./transform";

export const diffTableList = (
  originTableList: Table[],
  updatedTableList: Table[]
) => {
  const diffResult = detailedDiff(
    cloneDeep(originTableList),
    cloneDeep(updatedTableList)
  );

  const createTableList: CreateTableContext[] = [];
  const addedTableList = Object.values(diffResult.added) as Table[];
  for (const table of addedTableList) {
    const createTableContext = transformTableToCreateTableContext(table);
    const addColumnList = table.columnList.map((column) =>
      transformColumnToAddColumnContext(column)
    );
    createTableContext.addColumnList = addColumnList;
    createTableList.push(createTableContext);
  }
  // TODO(Steven): Do the deleted/updated table list checks later.

  return {
    createTableList,
  };
};
