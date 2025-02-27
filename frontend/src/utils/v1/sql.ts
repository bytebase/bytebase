import dayjs from "dayjs";
import Long from "long";
import { getDateForPbTimestamp } from "@/types";
import { NullValue } from "@/types/proto/google/protobuf/struct";
import { Engine } from "@/types/proto/v1/common";
import {
  RowValue,
  type RowValue_Timestamp,
  type RowValue_TimestampTZ,
} from "@/types/proto/v1/sql_service";
import { isNullOrUndefined } from "../util";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";

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
    
    // We only need special handling for PostgreSQL BIT/VARBIT types
    // Get database connection information, but handle case where it's not available
    try {
      // Check if we're in a context with access to the engine
      const { database } = useConnectionOfCurrentSQLEditorTab();
      const engine = database.value?.instanceResource?.engine;
      
      // For PostgreSQL, BIT/VARBIT values are represented as ASCII '0's and '1's
      if (engine === Engine.POSTGRES) {
        // We need to check if this looks like a bit string (all bytes are ASCII '0' or '1')
        // AND has a reasonable length for a bit representation (not thousands of characters)
        const isPgBitString = 
          byteArray.length < 1000 && // reasonable size check
          byteArray.every(byte => byte === 48 || byte === 49); // only '0' and '1' chars
        
        if (isPgBitString) {
          // Convert from ASCII codes to actual string
          return String.fromCharCode(...byteArray);
        }
      }
    } catch (e) {
      // If we can't access the connection info, fall back to default behavior
      console.debug("Could not determine database engine for bytes handling", e);
    }
    
    // Display binary data as hexadecimal with 0x prefix for better readability
    // Most database UIs display binary data in hex format with uppercase letters
    const hexString = byteArray
      .map((byte) => byte.toString(16).toUpperCase().padStart(2, "0"))
      .join("");
    return hexString.length === 0 ? "0x00" : `0x${hexString}`;
  }
  if (value.timestampValue && value.timestampValue.googleTimestamp) {
    return formatTimestamp(value.timestampValue);
  }
  if (value.timestampTzValue && value.timestampTzValue.googleTimestamp) {
    return formatTimestampWithTz(value.timestampTzValue);
  }
  const key = keys[0];
  return plainObject[key];
};

const formatTimestamp = (timestamp: RowValue_Timestamp) => {
  const fullDayjs = dayjs(getDateForPbTimestamp(timestamp.googleTimestamp)).utc();
  const microseconds = Math.floor((timestamp.googleTimestamp?.nanos ?? 0) / Math.pow(10, 9 - timestamp.accuracy));
  const formattedTimestamp =
    microseconds > 0
      ? `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}.${microseconds.toString().padStart(timestamp.accuracy, "0")}`
      : `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}`;
  return formattedTimestamp;
};

const formatTimestampWithTz = (timestampTzValue: RowValue_TimestampTZ) => {
  const fullDayjs = dayjs(getDateForPbTimestamp(timestampTzValue.googleTimestamp))
    .utc()
    .add(timestampTzValue.offset, "seconds");

  const hourOffset = Math.floor(timestampTzValue.offset / 60 / 60);
  const timezoneOffsetPrefix = Math.abs(hourOffset) < 10 ? "0" : "";
  const timezoneOffset =
    hourOffset > 0
      ? `+${timezoneOffsetPrefix}${hourOffset}`
      : `-${timezoneOffsetPrefix}${Math.abs(hourOffset)}`;
  timestampTzValue.accuracy = (timestampTzValue.accuracy === 0) ? 6 : timestampTzValue.accuracy;
  const microseconds = Math.floor(
    (timestampTzValue.googleTimestamp?.nanos ?? 0) / Math.pow(10, 9 - timestampTzValue.accuracy) 
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
  const schemaAndTable = generateSchemaAndTableNameInSQL(engine, schema, table);

  switch (engine) {
    case Engine.MONGODB:
      return `db.${table}.find().limit(${limit});`;
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
    case Engine.OCEANBASE_ORACLE:
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
