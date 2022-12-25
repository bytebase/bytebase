import { defaultTo } from "lodash-es";
import {
  AddColumnContext,
  CreateTableContext,
  ChangeColumnContext,
} from "@/types";
import { Column, Table } from "@/types/schemaEditor/atomType";

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
    primaryKeyList: [],
    addForeignKeyList: [],
    // As we don't have a CharacterSet field in table model,
    // set it as an empty string for now.
    characterSet: "",
  };
};
