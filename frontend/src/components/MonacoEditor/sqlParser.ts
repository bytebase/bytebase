import { isObject, isArray } from "lodash-es";
import { Parser, AST } from "node-sql-parser";

type ParseResult = {
  data: AST | AST[] | null;
  error: Error | null;
};

const DDL_TYPE = ["create", "alter", "drop"];
const DML_TYPE = ["insert", "update", "delete"];
const SELECT = "select";

export const parseSQL = (sql: string): ParseResult => {
  if (sql === "") {
    return { data: [], error: null };
  }

  const parser = new Parser();

  try {
    const ast = parser.astify(sql);
    return { data: ast, error: null };
  } catch (error) {
    return { data: null, error: error as Error };
  }
};

export const isValidStatement = (data: ParseResult["data"]) => {
  // if parser returns the null AST. it's an invalid sql statement
  return data !== null;
};

export const isSelectStatement = (data: ParseResult["data"]) => {
  // if the sql statement is an object, it's a single sql statement
  if (isObject(data) && !isArray(data)) {
    return data.type.toLowerCase() === SELECT;
  }
  // if the sql statement is an array, it's a multiple sql statements
  if (isArray(data)) {
    return data.every((statement) => statement.type.toLowerCase() === SELECT);
  }
};

export const isMultipleStatements = (data: ParseResult["data"]) => {
  // if the sql statement is an array and it's more than one statement
  return isArray(data) && data.length > 1;
};

export const isDDLStatement = (
  data: ParseResult["data"],
  method: "every" | "some" = "every"
) => {
  // if the sql statement is an object, it's a single sql statement
  if (isObject(data) && !isArray(data)) {
    return DDL_TYPE.includes(data.type.toLowerCase());
  }
  // if the sql statement is an array, it's a multiple sql statements
  if (isArray(data)) {
    return data[method]((statement) =>
      DDL_TYPE.includes(statement.type.toLowerCase())
    );
  }
};

export const isDMLStatement = (
  data: ParseResult["data"],
  method: "every" | "some" = "every"
) => {
  // if the sql statement is an object, it's a single sql statement
  if (isObject(data) && !isArray(data)) {
    return DML_TYPE.includes(data.type.toLowerCase());
  }
  // if the sql statement is an array, it's a multiple sql statements
  if (isArray(data)) {
    return data[method]((statement) =>
      DML_TYPE.includes(statement.type.toLowerCase())
    );
  }
};

export const transformSQL = (data: AST | AST[], database = "MySQL") => {
  const parser = new Parser();
  return parser.sqlify(data, { database });
};

export default parseSQL;
