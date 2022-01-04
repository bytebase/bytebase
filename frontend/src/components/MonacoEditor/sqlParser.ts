import { Parser, AST } from "node-sql-parser";

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

export const isSelectStatement = (sql: string) => {
  const { data } = parseSQL(sql);
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  // @ts-ignore
  // if parser returns the null AST. it's an invalid sql statement
  return data?.type.toLowerCase() === "select" || data === null;
};

export default parseSQL;
