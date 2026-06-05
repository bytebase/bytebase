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
  // Use a local rather than writing back to `timestampTzValue.accuracy`:
  // the RowValue comes from the immer-frozen Zustand result set, so an
  // in-place assignment throws "Cannot assign to read only property".
  const accuracy =
    timestampTzValue.accuracy === 0 ? 6 : timestampTzValue.accuracy;
  const microseconds = Math.floor(
    (timestampTzValue.googleTimestamp?.nanos ?? 0) / Math.pow(10, 9 - accuracy)
  );
  const formattedTimestamp =
    microseconds > 0
      ? `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}.${microseconds.toString().padStart(accuracy, "0")}${timezoneOffset}`
      : `${fullDayjs.format("YYYY-MM-DD HH:mm:ss")}${timezoneOffset}`;
  return formattedTimestamp;
};

const escapeMongoDBCollectionName = (name: string) => {
  // The backend wraps --eval in single quotes for shell execution and escapes
  // single quotes with '"'"'. To avoid conflicts, we:
  // 1. Use double quotes for the collection name string
  // 2. Escape backslashes and double quotes for JavaScript
  // 3. Replace single quotes with Unicode escape \u0027 so the shell doesn't see them
  return name
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/'/g, "\\u0027");
};

const wrapSQLIdentifier = (id: string, engine: Engine) => {
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

const generateSchemaAndTableNameInSQL = (
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
      return `db["${escapeMongoDBCollectionName(table)}"].find().limit(${limit});`;
    case Engine.COSMOSDB:
      return `SELECT * FROM c`;
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
    return `db["${escapeMongoDBCollectionName(table)}"].insert({ ${kvPairs} });`;
  }

  const schemaAndTable = generateSchemaAndTableNameInSQL(engine, schema, table);

  const columnNames = columns
    .map((column) => wrapSQLIdentifier(column, engine))
    .join(", ");
  const placeholders = columns.map((_) => "?").join(", ");

  return `INSERT INTO ${schemaAndTable} (${columnNames}) VALUES (${placeholders});`;
};

// MySQL-family engines use backtick identifiers, treat backslash as an escape
// character inside string literals, and render booleans as 1/0.
const MYSQL_FAMILY_ENGINES = new Set<Engine>([
  Engine.MYSQL,
  Engine.MARIADB,
  Engine.TIDB,
  Engine.OCEANBASE,
  Engine.CLICKHOUSE,
  Engine.STARROCKS,
  Engine.DORIS,
]);

const isMySQLFamilyEngine = (engine: Engine): boolean =>
  MYSQL_FAMILY_ENGINES.has(engine);

const escapeSQLStringLiteral = (value: string, engine: Engine): string => {
  let escaped = value;
  // Escape backslashes first so the quote-doubling below isn't affected.
  if (isMySQLFamilyEngine(engine)) {
    escaped = escaped.replaceAll("\\", "\\\\");
  }
  escaped = escaped.replaceAll("'", "''");
  return `'${escaped}'`;
};

const bytesToHex = (bytes: Uint8Array): string =>
  Array.from(bytes)
    .map((b) => b.toString(16).toUpperCase().padStart(2, "0"))
    .join("");

// rowValueToSQLLiteral renders a RowValue as an engine-aware SQL literal,
// suitable for inlining into an INSERT statement. It is type-aware: numbers
// stay bare, strings/JSON/timestamps are single-quoted and escaped, booleans
// follow the engine convention, and NULL is emitted for null/unset cells.
export const rowValueToSQLLiteral = (
  value: RowValue | undefined,
  engine: Engine
): string => {
  if (value === undefined || value.kind?.case === undefined) {
    return "NULL";
  }
  const { kind } = value;
  switch (kind.case) {
    case "nullValue":
      return "NULL";
    case "boolValue":
      if (isMySQLFamilyEngine(engine)) {
        return kind.value ? "1" : "0";
      }
      return kind.value ? "TRUE" : "FALSE";
    case "int32Value":
    case "uint32Value":
    case "doubleValue":
    case "floatValue":
      return String(kind.value);
    case "int64Value":
    case "uint64Value":
      return kind.value.toString();
    case "stringValue":
      return escapeSQLStringLiteral(kind.value, engine);
    case "bytesValue": {
      const hex = bytesToHex(kind.value);
      // Postgres-family bytea literal vs MySQL/MSSQL 0x literal.
      if (
        engine === Engine.POSTGRES ||
        engine === Engine.COCKROACHDB ||
        engine === Engine.REDSHIFT
      ) {
        return `'\\x${hex}'`;
      }
      return `0x${hex}`;
    }
    default: {
      // valueValue (JSON / composite), timestamps, and anything else fall
      // back to their plain display form, single-quoted.
      const plain = extractSQLRowValuePlain(value);
      if (plain === null || plain === undefined) {
        return "NULL";
      }
      return escapeSQLStringLiteral(String(plain), engine);
    }
  }
};

// generateInsertStatementFromRows builds a single batched INSERT statement for
// the given rows. Each `rows[i]` is the cell array of one result row, aligned
// with `columns`.
export const generateInsertStatementFromRows = (params: {
  engine: Engine;
  schema: string | undefined;
  table: string;
  columns: string[];
  rows: RowValue[][];
}): string => {
  const { engine, schema, table, columns, rows } = params;
  const schemaAndTable = generateSchemaAndTableNameInSQL(
    engine,
    schema ?? "",
    table
  );
  const columnNames = columns
    .map((column) => wrapSQLIdentifier(column, engine))
    .join(", ");
  const valueLines = rows
    .map(
      (row) =>
        `  (${row.map((cell) => rowValueToSQLLiteral(cell, engine)).join(", ")})`
    )
    .join(",\n");
  return `INSERT INTO ${schemaAndTable} (${columnNames}) VALUES\n${valueLines};`;
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
    return `db["${escapeMongoDBCollectionName(table)}"].updateOne({ /* <query> */ }, { $set: { /* ${kvPairs} */} });`;
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
    return `db["${escapeMongoDBCollectionName(table)}"].deleteOne({ /* query */ });`;
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
