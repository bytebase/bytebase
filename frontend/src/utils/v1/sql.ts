import dayjs from "dayjs";
import { NullValue } from "@/types/proto/google/protobuf/struct";
import { Engine } from "@/types/proto/v1/common";
import { RowValue } from "@/types/proto/v1/sql_service";

export const extractSQLRowValue = (value: RowValue | undefined) => {
  if (typeof value === "undefined") {
    return null;
  }
  if (value.nullValue === NullValue.NULL_VALUE) {
    return null;
  }

  const plainObject = RowValue.toJSON(value) as Record<string, any>;
  const keys = Object.keys(plainObject);
  if (keys.length === 0) {
    console.debug("empty row value", value);
    return null;
  }
  if (keys.length > 1) {
    console.debug("mixed type in row value", value);
  }
  if (value.bytesValue) {
    // convert byte arrays to binary 10101001 strings
    const byteArray = value.bytesValue;
    const parts: string[] = [];
    for (let i = 0; i < byteArray.length; i++) {
      const byte = byteArray[i];
      const part = byte.toString(2).padStart(8, "0");
      parts.push(part);
    }
    const binaryString = parts.join("").replace(/^0+/g, "");
    if (binaryString.length === 0) {
      return "0";
    }
    return binaryString;
  }
  const key = keys[0];
  return plainObject[key];
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

  const schemaAndTable = generateSchemaAndTableNameInSQL(engine, schema, table);

  if (engine === Engine.MSSQL) {
    return `SELECT TOP ${limit} * FROM ${schemaAndTable};`;
  }
  if (engine === Engine.ORACLE || engine === Engine.OCEANBASE_ORACLE) {
    return `SELECT * FROM ${schemaAndTable} WHERE ROWNUM <= ${limit};`;
  }

  return `SELECT * FROM ${schemaAndTable} LIMIT ${limit};`;
};

export const compareQueryRowValues = (
  type: string,
  a: RowValue,
  b: RowValue
): number => {
  const valueA = extractSQLRowValue(a);
  const valueB = extractSQLRowValue(b);
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
