import { MD5 } from "crypto-js";
import { pull, pullAt } from "lodash-es";
import { create } from "@bufbuild/protobuf";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  ForeignKeyMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { IndexMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { getFixedPrimaryKey, upsertArray } from "@/utils";

export const upsertColumnPrimaryKey = (
  engine: Engine,
  table: TableMetadata,
  columnName: string
) => {
  const pkIndex = table.indexes.findIndex((idx) => idx.primary);
  if (pkIndex < 0) {
    let name = getFixedPrimaryKey(engine);
    // If no fixed primary key, generate a unique name.
    if (!name) {
      // For Postgres, constraint name must be unique within the schema.
      // Format: table_pk_{md5(table_pk_timestamp).slice(0, 6)}, e.g. test_pk_d4402d
      const nameParts: string[] = [table.name, "pk"];
      const rawName = nameParts.join("_").toLowerCase();
      name = `${rawName}_${MD5(`${rawName}_${Date.now()}`).toString().slice(0, 6)}`;
    }
    table.indexes.push(
      create(IndexMetadataSchema, {
        name,
        primary: true,
        unique: true,
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
