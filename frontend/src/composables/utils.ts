import { create as createProto } from "@bufbuild/protobuf";
import { NullValue } from "@bufbuild/protobuf/wkt";
import { orderBy } from "lodash-es";
import { stringify } from "uuid";
import type { SQLResultSetV1 } from "@/types";
import type { QueryRow, RowValue } from "@/types/proto-es/v1/sql_service_pb";
import {
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";

type NoSQLRowData = {
  key: string;
  value: unknown;
};

const base64ToArrayBuffer = (base64: string): ArrayBuffer => {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  return bytes.buffer;
};

const decodeBase64ToUUID = (base64Encoded: string): string => {
  const uint8Array = new Uint8Array(base64ToArrayBuffer(base64Encoded));
  return stringify(uint8Array);
};

const flattenNoSQLColumn = (value: unknown): unknown => {
  if (typeof value !== "object") {
    return value;
  }
  if (value === null) {
    return value;
  }
  if (Array.isArray(value)) {
    return value.map(flattenNoSQLColumn);
  }

  const dict = value as { [key: string]: unknown };
  if (Object.keys(dict).length === 1 && Object.keys(dict)[0].startsWith("$")) {
    // Used by the MongoDB response.
    // https://www.mongodb.com/zh-cn/docs/manual/reference/mongodb-extended-json/#bson-data-types-and-associated-representations
    const key = Object.keys(dict)[0];
    switch (key) {
      case "$oid":
        return dict[key];
      case "$date":
        if (typeof dict[key] !== "object" || dict[key] === null) {
          return dict[key];
        }
        const dateObj = dict[key] as Record<string, unknown>;
        if (!dateObj["$numberLong"]) {
          return dict[key];
        }
        return new Date(parseInt(dateObj["$numberLong"] as string));
      case "$numberLong":
        return parseInt(dict[key] as string);
      case "$numberDouble":
        return parseFloat(dict[key] as string);
      case "$numberInt":
        return parseInt(dict[key] as string);
      case "$numberDecimal":
        return Number(dict[key]);
      case "$timestamp":
        return (dict[key] as { t: number; i: number }).t;
      case "$binary": {
        // https://www.mongodb.com/zh-cn/docs/manual/reference/bson-types/#binary-data
        const { base64, subType } = dict[key] as {
          base64: string;
          subType: string;
        };
        switch (subType) {
          case "03":
          case "04":
            try {
              return decodeBase64ToUUID(base64);
            } catch {
              return dict[key];
            }
          default:
            return dict[key];
        }
      }
      default:
        return dict[key];
    }
  }

  return Object.keys(dict).reduce<Record<string, unknown>>((d, key) => {
    d[key] = flattenNoSQLColumn(dict[key]);
    return d;
  }, {});
};

const convertAnyToRowValue = (
  value: unknown,
  nested: boolean
): { value: RowValue; type: string } => {
  switch (typeof value) {
    case "number": {
      if (Math.floor(value) === value) {
        return {
          value: createProto(RowValueSchema, {
            kind: {
              case: "int32Value",
              value: value,
            },
          }),
          type: "INTEGER",
        };
      }
      return {
        value: createProto(RowValueSchema, {
          kind: {
            case: "floatValue",
            value: value,
          },
        }),
        type: "FLOAT",
      };
    }
    case "string":
      return {
        value: createProto(RowValueSchema, {
          kind: {
            case: "stringValue",
            value: value,
          },
        }),
        type: "TEXT",
      };
    case "undefined":
      return {
        value: createProto(RowValueSchema, {
          kind: {
            case: "nullValue",
            value: NullValue.NULL_VALUE,
          },
        }),
        type: "NULL",
      };
    case "boolean":
      return {
        value: createProto(RowValueSchema, {
          kind: {
            case: "boolValue",
            value: value,
          },
        }),
        type: "BOOLEAN",
      };
    case "bigint":
      return {
        value: createProto(RowValueSchema, {
          kind: {
            case: "stringValue",
            value: value.toString(),
          },
        }),
        type: "TEXT",
      };
    case "object": {
      if (value === null) {
        return {
          value: createProto(RowValueSchema, {
            kind: {
              case: "nullValue",
              value: NullValue.NULL_VALUE,
            },
          }),
          type: "NULL",
        };
      }
      if (Array.isArray(value)) {
        return {
          value: createProto(RowValueSchema, {
            kind: {
              case: "stringValue",
              value: JSON.stringify(value.map(flattenNoSQLColumn)),
            },
          }),
          type: "OBJECT",
        };
      }
      if (value instanceof Date) {
        return {
          value: createProto(RowValueSchema, {
            kind: {
              case: "timestampValue",
              value: {
                googleTimestamp: {
                  seconds: BigInt(Math.floor(value.getTime() / 1000)),
                  nanos: (value.getTime() % 1000) * 1e6,
                },
                accuracy: 6,
              },
            },
          }),
          type: "DATETIME",
        };
      }

      if (nested) {
        const formatted = flattenNoSQLColumn(value);
        return convertAnyToRowValue(formatted, !nested);
      } else {
        return {
          value: createProto(RowValueSchema, {
            kind: {
              case: "stringValue",
              value: JSON.stringify(value),
            },
          }),
          type: "TEXT",
        };
      }
    }
    default:
      return {
        value: createProto(RowValueSchema, {
          kind: {
            case: "stringValue",
            value: JSON.stringify(value),
          },
        }),
        type: "TEXT",
      };
  }
};

export const flattenNoSQLResult = (resultSet: SQLResultSetV1) => {
  for (const result of resultSet.results) {
    const { columns, columnIndexMap } = getNoSQLColumns(result.rows);

    const rows: QueryRow[] = [];
    const columnTypeNames: string[] = Array.from({
      length: columns.length,
    }).map((_) => "TEXT");

    for (const row of result.rows) {
      if (
        row.values.length !== 1 ||
        row.values[0].kind.case !== "stringValue"
      ) {
        continue;
      }
      const data = JSON.parse(row.values[0].kind.value);
      const values: RowValue[] = Array.from({ length: columns.length }).map(
        (_) =>
          createProto(RowValueSchema, {
            kind: {
              case: "nullValue",
              value: NullValue.NULL_VALUE,
            },
          })
      );

      for (const [key, value] of Object.entries(data)) {
        const index = columnIndexMap.get(key) ?? 0;

        const { value: formatted, type } = convertAnyToRowValue(value, true);
        values[index] = formatted;
        columnTypeNames[index] = type;
      }

      rows.push(
        createProto(QueryRowSchema, {
          values: values,
        })
      );
    }

    result.rows = rows;
    result.columnNames = columns;
    result.columnTypeNames = columnTypeNames;
  }
};

const getNoSQLColumns = (rows: QueryRow[]) => {
  const columnSet = new Set<string>();
  const columnIndexMap = new Map<string, number>();

  for (const row of rows) {
    const parsedRows = getNoSQLRows(row);
    if (!parsedRows) {
      continue;
    }
    for (const item of parsedRows) {
      columnSet.add(item.key);
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
      (column) => (column === "id" || column === "_id" ? -1 : 1),
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

const getNoSQLRows = (row: QueryRow): NoSQLRowData[] | undefined => {
  if (row.values.length !== 1 || row.values[0].kind.case !== "stringValue") {
    return;
  }
  const parsedRow = JSON.parse(row.values[0].kind.value) as {
    [key: string]: Record<string, unknown>;
  };
  const results: NoSQLRowData[] = [];

  for (const [key, value] of Object.entries(parsedRow)) {
    results.push({
      key: key,
      value: value,
    });
  }
  return results;
};
