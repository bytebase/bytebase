import { pull, pullAt } from "lodash-es";
import {
  ForeignKeyMetadata,
  IndexMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { upsertArray } from "@/utils";

export const upsertColumnPrimaryKey = (
  table: TableMetadata,
  columnName: string
) => {
  const pkIndex = table.indexes.findIndex((idx) => idx.primary);
  if (pkIndex < 0) {
    table.indexes.push(
      IndexMetadata.fromPartial({
        primary: true,
        name: "PRIMARY",
        expressions: [columnName],
      })
    );
  } else {
    const pk = table.indexes[pkIndex];
    upsertArray(pk.expressions, columnName);
  }
};
export const removeColumnPrimaryKey = (
  table: TableMetadata,
  columnName: string
) => {
  const pkIndex = table.indexes.findIndex((idx) => idx.primary);
  if (pkIndex < 0) {
    return;
  }
  const pk = table.indexes[pkIndex];
  pull(pk.expressions, columnName);
  if (pk.expressions.length === 0) {
    pullAt(table.indexes, pkIndex);
  }
};
export const upsertColumnFromForeignKey = (
  fk: ForeignKeyMetadata,
  columnName: string,
  referencedColumnName: string
) => {
  const position = fk.columns.indexOf(columnName);
  if (position < 0) {
    fk.columns.push(columnName);
    fk.referencedColumns.push(referencedColumnName);
  } else {
    fk.referencedColumns[position] = referencedColumnName;
  }
};
export const removeColumnFromForeignKey = (
  table: TableMetadata,
  fk: ForeignKeyMetadata,
  columnName: string
) => {
  const position = fk.columns.indexOf(columnName);
  if (position < 0) {
    return;
  }
  pullAt(fk.columns, position);
  pullAt(fk.referencedColumns, position);
  if (fk.columns.length === 0) {
    const position = table.foreignKeys.findIndex((_fk) => _fk.name === fk.name);
    if (position >= 0) {
      pullAt(table.foreignKeys, position);
    }
  }
};
export const removeColumnFromAllForeignKeys = (
  table: TableMetadata,
  columnName: string
) => {
  for (let i = 0; i < table.foreignKeys.length; i++) {
    const fk = table.foreignKeys[i];
    const columnIndex = fk.columns.indexOf(columnName);
    if (columnIndex < 0) continue;
    pullAt(fk.columns, columnIndex);
    pullAt(fk.referencedColumns, columnIndex);
  }
  table.foreignKeys = table.foreignKeys.filter((fk) => fk.columns.length > 0);
};
