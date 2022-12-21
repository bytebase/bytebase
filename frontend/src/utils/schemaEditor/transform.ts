import { cloneDeep, defaultTo } from "lodash-es";
import {
  AddColumnContext,
  CreateTableContext,
  ChangeColumnContext,
} from "@/types";
import { Column, Table } from "@/types/schemaEditor/atomType";
import { ColumnMetadata, TableMetadata } from "@/types/proto/database";

export const transformTableDataToTable = (
  tableMetadata: TableMetadata
): Table => {
  const columnList = tableMetadata.columns.map((column) =>
    transformColumnDataToColumn(column)
  );

  return {
    oldName: tableMetadata.name,
    newName: tableMetadata.name,
    engine: tableMetadata.engine,
    collation: tableMetadata.collation,
    rowCount: tableMetadata.rowCount,
    dataSize: tableMetadata.dataSize,
    comment: tableMetadata.comment,
    originColumnList: columnList,
    columnList: cloneDeep(columnList),
    status: "normal",
  };
};

export const transformColumnDataToColumn = (
  columnMetadata: ColumnMetadata
): Column => {
  return {
    oldName: columnMetadata.name,
    newName: columnMetadata.name,
    type: columnMetadata.type,
    nullable: columnMetadata.nullable,
    comment: columnMetadata.comment,
    default: columnMetadata.default,
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
    engine: defaultTo(table.engine, ""),
    collation: defaultTo(table.collation, ""),
    comment: defaultTo(table.comment, ""),
    addColumnList: [],
    // As we don't have a CharacterSet field in table model,
    // set it as an empty string for now.
    characterSet: "",
  };
};
