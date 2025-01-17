import { orderBy } from "lodash-es";
import type { SQLResultSetV1 } from "@/types";
import { NullValue } from "@/types/proto/google/protobuf/struct";
import { Engine } from "@/types/proto/v1/common";
import { QueryRow, RowValue } from "@/types/proto/v1/sql_service";

type NoSQLRowData = {
  Key: string;
  Value: string | number | object | any;
};

const instanceOfNoSQLRowData = (data: any): data is NoSQLRowData => {
  return (
    typeof data === "object" &&
    Object.keys(data).length === 2 &&
    "Key" in data &&
    "Value" in data
  );
};

const isNoSQLRowArray = (data: object): boolean => {
  return Array.isArray(data) && data.every(instanceOfNoSQLRowData);
};

interface CosmosDBRawData {
  [key: string]: string | number | object | any;
}

const flattenNoSQLColumn = (data: any): any => {
  if (!Array.isArray(data)) {
    return data;
  }
  if (isNoSQLRowArray(data)) {
    const result: { [key: string]: any } = {};
    for (const row of data as NoSQLRowData[]) {
      result[row.Key] = flattenNoSQLColumn(row.Value);
    }
    return result;
  }

  const result: any[] = [];
  for (const item of data as any[]) {
    if (Array.isArray(item)) {
      result.push(flattenNoSQLColumn(item));
    } else {
      result.push(item);
    }
  }

  return result;
};

export const flattenNoSQLResult = (
  engine: Engine,
  resultSet: SQLResultSetV1
) => {
  for (const result of resultSet.results) {
    const { columns, columnIndexMap } = getNoSQLColumns(engine, result.rows);

    const rows: QueryRow[] = [];
    const columnTypeNames: string[] = Array.from({
      length: columns.length,
    }).map((_) => "TEXT");

    for (const row of result.rows) {
      let parsedRows: NoSQLRowData[] | undefined = undefined;
      switch (engine) {
        case Engine.MONGODB:
          parsedRows = getMongoDBRows(row);
          break;
        case Engine.COSMOSDB:
          parsedRows = getCosmosDBRows(row);
          break;
      }
      if (!parsedRows) {
        continue;
      }

      const values: RowValue[] = Array.from({ length: columns.length }).map(
        (_) =>
          RowValue.fromPartial({
            nullValue: NullValue.NULL_VALUE,
          })
      );
      for (const rawData of parsedRows) {
        const index = columnIndexMap.get(rawData.Key) ?? 0;
        switch (typeof rawData.Value) {
          case "object":
            const value = flattenNoSQLColumn(rawData.Value);
            values[index] = RowValue.fromPartial({
              stringValue: JSON.stringify(value),
            });
            columnTypeNames[index] = "OBJECT";
            break;
          case "number":
            values[index] = RowValue.fromPartial({
              int64Value: rawData.Value,
            });
            columnTypeNames[index] = "INTEGER";
            break;
          default:
            values[index] = RowValue.fromPartial({
              stringValue: rawData.Value,
            });
            columnTypeNames[index] = "TEXT";
            break;
        }
      }

      rows.push(
        QueryRow.fromPartial({
          values: values,
        })
      );
    }

    result.rows = rows;
    result.columnNames = columns;
    result.columnTypeNames = columnTypeNames;
  }
};

const getNoSQLColumns = (
  engine: Engine,
  rows: QueryRow[]
): {
  columns: string[];
  columnIndexMap: Map<string, number>;
} => {
  switch (engine) {
    case Engine.MONGODB:
      return getMongoDBColumns(rows);
    case Engine.COSMOSDB:
      return getCosmosDBColumns(rows);
    default:
      return {
        columns: [],
        columnIndexMap: new Map(),
      };
  }
};

const getMongoDBColumns = (rows: QueryRow[]) => {
  const columnSet = new Set<string>();
  const columnIndexMap = new Map<string, number>();

  for (const row of rows) {
    const parsedRows = getMongoDBRows(row);
    if (!parsedRows) {
      continue;
    }
    for (const rawData of parsedRows) {
      columnSet.add(rawData.Key);
    }
  }

  const sortedColumns = orderBy(
    [...columnSet],
    (column) => (column === "_id" ? -1 : column),
    "asc"
  );

  for (let i = 0; i < sortedColumns.length; i++) {
    columnIndexMap.set(sortedColumns[i], i);
  }

  return {
    columns: sortedColumns,
    columnIndexMap,
  };
};

const getCosmosDBColumns = (rows: QueryRow[]) => {
  const columnSet = new Set<string>();
  const columnIndexMap = new Map<string, number>();

  for (const row of rows) {
    const parsedRows = getCosmosDBRows(row);
    if (!parsedRows) {
      continue;
    }
    for (const item of parsedRows) {
      columnSet.add(item.Key);
    }
  }

  const builtInColumns = new Map<string, number>([
    ["_rid", 0],
    ["_self", 1],
    ["_etag", 2],
    ["_attachments", 3],
    ["_ts", 4],
  ]);

  const sortedColumns = orderBy(
    [...columnSet],
    [
      (column) => (column === "id" ? -1 : 1),
      (column) => (builtInColumns.has(column) ? 1 : 0),
      (column) => builtInColumns.get(column) ?? 0,
    ],
    ["asc", "asc"]
  );

  for (let i = 0; i < sortedColumns.length; i++) {
    columnIndexMap.set(sortedColumns[i], i);
  }

  return {
    columns: sortedColumns,
    columnIndexMap,
  };
};

const getMongoDBRows = (row: QueryRow): NoSQLRowData[] | undefined => {
  if (row.values.length !== 1 || !row.values[0].stringValue) {
    return;
  }
  const parsedRows = JSON.parse(row.values[0].stringValue) as NoSQLRowData[];
  return parsedRows;
};

const getCosmosDBRows = (row: QueryRow): NoSQLRowData[] | undefined => {
  if (row.values.length !== 1 || !row.values[0].stringValue) {
    return;
  }
  const parsedRow = JSON.parse(row.values[0].stringValue) as CosmosDBRawData;
  const results: NoSQLRowData[] = [];

  for (const [key, value] of Object.entries(parsedRow)) {
    results.push({
      Key: key,
      Value: value,
    });
  }
  return results;
};
