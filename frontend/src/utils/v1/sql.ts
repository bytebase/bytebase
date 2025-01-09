import dayjs from "dayjs";
import Long from "long";
import { getDateForPbTimestamp } from "@/types";
import { NullValue } from "@/types/proto/google/protobuf/struct";
import type { Timestamp } from "@/types/proto/google/protobuf/timestamp";
import { Engine } from "@/types/proto/v1/common";
import { RowValue } from "@/types/proto/v1/sql_service";
import { isNullOrUndefined } from "../util";

// extractSQLRowValuePlain extracts a plain value from a RowValue.
export const extractSQLRowValuePlain = (value: RowValue | undefined) => {
  if (
    typeof value === "undefined" ||
    value.nullValue === NullValue.NULL_VALUE
  ) {
    return null;
  }

  const plainObject = RowValue.toJSON(value) as Record<string, any>;
  const keys = Object.keys(plainObject);
  if (keys.length === 0) {
    return undefined; // Will bi displayed as "UNSET"
  }
  if (keys.length > 1) {
    console.debug("mixed type in row value", value);
  }
  if (value.bytesValue) {
    const byteArray = Array.from(value.bytesValue);
    const binaryString = byteArray
      .map((byte) => byte.toString(2).padStart(8, "0"))
      .join("")
      .replace(/^0+/g, "");
    return binaryString.length === 0 ? "0" : binaryString;
  }
  if (value.timestampValue) {
    return formatTimestamp(value.timestampValue);
  }
  if (value.timestampTzValue && value.timestampTzValue.timestamp) {
    return formatTimestampWithTz(value.timestampTzValue.timestamp);
  }
  const key = keys[0];
  return plainObject[key];
};

const formatTimestamp = (timestamp: Timestamp) => {
  const fullDayjs = dayjs(getDateForPbTimestamp(timestamp)).utc();
  const microseconds = Math.floor(timestamp.nanos / 1000);
  const formattedTimestamp =
    microseconds > 0
      ? `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}.${microseconds.toString().padStart(6, "0")}`
      : `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}`;
  return formattedTimestamp;
};

const formatTimestampWithTz = (timestamp: Timestamp) => {
  const fullDayjs = dayjs(getDateForPbTimestamp(timestamp));
  const microseconds = Math.floor(timestamp.nanos / 1000);
  let timezoneOffset = fullDayjs.format("Z");
  if (timezoneOffset.endsWith(":00")) {
    timezoneOffset = timezoneOffset.slice(0, -3);
  }
  const formattedTimestamp =
    microseconds > 0
      ? `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}.${microseconds.toString().padStart(6, "0")}${timezoneOffset}`
      : `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}${timezoneOffset}`;
  return formattedTimestamp;
};

export const wrapSQLIdentifier = (id: string, engine: Engine) => {
  if (engine === Engine.MSSQL) {
    return `[${id}]`;
  }
  if (
    [
      Engine.POSTGRES,
      Engine.SQLITE,
      Engine.SNOWFLAKE,
      Engine.ORACLE,
      Engine.OCEANBASE_ORACLE,
      Engine.REDSHIFT,
      Engine.COCKROACHDB,
    ].includes(engine)
  ) {
    return `"${id}"`;
  }

  return "`" + id + "`";
};

export const generateSchemaAndTableNameInSQL = (
  engine: Engine,
  schema: string,
  tableOrView: string
) => {
  const parts: string[] = [];
  if (schema) {
    parts.push(wrapSQLIdentifier(schema, engine));
  }
  parts.push(wrapSQLIdentifier(tableOrView, engine));
  return parts.join(".");
};

export const generateSimpleSelectAllStatement = (
  engine: Engine,
  schema: string,
  table: string,
  limit: number
) => {
  if (engine === Engine.MONGODB) {
    return `db.${table}.find().limit(${limit});`;
  }
  if (engine === Engine.COSMOSDB) {
    return `SELECT * FROM ${table}`;
  }

  const schemaAndTable = generateSchemaAndTableNameInSQL(engine, schema, table);

  if (engine === Engine.MSSQL) {
    return `SELECT TOP ${limit} * FROM ${schemaAndTable};`;
  }
  if (engine === Engine.ORACLE || engine === Engine.OCEANBASE_ORACLE) {
    return `SELECT * FROM ${schemaAndTable} WHERE ROWNUM <= ${limit};`;
  }

  return `SELECT * FROM ${schemaAndTable} LIMIT ${limit};`;
};

