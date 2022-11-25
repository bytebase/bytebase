import {
  Column,
  AddColumnContext,
  Table,
  CreateTableContext,
  ChangeColumnContext,
} from "@/types";
import { defaultTo } from "lodash-es";

export const transformColumnToAddColumnContext = (
  column: Column
): AddColumnContext => {
  return {
    name: defaultTo(column.name, ""),
    type: defaultTo(column.type, ""),
    characterSet: defaultTo(column.characterSet, ""),
    collation: defaultTo(column.collation, ""),
    comment: defaultTo(column.comment, ""),
    nullable: defaultTo(column.nullable, false),
    default: defaultTo(column.default, undefined),
  };
};

export const transformColumnToChangeColumnContext = (
  originColumn: Column,
  column: Column
): ChangeColumnContext => {
  return {
    oldName: defaultTo(originColumn.name, ""),
    newName: defaultTo(column.name, ""),
    type: defaultTo(column.type, ""),
    characterSet: defaultTo(column.characterSet, ""),
    collation: defaultTo(column.collation, ""),
    comment: defaultTo(column.comment, ""),
    nullable: defaultTo(column.nullable, false),
    default: defaultTo(column.default, undefined),
  };
};

export const transformTableToCreateTableContext = (
  table: Table
): CreateTableContext => {
  return {
    name: defaultTo(table.name, ""),
    type: defaultTo(table.type, ""),
    engine: defaultTo(table.engine, ""),
    // As we don't have a CharacterSet field in table model,
    // set it as an empty string for now.
    characterSet: "",
    collation: defaultTo(table.collation, ""),
    comment: defaultTo(table.comment, ""),
    addColumnList: [],
  };
};
