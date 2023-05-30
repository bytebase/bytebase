import { isEqual, isUndefined } from "lodash-es";
import type {
  AddColumnContext,
  DropColumnContext,
  ChangeColumnContext,
  AlterColumnContext,
} from "@/types";
import { Column } from "@/types/schemaEditor/atomType";
import {
  transformColumnToAddColumnContext,
  transformColumnToChangeColumnContext,
} from "./transform";

// diffColumnList gets the differences between column object list.
// Including addColumnList, modifyColumnList and dropColumnList.
export const diffColumnList = (
  originColumnList: Column[],
  columnList: Column[]
) => {
  const addColumnContextList: AddColumnContext[] = [];
  const addedColumnList = columnList.filter(
    (column) => column.status === "created"
  );
  for (const column of addedColumnList) {
    addColumnContextList.push(transformColumnToAddColumnContext(column));
  }

  const alterColumnContextList: AlterColumnContext[] = [];
  const changeColumnContextList: ChangeColumnContext[] = [];
  const changedColumnList = columnList.filter(
    (column) => column.status === "normal"
  );
  for (const column of changedColumnList) {
    const originColumn = originColumnList.find(
      (originColumn) => originColumn.id === column.id
    );
    if (isUndefined(originColumn)) {
      continue;
    }
    if (!isEqual(originColumn, column)) {
      changeColumnContextList.push(
        transformColumnToChangeColumnContext(originColumn, column)
      );

      const alterColumnContext: AlterColumnContext = {
        oldName: originColumn.name,
        newName: column.name,
        defaultChanged: false,
      };
      if (!isEqual(originColumn.type, column.type)) {
        alterColumnContext.type = column.type;
      }
      if (!isEqual(originColumn.comment, column.comment)) {
        alterColumnContext.comment = column.comment;
      }
      if (!isEqual(originColumn.nullable, column.nullable)) {
        alterColumnContext.nullable = column.nullable;
      }
      if (!isEqual(originColumn.default, column.default)) {
        alterColumnContext.defaultChanged = true;
        alterColumnContext.default = column.default;
      }
      alterColumnContextList.push(alterColumnContext);
    }
  }

  const dropColumnContextList: DropColumnContext[] = [];
  const droppedColumnList = columnList.filter(
    (column) => column.status === "dropped"
  );
  for (const column of droppedColumnList) {
    const originColumn = originColumnList.find((item) => item.id === column.id);
    if (originColumn) {
      dropColumnContextList.push({
        name: originColumn.name,
      });
    }
  }

  return {
    addColumnList: addColumnContextList,
    alterColumnList: alterColumnContextList,
    changeColumnList: changeColumnContextList,
    dropColumnList: dropColumnContextList,
  };
};