export const generateSimpleInsertStatement = (
  engine: Engine,
  schema: string,
  table: string,
  columns: string[]
) => {
  if (engine === Engine.MONGODB) {
    const kvPairs = columns
      .map((column, i) => `"${column}": <value${i + 1}>`)
      .join(", ");
    return `db.${table}.insert({ ${kvPairs} });`;
  }

  const schemaAndTable = generateSchemaAndTableNameInSQL(engine, schema, table);

  const columnNames = columns
    .map((column) => wrapSQLIdentifier(column, engine))
    .join(", ");
  const placeholders = columns.map((_) => "?").join(", ");

  return `INSERT INTO ${schemaAndTable} (${columnNames}) VALUES (${placeholders});`;
};

export const generateSimpleUpdateStatement = (
  engine: Engine,
  schema: string,
  table: string,
  columns: string[]
) => {
  if (engine === Engine.MONGODB) {
    const kvPairs = columns
      .map((column, i) => `"${column}": <value${i + 1}>`)
      .join(", ");
    return `db.${table}.updateOne({ /* <query> */ }, { $set: { /* ${kvPairs} */} });`;
  }

  const schemaAndTable = generateSchemaAndTableNameInSQL(engine, schema, table);

  const sets = columns
    .map((column) => `${wrapSQLIdentifier(column, engine)}=?`)
    .join(", ");

  return `UPDATE ${schemaAndTable} SET ${sets} WHERE 1=2 /* your condition here */;`;
};

export const generateSimpleDeleteStatement = (
  engine: Engine,
  schema: string,
  table: string
) => {
  if (engine === Engine.MONGODB) {
    return `db.${table}.deleteOne({ /* query */ });`;
  }

  const schemaAndTable = generateSchemaAndTableNameInSQL(engine, schema, table);

  return `DELETE FROM ${schemaAndTable} WHERE 1=2 /* your condition here */;`;
};

export const compareQueryRowValues = (
  type: string,
  a: RowValue,
  b: RowValue
): number => {
  const valueA = extractSQLRowValuePlain(a);
  const valueB = extractSQLRowValuePlain(b);

  // NULL or undefined values go behind
  if (isNullOrUndefined(valueA)) return 1;
  if (isNullOrUndefined(valueB)) return -1;

  // Check if the values are Longs and compare them.
  const rawA = extractSQLRowValueRaw(a);
  const rawB = extractSQLRowValueRaw(b);
  if (Long.isLong(rawA) && Long.isLong(rawB)) {
    return rawA.compare(rawB);
  }

  if (typeof valueA === "number" && typeof valueB === "number") {
    return valueA - valueB;
  }

  if (type === "INT" || type === "INTEGER") {
    const intA = toInt(valueA);
    const intB = toInt(valueB);
    return intA - intB;
  }
  if (type === "FLOAT") {
    const floatA = toFloat(valueA);
    const floatB = toFloat(valueB);
    return floatA - floatB;
  }
  if (type === "DATE" || type === "DATETIME") {
    const dateA = dayjs(valueA);
    const dateB = dayjs(valueB);
    return dateA.isBefore(dateB) ? -1 : dateA.isAfter(dateB) ? 1 : 0;
  }
  const stringA = String(valueA);
  const stringB = String(valueB);
  return stringA < stringB ? -1 : stringA > stringB ? 1 : 0;
};

// extractSQLRowValueRaw extracts a raw value from a RowValue.
const extractSQLRowValueRaw = (value: RowValue | undefined) => {
  if (
    typeof value === "undefined" ||
    value.nullValue === NullValue.NULL_VALUE
  ) {
    return null;
  }
  const keys = Object.keys(RowValue.toJSON(value) as Record<string, any>);
  if (keys.length === 0) {
    return undefined;
  }
  return (value as any)[keys[0]];
};

const toInt = (a: any) => {
  return typeof a === "number"
    ? a
    : typeof a === "string"
      ? parseInt(a, 10)
      : Number(a);
};

const toFloat = (a: any) => {
  return typeof a === "number"
    ? a
    : typeof a === "string"
      ? parseFloat(a)
      : Number(a);
};
