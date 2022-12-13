import { cloneDeep, defaultTo } from "lodash-es";
import {
  AddColumnContext,
  CreateTableContext,
  ChangeColumnContext,
  Column as ColumnData,
  Table as TableData,
} from "@/types";
import { Column, Table } from "@/types/schemaEditor";

export const transformTableDataToTable = (tableData: TableData): Table => {
  const columnList = tableData.columnList.map((columnData) =>
    transformColumnDataToColumn(columnData)
  );

  return {
    databaseId: tableData.database.id,
    oldName: tableData.name,
    newName: tableData.name,
    type: tableData.type,
    engine: tableData.engine,
    collation: tableData.collation,
    rowCount: tableData.rowCount,
    dataSize: tableData.dataSize,
    comment: tableData.comment,
    originColumnList: columnList,
    columnList: cloneDeep(columnList),
    status: "normal",
  };
};

export const transformColumnDataToColumn = (columnData: ColumnData): Column => {
  return {
    databaseId: columnData.databaseId,
    oldName: columnData.name,
    newName: columnData.name,
    type: columnData.type,
    nullable: columnData.nullable,
    comment: columnData.comment,
    default: columnData.default || null,
    status: "normal",
  };
};

export const transformColumnToAddColumnContext = (
  column: Column
): AddColumnContext => {
  return {
    name: defaultTo(column.newName, ""),
    type: defaultTo(column.type, ""),
    comment: defaultTo(column.comment, ""),
    nullable: defaultTo(column.nullable, false),
    default: defaultTo(column.default, undefined),
    characterSet: "",
    collation: "",
  };
};

export const transformColumnToChangeColumnContext = (
  originColumn: Column,
  column: Column
): ChangeColumnContext => {
  return {
    oldName: defaultTo(originColumn.oldName, ""),
    newName: defaultTo(column.newName, ""),
    type: defaultTo(column.type, ""),
    comment: defaultTo(column.comment, ""),
    nullable: defaultTo(column.nullable, false),
    default: defaultTo(column.default, undefined),
    characterSet: "",
    collation: "",
  };
};

export const transformTableToCreateTableContext = (
  table: Table
): CreateTableContext => {
  return {
    name: defaultTo(table.newName, ""),
    type: defaultTo(table.type, ""),
    engine: defaultTo(table.engine, ""),
    collation: defaultTo(table.collation, ""),
    comment: defaultTo(table.comment, ""),
    addColumnList: [],
    // As we don't have a CharacterSet field in table model,
    // set it as an empty string for now.
    characterSet: "",
  };
};
