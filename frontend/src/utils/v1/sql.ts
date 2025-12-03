import { toJson, toJsonString } from "@bufbuild/protobuf";
import { ValueSchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { getDateForPbTimestampProtoEs } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type {
  RowValue,
  RowValue_Timestamp,
  RowValue_TimestampTZ,
} from "@/types/proto-es/v1/sql_service_pb";
import { RowValueSchema } from "@/types/proto-es/v1/sql_service_pb";
import { isNullOrUndefined } from "../util";

// extractSQLRowValuePlain extracts a plain value from a RowValue.
export const extractSQLRowValuePlain = (value: RowValue | undefined) => {
  if (typeof value === "undefined" || value.kind?.case === "nullValue") {
    return null;
  }

  const plainObject = toJson(RowValueSchema, value);
  if (plainObject === null) {
    return undefined;
  }
  const keys = Object.keys(plainObject);
  if (keys.length === 0) {
    return undefined; // Will be displayed as "UNSET"
  }
  if (keys.length > 1) {
    console.debug("mixed type in row value", value);
  }

  // First check if there's a formatted stringValue which should take precedence
  if (value.kind?.case === "stringValue") {
    let stringValue = value.kind.value;
    if (stringValue.startsWith('"') && stringValue.endsWith('"')) {
      stringValue = stringValue.replace(/^"|"$/g, "");
    }
    return stringValue;
  }

  // Handle binary data with auto-format detection
  if (value.kind?.case === "bytesValue") {
    // Ensure bytesValue exists before converting to array
    const byteArray = Array.from(value.kind.value);

    // For single byte/bit values (could be boolean)
    if (byteArray.length === 1) {
      // If it's 0 or 1, display as boolean
      if (byteArray[0] === 0 || byteArray[0] === 1) {
        return byteArray[0] === 1 ? "true" : "false";
      }
    }

    // Check if it's readable text
    const isReadableText = byteArray.every(
      (byte: number) => byte >= 32 && byte <= 126
    );
    if (isReadableText) {
      try {
        return new TextDecoder().decode(new Uint8Array(byteArray));
      } catch {
        // If text decoding fails, fallback to hex
      }
    }

    // The column type isn't available in this context
    // Default to HEX format for most binary data as it's more compact
    return (
      "0x" +
      byteArray
        .map((byte: number) => byte.toString(16).toUpperCase().padStart(2, "0"))
        .join("")
    );
  }

  if (
    value.kind?.case === "timestampValue" &&
    value.kind.value.googleTimestamp
  ) {
    return formatTimestamp(value.kind.value);
  }
  if (
    value.kind?.case === "timestampTzValue" &&
    value.kind.value.googleTimestamp
  ) {
    return formatTimestampWithTz(value.kind.value);
  }
  if (value.kind?.case === "valueValue") {
    return toJsonString(ValueSchema, value.kind.value);
  }

  return Object.values(plainObject)[0];
};

const formatTimestamp = (timestamp: RowValue_Timestamp) => {
  const fullDayjs = dayjs(
    getDateForPbTimestampProtoEs(timestamp.googleTimestamp)
  ).utc();
  const microseconds = Math.floor(
    (timestamp.googleTimestamp?.nanos ?? 0) /
      Math.pow(10, 9 - timestamp.accuracy)
  );
  const formattedTimestamp =
    microseconds > 0
      ? `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}.${microseconds.toString().padStart(timestamp.accuracy, "0")}`
      : `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}`;
  return formattedTimestamp;
};

const formatTimestampWithTz = (timestampTzValue: RowValue_TimestampTZ) => {
  const fullDayjs = dayjs(
    getDateForPbTimestampProtoEs(timestampTzValue.googleTimestamp)
  )
    .utc()
    .add(timestampTzValue.offset, "seconds");

  const hourOffset = Math.floor(timestampTzValue.offset / 60 / 60);
  const timezoneOffsetPrefix = Math.abs(hourOffset) < 10 ? "0" : "";
  const timezoneOffset =
    hourOffset > 0
      ? `+${timezoneOffsetPrefix}${hourOffset}`
      : `-${timezoneOffsetPrefix}${Math.abs(hourOffset)}`;
  timestampTzValue.accuracy =
    timestampTzValue.accuracy === 0 ? 6 : timestampTzValue.accuracy;
  const microseconds = Math.floor(
    (timestampTzValue.googleTimestamp?.nanos ?? 0) /
      Math.pow(10, 9 - timestampTzValue.accuracy)
  );
  const formattedTimestamp =
    microseconds > 0
      ? `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}.${microseconds.toString().padStart(timestampTzValue.accuracy, "0")}${timezoneOffset}`
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
      Engine.REDSHIFT,
      Engine.COCKROACHDB,
      Engine.CASSANDRA,
      Engine.TRINO,
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
  const schemaAndTable = generateSchemaAndTableNameInSQL(engine, schema, table);

  switch (engine) {
    case Engine.MONGODB:
      return `db['${table}'].find().limit(${limit});`;
    case Engine.COSMOSDB:
      return `SELECT * FROM ${table}`;
    case Engine.ELASTICSEARCH:
      return `GET ${table}/_search?size=${limit}
{
	"query": {
		"match_all": {}
	}
}`;
    case Engine.MSSQL:
      return `SELECT TOP ${limit} * FROM ${schemaAndTable};`;
    case Engine.ORACLE:
      return `SELECT * FROM ${schemaAndTable} WHERE ROWNUM <= ${limit};`;
    default:
      return `SELECT * FROM ${schemaAndTable} LIMIT ${limit};`;
  }
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
    return `db['${table}'].insert({ ${kvPairs} });`;
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
    return `db['${table}'].updateOne({ /* <query> */ }, { $set: { /* ${kvPairs} */} });`;
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
    return `db['${table}'].deleteOne({ /* query */ });`;
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

  // Check if the values are bigints and compare them.
  const rawA = extractSQLRowValueRaw(a);
  const rawB = extractSQLRowValueRaw(b);
  if (typeof rawA === "bigint" && typeof rawB === "bigint") {
    return rawA < rawB ? -1 : rawA > rawB ? 1 : 0;
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
  if (typeof value === "undefined" || value.kind?.case === "nullValue") {
    return null;
  }
  const j = toJson(RowValueSchema, value);
  if (j === null) {
    return undefined;
  }
  const keys = Object.keys(j);
  if (keys.length === 0) {
    return undefined;
  }
  return value.kind?.value;
};

const toInt = (a: unknown) => {
  return typeof a === "number"
    ? a
    : typeof a === "string"
      ? parseInt(a, 10)
      : Number(a);
};

const toFloat = (a: unknown) => {
  return typeof a === "number"
    ? a
    : typeof a === "string"
      ? parseFloat(a)
      : Number(a);
};
