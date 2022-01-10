import { Parser, AST } from "node-sql-parser";
import { isObject, isArray } from "lodash-es";

type ParseResult = {
  data: AST | AST[] | null;
  error: Error | null;
};

const parseSQL = (sql: string): ParseResult => {
  const parser = new Parser();

  try {
    const ast = parser.astify(sql);
    return { data: ast, error: null };
  } catch (error) {
    return { data: null, error: error as Error };
  }
};

export const isValidStatement = (sql: string) => {
  const { data } = parseSQL(sql);
  return data !== null;
};

export const isSelectStatement = (sql: string) => {
  const { data } = parseSQL(sql);

  // if parser returns the null AST. it's an invalid sql statement
  if (data === null) return true;
  // if the sql statement is an object, it's a single sql statement
  if (isObject(data) && !isArray(data)) {
    return data.type.toLowerCase() === "select";
  }
  // if the sql statement is an array, it's a multiple sql statements
  if (isArray(data)) {
    return data.every((statement) => statement.type.toLowerCase() === "select");
  }
};

export default parseSQL;
