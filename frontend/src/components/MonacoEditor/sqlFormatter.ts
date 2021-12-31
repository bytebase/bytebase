import { format, FormatOptions } from "sql-formatter";

import { SqlDialect } from "../../types";

type FormatResult = {
  data: string;
  error: Error | null;
};

const formatSQL = (sql: string, dialect: SqlDialect): FormatResult => {
  const options: FormatOptions = {
    language: dialect,
  };
  try {
    const formatted = format(sql, options);
    return { data: formatted, error: null };
  } catch (error) {
    return { data: "", error: error as Error };
  }
};

export default formatSQL;
